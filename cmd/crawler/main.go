package main

import (
	"database/sql"
	"golangwebcrawler/cmd/crawler/internal/crawler"
	"golangwebcrawler/cmd/crawler/internal/fetcher"
	"golangwebcrawler/cmd/crawler/internal/parser"
	"golangwebcrawler/cmd/crawler/internal/storage"
	"log"
	"net/http"

	_ "github.com/lib/pq"
)

func main() {
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

// todo source connection string from enviorment variables and not hard coded in the codebase
func setupDatabase() (*sql.DB, error) {
	conStr := "postgres://myuser:mypassword@localhost:5433/jobs_webcrawler?sslmode=disable"
	conn, err := sql.Open("postgres", conStr)
	if err != nil {
		log.Fatalf("failed to open connection: %v", err)
	}

	if err := conn.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	return conn, nil
}
