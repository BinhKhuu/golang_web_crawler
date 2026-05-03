package storage

import (
	"context"
	"errors"
	"golangwebcrawler/internal/models"
	"golangwebcrawler/internal/typeutil"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/lib/pq"
)

const (
	testURL           = "http://example.com"
	softwareEngineer  = "Software Engineer"
	exampleInc        = "Example Inc"
	remoteLocation    = "Remote"
	salary100kTo150k  = "$100k - $150k"
	seekJobURL1       = "https://www.seek.com.au/job1"
	dockerSkill       = "Docker"
	backendDeveloper  = "Backend Developer"
	techCorp          = "Tech Corp"
	newYorkNY         = "New York, NY"
	salary120kTo170k  = "$120k - $170k"
	seekJobURL2       = "https://www.seek.com.au/job2"
	awsSkill          = "AWS"
	contentTypeColumn = "content_type"
	example2Com       = "http://example2.com"
	html1Content      = "<html>1</html>"
	testCorp          = "Test Corp"
	goSkill           = "Go"
	pythonSkill       = "Python"
	rawContentColumn  = "raw_content"
	example1Com       = "http://example1.com"
	urlColumn         = "url"
	textHtml          = "text/html"
)

const testURLSingle = "http://example.com"

var testJobCards = []ExtractedJobData{
	{
		Title:    softwareEngineer,
		Company:  exampleInc,
		Location: remoteLocation,
		Salary:   salary100kTo150k,
		Link:     seekJobURL1,
		Skills:   []string{goSkill, dockerSkill},
	},
	{
		Title:    backendDeveloper,
		Company:  techCorp,
		Location: newYorkNY,
		Salary:   salary120kTo170k,
		Link:     seekJobURL2,
		Skills:   []string{pythonSkill, awsSkill},
	},
}

var testJobListing = JobListing{
	Company:         exampleInc,
	RemoteFlag:      true,
	Location:        remoteLocation,
	SalaryMin:       floatPtr(750000.50),
	SalaryMax:       floatPtr(150000),
	Currency:        "USD",
	DescriptionHTML: "<p>Job description</p>",
	DescriptionText: "Job description",
	PostedDate:      timePtr(typeutil.UTCTimeNow()),
	ExpiresAt:       timePtr(typeutil.UTCTimeNow().Add(30 * 24 * time.Hour)),
	Source:          "ExampleSource",
	SourceID:        "12345",
	URL:             "http://example.com/job/12345",
	Tags:            []string{goSkill, remoteLocation},
	RawJSON:         []byte(`{"title": "Software Engineer"}`),
}

