package playwrightfetcher

import (
	"context"
	"errors"
	"fmt"
	"golangwebcrawler/cmd/crawler/internal/crawler"
	"log/slog"

	"github.com/playwright-community/playwright-go"
)

var (
	ErrPlaywrightClose = errors.New("closing playwright browser")
	ErrPlaywrightStop  = errors.New("stopping playwright")
)

const defaultTimeout = 10000

type PlaywrightFetcher struct {
	logger      *slog.Logger
	fetchConfig *PlaywrightFetcherConfig
	pw          *playwright.Playwright
	browser     playwright.Browser
	browserCtx  playwright.BrowserContext
}

/*
PlaywrightFetcherConfig
Leave query and selectors empty to skip the step.
*/
type PlaywrightFetcherConfig struct {
	URL                   string   `json:"url"`
	Headless              bool     `json:"headless"`
	SearchInputSelectors  []string `json:"searchInputSelectors"`
	SearchQuery           string   `json:"searchQuery"`
	SearchSubmitSelectors []string `json:"searchSubmitSelectors"`
	ResultsSelectors      []string `json:"resultsSelectors"`
	DataSelectors         []string `json:"dataSelectors"`
	SPAUpdateSelectors    []string `json:"spaUpdateSelectors"`
	Timeout               int      `json:"timeout"`
}

func NewPlaywrightFetcher(logger *slog.Logger, fetchConfig *PlaywrightFetcherConfig) (*PlaywrightFetcher, error) {
	f := &PlaywrightFetcher{
		logger:      logger,
		fetchConfig: fetchConfig,
	}

	err := f.configurePlaywrightBrowser()
	if err != nil {
		logger.Error("error configuring playwright browser", "error", err)
		return nil, err
	}
	return f, nil
}

// todo add channel and defer ctx close logic.
/*
	1. Introduce a playwright configuration (and default configuration)
		- inside configuration have selectors, searchquery, search submit selectors etc
		- all the information needed for a configuration to target a specific website
		- the fetcher will be generic and look for these selectors and store the information if it can
		- might need a SPA configuration or SPA function thats separate from the regular page refresher
			- spa will have better luck looking at network traffic to get the data
	GUARD against nil configuration

		if f.fetchConfig == nil {
			return crawler.FetchResult{}, errors.New("fetch config is nil")
		}

		// slices - check before ranging or indexing
		if len(f.fetchConfig.SearchInputSelectors) > 0 {
			// do search input logic
		}

		if len(f.fetchConfig.SearchSubmitSelectors) > 0 {
			// do search submit logic
		}

		if len(f.fetchConfig.ResultsSelectors) > 0 {
			// do results logic
		}

		if len(f.fetchConfig.DataSelectors) > 0 {
			// do data extraction logic
		}

		if len(f.fetchConfig.SPAUpdateSelectors) > 0 {
			// do SPA logic
		}
	2. Add random delays and human-like behavior
	3. the Fetch via playwright will be generice
		1. Load page
		2. Wait for dom to load and first selector to be available
		3. Add cookies and email and promoto click acceptors
		3. If Searching then run search confiruation
		4. If Clicking run clicking configuration
			- click selector
			- look for details
				- check html and network
				- if details store


*/
func (f *PlaywrightFetcher) Fetch(ctx context.Context, url string) (crawler.FetchResult, error) {
	if f.fetchConfig == nil {
		return crawler.FetchResult{}, nil
	}

	p, err := f.browserCtx.NewPage()
	if err != nil {
		return crawler.FetchResult{}, err
	}
	// 4. Add random delays and human-like behavior
	const timeoutInMs = 30000
	_, err = p.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(timeoutInMs),
	})
	if err != nil {
		return crawler.FetchResult{}, err
	}

	locator := p.Locator("a[id*='job-title']")

	err = locator.WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateVisible,
	})
	if err != nil {
		return crawler.FetchResult{}, err
	}

	entries, err := p.Locator(`a[id*='job-title']`).All()
	if err != nil {
		return crawler.FetchResult{}, err
	}

	for _, entry := range entries {
		text, err := entry.TextContent()
		if err != nil {
			text = "error getting text content"
		}
		f.logger.Info("playwright fetcher", "url", url, "content", text)
	}

	return crawler.FetchResult{}, nil
}

