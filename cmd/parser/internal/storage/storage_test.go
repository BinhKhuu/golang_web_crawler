package storage

import (
	"golangwebcrawler/internal/models"
	"golangwebcrawler/internal/typeutil"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/lib/pq"
)

func Test_GetLatestRawData_ReturnsRawData(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	defer db.Close()

	storageService := NewDBStorageService(db)
	timeParam := time.Now().Add(-time.Hour)

	rows := sqlmock.NewRows([]string{"url", "content_type", "raw_content"}).
		AddRow("http://example.com", "text/html", "<html></html>")

	mock.ExpectQuery(`SELECT .+ FROM raw_data`).
		WithArgs(timeParam).
		WillReturnRows(rows)

	result, err := storageService.GetLatestRawData(timeParam)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreParseData(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewDBStorageService(db)
	jobListing := models.JobListing{
		Company:         "Example Inc",
		RemoteFlag:      true,
		Location:        "Remote",
		SalaryMin:       typeutil.FloatPtr(750000.50),
		SalaryMax:       typeutil.FloatPtr(150000),
		Currency:        "USD",
		DescriptionHTML: "<p>Job description</p>",
		DescriptionText: "Job description",
		PostedDate:      typeutil.TimePtr(time.Now()),
		ExpiresAt:       typeutil.TimePtr(time.Now().Add(30 * 24 * time.Hour)),
		Source:          "ExampleSource",
		SourceID:        "12345",
		URL:             "http://example.com/job/12345",
		Tags:            []string{"Go", "Remote"},
		RawJSON:         []byte(`{"title": "Software Engineer"}`),
	}

	mock.ExpectExec("INSERT INTO job_listings").
		WithArgs(jobListing.Title, jobListing.Company, jobListing.Location, jobListing.RemoteFlag, jobListing.SalaryMin, jobListing.SalaryMax, jobListing.Currency, jobListing.DescriptionHTML, jobListing.DescriptionText, jobListing.PostedDate, jobListing.ExpiresAt, jobListing.Source, jobListing.SourceID, jobListing.URL, sqlmock.AnyArg(), jobListing.RawJSON).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := storageService.StoreParsedData(jobListing); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
