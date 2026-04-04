package storage

import (
	"context"
	"database/sql"
	"golangwebcrawler/cmd/crawler/internal/config"
	"golangwebcrawler/internal/dbstore"
	"golangwebcrawler/internal/models"
	"golangwebcrawler/internal/testhelpers"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/lib/pq" // postgres driver
)

func Test_Migrations(t *testing.T) {
	testhelpers.SetTestEnvs(t)
	conStr, err := dbstore.GetConnectionString()
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), config.QueryTimeout)
	conn, err := sql.Open("postgres", conStr)
	defer cancel()
	if err != nil {
		t.Fatalf("failed to open connection: %v", err)
	}
	defer conn.Close()

	if conErr := conn.PingContext(ctx); conErr != nil {
		t.Fatalf("failed to ping database: %v", conErr)
	}

	var count int
	defer cancel()
	err = conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'raw_data'`).Scan(&count)
	if err != nil {
		t.Fatalf("failed to check table existence: %v", err)
	}

	if count == 0 {
		t.Fatal("table raw_data does not exist")
	}

	err = conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'job_listings'`).Scan(&count)
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
	defer db.Close()

	storage := NewDBStorageService(db)
	rawData := models.RawData{
		URL:         "http://example.com",
		ContentType: "text/html",
		RawContent:  "<html><body>Example</body></html>",
	}

	mock.ExpectExec("INSERT INTO raw_data").
		WithArgs(rawData.URL, rawData.ContentType, rawData.RawContent).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := storage.StoreRawData(context.Background(), rawData); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
