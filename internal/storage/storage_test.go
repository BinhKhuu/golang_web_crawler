package storage

import (
	"context"
	"golangwebcrawler/internal/models"
	"log/slog"
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

	storageService := NewService(db, slog.Default())
	timeParam := time.Now().Add(-time.Hour)

	rows := sqlmock.NewRows([]string{"url", "content_type", "raw_content"}).
		AddRow("http://example.com", "text/html", "<html></html>")

	mock.ExpectQuery(`SELECT .+ FROM raw_data`).
		WithArgs(timeParam).
		WillReturnRows(rows)

	result, err := storageService.GetLatestRawData(context.Background(), timeParam)
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

func Test_StoreRawDataBatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	items := []models.RawDataItem{
		{URL: "http://example1.com", ContentType: "text/html", RawContent: "<html>1</html>"},
		{URL: "http://example2.com", ContentType: "text/html", RawContent: "<html>2</html>"},
	}

	mock.ExpectBegin()
	mock.ExpectExec(`CREATE TEMP TABLE raw_data_batch`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPrepare(`COPY "raw_data_batch" \("url", "content_type", "raw_content"\) FROM STDIN`)

	for _, item := range items {
		mock.ExpectExec(`COPY "raw_data_batch"`).
			WithArgs(item.URL, item.ContentType, item.RawContent).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}

	// pq.CopyIn requires a final Exec() with no args to flush the COPY stream.
	mock.ExpectExec(`COPY "raw_data_batch"`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectExec(`INSERT INTO raw_data \(url, content_type, raw_content\)\s+SELECT url, content_type, raw_content FROM raw_data_batch\s+ON CONFLICT \(url\)\s+DO UPDATE SET content_type = EXCLUDED.content_type,\s+raw_content = EXCLUDED.raw_content,\s+fetched_at = NOW\(\)`).
		WillReturnResult(sqlmock.NewResult(0, int64(len(items))))
	mock.ExpectCommit()

	if err := storageService.StoreRawDataBatch(context.Background(), items); err != nil {
		t.Fatalf("expected no error, got %v", err)
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

	storageService := NewService(db, slog.Default())
	jobCards := mockJobCards()
	mock.ExpectBegin()

	mock.ExpectPrepare(`COPY "extracted_jobdata" \("title", "company", "location", "salary", "description", "link", "skills"\) FROM STDIN`)

	for _, jobCard := range jobCards {
		expectedSkills := strings.Join(jobCard.Skills, ",")
		mock.ExpectExec(`COPY "extracted_jobdata"`).
			WithArgs(jobCard.Title, jobCard.Company, jobCard.Location, jobCard.Salary, jobCard.Description, jobCard.Link, expectedSkills).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()

	if err := storageService.StoreExtractedJobDataBatch(context.Background(), jobCards); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func mockJobCards() []ExtractedJobData {
	return []ExtractedJobData{
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
}

func Test_StoreExtractedJobDataBatchUpSert(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	jobCards := mockJobCards()

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO extracted_jobdata \(title, company, location, salary, description, link, skills\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\) ON CONFLICT \(link\) DO UPDATE SET title = EXCLUDED.title, company = EXCLUDED.company, location = EXCLUDED.location, salary = EXCLUDED.salary, description = EXCLUDED.description, skills = EXCLUDED.skills`)
	for _, jobCard := range jobCards {
		expectedSkills := strings.Join(jobCard.Skills, ",")
		mock.ExpectExec(`INSERT INTO extracted_jobdata`).
			WithArgs(jobCard.Title, jobCard.Company, jobCard.Location, jobCard.Salary, jobCard.Description, jobCard.Link, expectedSkills).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectCommit()
	if err := storageService.StoreExtractedJobDataBatchUpSert(context.Background(), jobCards); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func Test_StoreJobListingData(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	jobListing := getMockJobListing()

	mock.ExpectExec("INSERT INTO job_listings").
		WithArgs(jobListing.Title, jobListing.Company, jobListing.Location, jobListing.RemoteFlag, jobListing.SalaryMin, jobListing.SalaryMax, jobListing.Currency, jobListing.DescriptionHTML, jobListing.DescriptionText, jobListing.PostedDate, jobListing.ExpiresAt, jobListing.Source, jobListing.SourceID, jobListing.URL, sqlmock.AnyArg(), jobListing.RawJSON).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := storageService.StoreJobListingData(context.Background(), jobListing); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func getMockJobListing() JobListing {
	return JobListing{
		Company:         "Example Inc",
		RemoteFlag:      true,
		Location:        "Remote",
		SalaryMin:       floatPtr(750000.50),
		SalaryMax:       floatPtr(150000),
		Currency:        "USD",
		DescriptionHTML: "<p>Job description</p>",
		DescriptionText: "Job description",
		PostedDate:      timePtr(time.Now()),
		ExpiresAt:       timePtr(time.Now().Add(30 * 24 * time.Hour)),
		Source:          "ExampleSource",
		SourceID:        "12345",
		URL:             "http://example.com/job/12345",
		Tags:            []string{"Go", "Remote"},
		RawJSON:         []byte(`{"title": "Software Engineer"}`),
	}
}

func floatPtr(f float64) *float64 {
	return &f
}

func timePtr(t time.Time) *time.Time {
	return &t
}
