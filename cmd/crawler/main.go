package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"golangwebcrawler/cmd/crawler/internal/crawler"
	"golangwebcrawler/cmd/crawler/internal/fetcher"
	"golangwebcrawler/cmd/crawler/internal/parser"
	"golangwebcrawler/cmd/crawler/internal/storage"
	"log"
	"net"
	"net/http"
	"os"
	"time"

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
	defer db.Close()
	httpClient := http.DefaultClient
	allowedDomains := []string{"example.com", "iana.org"}
	crawler := crawler.NewCrawler(maxDepth, allowedDomains)
	storage := storage.NewDBStorageService(db)

	parser := parser.NewHTTPParser()

	fetcher := fetcher.NewHTTPFetcher(httpClient)

	err = crawler.CrawlAsync("http://example.com", maxDepth, fetcher, parser, storage)
	if err != nil {
		log.Fatalf("failed to crawl: %v", err)
	}
	crawler.Wait()
}

func setupDatabase() (*sql.DB, error) {
	conStr, err := getConnectionString()
	if err != nil {
		return nil, fmt.Errorf("failed to load database settings: %w", err)
	}
	conn, err := sql.Open("postgres", conStr)
	if err != nil {
		log.Fatalf("failed to open connection: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := conn.PingContext(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	return conn, nil
}

func getConnectionString() (string, error) {
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		return "", errors.New("DB_USER environment variable is not set")
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		return "", errors.New("DB_PASSWORD environment variable is not set")
	}
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		return "", errors.New("DB_HOST environment variable is not set")
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		return "", errors.New("DB_PORT environment variable is not set")
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		return "", errors.New("DB_NAME environment variable is not set")
	}
	dbSslmode := os.Getenv("DB_SSLMODE")
	if dbSslmode == "" {
		return "", errors.New("DB_SSLMODE environment variable is not set")
	}

	hostPort := net.JoinHostPort(dbHost, dbPort)
	conStr := fmt.Sprintf(`postgres://%s:%s@%s/%s?sslmode=%s`, dbUser, dbPassword, hostPort, dbName, dbSslmode)
	return conStr, nil
}
