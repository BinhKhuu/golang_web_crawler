package commands

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"golangwebcrawler/internal/dbstore"
	"golangwebcrawler/internal/fetcher/playwrightfetcher"
	"log/slog"
	"os"
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

	URL         string `default:""                                                  description:"Target URL to crawl (overrides config file)"          long:"url"         short:"u"`
	MaxDepth    int    `default:"3"                                                 description:"Maximum crawl depth"                                  long:"max-depth"   short:"D"`
	Concurrency int    `default:"10"                                                description:"Number of concurrent crawls"                          long:"concurrency" short:"c"`
	Mode        string `default:"sequential"                                        description:"Execution mode (sequential, concurrent, independent)" long:"mode"        short:"m"`
	Headless    bool   `default:"true"                                              description:"Run browser in headless mode"                         long:"headless"`
	Query       string `default:""                                                  description:"Search query (overrides config file)"                 long:"query"       short:"q"`
	Timeout     int    `default:"0"                                                 description:"Playwright timeout in ms (overrides config file)"     long:"timeout"     short:"t"`
	ParseAfter  bool   `description:"Automatically run parse after crawl completes" long:"parse"`
	ConfigFile  string `default:"configs/seek.json"                                 description:"Path to site configuration JSON file"                 long:"config"      short:"f"`
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
	// TODO: pass pwConfig to orchestrator / fetcher creation
	_ = pwConfig
	// 4. Create CrawlJob with configured parameters
	// 5. If ParseAfter, also create ParseJob
	// 6. Parse Mode string to orchestrator.Mode
	// 7. Run orchestrator

	c.Logger.Info("Finished crawl command")
	return nil
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
