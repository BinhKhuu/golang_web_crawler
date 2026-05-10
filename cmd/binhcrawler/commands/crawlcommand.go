package commands

import (
	"errors"
	"golangwebcrawler/internal/dbstore"
)

const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

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
	_, dbErr := dbstore.SetupDatabase()
	if dbErr != nil {
		c.Logger.Error("error setting up database")
		return errors.New("failed to set up database")
	}
	// 3. Build PlaywrightFetcherConfig from flags
	// 4. Create CrawlJob with configured parameters
	// 5. If ParseAfter, also create ParseJob
	// 6. Parse Mode string to orchestrator.Mode
	// 7. Run orchestrator

	c.Logger.Info("Finished crawl command")
	return nil
}
