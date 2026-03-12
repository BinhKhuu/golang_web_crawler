package storage

import (
	"database/sql"
	"golangwebcrawler/cmd/crawler/internal/models"
)

type DBStorageService struct {
	DB *sql.DB
}

func NewDBStorageService(db *sql.DB) *DBStorageService {
	return &DBStorageService{DB: db}
}

func (s *DBStorageService) StoreRawData(result models.RawData) error {
	_, err := s.DB.Exec(
		`INSERT INTO raw_data (url, content_type, raw_content) 
		VALUES ($1, $2, $3)
		ON CONFLICT (url) 
		DO UPDATE SET content_type = $2, raw_content = $3, fetched_at = NOW()`,
		result.URL, result.ContentType, string(result.Raw_content),
	)
	return err
}
