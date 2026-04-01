package parser

import (
	"database/sql"
	"errors"
	"fmt"
	"golangwebcrawler/cmd/parser/internal/storage"
	"golangwebcrawler/internal/llm"
	"golangwebcrawler/internal/models"
)

type Parser[T any] interface {
	Parse(html string) (T, error)
}

var (
	ErrUnsupportedParserType = errors.New("unsupported parser type")
	ErrCastingParserType     = errors.New("failed to cast parser to requested type")
)

func NewParser[T any](db *sql.DB) (Parser[T], error) {
	var zero T
	storageService := storage.NewDBStorageService(db) // todo maybe this should be a parameter to newparser
	llmService, err := llm.NewLLMService()
	if err != nil {
		return nil, err
	}
	switch any(zero).(type) {
	case models.JobListing:
		p := NewJobListingParser(storageService, llmService)
		if typed, ok := any(p).(Parser[T]); ok {
			return typed, nil
		}
		return nil, fmt.Errorf("NewJobListingParser type %T: %w", zero, ErrCastingParserType)
	default:
		return nil, fmt.Errorf("NewParser: type %T: %w", zero, ErrUnsupportedParserType)
	}
}
