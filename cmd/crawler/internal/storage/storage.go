package storage

import (
	"context"
	"database/sql"
	"golangwebcrawler/cmd/crawler/internal/config"
	"golangwebcrawler/cmd/crawler/internal/models"
)

type CrawlerStorageService struct {
	db *sql.DB
}

func NewDBStorageService(db *sql.DB) *CrawlerStorageService {
	return &CrawlerStorageService{db: db}
}

func (s *CrawlerStorageService) StoreRawData(result models.RawData) error {
	ctx, cancel := context.WithTimeout(context.Background(), config.QueryTimeout)
	defer cancel()
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO raw_data (url, content_type, raw_content) 
		VALUES ($1, $2, $3)
		ON CONFLICT (url) 
		DO UPDATE SET content_type = $2, raw_content = $3, fetched_at = NOW()`,
		result.URL, result.ContentType, result.Raw_content,
	)
	return err
}