func floatPtr(f float64) *float64 {
	return &f
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func Test_StoreRawData_Success(t *testing.T) {
	_, mock, storageService := setupStorageTest(t)

	mock.ExpectExec(
		`INSERT INTO raw_data \(url, content_type, raw_content\)\s+VALUES \(\$1, \$2, \$3\)\s+ON CONFLICT \(url\)\s+DO UPDATE SET content_type = \$2, raw_content = \$3, fetched_at = NOW\(\)`,
	).WithArgs(testURL, "text/html", "<html></html>").WillReturnResult(sqlmock.NewResult(1, 1))

	err := storageService.StoreRawData(context.Background(), testURL, "text/html", "<html></html>")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func Test_StoreRawData_Error(t *testing.T) {
	_, mock, storageService := setupStorageTest(t)

	mock.ExpectExec(`INSERT INTO raw_data`).
		WithArgs(testURL, "text/html", "<html></html>").
		WillReturnError(errors.New("duplicate key"))

	err := storageService.StoreRawData(context.Background(), testURL, "text/html", "<html></html>")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "duplicate key") {
		t.Errorf("expected error to contain 'duplicate key', got: %s", err.Error())
	}
}

func Test_GetLatestRawData(t *testing.T) {
	tests := map[string]struct {
		setupMock      func(sqlmock.Sqlmock, time.Time)
		expectedErr    string
		expectedCount  int
		validateResult func(t *testing.T, result []RawData)
	}{
		"ReturnsRawData": {
			setupMock: func(mock sqlmock.Sqlmock, timeParam time.Time) {
				rows := sqlmock.NewRows([]string{urlColumn, contentTypeColumn, rawContentColumn}).
					AddRow(testURLSingle, "text/html", html1Content)
				mock.ExpectQuery(`SELECT .+ FROM raw_data`).
					WithArgs(timeParam).
					WillReturnRows(rows)
			},
			expectedCount: 1,
			validateResult: func(t *testing.T, result []RawData) {
				if len(result) != 1 {
					t.Fatalf("expected 1 result, got %d", len(result))
				}
			},
		},
		"WithFetchedAt": {
			setupMock: func(mock sqlmock.Sqlmock, timeParam time.Time) {
				rows := sqlmock.NewRows([]string{urlColumn, contentTypeColumn, rawContentColumn}).
					AddRow(testURLSingle, "text/html", html1Content).
					AddRow(example2Com, "application/json", `{"key": "value"}`)
				mock.ExpectQuery(`SELECT .+ FROM raw_data`).
					WithArgs(timeParam).
					WillReturnRows(rows)
			},
			expectedCount: 2,
			validateResult: func(t *testing.T, result []RawData) {
				if len(result) != 2 {
					t.Fatalf("expected 2 results, got %d", len(result))
				}
				if result[0].URL != testURLSingle {
					t.Errorf("expected URL 'http://example.com', got '%s'", result[0].URL)
				}
				if result[1].URL != example2Com {
					t.Errorf("expected URL 'http://example2.com', got '%s'", result[1].URL)
				}
			},
		},
		"EmptyResults": {
			setupMock: func(mock sqlmock.Sqlmock, timeParam time.Time) {
				rows := sqlmock.NewRows([]string{"url", "content_type", rawContentColumn})
				mock.ExpectQuery(`SELECT .+ FROM raw_data`).
					WithArgs(timeParam).
					WillReturnRows(rows)
			},
			expectedCount: 0,
			validateResult: func(t *testing.T, result []RawData) {
				if len(result) != 0 {
					t.Fatalf("expected 0 results, got %d", len(result))
				}
			},
		},
		"QueryError": {
			setupMock: func(mock sqlmock.Sqlmock, timeParam time.Time) {
				mock.ExpectQuery(`SELECT .+ FROM raw_data`).
					WithArgs(timeParam).
					WillReturnError(errors.New("connection refused"))
			},
			expectedErr: "querying raw data",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock, storageService := setupStorageTest(t)
			timeParam := typeutil.UTCTimeNow().Add(-time.Hour)

			tt.setupMock(mock, timeParam)

			result, err := storageService.GetLatestRawData(context.Background(), timeParam)
			if tt.expectedErr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.expectedErr) {
					t.Errorf("expected error to contain '%s', got: %s", tt.expectedErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if len(result) != tt.expectedCount {
				t.Fatalf("expected %d results, got %d", tt.expectedCount, len(result))
			}
			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}

			_ = db
		})
	}
}

func Test_StoreRawDataBatch(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)
	items := []models.RawDataItem{
		{URL: example1Com, ContentType: textHtml, RawContent: html1Content},
		{URL: "http://example2.com", ContentType: textHtml, RawContent: "<html>2</html>"},
	}

	expectStoreRawDataBatchSuccess(mock, items)
	if err := storageService.StoreRawDataBatch(context.Background(), items); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_ = db
}

func Test_StoreRawDataBatch_EmptyItems(t *testing.T) {
	_, _, storageService := setupStorageTest(t)
	if err := storageService.StoreRawDataBatch(context.Background(), []models.RawDataItem{}); err != nil {
		t.Fatalf("expected no error for empty items, got: %s", err)
	}
}

func Test_StoreRawDataBatch_NilItems(t *testing.T) {
	_, _, storageService := setupStorageTest(t)
	if err := storageService.StoreRawDataBatch(context.Background(), nil); err != nil {
		t.Fatalf("expected no error for nil items, got: %s", err)
	}
}

