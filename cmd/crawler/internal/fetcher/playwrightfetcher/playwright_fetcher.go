package playwrightfetcher

import (
	"context"
	"errors"
	"fmt"
	"golangwebcrawler/cmd/crawler/internal/crawler"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/playwright-community/playwright-go"
)

var (
	ErrPlaywrightClose = errors.New("closing playwright browser error")
	ErrPlaywrightStop  = errors.New("stopping playwright error")
)

const defaultTimeout = 10000

type PlaywrightFetcher struct {
	logger      *slog.Logger
	fetchConfig *PlaywrightFetcherConfig
	pw          *playwright.Playwright
	browser     playwright.Browser
	browserCtx  playwright.BrowserContext
	fetchFn     func(ctx context.Context, url string) ([]crawler.FetchResult, error)
}

/*
PlaywrightFetcherConfig
Leave query and selectors empty to skip the step.

the first matching selector is used for each of the following steps:
SearchInputSelectors
SearchQuery
SearchSubmitSelectors
ResultsSelectors

matches multiple selectors storing results
DataSelectors.
*/
type PlaywrightFetcherConfig struct {
	URL                   string   `json:"url"`
	Headless              bool     `json:"headless"`
	SearchInputSelectors  []string `json:"searchInputSelectors"`
	SearchQuery           string   `json:"searchQuery"`
	SearchSubmitSelectors []string `json:"searchSubmitSelectors"`
	ResultsSelectors      []string `json:"resultsSelectors"`
	DataSelectors         []string `json:"dataSelectors"`
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

func (f *PlaywrightFetcher) Fetch(ctx context.Context, url string) ([]crawler.FetchResult, error) {
	return f.fetchFn(ctx, url)
}

// FetchDefault fetches the page using Playwright's default settings. Only links are traversed.
func (f *PlaywrightFetcher) FetchDefault(ctx context.Context, url string) ([]crawler.FetchResult, error) {
	if err := ctx.Err(); err != nil {
		return []crawler.FetchResult{}, err
	}

	p, err := f.browserCtx.NewPage()
	if err != nil {
		return []crawler.FetchResult{}, err
	}
	defer func() {
		if closeErr := p.Clock(); err != nil {
			f.logger.Error("error closing page", "error", closeErr)
		}
	}()

	const timeoutInMs = 30000
	if ctxErr := ctx.Err(); ctxErr != nil {
		return []crawler.FetchResult{}, ctxErr
	}
	_, err = p.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(timeoutInMs),
	})
	if err != nil {
		return []crawler.FetchResult{}, err
	}

	entries, err := p.Locator(`a[id*='job-title']`).All()
	if err != nil {
		return []crawler.FetchResult{}, err
	}

	// todo store the actual body by going to the link or look at the network
	results := []crawler.FetchResult{}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return results, err
		}

		text, err := entry.TextContent()
		if err != nil {
			text = "error getting text content"
		}
		f.logger.Info("playwright fetcher", "url", url, "content", text)
		results = append(results, crawler.FetchResult{
			URL:        url,
			Body:       []byte(text),
			StatusCode: http.StatusOK,
		})
	}

	return results, nil
}

func (f *PlaywrightFetcher) FetchSPAConfig(ctx context.Context, url string) ([]crawler.FetchResult, error) {
	if err := ctx.Err(); err != nil {
		return []crawler.FetchResult{}, err
	}

	p, err := f.browserCtx.NewPage()
	if err != nil {
		return []crawler.FetchResult{}, err
	}
	defer func() {
		if closeErr := p.Close(); err != nil {
			f.logger.Error("error closing page", "error", closeErr)
		}
	}()

	if f.fetchConfig == nil {
		return []crawler.FetchResult{}, errors.New("fetch config is nil")
	}

	const timeoutInMs = 30000
	if ctxErr := ctx.Err(); ctxErr != nil {
		return []crawler.FetchResult{}, ctxErr
	}
	_, err = p.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(timeoutInMs),
	})
	if err != nil {
		return []crawler.FetchResult{}, err
	}

	if loadErr := f.fillSearchInput(ctx, p); loadErr != nil {
		return []crawler.FetchResult{}, loadErr
	}
	if searchErr := f.submitSearch(ctx, p); searchErr != nil {
		return []crawler.FetchResult{}, searchErr
	}
	results, err := f.waitAndCollectResults(ctx, p)
	if err != nil {
		return results, err
	}
	return results, nil
}

