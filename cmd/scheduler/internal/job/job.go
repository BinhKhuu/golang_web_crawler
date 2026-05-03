package job

import (
	"context"
	"database/sql"
	"fmt"
	"golangwebcrawler/internal/models"
	"golangwebcrawler/internal/storage"
	"log/slog"
	"time"
)

// JobType represents the type of job to be executed.
type JobType int

const (
	Crawl JobType = iota
	Parse
)

func (jt JobType) String() string {
	return [...]string{"Crawl", "Parse"}[jt]
}

// Job is the interface for all schedulable units of work.
type Job interface {
	// Type returns the job type.
	Type() JobType
	// Execute runs the job to completion.
	Execute(ctx context.Context) error
}

// CrawlFunc performs the actual crawl operation.
type CrawlFunc func(ctx context.Context) error

// CrawlJob wraps a crawl execution function.
type CrawlJob struct {
	ExecuteFn CrawlFunc
	Logger    *slog.Logger
}

// Type returns Crawl.
func (j *CrawlJob) Type() JobType {
	return Crawl
}

// Execute runs the crawl operation.
func (j *CrawlJob) Execute(ctx context.Context) error {
	j.Logger.Info("starting crawl job")

	if err := j.ExecuteFn(ctx); err != nil {
		return fmt.Errorf("crawl job failed: %w", err)
	}

	j.Logger.Info("crawl job completed")
	return nil
}

// ParseFunc creates a parser and executes parsing on raw data.
type ParseFunc func(ctx context.Context) error

// ParseJob wraps a parse execution function.
type ParseJob struct {
	ExecuteFn ParseFunc
	Logger    *slog.Logger
}

// NewParseJob creates a ParseJob with the given configuration.
func NewParseJob(cfg *ParseConfig) *ParseJob {
	executeFn := func(ctx context.Context) error {
		return executeParse(ctx, cfg)
	}
	return &ParseJob{
		ExecuteFn: executeFn,
		Logger:    cfg.Logger,
	}
}

// Type returns Parse.
func (j *ParseJob) Type() JobType {
	return Parse
}

// Execute runs the parse operation.
func (j *ParseJob) Execute(ctx context.Context) error {
	j.Logger.Info("starting parse job")

	if err := j.ExecuteFn(ctx); err != nil {
		return fmt.Errorf("parse job failed: %w", err)
	}

	j.Logger.Info("parse job completed")
	return nil
}

// StorageJob abstracts the storage layer for parse jobs.
type StorageJob interface {
	GetLatestRawData(ctx context.Context, startDate time.Time) ([]storage.RawData, error)
	StoreExtractedJobDataBatchUpSert(ctx context.Context, results []storage.ExtractedJobData) error
}

// ParserJob abstracts the LLM-based parser.
type ParserJob interface {
	ParseLLM(ctx context.Context, html string) ([]models.ExtractedJobData, error)
}

// ParseConfig holds configuration for a parse job.
type ParseConfig struct {
	Storage   StorageJob
	ParserFn  func() (ParserJob, error)
	Logger    *slog.Logger
	StartDate time.Time
	BatchSize int
}

func executeParse(ctx context.Context, cfg *ParseConfig) error {
	rawData, err := cfg.Storage.GetLatestRawData(ctx, cfg.StartDate)
	if err != nil {
		return fmt.Errorf("failed to fetch raw data: %w", err)
	}

	if len(rawData) == 0 {
		cfg.Logger.Info("no raw data to parse")
		return nil
	}

	p, err := cfg.ParserFn()
	if err != nil {
		return fmt.Errorf("failed to create parser: %w", err)
	}

	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}

	batch := make([]storage.ExtractedJobData, 0, batchSize)
	for _, item := range rawData {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("parse cancelled: %w", err)
		}

		results, err := p.ParseLLM(ctx, item.RawContent)
		if err != nil {
			cfg.Logger.Warn("failed to parse raw data", "url", item.URL, "error", err)
			continue
		}

		for _, r := range results {
			batch = append(batch, storage.ExtractedJobData{
				Title:       r.Title,
				Company:     r.Company,
				Location:    r.Location,
				Salary:      r.Salary,
				Description: r.Description,
				Skills:      r.Skills,
				Link:        item.URL,
			})
		}

		if len(batch) >= batchSize {
			if storeErr := cfg.Storage.StoreExtractedJobDataBatchUpSert(ctx, batch); storeErr != nil {
				cfg.Logger.Warn("failed to store batch", "error", storeErr)
			}
			batch = make([]storage.ExtractedJobData, 0, batchSize)
		}
	}

	if len(batch) > 0 {
		if err := cfg.Storage.StoreExtractedJobDataBatchUpSert(ctx, batch); err != nil {
			cfg.Logger.Warn("failed to store final batch", "error", err)
		}
	}

	return nil
}

// NewDBParserCreator returns a function that creates a parser from the database.
func NewDBParserCreator(db *sql.DB) func() (ParserJob, error) {
	return func() (ParserJob, error) {
		return NewDBParser(db)
	}
}

// DBParser wraps the generic parser for job listings.
type DBParser struct {
	db *sql.DB
}

// NewDBParser creates a new database-backed parser.
func NewDBParser(db *sql.DB) (*DBParser, error) {
	return &DBParser{db: db}, nil
}

// ParseLLM delegates to the underlying parser implementation.
func (p *DBParser) ParseLLM(ctx context.Context, html string) ([]models.ExtractedJobData, error) {
	return parseJobListing(ctx, p.db, html)
}

// parseJobListing is a placeholder that will be wired to the actual parser.
// It's defined here to avoid importing internal packages.
var parseJobListing func(ctx context.Context, db *sql.DB, html string) ([]models.ExtractedJobData, error)

// SetParseJobListing sets the actual parser implementation.
func SetParseJobListing(fn func(ctx context.Context, db *sql.DB, html string) ([]models.ExtractedJobData, error)) {
	parseJobListing = fn
}

const defaultBatchSize = 100
