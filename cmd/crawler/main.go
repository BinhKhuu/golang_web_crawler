package main

import (
	"database/sql"
	"fmt"
	"golangwebcrawler/cmd/crawler/internal/crawler"
	"golangwebcrawler/cmd/crawler/internal/fetcher"
	"golangwebcrawler/cmd/crawler/internal/parser"
	"golangwebcrawler/cmd/crawler/internal/storage"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Fatal("error loading .env file")
	}

	maxDepth := 3
	db, err := setupDatabase()
	if err != nil {
		log.Fatalf("failed to set up database: %v", err)
		return
	}
	defer db.Close() //nolint:errcheck
	httpClient := http.DefaultClient
	allowedDomains := []string{"example.com", "iana.org"}
	crawler := crawler.NewCrawler(maxDepth, allowedDomains)
	storage := storage.NewDBStorageService(db)

	parser := parser.NewHttpParser()

	fetcher := fetcher.NewHTTPFetcher(httpClient)

	err = crawler.CrawlAsync("http://example.com", maxDepth, fetcher, parser, storage)
	if err != nil {
		log.Fatalf("failed to crawl: %v", err)
	}
	crawler.Wait()

}

func setupDatabase() (*sql.DB, error) {
	DB_USER, DB_PASSWORD, DB_HOST, DB_PORT, DB_NAME, DB_SSLMODE, result, err, shouldReturn := loadDBSettings()
	if shouldReturn {
		return result, err
	}
	conStr := fmt.Sprintf(`postgres://%s:%s@%s:%s/%s?sslmode=%s`, DB_USER, DB_PASSWORD, DB_HOST, DB_PORT, DB_NAME, DB_SSLMODE) //`postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}`
	conn, err := sql.Open("postgres", conStr)
	if err != nil {
		log.Fatalf("failed to open connection: %v", err)
	}

	if err := conn.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	return conn, nil
}

func loadDBSettings() (string, string, string, string, string, string, *sql.DB, error, bool) {
	DB_USER := os.Getenv("DB_USER")
	if DB_USER == "" {
		return "", "", "", "", "", "", nil, fmt.Errorf("DB_USER environment variable is not set"), true
	}
	DB_PASSWORD := os.Getenv("DB_PASSWORD")
	if DB_PASSWORD == "" {
		return "", "", "", "", "", "", nil, fmt.Errorf("DB_PASSWORD environment variable is not set"), true
	}
	DB_HOST := os.Getenv("DB_HOST")
	if DB_HOST == "" {
		return "", "", "", "", "", "", nil, fmt.Errorf("DB_HOST environment variable is not set"), true
	}
	DB_PORT := os.Getenv("DB_PORT")
	if DB_PORT == "" {
		return "", "", "", "", "", "", nil, fmt.Errorf("DB_PORT environment variable is not set"), true
	}
	DB_NAME := os.Getenv("DB_NAME")
	if DB_NAME == "" {
		return "", "", "", "", "", "", nil, fmt.Errorf("DB_NAME environment variable is not set"), true
	}
	DB_SSLMODE := os.Getenv("DB_SSLMODE")
	if DB_SSLMODE == "" {
		return "", "", "", "", "", "", nil, fmt.Errorf("DB_SSLMODE environment variable is not set"), true
	}
	return DB_USER, DB_PASSWORD, DB_HOST, DB_PORT, DB_NAME, DB_SSLMODE, nil, nil, false
}
