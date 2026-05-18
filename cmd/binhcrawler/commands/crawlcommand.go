package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"golangwebcrawler/cmd/binhcrawler/internal/job"
	"golangwebcrawler/cmd/binhcrawler/internal/orchestrator"
	"golangwebcrawler/internal/crawler"
	"golangwebcrawler/internal/dbstore"
	"golangwebcrawler/internal/fetcher/playwrightfetcher"
	"golangwebcrawler/internal/storage"
	"golangwebcrawler/internal/typeutil"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	crawlerparser "golangwebcrawler/internal/crawlerparser"
)

const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"

	DefaultConfigPath = "configs/seek.json"
)

var setupDatabaseFn = dbstore.SetupDatabase

// CrawlCommand defines the 'crawl' subcommand.
type CrawlCommand struct {
	BaseCommand
	GlobalOpts `group:"Global Options"`

	URL         string `default:""           description:"Target URL to crawl (overrides config file)"          long:"url"         short:"u"`
	MaxDepth    int    `default:"3"          description:"Maximum crawl depth"                                  long:"max-depth"   short:"D"`
	Concurrency int    `default:"10"         description:"Number of concurrent crawls"                          long:"concurrency" short:"c"`
	Mode        string `default:"sequential" description:"Execution mode (sequential, concurrent, independent)" long:"mode"        short:"m"`
	// Headless defaults to false (headed mode) to avoid bot detection.
	// Sites like Seek use browser fingerprinting; a visible browser window
	// with a real user profile is less likely to be flagged as automated.
	// Pass --headless to enable headless mode for CI/CD environments.
	Headless   bool   `description:"Run browser in headless mode"                  long:"headless"`
	Query      string `default:""                                                  description:"Search query (overrides config file)"             long:"query"   short:"q"`
	Timeout    int    `default:"0"                                                 description:"Playwright timeout in ms (overrides config file)" long:"timeout" short:"t"`
	ParseAfter bool   `description:"Automatically run parse after crawl completes" long:"parse"`
	ConfigFile string `default:"configs/seek.json"                                 description:"Path to site configuration JSON file"             long:"config"  short:"f"`
}

func (c *CrawlCommand) Execute(_ []string) error {
	c.Logger.Info("Starting crawl command")

	// todo name _ db when its ready for use
	db, dbErr := InitDb()
	if dbErr != nil {
		c.Logger.Error("error setting up database")
		return dbErr
	}
	defer func() {
		if dbCloseErr := db.Close(); dbCloseErr != nil {
			c.Logger.Error("error closing database")
		} else {
			c.Logger.Info("closed database")
		}
	}()

	pwConfig, pwErr := buildPlaywrightFetcherConfig(c, c.Logger)
	if pwErr != nil {
		c.Logger.Error("failed to build playwright config", "error", pwErr)
		return pwErr
	}
	// 4. Create CrawlJob with configured parameters
	fetcher, fetchErr := playwrightfetcher.NewConfiguredPlaywrightFetcher(c.Logger, &pwConfig)
	if fetchErr != nil {
		c.Logger.Error("failed to create playwright fetcher", "error", fetchErr)
		return fmt.Errorf("create playwright fetcher: %w", fetchErr)
	}
	defer fetcher.Close()

	storageSvc := storage.NewService(db, c.Logger)

	allowedDomains := extractAllowedDomains(pwConfig.URL)
	crawlJob := newCrawlJob(pwConfig.URL, fetcher, storageSvc, c.MaxDepth, allowedDomains, c.Concurrency, c.Logger)
	jobs := []job.Job{crawlJob}

	if c.ParseAfter {
		parseJob := newParseJob(storageSvc, db, c.Logger)
		jobs = append(jobs, parseJob)
	}

	mode, modeErr := parseMode(c.Mode)
	if modeErr != nil {
		c.Logger.Error("invalid execution mode", "error", modeErr)
		return fmt.Errorf("parse mode: %w", modeErr)
	}

	orch := orchestrator.New(jobs, mode, c.Logger)

	if err := orch.Run(context.Background()); err != nil {
		return fmt.Errorf("orchestrator run: %w", err)
	}

	c.Logger.Info("Finished crawl command")
	return nil
}

// extractAllowedDomains derives the allowed domain from the target URL.
// It strips the first hostname label so subdomains are included in scope
// (e.g. "www.seek.com.au" -> "seek.com.au"). Returns []string to match
// crawler.NewCrawler's allowedDomains parameter — only one domain is ever
// derived from a single URL. Multiple domains would require an explicit CLI flag.
func extractAllowedDomains(targetURL string) []string {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return []string{}
	}
	host := parsed.Hostname()
	if _, after, ok := strings.Cut(host, "."); ok {
		return []string{after}
	}
	return []string{host}
}

func newCrawlJob(startURL string, fetcher crawler.Fetcher, stor *storage.Service, maxDepth int, allowedDomains []string, concurrency int, logger *slog.Logger) *job.CrawlJob {
	crawlFn := func(ctx context.Context) error {
		c := crawler.NewCrawler(maxDepth, allowedDomains, logger)
		p := crawlerparser.NewHTTPParser()
		return c.Crawl(ctx, startURL, fetcher, p, stor, concurrency)
	}
	return &job.CrawlJob{
		ExecuteFn: crawlFn,
		Logger:    logger,
	}
}

const defaultBatchSize = 100

func newParseJob(stor *storage.Service, db *sql.DB, logger *slog.Logger) *job.ParseJob {
	startTime := typeutil.UTCTimeNow().Add(-1 * time.Minute)
	return job.NewParseJob(&job.ParseConfig{
		Storage:   stor,
		ParserFn:  job.NewDBParserCreator(db),
		Logger:    logger,
		StartDate: startTime,
		BatchSize: defaultBatchSize,
	})
}

func parseMode(modeStr string) (orchestrator.Mode, error) {
	switch strings.ToLower(modeStr) {
	case "", "sequential":
		return orchestrator.Sequential, nil
	case "concurrent":
		return orchestrator.Concurrent, nil
	case "independent":
		return orchestrator.Independent, nil
	default:
		return orchestrator.Sequential, fmt.Errorf("invalid mode %q: must be sequential, concurrent, or independent", modeStr)
	}
}

func InitDb() (*sql.DB, error) {
	db, dbErr := setupDatabaseFn()
	if dbErr != nil {
		return db, dbErr
	}
	return db, dbErr
}

func buildPlaywrightFetcherConfig(c *CrawlCommand, logger *slog.Logger) (playwrightfetcher.PlaywrightFetcherConfig, error) {
	config := playwrightfetcher.DefaultConfig()

	if c.ConfigFile != "" {
		if err := loadJSONConfig(c.ConfigFile, &config); err != nil {
			if c.ConfigFile == DefaultConfigPath {
				logger.Warn("no config file at " + DefaultConfigPath + ", using built-in defaults for seek.com.au — see cmd/binhcrawler/configs/seek.json for reference")
			} else {
				return playwrightfetcher.PlaywrightFetcherConfig{}, err
			}
		}
	}

	if c.URL != "" {
		config.URL = c.URL
	}
	if c.Timeout > 0 {
		config.Timeout = c.Timeout
	}
	if c.Query != "" {
		config.Search.Query = c.Query
	}
	if c.Headless {
		config.Headless = c.Headless
	}

	return config, nil
}

func loadJSONConfig(path string, target *playwrightfetcher.PlaywrightFetcherConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file %q: %w", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("parse config file %q: %w", path, err)
	}
	return nil
}
