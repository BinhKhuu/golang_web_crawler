package storage

import (
	"context"
	"database/sql"
	"errors"
	"golangwebcrawler/internal/dbstore"
	"golangwebcrawler/internal/models"
	"log/slog"
	"strings"
	"time"

	"github.com/lib/pq"
)

type ParserStorageService struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewDBStorageService(db *sql.DB, logger *slog.Logger) *ParserStorageService {
	return &ParserStorageService{db: db, logger: logger}
}

func (s *ParserStorageService) GetLatestRawData(ctx context.Context, startDate time.Time) ([]models.RawData, error) {
	ctx, cancel := context.WithTimeout(ctx, dbstore.QueryTimeout)
	defer cancel()

	rows, err := s.db.QueryContext(ctx,
		`SELECT url, content_type, raw_content
		FROM raw_data
		WHERE fetched_at >= $1`, startDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.RawData
	for rows.Next() {
		var r models.RawData
		if err := rows.Scan(&r.URL, &r.ContentType, &r.RawContent); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func (s *ParserStorageService) StoreJobListingData(ctx context.Context, result models.JobListing) error {
	ctx, cancel := context.WithTimeout(ctx, dbstore.QueryTimeout)
	defer cancel()

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO job_listings (title, company, location, remote_flag, salary_min, salary_max, currency, description_html, description_text, posted_date, expires_at, source, source_id, url, tags, raw_json)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		result.Title, result.Company, result.Location, result.RemoteFlag, result.SalaryMin, result.SalaryMax, result.Currency, result.DescriptionHTML, result.DescriptionText, result.PostedDate, result.ExpiresAt, result.Source, result.SourceID, result.URL, pq.Array(result.Tags), result.RawJSON,
	)
	return err
}

func (s *ParserStorageService) StoreExtractedJobDataBatchUpSert(ctx context.Context, results []models.ExtractedJobData) error {
	if len(results) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, dbstore.QueryTimeout)
	defer cancel()

	txn, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		rollbackErr := txn.Rollback()
		if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			s.logger.Error("Failed to rollback transaction", "error", rollbackErr)
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
		return err
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
			return err
		}
	}

	return txn.Commit()
}

func (s *ParserStorageService) StoreExtractedJobDataBatch(ctx context.Context, results []models.ExtractedJobData) error {
	if len(results) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, dbstore.QueryTimeout)
	defer cancel()

	txn, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		rollbackErr := txn.Rollback()
		if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			s.logger.Error("Failed to rollback transaction", "error", rollbackErr)
		}
	}()

	stmt, err := txn.PrepareContext(ctx, pq.CopyIn("extracted_jobdata",
		"title", "company", "location", "salary", "description", "link", "skills"))
	if err != nil {
		return err
	}

	for _, result := range results {
		skills := strings.Join(result.Skills, ",")
		_, err = stmt.ExecContext(ctx, result.Title, result.Company, result.Location,
			result.Salary, result.Description, result.Link, skills)
		if err != nil {
			return err
		}
	}

	defer func() {
		if err = stmt.Close(); err != nil {
			s.logger.Error("Failed to close statement", "error", err)
			return
		}
	}()

	return txn.Commit()
}
