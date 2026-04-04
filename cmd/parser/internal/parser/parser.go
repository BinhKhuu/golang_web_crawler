package parser

import (
	"database/sql"
	"errors"
	"fmt"
	"golangwebcrawler/internal/llm"
	"golangwebcrawler/internal/models"
)

type Parser[T any] interface {
	ParseLLM(html string) ([]models.ExtractedJobData, error)
	ParseQuery(html string) ([]models.ExtractedJobData, error)
}

var (
	ErrUnsupportedParserType = errors.New("unsupported parser type")
	ErrCastingParserType     = errors.New("failed to cast parser to requested type")
)

func NewParser[T any](db *sql.DB) (Parser[T], error) {
	var zero T
	llmService, err := llm.NewLLMService()
	if err != nil {
		return nil, err
	}
	switch any(zero).(type) {
	case models.JobListing:
		p := NewJobListingParser(llmService)
		if typed, ok := any(p).(Parser[T]); ok {
			return typed, nil
		}
		return nil, fmt.Errorf("NewJobListingParser type %T: %w", zero, ErrCastingParserType)
	default:
		return nil, fmt.Errorf("NewParser: type %T: %w", zero, ErrUnsupportedParserType)
	}
}