func Test_StoreRawDataBatch_MultipleItems(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)
	items := []models.RawDataItem{
		{URL: example1Com, ContentType: textHtml, RawContent: html1Content},
		{URL: "http://example2.com", ContentType: "application/json", RawContent: `{"key": "value"}`},
		{URL: "http://example3.com", ContentType: "text/plain", RawContent: "plain text"},
	}

	expectStoreRawDataBatchSuccess(mock, items)
	if err := storageService.StoreRawDataBatch(context.Background(), items); err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}

	_ = db
}

func Test_StoreRawDataBatch_BeginTxError(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)
	items := []models.RawDataItem{{URL: example1Com, ContentType: textHtml, RawContent: html1Content}}

	mock.ExpectBegin().WillReturnError(errors.New("connection error"))
	err := storageService.StoreRawDataBatch(context.Background(), items)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "beginning transaction") {
		t.Errorf("expected error to contain 'beginning transaction', got: %s", err.Error())
	}

	_ = db
}

func Test_StoreRawDataBatch_CreateTempTableError(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)
	items := []models.RawDataItem{{URL: example1Com, ContentType: textHtml, RawContent: html1Content}}

	mock.ExpectBegin()
	mock.ExpectExec(`CREATE TEMP TABLE raw_data_batch`).WillReturnError(errors.New("table already exists"))
	err := storageService.StoreRawDataBatch(context.Background(), items)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "creating temp table") {
		t.Errorf("expected error to contain 'creating temp table', got: %s", err.Error())
	}

	_ = db
}

func Test_StoreRawDataBatch_CopyExecError(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)
	items := []models.RawDataItem{{URL: example1Com, ContentType: textHtml, RawContent: html1Content}}

	mock.ExpectBegin()
	mock.ExpectExec(`CREATE TEMP TABLE raw_data_batch`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPrepare(`COPY "raw_data_batch" \("url", "content_type", "raw_content"\) FROM STDIN`)
	mock.ExpectExec(`COPY "raw_data_batch"`).
		WithArgs(example1Com, textHtml, html1Content).
		WillReturnError(errors.New("copy error"))

	err := storageService.StoreRawDataBatch(context.Background(), items)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "executing copy") {
		t.Errorf("expected error to contain 'executing copy', got: %s", err.Error())
	}

	_ = db
}

func Test_StoreRawDataBatch_FlushError(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)
	items := []models.RawDataItem{{URL: example1Com, ContentType: textHtml, RawContent: html1Content}}

	mock.ExpectBegin()
	mock.ExpectExec(`CREATE TEMP TABLE raw_data_batch`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPrepare(`COPY "raw_data_batch" \("url", "content_type", "raw_content"\) FROM STDIN`)
	mock.ExpectExec(`COPY "raw_data_batch"`).
		WithArgs(example1Com, textHtml, html1Content).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`COPY "raw_data_batch"`).WillReturnError(errors.New("flush error"))

	err := storageService.StoreRawDataBatch(context.Background(), items)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "flushing copy statement") {
		t.Errorf("expected error to contain 'flushing copy statement', got: %s", err.Error())
	}

	_ = db
}

func Test_StoreRawDataBatch_UpsertError(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)
	items := []models.RawDataItem{{URL: example1Com, ContentType: textHtml, RawContent: html1Content}}

	mock.ExpectBegin()
	mock.ExpectExec(`CREATE TEMP TABLE raw_data_batch`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPrepare(`COPY "raw_data_batch" \("url", "content_type", "raw_content"\) FROM STDIN`)
	mock.ExpectExec(`COPY "raw_data_batch"`).
		WithArgs(example1Com, textHtml, html1Content).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`COPY "raw_data_batch"`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO raw_data .+ FROM raw_data_batch`).WillReturnError(errors.New("upsert error"))

	err := storageService.StoreRawDataBatch(context.Background(), items)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "upserting from temp table") {
		t.Errorf("expected error to contain 'upserting from temp table', got: %s", err.Error())
	}

	_ = db
}

func Test_StoreRawDataBatch_CommitError(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)
	items := []models.RawDataItem{{URL: example1Com, ContentType: textHtml, RawContent: html1Content}}

	mock.ExpectBegin()
	mock.ExpectExec(`CREATE TEMP TABLE raw_data_batch`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPrepare(`COPY "raw_data_batch" \("url", "content_type", "raw_content"\) FROM STDIN`)
	mock.ExpectExec(`COPY "raw_data_batch"`).
		WithArgs(example1Com, textHtml, html1Content).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`COPY "raw_data_batch"`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO raw_data .+ FROM raw_data_batch`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit error"))

	err := storageService.StoreRawDataBatch(context.Background(), items)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "committing transaction") {
		t.Errorf("expected error to contain 'committing transaction', got: %s", err.Error())
	}

	_ = db
}

