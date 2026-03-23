package parser

import (
	"errors"
	"fmt"
	"golangwebcrawler/cmd/parser/models"
)

type Parser[T any] interface {
	Parse(html string) (T, error)
}

var (
	ErrUnsupportedParserType = errors.New("unsupported parser type")
	ErrCastingParserType     = errors.New("failed to cast parser to requested type")
)

// Todo Test this, When type does not exist, when type exist but cannot be casted correctly (might be too hard to do) and successful parse
func NewParser[T any]() (Parser[T], error) {
	var zero T

	switch any(zero).(type) {
	case models.JobListing:
		p := NewJobListingParser()
		if typed, ok := any(p).(Parser[T]); ok {
			return typed, nil
		}
		return nil, fmt.Errorf("NewJobListingParser type %T: %w", zero, ErrCastingParserType)
	default:
		return nil, fmt.Errorf("NewParser: type %T: %w", zero, ErrUnsupportedParserType)
	}
}
