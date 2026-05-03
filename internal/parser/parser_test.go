package parser

import (
	"database/sql"
	"errors"
	"testing"

	"golangwebcrawler/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
)

type UnsupportedParser struct{}

func Test_NewParser_JobListing(t *testing.T) {
	db, _, dbClose := mockDb()
	defer dbClose()

	p, err := NewParser[models.JobListing](db)
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
	db, _, dbClose := mockDb()
	defer dbClose()
	_, err := NewParser[UnsupportedParser](db)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expectedErr := ErrUnsupportedParserType
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func mockDb() (*sql.DB, *sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		panic("failed to create mock database: " + err.Error())
	}
	return db, &mock, func() { db.Close() }
}
