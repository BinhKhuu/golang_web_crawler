package storage_test

import (
	"context"
	"database/sql"
	"golangwebcrawler/cmd/crawler/internal/config"
	"golangwebcrawler/internal/dbstore"
	"golangwebcrawler/internal/models"
	"golangwebcrawler/internal/storage"
	"golangwebcrawler/internal/testhelpers"
	"log/slog"
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

func Test_StoreRawDataBatch_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	svc := storage.NewService(db, slog.Default())
	items := []models.RawDataItem{
		{URL: "http://example.com/1", ContentType: "text/html", RawContent: "<html>1</html>"},
		{URL: "http://example.com/2", ContentType: "text/html", RawContent: "<html>2</html>"},
		{URL: "http://example.com/3", ContentType: "text/html", RawContent: "<html>3</html>"},
	}

	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO raw_data")
	for _, item := range items {
		mock.ExpectExec("INSERT INTO raw_data").
			WithArgs(item.URL, item.ContentType, item.RawContent).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()

	if err := svc.StoreRawDataBatch(context.Background(), items); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func Test_StoreRawDataBatch_EmptyItems(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	svc := storage.NewService(db, slog.Default())

	if err := svc.StoreRawDataBatch(context.Background(), []models.RawDataItem{}); err != nil {
		t.Fatalf("expected no error for empty items, got %v", err)
	}
}

func Test_StoreRawDataBatch_RollbackOnError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	svc := storage.NewService(db, slog.Default())
	items := []models.RawDataItem{
		{URL: "http://example.com/1", ContentType: "text/html", RawContent: "<html>1</html>"},
		{URL: "http://example.com/2", ContentType: "text/html", RawContent: "<html>2</html>"},
	}

	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO raw_data")
	mock.ExpectExec("INSERT INTO raw_data").
		WithArgs(items[0].URL, items[0].ContentType, items[0].RawContent).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO raw_data").
		WithArgs(items[1].URL, items[1].ContentType, items[1].RawContent).
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	err = svc.StoreRawDataBatch(context.Background(), items)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
