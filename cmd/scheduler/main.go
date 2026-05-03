package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golangwebcrawler/cmd/scheduler/internal/job"
	"golangwebcrawler/cmd/scheduler/internal/orchestrator"
	"golangwebcrawler/internal/crawler"
	crawlerparser "golangwebcrawler/internal/crawlerparser"
	"golangwebcrawler/internal/dbstore"
	"golangwebcrawler/internal/fetcher/playwrightfetcher"
	"golangwebcrawler/internal/models"
	parserpkg "golangwebcrawler/internal/parser"
	"golangwebcrawler/internal/storage"

	"github.com/joho/godotenv"
)

const (
	defaultMaxDepth    = 3
	defaultConcurrency = 10
	defaultBatchSize   = 100
)

func init() {
	job.SetParseJobListing(func(ctx context.Context, db *sql.DB, html string) ([]models.ExtractedJobData, error) {
		p, err := parserpkg.NewParser[models.JobListing](db)
		if err != nil {
			return nil, err
		}
		return p.ParseLLM(ctx, html)
	})
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := godotenv.Load(".env"); err != nil {
		logger.Warn("failed to load .env file", "error", err)
	}

	database, err := dbstore.SetupDatabase()
	if err != nil {
		logger.Error("error setting up database", "error", err)
		return
	}
	defer database.Close()

	storageSvc := storage.NewService(database, logger)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := runScheduler(ctx, logger, database, storageSvc); err != nil {
		logger.Error("scheduler failed", "error", err)
	}
}

func runScheduler(ctx context.Context, logger *slog.Logger, database *sql.DB, storageSvc *storage.Service) error {
	pwCfg := playwrightfetcher.GetSeekConfiguration()
	fetcher, err := playwrightfetcher.NewConfiguredPlaywrightFetcher(logger, &pwCfg)
	if err != nil {
		return fmt.Errorf("failed to create playwright fetcher: %w", err)
	}
	defer fetcher.Close()

	startTime := time.Now().UTC().Add(-1 * time.Minute)

	crawlJob := newCrawlJob(pwCfg.URL, fetcher, storageSvc, logger)
	parseJob := newParseJob(storageSvc, database, logger, startTime)

	orch := orchestrator.New([]job.Job{crawlJob, parseJob}, orchestrator.Sequential, logger)

	return orch.Run(ctx)
}

func newCrawlJob(url string, fetcher crawler.Fetcher, stor *storage.Service, logger *slog.Logger) *job.CrawlJob {
	crawlFn := func(ctx context.Context) error {
		c := crawler.NewCrawler(defaultMaxDepth, []string{"seek.com.au"}, logger)
		p := crawlerparser.NewHTTPParser()

		return c.Crawl(ctx, url, fetcher, p, stor, defaultConcurrency)
	}
	return &job.CrawlJob{
		ExecuteFn: crawlFn,
		Logger:    logger,
	}
}

func newParseJob(stor *storage.Service, db *sql.DB, logger *slog.Logger, startTime time.Time) *job.ParseJob {
	return job.NewParseJob(&job.ParseConfig{
		Storage:   stor,
		ParserFn:  job.NewDBParserCreator(db),
		Logger:    logger,
		StartDate: startTime,
		BatchSize: defaultBatchSize,
	})
}
