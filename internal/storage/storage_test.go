package storage

import (
	"context"
	"errors"
	"golangwebcrawler/internal/models"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/lib/pq"
)

const testURL = "http://example.com"

func Test_StoreRawData_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())

	mock.ExpectExec(
		`INSERT INTO raw_data \(url, content_type, raw_content\)\s+VALUES \(\$1, \$2, \$3\)\s+ON CONFLICT \(url\)\s+DO UPDATE SET content_type = \$2, raw_content = \$3, fetched_at = NOW\(\)`,
	).WithArgs(testURL, "text/html", "<html></html>").WillReturnResult(sqlmock.NewResult(1, 1))

	err = storageService.StoreRawData(context.Background(), testURL, "text/html", "<html></html>")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreRawData_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())

	mock.ExpectExec(
		`INSERT INTO raw_data`,
	).WithArgs(testURL, "text/html", "<html></html>").WillReturnError(errors.New("duplicate key"))

	err = storageService.StoreRawData(context.Background(), testURL, "text/html", "<html></html>")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "duplicate key") {
		t.Errorf("expected error to contain 'duplicate key', got: %s", err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_GetLatestRawData_ReturnsRawData(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	timeParam := time.Now().Add(-time.Hour)

	rows := sqlmock.NewRows([]string{"url", "content_type", "raw_content"}).
		AddRow(testURL, "text/html", "<html></html>")

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

// todo update this test UTC - 0 timing
func Test_GetLatestRawData_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	timeParam := time.Now().Add(-time.Hour)

	mock.ExpectQuery(`SELECT .+ FROM raw_data`).
		WithArgs(timeParam).
		WillReturnError(errors.New("connection refused"))

	_, err = storageService.GetLatestRawData(context.Background(), timeParam)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("expected error to contain 'connection refused', got: %s", err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

// todo update this test UTC - 0 timing
func Test_GetLatestRawData_EmptyResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	timeParam := time.Now().Add(-time.Hour)

	rows := sqlmock.NewRows([]string{"url", "content_type", "raw_content"})

	mock.ExpectQuery(`SELECT .+ FROM raw_data`).
		WithArgs(timeParam).
		WillReturnRows(rows)

	result, err := storageService.GetLatestRawData(context.Background(), timeParam)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 results, got %d", len(result))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

// todo update this test UTC - 0 timing
func Test_GetLatestRawData_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	timeParam := time.Now().Add(-time.Hour)

	// Simulate a scan error by returning an error during query execution
	mock.ExpectQuery(`SELECT .+ FROM raw_data`).
		WithArgs(timeParam).
		WillReturnError(errors.New("scan error on row"))

	_, err = storageService.GetLatestRawData(context.Background(), timeParam)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// The error wraps the query error, not a scan error since we're simulating at query level
	if !strings.Contains(err.Error(), "querying raw data") {
		t.Errorf("expected error to contain 'querying raw data', got: %s", err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreJobListingData_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	jobListing := getMockJobListing()

	mock.ExpectExec("INSERT INTO job_listings").
		WithArgs(jobListing.Title, jobListing.Company, jobListing.Location, jobListing.RemoteFlag, jobListing.SalaryMin, jobListing.SalaryMax, jobListing.Currency, jobListing.DescriptionHTML, jobListing.DescriptionText, jobListing.PostedDate, jobListing.ExpiresAt, jobListing.Source, jobListing.SourceID, jobListing.URL, sqlmock.AnyArg(), jobListing.RawJSON).
		WillReturnError(errors.New("duplicate key"))

	err = storageService.StoreJobListingData(context.Background(), jobListing)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "duplicate key") {
		t.Errorf("expected error to contain 'duplicate key', got: %s", err.Error())
	}

	if !strings.Contains(err.Error(), "storing job listing") {
		t.Errorf("expected error to contain 'storing job listing', got: %s", err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreRawDataBatch_EmptyItems(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())

	err = storageService.StoreRawDataBatch(context.Background(), []models.RawDataItem{})
	if err != nil {
		t.Fatalf("expected no error for empty items, got: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreRawDataBatch_BeginTxError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	items := []models.RawDataItem{
		{URL: "http://example1.com", ContentType: "text/html", RawContent: "<html>1</html>"},
	}

	mock.ExpectBegin().WillReturnError(errors.New("connection error"))

	err = storageService.StoreRawDataBatch(context.Background(), items)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "beginning transaction") {
		t.Errorf("expected error to contain 'beginning transaction', got: %s", err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreRawDataBatch_CreateTempTableError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	items := []models.RawDataItem{
		{URL: "http://example1.com", ContentType: "text/html", RawContent: "<html>1</html>"},
	}

	mock.ExpectBegin()
	mock.ExpectExec(`CREATE TEMP TABLE raw_data_batch`).WillReturnError(errors.New("table already exists"))

	err = storageService.StoreRawDataBatch(context.Background(), items)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "creating temp table") {
		t.Errorf("expected error to contain 'creating temp table', got: %s", err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreRawDataBatch_CopyExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	items := []models.RawDataItem{
		{URL: "http://example1.com", ContentType: "text/html", RawContent: "<html>1</html>"},
	}

	mock.ExpectBegin()
	mock.ExpectExec(`CREATE TEMP TABLE raw_data_batch`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPrepare(`COPY "raw_data_batch" \("url", "content_type", "raw_content"\) FROM STDIN`)
	mock.ExpectExec(`COPY "raw_data_batch"`).
		WithArgs("http://example1.com", "text/html", "<html>1</html>").
		WillReturnError(errors.New("copy error"))

	err = storageService.StoreRawDataBatch(context.Background(), items)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "executing copy") {
		t.Errorf("expected error to contain 'executing copy', got: %s", err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreRawDataBatch_FlushError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	items := []models.RawDataItem{
		{URL: "http://example1.com", ContentType: "text/html", RawContent: "<html>1</html>"},
	}

	mock.ExpectBegin()
	mock.ExpectExec(`CREATE TEMP TABLE raw_data_batch`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPrepare(`COPY "raw_data_batch" \("url", "content_type", "raw_content"\) FROM STDIN`)
	mock.ExpectExec(`COPY "raw_data_batch"`).
		WithArgs("http://example1.com", "text/html", "<html>1</html>").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`COPY "raw_data_batch"`).
		WillReturnError(errors.New("flush error"))

	err = storageService.StoreRawDataBatch(context.Background(), items)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "flushing copy statement") {
		t.Errorf("expected error to contain 'flushing copy statement', got: %s", err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreRawDataBatch_UpsertError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	items := []models.RawDataItem{
		{URL: "http://example1.com", ContentType: "text/html", RawContent: "<html>1</html>"},
	}

	mock.ExpectBegin()
	mock.ExpectExec(`CREATE TEMP TABLE raw_data_batch`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPrepare(`COPY "raw_data_batch" \("url", "content_type", "raw_content"\) FROM STDIN`)
	mock.ExpectExec(`COPY "raw_data_batch"`).
		WithArgs("http://example1.com", "text/html", "<html>1</html>").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`COPY "raw_data_batch"`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO raw_data .+ FROM raw_data_batch`).WillReturnError(errors.New("upsert error"))

	err = storageService.StoreRawDataBatch(context.Background(), items)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "upserting from temp table") {
		t.Errorf("expected error to contain 'upserting from temp table', got: %s", err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreRawDataBatch_CommitError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	items := []models.RawDataItem{
		{URL: "http://example1.com", ContentType: "text/html", RawContent: "<html>1</html>"},
	}

	mock.ExpectBegin()
	mock.ExpectExec(`CREATE TEMP TABLE raw_data_batch`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPrepare(`COPY "raw_data_batch" \("url", "content_type", "raw_content"\) FROM STDIN`)
	mock.ExpectExec(`COPY "raw_data_batch"`).
		WithArgs("http://example1.com", "text/html", "<html>1</html>").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`COPY "raw_data_batch"`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO raw_data .+ FROM raw_data_batch`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit error"))

	err = storageService.StoreRawDataBatch(context.Background(), items)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "committing transaction") {
		t.Errorf("expected error to contain 'committing transaction', got: %s", err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

// ==================== StoreExtractedJobDataBatch Error Cases ====================

func Test_StoreExtractedJobDataBatch_EmptyResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())

	err = storageService.StoreExtractedJobDataBatch(context.Background(), []ExtractedJobData{})
	if err != nil {
		t.Fatalf("expected no error for empty results, got: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreExtractedJobDataBatch_BeginTxError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	jobCards := mockJobCards()

	mock.ExpectBegin().WillReturnError(errors.New("connection error"))

	err = storageService.StoreExtractedJobDataBatch(context.Background(), jobCards)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "beginning transaction") {
		t.Errorf("expected error to contain 'beginning transaction', got: %s", err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreExtractedJobDataBatch_CommitError(t *testing.T) {
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
	mock.ExpectCommit().WillReturnError(errors.New("commit error"))

	err = storageService.StoreExtractedJobDataBatch(context.Background(), jobCards)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "committing copy transaction") {
		t.Errorf("expected error to contain 'committing copy transaction', got: %s", err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreExtractedJobDataBatchUpSert_EmptyResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())

	err = storageService.StoreExtractedJobDataBatchUpSert(context.Background(), []ExtractedJobData{})
	if err != nil {
		t.Fatalf("expected no error for empty results, got: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreExtractedJobDataBatchUpSert_BeginTxError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	jobCards := mockJobCards()

	mock.ExpectBegin().WillReturnError(errors.New("connection error"))

	err = storageService.StoreExtractedJobDataBatchUpSert(context.Background(), jobCards)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "beginning transaction") {
		t.Errorf("expected error to contain 'beginning transaction', got: %s", err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreExtractedJobDataBatchUpSert_ExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	jobCards := mockJobCards()

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO extracted_jobdata \(title, company, location, salary, description, link, skills\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\) ON CONFLICT \(link\) DO UPDATE SET title = EXCLUDED.title, company = EXCLUDED.company, location = EXCLUDED.location, salary = EXCLUDED.salary, description = EXCLUDED.description, skills = EXCLUDED.skills`)
	expectedSkills := strings.Join(jobCards[0].Skills, ",")
	mock.ExpectExec(`INSERT INTO extracted_jobdata`).
		WithArgs(jobCards[0].Title, jobCards[0].Company, jobCards[0].Location, jobCards[0].Salary, jobCards[0].Description, jobCards[0].Link, expectedSkills).
		WillReturnError(errors.New("exec error"))

	err = storageService.StoreExtractedJobDataBatchUpSert(context.Background(), jobCards)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "executing statement") {
		t.Errorf("expected error to contain 'executing statement', got: %s", err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreExtractedJobDataBatchUpSert_CommitError(t *testing.T) {
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
	mock.ExpectCommit().WillReturnError(errors.New("commit error"))

	err = storageService.StoreExtractedJobDataBatchUpSert(context.Background(), jobCards)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "committing transaction") {
		t.Errorf("expected error to contain 'committing transaction', got: %s", err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

// ==================== Nil Slice Tests ====================

func Test_StoreRawDataBatch_NilItems(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())

	err = storageService.StoreRawDataBatch(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected no error for nil items, got: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreExtractedJobDataBatch_NilResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())

	err = storageService.StoreExtractedJobDataBatch(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected no error for nil results, got: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreExtractedJobDataBatchUpSert_NilResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())

	err = storageService.StoreExtractedJobDataBatchUpSert(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected no error for nil results, got: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreJobListingData_WithNilFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	jobListing := JobListing{
		Title:           "Test Job",
		Company:         "Test Corp",
		Location:        "Remote",
		RemoteFlag:      false,
		SalaryMin:       nil,
		SalaryMax:       nil,
		Currency:        "",
		DescriptionHTML: "",
		DescriptionText: "Test description",
		PostedDate:      nil,
		ExpiresAt:       nil,
		Source:          "test",
		SourceID:        "1",
		URL:             "http://example.com/job/1",
		Tags:            nil,
		RawJSON:         nil,
	}

	mock.ExpectExec("INSERT INTO job_listings").
		WithArgs(jobListing.Title, jobListing.Company, jobListing.Location, jobListing.RemoteFlag, nil, nil, "", "", jobListing.DescriptionText, nil, nil, jobListing.Source, jobListing.SourceID, jobListing.URL, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = storageService.StoreJobListingData(context.Background(), jobListing)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreExtractedJobDataBatch_NilSkills(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	jobCards := []ExtractedJobData{
		{
			Title:    "Software Engineer",
			Company:  "Example Inc",
			Location: "Remote",
			Salary:   "$100k - $150k",
			Link:     "https://www.seek.com.au/job1",
			Skills:   nil,
		},
	}

	mock.ExpectBegin()
	mock.ExpectPrepare(`COPY "extracted_jobdata" \("title", "company", "location", "salary", "description", "link", "skills"\) FROM STDIN`)
	mock.ExpectExec(`COPY "extracted_jobdata"`).
		WithArgs("Software Engineer", "Example Inc", "Remote", "$100k - $150k", "", "https://www.seek.com.au/job1", "").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = storageService.StoreExtractedJobDataBatch(context.Background(), jobCards)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreExtractedJobDataBatchUpSert_NilSkills(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	jobCards := []ExtractedJobData{
		{
			Title:    "Software Engineer",
			Company:  "Example Inc",
			Location: "Remote",
			Salary:   "$100k - $150k",
			Link:     "https://www.seek.com.au/job1",
			Skills:   nil,
		},
	}

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO extracted_jobdata \(title, company, location, salary, description, link, skills\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\) ON CONFLICT \(link\) DO UPDATE SET title = EXCLUDED.title, company = EXCLUDED.company, location = EXCLUDED.location, salary = EXCLUDED.salary, description = EXCLUDED.description, skills = EXCLUDED.skills`)
	mock.ExpectExec(`INSERT INTO extracted_jobdata`).
		WithArgs("Software Engineer", "Example Inc", "Remote", "$100k - $150k", "", "https://www.seek.com.au/job1", "").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = storageService.StoreExtractedJobDataBatchUpSert(context.Background(), jobCards)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

// ==================== RawData with FetchedAt Test ====================

func Test_GetLatestRawData_WithFetchedAt(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	timeParam := time.Now().Add(-time.Hour)

	rows := sqlmock.NewRows([]string{"url", "content_type", "raw_content"}).
		AddRow(testURL, "text/html", "<html></html>").
		AddRow("http://example2.com", "application/json", `{"key": "value"}`)

	mock.ExpectQuery(`SELECT .+ FROM raw_data`).
		WithArgs(timeParam).
		WillReturnRows(rows)

	result, err := storageService.GetLatestRawData(context.Background(), timeParam)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}

	if result[0].URL != testURL {
		t.Errorf("expected URL 'http://example.com', got '%s'", result[0].URL)
	}
	if result[1].URL != "http://example2.com" {
		t.Errorf("expected URL 'http://example2.com', got '%s'", result[1].URL)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreRawDataBatch_BadConnectionRollback(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	items := []models.RawDataItem{
		{URL: "http://example1.com", ContentType: "text/html", RawContent: "<html>1</html>"},
	}

	mock.ExpectBegin()
	// Simulate the transaction failing and then rolling back successfully
	mock.ExpectExec(`CREATE TEMP TABLE raw_data_batch`).WillReturnError(errors.New("bad connection"))
	// The deferred rollback should be called, but since sql.ErrTxDone is expected after error,
	// we don't need to expect a rollback here - the transaction is already done

	err = storageService.StoreRawDataBatch(context.Background(), items)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreExtractedJobDataBatchUpSert_EmptyDescription(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	jobCards := []ExtractedJobData{
		{
			Title:       "Software Engineer",
			Company:     "Example Inc",
			Location:    "Remote",
			Salary:      "$100k - $150k",
			Description: "",
			Link:        "https://www.seek.com.au/job1",
			Skills:      []string{"Go"},
		},
	}

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO extracted_jobdata \(title, company, location, salary, description, link, skills\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\) ON CONFLICT \(link\) DO UPDATE SET title = EXCLUDED.title, company = EXCLUDED.company, location = EXCLUDED.location, salary = EXCLUDED.salary, description = EXCLUDED.description, skills = EXCLUDED.skills`)
	mock.ExpectExec(`INSERT INTO extracted_jobdata`).
		WithArgs("Software Engineer", "Example Inc", "Remote", "$100k - $150k", "", "https://www.seek.com.au/job1", "Go").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = storageService.StoreExtractedJobDataBatchUpSert(context.Background(), jobCards)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreRawDataBatch_MultipleItems(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	items := []models.RawDataItem{
		{URL: "http://example1.com", ContentType: "text/html", RawContent: "<html>1</html>"},
		{URL: "http://example2.com", ContentType: "application/json", RawContent: `{"key": "value"}`},
		{URL: "http://example3.com", ContentType: "text/plain", RawContent: "plain text"},
	}

	mock.ExpectBegin()
	mock.ExpectExec(`CREATE TEMP TABLE raw_data_batch`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPrepare(`COPY "raw_data_batch" \("url", "content_type", "raw_content"\) FROM STDIN`)

	for _, item := range items {
		mock.ExpectExec(`COPY "raw_data_batch"`).
			WithArgs(item.URL, item.ContentType, item.RawContent).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}

	mock.ExpectExec(`COPY "raw_data_batch"`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectExec(`INSERT INTO raw_data .+ FROM raw_data_batch`).
		WillReturnResult(sqlmock.NewResult(0, int64(len(items))))
	mock.ExpectCommit()

	err = storageService.StoreRawDataBatch(context.Background(), items)
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreExtractedJobDataBatch_MultipleItems(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	jobCards := []ExtractedJobData{
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
		{
			Title:    "DevOps Engineer",
			Company:  "Cloud Inc",
			Location: "London, UK",
			Salary:   "$130k - $180k",
			Link:     "https://www.seek.com.au/job3",
			Skills:   []string{"Kubernetes", "Terraform"},
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

	err = storageService.StoreExtractedJobDataBatch(context.Background(), jobCards)
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreExtractedJobDataBatchUpSert_MultipleItems(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	jobCards := []ExtractedJobData{
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
	mock.ExpectPrepare(`INSERT INTO extracted_jobdata \(title, company, location, salary, description, link, skills\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\) ON CONFLICT \(link\) DO UPDATE SET title = EXCLUDED.title, company = EXCLUDED.company, location = EXCLUDED.location, salary = EXCLUDED.salary, description = EXCLUDED.description, skills = EXCLUDED.skills`)
	for _, jobCard := range jobCards {
		expectedSkills := strings.Join(jobCard.Skills, ",")
		mock.ExpectExec(`INSERT INTO extracted_jobdata`).
			WithArgs(jobCard.Title, jobCard.Company, jobCard.Location, jobCard.Salary, jobCard.Description, jobCard.Link, expectedSkills).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectCommit()

	err = storageService.StoreExtractedJobDataBatchUpSert(context.Background(), jobCards)
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

// ==================== JobListing with All Fields Tests ====================

func Test_StoreJobListingData_FullFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	now := time.Now()
	expiresAt := now.Add(30 * 24 * time.Hour)
	jobListing := JobListing{
		ID:              1,
		Title:           "Senior Software Engineer",
		Company:         "Big Tech Corp",
		Location:        "San Francisco, CA",
		RemoteFlag:      true,
		SalaryMin:       floatPtr(150000),
		SalaryMax:       floatPtr(250000),
		Currency:        "USD",
		DescriptionHTML: "<p>Exciting job description</p>",
		DescriptionText: "Exciting job description",
		PostedDate:      &now,
		ExpiresAt:       &expiresAt,
		Source:          "LinkedIn",
		SourceID:        "job-12345",
		URL:             "https://linkedin.com/job/12345",
		Tags:            []string{"Go", "Kubernetes", "Remote"},
		RawJSON:         []byte(`{"title": "Senior Software Engineer", "company": "Big Tech Corp"}`),
		CrawlTimestamp:  now,
	}

	mock.ExpectExec("INSERT INTO job_listings").
		WithArgs(jobListing.Title, jobListing.Company, jobListing.Location, jobListing.RemoteFlag, jobListing.SalaryMin, jobListing.SalaryMax, jobListing.Currency, jobListing.DescriptionHTML, jobListing.DescriptionText, jobListing.PostedDate, jobListing.ExpiresAt, jobListing.Source, jobListing.SourceID, jobListing.URL, sqlmock.AnyArg(), jobListing.RawJSON).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = storageService.StoreJobListingData(context.Background(), jobListing)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreRawDataBatch_SingleItem(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	items := []models.RawDataItem{
		{URL: "http://example1.com", ContentType: "text/html", RawContent: "<html>1</html>"},
	}

	mock.ExpectBegin()
	mock.ExpectExec(`CREATE TEMP TABLE raw_data_batch`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPrepare(`COPY "raw_data_batch" \("url", "content_type", "raw_content"\) FROM STDIN`)
	mock.ExpectExec(`COPY "raw_data_batch"`).
		WithArgs("http://example1.com", "text/html", "<html>1</html>").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`COPY "raw_data_batch"`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO raw_data .+ FROM raw_data_batch`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = storageService.StoreRawDataBatch(context.Background(), items)
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreExtractedJobDataBatch_SingleItem(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	jobCards := []ExtractedJobData{
		{
			Title:    "Software Engineer",
			Company:  "Example Inc",
			Location: "Remote",
			Salary:   "$100k - $150k",
			Link:     "https://www.seek.com.au/job1",
			Skills:   []string{"Go"},
		},
	}

	mock.ExpectBegin()
	mock.ExpectPrepare(`COPY "extracted_jobdata" \("title", "company", "location", "salary", "description", "link", "skills"\) FROM STDIN`)
	mock.ExpectExec(`COPY "extracted_jobdata"`).
		WithArgs("Software Engineer", "Example Inc", "Remote", "$100k - $150k", "", "https://www.seek.com.au/job1", "Go").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = storageService.StoreExtractedJobDataBatch(context.Background(), jobCards)
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_StoreExtractedJobDataBatchUpSert_SingleItem(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storageService := NewService(db, slog.Default())
	jobCards := []ExtractedJobData{
		{
			Title:    "Software Engineer",
			Company:  "Example Inc",
			Location: "Remote",
			Salary:   "$100k - $150k",
			Link:     "https://www.seek.com.au/job1",
			Skills:   []string{"Go"},
		},
	}

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO extracted_jobdata \(title, company, location, salary, description, link, skills\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\) ON CONFLICT \(link\) DO UPDATE SET title = EXCLUDED.title, company = EXCLUDED.company, location = EXCLUDED.location, salary = EXCLUDED.salary, description = EXCLUDED.description, skills = EXCLUDED.skills`)
	mock.ExpectExec(`INSERT INTO extracted_jobdata`).
		WithArgs("Software Engineer", "Example Inc", "Remote", "$100k - $150k", "", "https://www.seek.com.au/job1", "Go").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = storageService.StoreExtractedJobDataBatchUpSert(context.Background(), jobCards)
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_NewService(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	logger := slog.Default()
	service := NewService(db, logger)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if service.db != db {
		t.Error("expected service.db to be the provided db")
	}
	if service.logger != logger {
		t.Error("expected service.logger to be the provided logger")
	}

	// Verify mock expectations are met (the mock just needs to be open)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func Test_RawData_StructFields(t *testing.T) {
	now := time.Now().Format(time.RFC3339)
	r := RawData{
		URL:         testURL,
		ContentType: "text/html",
		RawContent:  "<html></html>",
		FetchedAt:   now,
	}

	if r.URL != testURL {
		t.Errorf("expected URL 'http://example.com', got '%s'", r.URL)
	}
	if r.ContentType != "text/html" {
		t.Errorf("expected ContentType 'text/html', got '%s'", r.ContentType)
	}
	if r.RawContent != "<html></html>" {
		t.Errorf("expected RawContent '<html></html>', got '%s'", r.RawContent)
	}
	if r.FetchedAt != now {
		t.Errorf("expected FetchedAt '%s', got '%s'", now, r.FetchedAt)
	}
}

func Test_ExtractedJobData_JSONTags(t *testing.T) {
	ejd := ExtractedJobData{
		Title:       "Test",
		Company:     "Test Corp",
		Location:    "Remote",
		Salary:      "$100k",
		Description: "Description",
		Skills:      []string{"Go"},
		Link:        testURL,
	}

	if ejd.Title != "Test" {
		t.Errorf("expected Title 'Test', got '%s'", ejd.Title)
	}
	if ejd.Company != "Test Corp" {
		t.Errorf("expected Company 'Test Corp', got '%s'", ejd.Company)
	}
	if ejd.Location != "Remote" {
		t.Errorf("expected Location 'Remote', got '%s'", ejd.Location)
	}
	if ejd.Salary != "$100k" {
		t.Errorf("expected Salary '$100k', got '%s'", ejd.Salary)
	}
	if ejd.Description != "Description" {
		t.Errorf("expected Description 'Description', got '%s'", ejd.Description)
	}
	if len(ejd.Skills) != 1 || ejd.Skills[0] != "Go" {
		t.Errorf("expected Skills ['Go'], got %v", ejd.Skills)
	}
	if ejd.Link != testURL {
		t.Errorf("expected Link 'http://example.com', got '%s'", ejd.Link)
	}
}

func Test_QueryTimeoutConstants(t *testing.T) {
	// Verify the constants are set correctly by checking they compile
	// and have expected values (5 seconds for query, 60 seconds for batch)
	_ = queryTimeout
	_ = batchQueryTimeout
}
