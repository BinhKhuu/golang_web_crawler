package commands

import (
	"database/sql"
	"golangwebcrawler/internal/dbstore"
	"golangwebcrawler/internal/fetcher/playwrightfetcher"
)

const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

var setupDatabaseFn = dbstore.SetupDatabase

// CrawlCommand defines the 'crawl' subcommand.
type CrawlCommand struct {
	BaseCommand
	GlobalOpts `group:"Global Options"`

	URL         string `default:"https://www.seek.com.au/software-engineer-jobs"    description:"Target URL to crawl"                                  long:"url"         short:"u"`
	MaxDepth    int    `default:"3"                                                 description:"Maximum crawl depth"                                  long:"max-depth"   short:"D"`
	Concurrency int    `default:"10"                                                description:"Number of concurrent crawls"                          long:"concurrency" short:"c"`
	Mode        string `default:"sequential"                                        description:"Execution mode (sequential, concurrent, independent)" long:"mode"        short:"m"`
	Headless    bool   `default:"true"                                              description:"Run browser in headless mode"                         long:"headless"`
	Query       string `default:"Software Engineer Jobs"                            description:"Search query"                                         long:"query"       short:"q"`
	Timeout     int    `default:"10000"                                             description:"Playwright timeout in ms"                             long:"timeout"     short:"t"`
	ParseAfter  bool   `description:"Automatically run parse after crawl completes" long:"parse"`
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

	// 3. Build PlaywrightFetcherConfig from flags
	buildPlaywrightFetcherConfig(c)
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

// todo implement /plans/playwright-config-plan.md
func buildPlaywrightFetcherConfig(c *CrawlCommand) (playwrightfetcher.PlaywrightFetcherConfig, error) {

	// if args are null use defult
	// pwCfg := playwrightfetcher.GetSeekConfiguration()
	var config playwrightfetcher.PlaywrightFetcherConfig

	return config, nil
}