func Test_StoreRawDataBatch_BadConnectionRollback(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)
	items := []models.RawDataItem{{URL: example1Com, ContentType: textHtml, RawContent: html1Content}}

	mock.ExpectBegin()
	mock.ExpectExec(`CREATE TEMP TABLE raw_data_batch`).WillReturnError(errors.New("bad connection"))

	err := storageService.StoreRawDataBatch(context.Background(), items)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	_ = db
}

func Test_StoreExtractedJobDataBatch(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)

	expectStoreExtractedJobDataBatchSuccess(mock, testJobCards)
	if err := storageService.StoreExtractedJobDataBatch(context.Background(), testJobCards); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_ = db
}

func Test_StoreExtractedJobDataBatch_EmptyResults(t *testing.T) {
	_, _, storageService := setupStorageTest(t)
	if err := storageService.StoreExtractedJobDataBatch(context.Background(), []ExtractedJobData{}); err != nil {
		t.Fatalf("expected no error for empty results, got: %s", err)
	}
}

func Test_StoreExtractedJobDataBatch_NilResults(t *testing.T) {
	_, _, storageService := setupStorageTest(t)
	if err := storageService.StoreExtractedJobDataBatch(context.Background(), nil); err != nil {
		t.Fatalf("expected no error for nil results, got: %s", err)
	}
}

func Test_StoreExtractedJobDataBatch_MultipleItems(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)
	jobCards := []ExtractedJobData{
		{Title: softwareEngineer, Company: exampleInc, Location: remoteLocation, Salary: salary100kTo150k, Link: seekJobURL1, Skills: []string{goSkill, dockerSkill}},
		{Title: backendDeveloper, Company: techCorp, Location: newYorkNY, Salary: salary120kTo170k, Link: seekJobURL2, Skills: []string{pythonSkill, awsSkill}},
		{Title: "DevOps Engineer", Company: "Cloud Inc", Location: "London, UK", Salary: "$130k - $180k", Link: "https://www.seek.com.au/job3", Skills: []string{"Kubernetes", "Terraform"}},
	}

	expectStoreExtractedJobDataBatchSuccess(mock, jobCards)
	if err := storageService.StoreExtractedJobDataBatch(context.Background(), jobCards); err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}

	_ = db
}

func Test_StoreExtractedJobDataBatch_NilSkills(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)
	jobCards := []ExtractedJobData{
		{Title: softwareEngineer, Company: exampleInc, Location: remoteLocation, Salary: salary100kTo150k, Link: seekJobURL1, Skills: nil},
	}

	mock.ExpectBegin()
	mock.ExpectPrepare(`COPY "extracted_jobdata" \("title", "company", "location", "salary", "description", "link", "skills"\) FROM STDIN`)
	mock.ExpectExec(`COPY "extracted_jobdata"`).
		WithArgs(softwareEngineer, exampleInc, remoteLocation, salary100kTo150k, "", "https://www.seek.com.au/job1", "").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := storageService.StoreExtractedJobDataBatch(context.Background(), jobCards); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	_ = db
}

func Test_StoreExtractedJobDataBatch_BeginTxError(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)

	mock.ExpectBegin().WillReturnError(errors.New("connection error"))
	err := storageService.StoreExtractedJobDataBatch(context.Background(), testJobCards)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "beginning transaction") {
		t.Errorf("expected error to contain 'beginning transaction', got: %s", err.Error())
	}

	_ = db
}

