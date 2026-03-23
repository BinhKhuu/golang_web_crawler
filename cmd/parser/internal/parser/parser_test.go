package parser

import (
	"errors"
	"golangwebcrawler/cmd/parser/models"
	"testing"
)

type UnsupportedParser struct{}

func Test_NewParser_JobListing(t *testing.T) {
	p, err := NewParser[models.JobListing]()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, ok := any(p).(Parser[models.JobListing]); ok {
		// Successfully casted to Parser[models.JobListing]
	} else {
		t.Fatalf("expected type Parser[models.JobListing], got %T", p)
	}
}

func Test_NewParser_Unsupported(t *testing.T) {
	_, err := NewParser[UnsupportedParser]()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expectedErr := ErrUnsupportedParserType
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}
