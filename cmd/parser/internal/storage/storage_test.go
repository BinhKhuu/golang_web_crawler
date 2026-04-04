package storage

import (
	"golangwebcrawler/internal/models"
	"golangwebcrawler/internal/typeutil"
	"strings"
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

func Test_StoreExtractedJobDataBatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewDBStorageService(db)
	jobCards := []models.ExtractedJobData{
		{
			Title:    "Software Engineer",
			Company:  "Example Inc",
			Location: "Remote",
			Salary:   "$100k - $150k",
			Link:     "https://www.seek.com.au/job1",
			Skills:   []string{"Go", "Docker"},
		},
		{
			Title:    "Backend Developer",
			Company:  "Tech Corp",
			Location: "New York, NY",
			Salary:   "$120k - $170k",
			Link:     "https://www.seek.com.au/job2",
			Skills:   []string{"Python", "AWS"},
		},
	}
	mock.ExpectBegin()

	mock.ExpectPrepare(`COPY "extracted_jobdata" \("title", "company", "location", "salary", "description", "link", "skills"\) FROM STDIN`)

	for _, jobCard := range jobCards {
		expectedSkills := strings.Join(jobCard.Skills, ",")
		mock.ExpectExec(`COPY "extracted_jobdata"`).
			WithArgs(jobCard.Title, jobCard.Company, jobCard.Location, jobCard.Salary, jobCard.Description, jobCard.Link, expectedSkills).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()

	if err := storageService.StoreExtractedJobDataBatch(jobCards); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func Test_StoreJobListingData(t *testing.T) {
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

	if err := storageService.StoreJobListingData(jobListing); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
