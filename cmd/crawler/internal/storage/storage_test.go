package storage

import (
	"golangwebcrawler/cmd/crawler/internal/models"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/lib/pq" // postgres driver
)

// this test is persistant and needs a database to spin up maybe use docker in the tests but for now ignore and comment out
// func Test_Store(t *testing.T) {
// 	conStr := "postgres://myuser:mypassword@localhost:5433/jobs_webcrawler?sslmode=disable"
// 	conn, err := sql.Open("postgres", conStr)
// 	if err != nil {
// 		t.Fatalf("failed to open connection: %v", err)
// 	}
// 	defer conn.Close()

// 	if err := conn.Ping(); err != nil {
// 		t.Fatalf("failed to ping database: %v", err)
// 	}
// }

func Test_StoreRawData_Upserts(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close() //nolint:errcheck

	storage := NewDBStorageService(db)
	rawData := models.RawData{
		URL:         "http://example.com",
		ContentType: "text/html",
		Raw_content: "<html><body>Example</body></html>",
	}

	mock.ExpectExec("INSERT INTO raw_data").
		WithArgs(rawData.URL, rawData.ContentType, string(rawData.Raw_content)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := storage.StoreRawData(rawData); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
