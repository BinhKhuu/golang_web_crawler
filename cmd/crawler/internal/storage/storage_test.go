package storage

import (
	"database/sql"
	"golangwebcrawler/cmd/crawler/internal/models"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/lib/pq" // postgres driver
)

// Test_Migrations connects to the test database in workflows the credentials are in test.yaml
func Test_Migrations(t *testing.T) {
	conStr := "postgres://postgres:postgres@localhost:5432/testdb?sslmode=disable"
	conn, err := sql.Open("postgres", conStr)
	if err != nil {
		t.Fatalf("failed to open connection: %v", err)
	}
	defer conn.Close() //nolint:errcheck

	if err := conn.Ping(); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}

	var count int
	err = conn.QueryRow(`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'raw_data'`).Scan(&count)
	if err != nil {
		t.Fatalf("failed to check table existence: %v", err)
	}

	if count == 0 {
		t.Fatal("table raw_data does not exist")
	}

	err = conn.QueryRow(`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'job_listings'`).Scan(&count)
	if err != nil {
		t.Fatalf("failed to check table existence: %v", err)
	}

	if count == 0 {
		t.Fatal("table job_listings does not exist")
	}

}

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