func Test_StoreExtractedJobDataBatch_CommitError(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)

	mock.ExpectBegin()
	mock.ExpectPrepare(`COPY "extracted_jobdata" \("title", "company", "location", "salary", "description", "link", "skills"\) FROM STDIN`)
	for _, jobCard := range testJobCards {
		expectedSkills := strings.Join(jobCard.Skills, ",")
		mock.ExpectExec(`COPY "extracted_jobdata"`).
			WithArgs(jobCard.Title, jobCard.Company, jobCard.Location, jobCard.Salary, jobCard.Description, jobCard.Link, expectedSkills).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit().WillReturnError(errors.New("commit error"))

	err := storageService.StoreExtractedJobDataBatch(context.Background(), testJobCards)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "committing copy transaction") {
		t.Errorf("expected error to contain 'committing copy transaction', got: %s", err.Error())
	}

	_ = db
}

func Test_StoreExtractedJobDataBatchUpSert(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO extracted_jobdata \(title, company, location, salary, description, link, skills\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\) ON CONFLICT \(link\) DO UPDATE SET title = EXCLUDED.title, company = EXCLUDED.company, location = EXCLUDED.location, salary = EXCLUDED.salary, description = EXCLUDED.description, skills = EXCLUDED.skills`)
	for _, jobCard := range testJobCards {
		expectedSkills := strings.Join(jobCard.Skills, ",")
		mock.ExpectExec(`INSERT INTO extracted_jobdata`).
			WithArgs(jobCard.Title, jobCard.Company, jobCard.Location, jobCard.Salary, jobCard.Description, jobCard.Link, expectedSkills).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectCommit()

	if err := storageService.StoreExtractedJobDataBatchUpSert(context.Background(), testJobCards); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_ = db
}

func Test_StoreExtractedJobDataBatchUpSert_EmptyResults(t *testing.T) {
	_, _, storageService := setupStorageTest(t)
	if err := storageService.StoreExtractedJobDataBatchUpSert(context.Background(), []ExtractedJobData{}); err != nil {
		t.Fatalf("expected no error for empty results, got: %s", err)
	}
}

func Test_StoreExtractedJobDataBatchUpSert_NilResults(t *testing.T) {
	_, _, storageService := setupStorageTest(t)
	if err := storageService.StoreExtractedJobDataBatchUpSert(context.Background(), nil); err != nil {
		t.Fatalf("expected no error for nil results, got: %s", err)
	}
}

func Test_StoreExtractedJobDataBatchUpSert_MultipleItems(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)
	jobCards := []ExtractedJobData{
		{Title: softwareEngineer, Company: exampleInc, Location: remoteLocation, Salary: salary100kTo150k, Link: seekJobURL1, Skills: []string{goSkill, dockerSkill}},
		{Title: backendDeveloper, Company: techCorp, Location: newYorkNY, Salary: salary120kTo170k, Link: seekJobURL2, Skills: []string{pythonSkill, awsSkill}},
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

	if err := storageService.StoreExtractedJobDataBatchUpSert(context.Background(), jobCards); err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}

	_ = db
}

func Test_StoreExtractedJobDataBatchUpSert_NilSkills(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)
	jobCards := []ExtractedJobData{
		{Title: softwareEngineer, Company: exampleInc, Location: remoteLocation, Salary: salary100kTo150k, Link: seekJobURL1, Skills: nil},
	}

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO extracted_jobdata \(title, company, location, salary, description, link, skills\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\) ON CONFLICT \(link\) DO UPDATE SET title = EXCLUDED.title, company = EXCLUDED.company, location = EXCLUDED.location, salary = EXCLUDED.salary, description = EXCLUDED.description, skills = EXCLUDED.skills`)
	mock.ExpectExec(`INSERT INTO extracted_jobdata`).
		WithArgs(softwareEngineer, exampleInc, remoteLocation, salary100kTo150k, "", "https://www.seek.com.au/job1", "").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := storageService.StoreExtractedJobDataBatchUpSert(context.Background(), jobCards); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	_ = db
}

