package playwrightfetcher

import (
	"context"
	"errors"
	"fmt"
	"golangwebcrawler/cmd/crawler/internal/crawler"
	"log/slog"
	"math/rand"
	"strings"
	"time"

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
	fetchFn     func(ctx context.Context, url string) (crawler.FetchResult, error)
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

	f.fetchFn = f.FetchDefault

	return configurePlaywright(f, logger)
}

func NewConfiguredPlaywrightFetcher(logger *slog.Logger, config *PlaywrightFetcherConfig) (*PlaywrightFetcher, error) {
	f := &PlaywrightFetcher{
		logger:      logger,
		fetchConfig: config,
	}
	f.fetchFn = f.FetchSPAConfig

	return configurePlaywright(f, logger)
}

func configurePlaywright(f *PlaywrightFetcher, logger *slog.Logger) (*PlaywrightFetcher, error) {
	err := f.configurePlaywrightBrowser()
	if err != nil {
		logger.Error("error configuring playwright browser", "error", err)
		return nil, err
	}
	return f, nil
}

// todo add channel and defer ctx close logic.
/*
	1. Playwright Fetcher can return multiple crawler results. Need to design now Crawler can handle an array of returned items
		- might be best to change the return type to slice of FetchResults this can allow for 0 - many results
	2. Look into inspecing the network for data


*/

// Fetch implements the Fetcher interface using Playwright to fetch and render web pages, allowing for dynamic content extraction and interaction with JavaScript-heavy sites. It uses the provided configuration to determine how to navigate and extract data from the target website.
// The fetch constructors determine which Fetch implementation will be used.
func (f *PlaywrightFetcher) Fetch(ctx context.Context, url string) (crawler.FetchResult, error) {
	return f.fetchFn(ctx, url)
}

func (f *PlaywrightFetcher) FetchDefault(ctx context.Context, url string) (crawler.FetchResult, error) {
	p, err := f.browserCtx.NewPage()
	if err != nil {
		return crawler.FetchResult{}, err
	}
	defer func() {
		if err := p.Close(); err != nil {
			f.logger.Error("error closing page", "error", err)
		}
	}()
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

func (f *PlaywrightFetcher) FetchSPAConfig(ctx context.Context, url string) (crawler.FetchResult, error) {
	p, err := f.browserCtx.NewPage()
	if err != nil {
		return crawler.FetchResult{}, err
	}
	defer func() {
		if err := p.Close(); err != nil {
			f.logger.Error("error closing page", "error", err)
		}
	}()

	if f.fetchConfig == nil {
		return crawler.FetchResult{}, errors.New("fetch config is nil")
	}

	const timeoutInMs = 30000
	_, err = p.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(timeoutInMs),
	})
	if err != nil {
		return crawler.FetchResult{}, err
	}

	// stop at first successful search input selector
	if len(f.fetchConfig.SearchInputSelectors) > 0 {
		for _, sel := range f.fetchConfig.SearchInputSelectors {
			err = p.Locator(sel).Fill(f.fetchConfig.SearchQuery)
			if err == nil {
				break
			}
		}
	}

	// stop at first successful search submit selector
	if len(f.fetchConfig.SearchSubmitSelectors) > 0 {
		for _, btn := range f.fetchConfig.SearchSubmitSelectors {
			err = p.Locator(btn).Click()
			if err == nil {
				break
			}
		}
	}

	if len(f.fetchConfig.ResultsSelectors) > 0 {
		for _, sel := range f.fetchConfig.ResultsSelectors {
			timeout := float64(f.fetchConfig.Timeout)
			locator := p.Locator(sel)
			err = locator.First().WaitFor(playwright.LocatorWaitForOptions{
				State:   playwright.WaitForSelectorStateVisible,
				Timeout: playwright.Float(timeout),
			})
			if err != nil {
				continue
			}

			entries, err := p.Locator(sel).All()
			if err == nil {
				for _, entry := range entries {
					err := entry.Click()
					randomDelay()
					if err != nil {
						f.logger.Error("error clicking entry", "error", err)
						continue
					}
					// todo store results in crawler results
					_ = fetchSPAConfigDataSelectors(f, p)
				}
			}
			break
		}
	}

	if len(f.fetchConfig.SPAUpdateSelectors) > 0 {
		// do SPA logic
	}

	return crawler.FetchResult{}, nil
}

func randomDelay() {
	// random delay between 1-3 seconds to mimic human behavior
	delay := time.Duration(1000+rand.Intn(2000)) * time.Millisecond
	time.Sleep(delay)
}

func fetchSPAConfigDataSelectors(f *PlaywrightFetcher, p playwright.Page) crawler.FetchResult {
	var texts []string
	if len(f.fetchConfig.DataSelectors) > 0 {
		for _, sel := range f.fetchConfig.DataSelectors {
			entries, err := p.Locator(sel).All()
			if err != nil {
				continue
			}

			for _, entry := range entries {
				textContent, err := entry.TextContent()
				if err != nil {
					textContent = "error getting text content"
				}
				f.logger.Info("playwright fetcher", "content", textContent)
				texts = append(texts, textContent)
			}

			break
		}
	}
	body := []byte(strings.Join(texts, "\n"))
	return crawler.FetchResult{
		URL:        p.URL(), // todo URL should be the URL of the job details page, not the search results page. need to click into the job details and extract from there.
		Body:       body,    // todo extract body content
		StatusCode: 200,     // todo extract status code
	}
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
		pw.Stop()
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
		b.Close()
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
		bctx.Close()
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
