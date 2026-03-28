package storage

import (
	"context"
	"database/sql"
	"golangwebcrawler/internal/dbstore"
	"golangwebcrawler/internal/models"
	"time"

	"github.com/lib/pq"
)

type ParserStorageService struct {
	db *sql.DB
}

// NewDBStorageService Todo return error.
func NewDBStorageService(db *sql.DB) *ParserStorageService {
	return &ParserStorageService{db: db}
}

func (s *ParserStorageService) GetLatestRawData(startDate time.Time) ([]models.RawData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbstore.QueryTimeout)
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
		if err := rows.Scan(&r.URL, &r.ContentType, &r.Raw_content); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func (s *ParserStorageService) StoreJobListingData(result models.JobListing) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbstore.QueryTimeout)
	defer cancel()

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO job_listings (title, company, location, remote_flag, salary_min, salary_max, currency, description_html, description_text, posted_date, expires_at, source, source_id, url, tags, raw_json)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		result.Title, result.Company, result.Location, result.RemoteFlag, result.SalaryMin, result.SalaryMax, result.Currency, result.DescriptionHTML, result.DescriptionText, result.PostedDate, result.ExpiresAt, result.Source, result.SourceID, result.URL, pq.Array(result.Tags), result.RawJSON,
	)
	return err
}

func (s *ParserStorageService) StoreJobCardData(result models.JobCard) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbstore.QueryTimeout)
	defer cancel()

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO job_cards (title, company, location, salary, description, url, link, classification, update_date, scrape_date, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())`,
		result.Title, result.Company, result.Location, result.Salary, result.Description, result.URL, result.Link, result.Classification, result.UpdateDate, result.ScrapeDate,
	)

	return err
}