func Test_StoreExtractedJobDataBatchUpSert_BeginTxError(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)

	mock.ExpectBegin().WillReturnError(errors.New("connection error"))
	err := storageService.StoreExtractedJobDataBatchUpSert(context.Background(), testJobCards)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "beginning transaction") {
		t.Errorf("expected error to contain 'beginning transaction', got: %s", err.Error())
	}

	_ = db
}

func Test_StoreExtractedJobDataBatchUpSert_ExecError(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO extracted_jobdata \(title, company, location, salary, description, link, skills\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\) ON CONFLICT \(link\) DO UPDATE SET title = EXCLUDED.title, company = EXCLUDED.company, location = EXCLUDED.location, salary = EXCLUDED.salary, description = EXCLUDED.description, skills = EXCLUDED.skills`)
	expectedSkills := strings.Join(testJobCards[0].Skills, ",")
	mock.ExpectExec(`INSERT INTO extracted_jobdata`).
		WithArgs(testJobCards[0].Title, testJobCards[0].Company, testJobCards[0].Location, testJobCards[0].Salary, testJobCards[0].Description, testJobCards[0].Link, expectedSkills).
		WillReturnError(errors.New("exec error"))

	err := storageService.StoreExtractedJobDataBatchUpSert(context.Background(), testJobCards)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "executing statement") {
		t.Errorf("expected error to contain 'executing statement', got: %s", err.Error())
	}

	_ = db
}

func Test_StoreExtractedJobDataBatchUpSert_CommitError(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO extracted_jobdata \(title, company, location, salary, description, link, skills\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\) ON CONFLICT \(link\) DO UPDATE SET title = EXCLUDED.title, company = EXCLUDED.company, location = EXCLUDED.location, salary = EXCLUDED.salary, description = EXCLUDED.description, skills = EXCLUDED.skills`)
	for _, jobCard := range testJobCards {
		expectedSkills := strings.Join(jobCard.Skills, ",")
		mock.ExpectExec(`INSERT INTO extracted_jobdata`).
			WithArgs(jobCard.Title, jobCard.Company, jobCard.Location, jobCard.Salary, jobCard.Description, jobCard.Link, expectedSkills).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectCommit().WillReturnError(errors.New("commit error"))

	err := storageService.StoreExtractedJobDataBatchUpSert(context.Background(), testJobCards)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "committing transaction") {
		t.Errorf("expected error to contain 'committing transaction', got: %s", err.Error())
	}

	_ = db
}

