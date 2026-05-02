package main

import (
	"context"
	"database/sql"
	"fmt"
	"golangwebcrawler/internal/crawler"
	"golangwebcrawler/internal/fetcher/httpfetcher"
	"golangwebcrawler/internal/fetcher/playwrightfetcher"
	"golangwebcrawler/internal/crawlerparser"
	"golangwebcrawler/internal/dbstore"
	"golangwebcrawler/internal/storage"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

const (
	httpTimeout             = 30 * time.Second
	httpMaxIdleConns        = 100
	httpMaxIdleConnsPerHost = 10
	httpIdleConnTimeout     = 90 * time.Second
	defaultConcurrency      = 100
	defaultMaxDepth         = 3
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := Load(".env")
	if err != nil {
		logger.Error("Error loading .env file", "error", err)
		return
	}

	database, err := dbstore.SetupDatabase()
	if err != nil {
		logger.Error("error setting up database", "error", err)
		return
	}
	defer func() {
		if dbCloseErr := database.Close(); err != nil {
			logger.Error("error closing database connection", "error", dbCloseErr)
		}
	}()

	crawlSPA(cfg, logger, database)
	crawlHttp(cfg, logger, database)
	logger.Info("Crawling completed.")
}

func crawlSPA(cfg *CrawlerConfig, logger *slog.Logger, database *sql.DB) {
	pwCfg := playwrightfetcher.GetSeekConfiguration()
	f, err := playwrightfetcher.NewConfiguredPlaywrightFetcher(logger, &pwCfg)
	if err != nil {
		logger.Error("error creating Playwright fetcher", "error", err)
		return
	}
	p := parser.NewHTTPParser()

	c := crawler.NewCrawler(cfg.MaxDepth, cfg.AllowedDomains, logger)
	storageSvc := storage.NewService(database, logger)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	err = c.Crawl(ctx, pwCfg.URL, f, p, storageSvc, defaultConcurrency)
	if err != nil {
		logger.Error("error crawling", "error", err)
		return
	}
}

func crawlHttp(cfg *CrawlerConfig, logger *slog.Logger, database *sql.DB) {
	httpClient := &http.Client{
		Timeout: httpTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        httpMaxIdleConns,
			MaxIdleConnsPerHost: httpMaxIdleConnsPerHost,
			IdleConnTimeout:     httpIdleConnTimeout,
		},
	}

	c := crawler.NewCrawler(cfg.MaxDepth, cfg.AllowedDomains, logger)
	storageSvc := storage.NewService(database, logger)
	p := parser.NewHTTPParser()
	f := httpfetcher.NewHTTPFetcher(httpClient)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	crawlErr := c.Crawl(ctx, "https://example.com", f, p, storageSvc, defaultConcurrency)
	if crawlErr != nil {
		logger.Error("error during crawl", "error", crawlErr)
	}
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

	maxDepth := defaultMaxDepth
	if val := os.Getenv("CRAWLER_MAX_DEPTH"); val != "" {
		if _, err := fmt.Sscanf(val, "%d", &maxDepth); err != nil {
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			logger.Warn("invalid CRAWLER_MAX_DEPTH value, using default", "value", val, "default", maxDepth)
		}
	}

	allowedDomains := []string{"seek.com.au", "example.com", "iana.org"}
	if val := os.Getenv("CRAWLER_ALLOWED_DOMAINS"); val != "" {
		allowedDomains = nil
		for d := range strings.SplitSeq(val, ",") {
			if d = strings.TrimSpace(d); d != "" {
				allowedDomains = append(allowedDomains, d)
			}
		}
	}

	return &CrawlerConfig{
		DBHost:         os.Getenv("DB_HOST"),
		DBPort:         os.Getenv("DB_PORT"),
		MaxDepth:       maxDepth,
		AllowedDomains: allowedDomains,
	}, nil
}
