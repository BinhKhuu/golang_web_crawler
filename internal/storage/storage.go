package storage

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"golangwebcrawler/internal/models"

	"github.com/lib/pq"
)

const (
	queryTimeout      = 5 * time.Second
	batchQueryTimeout = 60 * time.Second
)

type Service struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewService(db *sql.DB, logger *slog.Logger) *Service {
	return &Service{db: db, logger: logger}
}

func (s *Service) StoreRawData(ctx context.Context, url, contentType, rawContent string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO raw_data (url, content_type, raw_content)
		VALUES ($1, $2, $3)
		ON CONFLICT (url)
		DO UPDATE SET content_type = $2, raw_content = $3, fetched_at = NOW()`,
		url, contentType, rawContent,
	)
	if err != nil {
		return fmt.Errorf("storing raw data for %s: %w", url, err)
	}
	return nil
}

func (s *Service) StoreRawDataBatch(ctx context.Context, items []models.RawDataItem) error {
	if len(items) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, batchQueryTimeout)
	defer cancel()

	txn, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	defer func() {
		rollbackErr := txn.Rollback()
		if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) && !errors.Is(rollbackErr, driver.ErrBadConn) {
			s.logger.Error("failed to rollback transaction", "error", rollbackErr)
		}
	}()

	if _, ctxErr := txn.ExecContext(ctx,
		`CREATE TEMP TABLE raw_data_batch (url TEXT, content_type TEXT, raw_content TEXT) ON COMMIT DROP`); ctxErr != nil {
		return fmt.Errorf("creating temp table: %w", ctxErr)
	}

	stmt, err := txn.PrepareContext(ctx, pq.CopyIn("raw_data_batch",
		"url", "content_type", "raw_content"))
	if err != nil {
		return fmt.Errorf("preparing copy statement: %w", err)
	}
	defer func() {
		if closeErr := stmt.Close(); closeErr != nil {
			s.logger.Error("finishing copy:", "error", closeErr)
		}
	}()

	for _, item := range items {
		_, err = stmt.ExecContext(ctx, item.URL, item.ContentType, item.RawContent)
		if err != nil {
			return fmt.Errorf("executing copy for %s: %w", item.URL, err)
		}
	}

	if _, err = stmt.ExecContext(ctx); err != nil {
		return fmt.Errorf("flushing copy statement: %w", err)
	}

	if _, err := txn.ExecContext(ctx,
		`INSERT INTO raw_data (url, content_type, raw_content)
		SELECT url, content_type, raw_content FROM raw_data_batch
		ON CONFLICT (url)
		DO UPDATE SET content_type = EXCLUDED.content_type,
		              raw_content = EXCLUDED.raw_content,
		              fetched_at = NOW()`); err != nil {
		return fmt.Errorf("upserting from temp table: %w", err)
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

func (s *Service) GetLatestRawData(ctx context.Context, startDate time.Time) ([]RawData, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := s.db.QueryContext(ctx,
		`SELECT url, content_type, raw_content
		FROM raw_data
		WHERE fetched_at >= $1`, startDate)
	if err != nil {
		return nil, fmt.Errorf("querying raw data: %w", err)
	}
	defer rows.Close()

	var results []RawData
	for rows.Next() {
		var r RawData
		if err := rows.Scan(&r.URL, &r.ContentType, &r.RawContent); err != nil {
			return nil, fmt.Errorf("scanning raw data row: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating raw data rows: %w", err)
	}

	return results, nil
}

func (s *Service) StoreJobListingData(ctx context.Context, job JobListing) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO job_listings (title, company, location, remote_flag, salary_min, salary_max, currency, description_html, description_text, posted_date, expires_at, source, source_id, url, tags, raw_json)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		job.Title, job.Company, job.Location, job.RemoteFlag, job.SalaryMin, job.SalaryMax, job.Currency, job.DescriptionHTML, job.DescriptionText, job.PostedDate, job.ExpiresAt, job.Source, job.SourceID, job.URL, pq.Array(job.Tags), job.RawJSON,
	)
	if err != nil {
		return fmt.Errorf("storing job listing %s: %w", job.URL, err)
	}
	return nil
}

func (s *Service) StoreExtractedJobDataBatchUpSert(ctx context.Context, results []ExtractedJobData) error {
	if len(results) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, batchQueryTimeout)
	defer cancel()

	txn, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	defer func() {
		rollbackErr := txn.Rollback()
		if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) && !errors.Is(rollbackErr, driver.ErrBadConn) {
			s.logger.Error("failed to rollback transaction", "error", rollbackErr)
		}
	}()

	query := `
		INSERT INTO extracted_jobdata (title, company, location, salary, description, link, skills)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (link)
		DO UPDATE SET
			title = EXCLUDED.title,
			company = EXCLUDED.company,
			location = EXCLUDED.location,
			salary = EXCLUDED.salary,
			description = EXCLUDED.description,
			skills = EXCLUDED.skills,
			updated_at = NOW();
	`

	stmt, err := txn.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("preparing statement: %w", err)
	}
	defer stmt.Close()

	for _, result := range results {
		skills := strings.Join(result.Skills, ",")
		_, err = stmt.ExecContext(ctx,
			result.Title,
			result.Company,
			result.Location,
			result.Salary,
			result.Description,
			result.Link,
			skills,
		)
		if err != nil {
			return fmt.Errorf("executing statement for %s: %w", result.Link, err)
		}
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

func (s *Service) StoreExtractedJobDataBatch(ctx context.Context, results []ExtractedJobData) error {
	if len(results) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, batchQueryTimeout)
	defer cancel()

	txn, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() {
		rollbackErr := txn.Rollback()
		if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) && !errors.Is(rollbackErr, driver.ErrBadConn) {
			s.logger.Error("failed to rollback transaction", "error", rollbackErr)
		}
	}()

	stmt, err := txn.PrepareContext(ctx, pq.CopyIn("extracted_jobdata",
		"title", "company", "location", "salary", "description", "link", "skills"))
	if err != nil {
		return fmt.Errorf("preparing copy statement: %w", err)
	}
	defer stmt.Close()

	for _, result := range results {
		skills := strings.Join(result.Skills, ",")
		_, err = stmt.ExecContext(ctx, result.Title, result.Company, result.Location,
			result.Salary, result.Description, result.Link, skills)
		if err != nil {
			return fmt.Errorf("executing copy for %s: %w", result.Link, err)
		}
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing copy transaction: %w", err)
	}
	return nil
}

type RawData struct {
	URL         string
	ContentType string
	RawContent  string
	FetchedAt   string
}

type JobListing struct {
	ID              int64
	Title           string
	Company         string
	Location        string
	RemoteFlag      bool
	SalaryMin       *float64
	SalaryMax       *float64
	Currency        string
	DescriptionHTML string
	DescriptionText string
	PostedDate      *time.Time
	ExpiresAt       *time.Time
	Source          string
	SourceID        string
	URL             string
	Tags            []string
	RawJSON         []byte
	CrawlTimestamp  time.Time
}

type ExtractedJobData struct {
	Title       string   `json:"job_title"`
	Company     string   `json:"company_name"`
	Location    string   `json:"location"`
	Salary      string   `json:"salary_range"`
	Description string   `json:"description"`
	Skills      []string `json:"required_skills"`
	Link        string   `json:"links"`
}
