package storage

import (
	"database/sql"
	"log/slog"
	"strings"
	"testing"

	"golangwebcrawler/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
)

// ==================== Mock Expectation Helpers ====================
// These helpers are only compiled during tests (file ends with _test.go),
// ensuring they never leak into production builds.

func setupStorageTest(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *Service) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	t.Cleanup(func() { db.Close() })
	return db, mock, NewService(db, slog.Default())
}

func expectStoreRawDataBatchSuccess(mock sqlmock.Sqlmock, items []models.RawDataItem) {
	mock.ExpectBegin()
	mock.ExpectExec(`CREATE TEMP TABLE raw_data_batch`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPrepare(`COPY "raw_data_batch" \("url", "content_type", "raw_content"\) FROM STDIN`)

	for _, item := range items {
		mock.ExpectExec(`COPY "raw_data_batch"`).
			WithArgs(item.URL, item.ContentType, item.RawContent).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}

	// pq.CopyIn requires a final Exec() with no args to flush the COPY stream.
	mock.ExpectExec(`COPY "raw_data_batch"`).WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectExec(`INSERT INTO raw_data \(url, content_type, raw_content\)\s+SELECT url, content_type, raw_content FROM raw_data_batch\s+ON CONFLICT \(url\)\s+DO UPDATE SET content_type = EXCLUDED.content_type,\s+raw_content = EXCLUDED.raw_content,\s+fetched_at = NOW\(\)`).
		WillReturnResult(sqlmock.NewResult(0, int64(len(items))))
	mock.ExpectCommit()
}

func expectStoreExtractedJobDataBatchSuccess(mock sqlmock.Sqlmock, jobCards []ExtractedJobData) {
	mock.ExpectBegin()
	mock.ExpectPrepare(`COPY "extracted_jobdata" \("title", "company", "location", "salary", "description", "link", "skills"\) FROM STDIN`)

	for _, jobCard := range jobCards {
		expectedSkills := joinStringSlice(jobCard.Skills, ",")
		mock.ExpectExec(`COPY "extracted_jobdata"`).
			WithArgs(jobCard.Title, jobCard.Company, jobCard.Location, jobCard.Salary, jobCard.Description, jobCard.Link, expectedSkills).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()
}

// joinStringSlice joins a string slice with the given separator.
func joinStringSlice(slice []string, sep string) string {
	if len(slice) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(slice[0])
	for _, s := range slice[1:] {
		b.WriteString(sep)
		b.WriteString(s)
	}
	return b.String()
}