// Close Call to prevent resource leaks. Should be deferred right after creating the fetcher instance.
func (f *PlaywrightFetcher) Close() error {
	if f.browser != nil {
		if err := f.browser.Close(); err != nil {
			return fmt.Errorf("%w: %w", ErrPlaywrightClose, err)
		}
	}
	if f.pw != nil {
		if err := f.pw.Stop(); err != nil {
			return fmt.Errorf("%w: %w", ErrPlaywrightStop, err)
		}
	}
	return nil
}

// configurePlaywrightBrowser sets up a Playwright browser instance with enhanced stealth options to better mimic human behavior and avoid detection by anti-bot measures.
// will launch a browser in headed mode to prevent bot detection.
func (f *PlaywrightFetcher) configurePlaywrightBrowser() error {
	pw, err := playwright.Run()
	if err != nil {
		return err
	}

	b, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--disable-features=IsolateOrigins,site-per-process",
			"--disable-site-isolation-trials",
			"--disable-web-security",
			"--disable-features=BlockInsecurePrivateNetworkRequests",
			"--user-agent=Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		},
	})
	if err != nil {
		return err
	}

	const width = 1920
	const height = 1080
	ops := playwright.BrowserNewContextOptions{
		UserAgent:         playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"),
		Viewport:          &playwright.Size{Width: width, Height: height}, // Use ViewportSize pointer
		Locale:            playwright.String("en-US"),
		TimezoneId:        playwright.String("America/New_York"),
		Permissions:       []string{"geolocation", "notifications"}, // Add notifications to look more "human"
		JavaScriptEnabled: playwright.Bool(true),
		IgnoreHttpsErrors: playwright.Bool(true),
		HasTouch:          playwright.Bool(false),
	}

	// 2. More comprehensive script injection
	bctx, err := b.NewContext(ops)
	if err != nil {
		return err
	}

	// 3. Enhanced stealth scripts
	err = bctx.AddInitScript(playwright.Script{
		Content: playwright.String(`
            // Remove webdriver property
            Object.defineProperty(navigator, 'webdriver', {get: () => undefined});
            
            // Override plugins length
            Object.defineProperty(navigator, 'plugins', {get: () => [1, 2, 3, 4, 5]});
            
            // Override languages
            Object.defineProperty(navigator, 'languages', {get: () => ['en-US', 'en']});
            
            // Chrome runtime
            window.chrome = {
                runtime: {}
            };
            
            // Permissions
            const originalQuery = window.navigator.permissions.query;
            window.navigator.permissions.query = (parameters) => (
                parameters.name === 'notifications' ?
                    Promise.resolve({state: Notification.permission}) :
                    originalQuery(parameters)
            );
        `),
	})
	if err != nil {
		return err
	}

	f.pw = pw
	f.browser = b
	f.browserCtx = bctx
	return nil
}

func DefaultConfig() PlaywrightFetcherConfig {
	return PlaywrightFetcherConfig{
		URL:      "https://www.seek.com.au",
		Headless: true,
		SearchInputSelectors: []string{
			"input[name=keywords]",
			"input[placeholder*='Search']",
		},
		SearchQuery: "Software Engineer Jobs",
		SearchSubmitSelectors: []string{
			"button[type=submit]",
			"button[aria-label='Search']",
		},
		ResultsSelectors: []string{
			"a[data-automation='jobTitle']",
			"a.job-link",
			"a[data-testid='job-result']",
		},
		DataSelectors: []string{
			"a[id*='job-title']",
			"a[data-automation='jobTitle']",
			".job-title a",
			"article a[href*='/job/']",
		},
		SPAUpdateSelectors: []string{
			"#job-details",
			".JobDetail",
			"[data-automation='jobDetail']",
			".job-detail-content",
		},
		Timeout: defaultTimeout,
	}
}
