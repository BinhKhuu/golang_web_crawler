package main

import (
	"fmt"
	"golangwebcrawler/cmd/crawler/internal/crawler"
	"golangwebcrawler/cmd/crawler/internal/fetcher"
	"golangwebcrawler/cmd/crawler/internal/parser"
	"golangwebcrawler/cmd/crawler/internal/storage"
	"golangwebcrawler/internal/dbstore"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	cfg, err := Load("../../.env")
	if err != nil {
		log.Printf("Error loading .env file: %v\n", err)
		return
	}

	database, err := dbstore.SetupDatabase()
	if err != nil {
		log.Printf("error setting up database: %v\n", err)
		return
	}
	defer database.Close()

	httpClient := http.DefaultClient
	crawler := crawler.NewCrawler(cfg.MaxDepth, cfg.AllowedDomains)
	storage := storage.NewDBStorageService(database)
	parser := parser.NewHTTPParser()
	fetcher := fetcher.NewHTTPFetcher(httpClient)

	err = crawler.CrawlAsync("https://www.seek.com.au/software-engineer-jobs", 1, fetcher, parser, storage)
	if err != nil {
		log.Printf("error starting crawl: %v\n", err)
	}
	crawler.Wait()
	log.Printf("Crawling completed.")
}

type CrawlerConfig struct {
	DBHost         string
	DBPort         string
	MaxDepth       int
	AllowedDomains []string
}

func Load(envFile string) (*CrawlerConfig, error) {
	if err := godotenv.Load(envFile); err != nil {
		return nil, fmt.Errorf("failed to load env file: %w", err)
	}

	const maxDepth = 3
	return &CrawlerConfig{
		DBHost:         os.Getenv("DB_HOST"),
		DBPort:         os.Getenv("DB_PORT"),
		MaxDepth:       maxDepth,
		AllowedDomains: []string{"seek.com.au"},
	}, nil
}