func randomDelay(ctx context.Context) error {
	const randValue = 1000
	const randRange = 2000
	// #nosec G404 - math/rand is sufficient for network jitter
	delay := time.Duration(randValue+rand.Intn(randRange)) * time.Millisecond
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Close Call to prevent resource leaks. Should be deferred right after creating the fetcher instance.
func (f *PlaywrightFetcher) Close() error {
	if f.browserCtx != nil {
		if ctxErr := f.browserCtx.Close(); ctxErr != nil {
			return fmt.Errorf("%w: %w", ErrPlaywrightClose, ctxErr)
		}
	}
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

// waitAndCollectResults stops on first matching selector.
func (f *PlaywrightFetcher) waitAndCollectResults(ctx context.Context, p playwright.Page) ([]crawler.FetchResult, error) {
	var results []crawler.FetchResult
	if len(f.fetchConfig.ResultsSelectors) > 0 {
		for _, sel := range f.fetchConfig.ResultsSelectors {
			if err := ctx.Err(); err != nil {
				return results, err
			}

			if err := waitForElementVisibility(f, p, sel); err != nil {
				continue
			}

			entries, err := p.Locator(sel).All()
			if err == nil {
				for _, entry := range entries {
					if ctxErr := ctx.Err(); ctxErr != nil {
						return results, ctxErr
					}
					r, selectErr := f.fetchSPAConfigClickAction(ctx, entry, p)
					if selectErr != nil || r == nil {
						continue
					}
					results = append(results, r...)
				}
			}
			break
		}
	}
	return results, nil
}

func waitForElementVisibility(f *PlaywrightFetcher, p playwright.Page, sel string) error {
	timeout := float64(f.fetchConfig.Timeout)
	locator := p.Locator(sel)
	err := locator.First().WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(timeout),
	})
	return err
}

func (f *PlaywrightFetcher) fetchSPAConfigClickAction(ctx context.Context, entry playwright.Locator, p playwright.Page) ([]crawler.FetchResult, error) {
	id := createFetchId(entry, p)
	err := entry.Click()
	if err != nil {
		f.logger.Error("error clicking entry", "error", err)
		return nil, err
	}
	delayErr := randomDelay(ctx)
	if delayErr != nil {
		return nil, delayErr
	}

	r, fetchErr := f.fetchSPAConfigDataSelectors(ctx, p, id)
	if fetchErr != nil {
		return nil, fetchErr
	}

	return r, nil
}

// todo ID for fetched data in generation have a few selectors to look for when generating the id in playwrightfetcher.
func createFetchId(entry playwright.Locator, p playwright.Page) string {
	href, err := entry.GetAttribute("href")
	if err != nil {
		href = p.URL() + uuid.New().String() // fallback to current page URL if href is not available
	}
	return href
}

// submitSearch stops on the first matching selector.
func (f *PlaywrightFetcher) submitSearch(ctx context.Context, p playwright.Page) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if len(f.fetchConfig.SearchSubmitSelectors) > 0 {
		for _, btn := range f.fetchConfig.SearchSubmitSelectors {
			if err := ctx.Err(); err != nil {
				return err
			}

			err := p.Locator(btn).Click()
			if err == nil {
				break
			}
		}
	}

	return nil
}

// fillSearchInput stops on the first matching selector.
func (f *PlaywrightFetcher) fillSearchInput(ctx context.Context, p playwright.Page) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if len(f.fetchConfig.SearchInputSelectors) > 0 {
		for _, sel := range f.fetchConfig.SearchInputSelectors {
			if err := ctx.Err(); err != nil {
				return err
			}

			err := p.Locator(sel).Fill(f.fetchConfig.SearchQuery)
			if err == nil {
				break
			}
		}
	}

	return nil
}

// fetchSPAConfigDataSelectors iterates through the provided data selectors in the fetch configuration,
// attempting to locate and extract text content from elements matching those selectors on the current page.
func (f *PlaywrightFetcher) fetchSPAConfigDataSelectors(ctx context.Context, p playwright.Page, id string) ([]crawler.FetchResult, error) {
	var results []crawler.FetchResult
	if len(f.fetchConfig.DataSelectors) > 0 {
		for _, sel := range f.fetchConfig.DataSelectors {
			if err := ctx.Err(); err != nil {
				return results, err
			}

			entries, err := p.Locator(sel).All()
			if err != nil {
				continue
			}

			for _, entry := range entries {
				if err := ctx.Err(); err != nil {
					return results, err
				}

				textContent, err := entry.TextContent()
				if err != nil {
					// todo test this path
					continue
				}
				f.logger.Info("playwright fetcher", "content", textContent)
				results = append(results, crawler.FetchResult{
					URL:        id, // todo get url if possible
					Body:       []byte(textContent),
					StatusCode: http.StatusOK, // todo get status code if possible
				})
			}
		}
	}

	return results, nil
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
		if pwErr := pw.Stop(); pwErr != nil {
			f.logger.Error("error stopping Playwright after browser launch failure", "error", pwErr)
		}
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
		Timeout: defaultTimeout,
	}
}
