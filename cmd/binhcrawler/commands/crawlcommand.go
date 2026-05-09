package commands

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
	// 1. Setup logger with configured log level
	// 2. Setup database connection
	// 3. Build PlaywrightFetcherConfig from flags
	// 4. Create CrawlJob with configured parameters
	// 5. If ParseAfter, also create ParseJob
	// 6. Parse Mode string to orchestrator.Mode
	// 7. Run orchestrator

	c.Logger.Info("Finished crawl command")
	return nil
}