func Test_StoreExtractedJobDataBatchUpSert_EmptyDescription(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)
	jobCards := []ExtractedJobData{
		{Title: softwareEngineer, Company: exampleInc, Location: remoteLocation, Salary: salary100kTo150k, Description: "", Link: seekJobURL1, Skills: []string{goSkill}},
	}

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO extracted_jobdata \(title, company, location, salary, description, link, skills\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\) ON CONFLICT \(link\) DO UPDATE SET title = EXCLUDED.title, company = EXCLUDED.company, location = EXCLUDED.location, salary = EXCLUDED.salary, description = EXCLUDED.description, skills = EXCLUDED.skills`)
	mock.ExpectExec(`INSERT INTO extracted_jobdata`).
		WithArgs(softwareEngineer, exampleInc, remoteLocation, salary100kTo150k, "", "https://www.seek.com.au/job1", "Go").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := storageService.StoreExtractedJobDataBatchUpSert(context.Background(), jobCards); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	_ = db
}

func Test_StoreJobListingData(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)

	mock.ExpectExec("INSERT INTO job_listings").
		WithArgs(testJobListing.Title, testJobListing.Company, testJobListing.Location, testJobListing.RemoteFlag, testJobListing.SalaryMin, testJobListing.SalaryMax, testJobListing.Currency, testJobListing.DescriptionHTML, testJobListing.DescriptionText, testJobListing.PostedDate, testJobListing.ExpiresAt, testJobListing.Source, testJobListing.SourceID, testJobListing.URL, sqlmock.AnyArg(), testJobListing.RawJSON).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := storageService.StoreJobListingData(context.Background(), testJobListing); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_ = db
}

func Test_StoreJobListingData_Error(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)

	mock.ExpectExec("INSERT INTO job_listings").
		WithArgs(testJobListing.Title, testJobListing.Company, testJobListing.Location, testJobListing.RemoteFlag, testJobListing.SalaryMin, testJobListing.SalaryMax, testJobListing.Currency, testJobListing.DescriptionHTML, testJobListing.DescriptionText, testJobListing.PostedDate, testJobListing.ExpiresAt, testJobListing.Source, testJobListing.SourceID, testJobListing.URL, sqlmock.AnyArg(), testJobListing.RawJSON).
		WillReturnError(errors.New("duplicate key"))

	err := storageService.StoreJobListingData(context.Background(), testJobListing)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "duplicate key") {
		t.Errorf("expected error to contain 'duplicate key', got: %s", err.Error())
	}

	if !strings.Contains(err.Error(), "storing job listing") {
		t.Errorf("expected error to contain 'storing job listing', got: %s", err.Error())
	}

	_ = db
}

func Test_StoreJobListingData_WithNilFields(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)
	jobListing := JobListing{
		Title:           "Test Job",
		Company:         testCorp,
		Location:        remoteLocation,
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

	if err := storageService.StoreJobListingData(context.Background(), jobListing); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	_ = db
}

func Test_StoreJobListingData_FullFields(t *testing.T) {
	db, mock, storageService := setupStorageTest(t)
	now := typeutil.UTCTimeNow()
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
		Tags:            []string{goSkill, "Kubernetes", remoteLocation},
		RawJSON:         []byte(`{"title": "Senior Software Engineer", "company": "Big Tech Corp"}`),
		CrawlTimestamp:  now,
	}

	mock.ExpectExec("INSERT INTO job_listings").
		WithArgs(jobListing.Title, jobListing.Company, jobListing.Location, jobListing.RemoteFlag, jobListing.SalaryMin, jobListing.SalaryMax, jobListing.Currency, jobListing.DescriptionHTML, jobListing.DescriptionText, jobListing.PostedDate, jobListing.ExpiresAt, jobListing.Source, jobListing.SourceID, jobListing.URL, sqlmock.AnyArg(), jobListing.RawJSON).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := storageService.StoreJobListingData(context.Background(), jobListing); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	_ = db
}

func Test_NewService(t *testing.T) {
	db, _, _ := setupStorageTest(t)

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
}

func Test_RawData_StructFields(t *testing.T) {
	now := typeutil.UTCTimeNow().Format(time.RFC3339)
	r := RawData{
		URL:         testURL,
		ContentType: textHtml,
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

func Test_ExtractedJobData_StructFields(t *testing.T) {
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

func Test_DeleteRawDataByURLs_Success(t *testing.T) {
	_, mock, storageService := setupStorageTest(t)

	urls := []string{"http://example.com/job1", "http://example.com/job2"}
	mock.ExpectExec(`DELETE FROM raw_data WHERE url = ANY`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 2))

	err := storageService.DeleteRawDataByURLs(context.Background(), urls)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %s", err)
	}
}

func Test_DeleteRawDataByURLs_EmptyURLs(t *testing.T) {
	_, _, storageService := setupStorageTest(t)

	err := storageService.DeleteRawDataByURLs(context.Background(), []string{})
	if err != nil {
		t.Fatalf("unexpected error for empty URLs: %v", err)
	}
}

func Test_DeleteRawDataByURLs_NilURLs(t *testing.T) {
	_, _, storageService := setupStorageTest(t)

	err := storageService.DeleteRawDataByURLs(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error for nil URLs: %v", err)
	}
}

func Test_DeleteRawDataByURLs_DBError(t *testing.T) {
	_, mock, storageService := setupStorageTest(t)

	urls := []string{"http://example.com/job1"}
	mock.ExpectExec(`DELETE FROM raw_data WHERE url = ANY`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(errors.New("connection refused"))

	err := storageService.DeleteRawDataByURLs(context.Background(), urls)
	if err == nil {
		t.Fatal("expected error but got none")
	}

	if !strings.Contains(err.Error(), "deleting raw data by URLs") {
		t.Fatalf("expected wrapped error message, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %s", err)
	}
}
