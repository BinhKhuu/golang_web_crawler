package playwrightfetcher

import (
	"context"
	"errors"
	"fmt"
	"golangwebcrawler/internal/crawler"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/playwright-community/playwright-go"
)

var (
	ErrPlaywrightClose = errors.New("closing playwright browser error")
	ErrPlaywrightStop  = errors.New("stopping playwright error")
)

const defaultTimeout = 10000

const (
	paginationStrategyAuto    = "auto"
	paginationStrategyNext    = "next-button"
	paginationStrategyNumbers = "page-numbers"
	defaultMaxRetries         = 2
)

type PlaywrightFetcher struct {
	logger      *slog.Logger
	fetchConfig *PlaywrightFetcherConfig
	pw          *playwright.Playwright
	browser     playwright.Browser
	browserCtx  playwright.BrowserContext
	fetchFn     func(ctx context.Context, url string) ([]crawler.FetchResult, error)
}

// SearchConfig holds selectors and query for the search interaction.
// Leave fields empty to skip the step. The first matching selector is used.
type SearchConfig struct {
	InputSelectors  []string `json:"inputSelectors"`
	Query           string   `json:"query"`
	SubmitSelectors []string `json:"submitSelectors"`
}

// ResultsConfig holds selectors for collecting results after search.
// ListingSelectors: first match is used to locate job listing links.
// DataSelectors: all matches are used to extract content after clicking a listing.
type ResultsConfig struct {
	ListingSelectors []string `json:"listingSelectors"`
	DataSelectors    []string `json:"dataSelectors"`
}

// CanonicalizationConfig controls how fetched URLs are normalized for deduplication.
type CanonicalizationConfig struct {
	// IgnoreQueryParams strips these query parameters (e.g. tracking params like "ref", "sol").
	IgnoreQueryParams []string `json:"ignoreQueryParams"`
	// RootRelativePrefixes treats hrefs starting with these prefixes as root-relative (e.g. "job/" → "/job/").
	RootRelativePrefixes []string `json:"rootRelativePrefixes"`
}

// PaginationConfig controls how the fetcher navigates through paginated results.
type PaginationConfig struct {
	// ContainerSelectors: selectors that indicate a pagination section exists.
	ContainerSelectors []string `json:"containerSelectors"`
	// NextSelectors: selectors for "next page" button (tried in order).
	NextSelectors []string `json:"nextSelectors"`
	// PageNumberSelectors: selectors for numbered page buttons.
	PageNumberSelectors []string `json:"pageNumberSelectors"`
	// DisabledSelector: attribute/selector indicating "next" is disabled.
	DisabledSelector string `json:"disabledSelector"`
	// WaitForSelectors: after clicking next, wait for these to appear.
	WaitForSelectors []string `json:"waitForSelectors"`
	// Strategy: "auto" (default) | "next-button" | "page-numbers".
	Strategy string `json:"strategy"`
	// MaxRetries: retries on next-page click failure (default: 2).
	MaxRetries int `json:"maxRetries"`
	MaxPages   int `json:"maxPages"`
}

type PlaywrightFetcherConfig struct {
	URL      string `json:"url"`
	Headless bool   `json:"headless"`
	Timeout  int    `json:"timeout"`
	// MaxItems limits the number of scraped results (0 = unlimited).
	MaxItems int `json:"maxItems"`

	Search           SearchConfig           `json:"search"`
	Results          ResultsConfig          `json:"results"`
	Canonicalization CanonicalizationConfig `json:"canonicalization"`
	Pagination       PaginationConfig       `json:"pagination"`
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

	if ctxErr := ctx.Err(); ctxErr != nil {
		return []crawler.FetchResult{}, ctxErr
	}
	_, err = p.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(f.timeoutInMs()),
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

	if ctxErr := ctx.Err(); ctxErr != nil {
		return []crawler.FetchResult{}, ctxErr
	}
	_, err = p.Goto(url, playwright.PageGotoOptions{
		Timeout: playwright.Float(f.timeoutInMs()),
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

	res, err := collectPageResults(ctx, f, p, f.waitAndCollectResults)
	if err != nil {
		f.logger.Error("Page collection error")
		return nil, err
	}
	return res, nil
}

func collectPageResults(ctx context.Context, f *PlaywrightFetcher, p playwright.Page, collectionFn func(ctx context.Context, p playwright.Page) ([]crawler.FetchResult, error)) ([]crawler.FetchResult, error) {
	var allResults []crawler.FetchResult
	for page := 1; page <= f.fetchConfig.Pagination.MaxPages; page++ {
		f.logger.Debug("Collecting page", "page", page)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return allResults, ctxErr
		}

		pageResults, err := collectionFn(ctx, p)
		if err != nil {
			f.logger.Warn("error collecting results from page", "error", err)
		}
		allResults = append(allResults, pageResults...)

		if f.shouldStopPagination(allResults) {
			if f.fetchConfig.MaxItems > 0 && len(allResults) > f.fetchConfig.MaxItems {
				allResults = allResults[:f.fetchConfig.MaxItems]
			}
			break
		}

		if !f.hasPaginationSection(p) {
			break
		}

		if f.isNextDisabled(p) {
			break
		}

		if clickErr := f.clickNextPageWithRetry(ctx, p); clickErr != nil {
			f.logger.Warn("failed to navigate to next page after retries", "error", clickErr)
			break
		}

		if waitErr := f.waitForNextPageLoad(ctx, p); waitErr != nil {
			f.logger.Warn("timeout waiting for next page content", "error", waitErr)
			break
		}

		if delayErr := randomDelay(ctx); delayErr != nil {
			return allResults, delayErr
		}
	}

	return allResults, nil
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

func (f *PlaywrightFetcher) timeoutInMs() float64 {
	if f.fetchConfig != nil && f.fetchConfig.Timeout > 0 {
		return float64(f.fetchConfig.Timeout)
	}
	return float64(defaultTimeout)
}

// clickWithTimeout passes the timeout explicitly because relying on Playwright
// defaults in these retry-heavy paths led to slow failures for missing or
// invalid selectors.
func (f *PlaywrightFetcher) clickWithTimeout(locator playwright.Locator) error {
	return locator.Click(playwright.LocatorClickOptions{
		Timeout: playwright.Float(f.timeoutInMs()),
	})
}

// waitAndCollectResults stops on first matching selector.
func (f *PlaywrightFetcher) waitAndCollectResults(ctx context.Context, p playwright.Page) ([]crawler.FetchResult, error) {
	var results []crawler.FetchResult
	if len(f.fetchConfig.Results.ListingSelectors) > 0 {
		for _, sel := range f.fetchConfig.Results.ListingSelectors {
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
	locator := p.Locator(sel)
	err := locator.First().WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(f.timeoutInMs()),
	})
	return err
}

func (f *PlaywrightFetcher) fetchSPAConfigClickAction(ctx context.Context, entry playwright.Locator, p playwright.Page) ([]crawler.FetchResult, error) {
	id := f.createFetchID(entry, p)
	err := f.clickWithTimeout(entry)
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

func (f *PlaywrightFetcher) createFetchID(entry playwright.Locator, p playwright.Page) string {
	href, err := entry.GetAttribute("href")
	if err != nil || strings.TrimSpace(href) == "" {
		return p.URL() + uuid.New().String()
	}

	ignoreQueryParams := []string(nil)
	rootRelativePrefixes := []string(nil)
	if f.fetchConfig != nil {
		ignoreQueryParams = f.fetchConfig.Canonicalization.IgnoreQueryParams
		rootRelativePrefixes = f.fetchConfig.Canonicalization.RootRelativePrefixes
	}

	canonical := canonicalizeFetchedURL(p.URL(), href, ignoreQueryParams, rootRelativePrefixes)
	if canonical == "" {
		return p.URL() + uuid.New().String()
	}

	return canonical
}

// submitSearch stops on the first matching selector.
func (f *PlaywrightFetcher) submitSearch(ctx context.Context, p playwright.Page) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if len(f.fetchConfig.Search.SubmitSelectors) > 0 {
		for _, btn := range f.fetchConfig.Search.SubmitSelectors {
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

	if len(f.fetchConfig.Search.InputSelectors) > 0 {
		for _, sel := range f.fetchConfig.Search.InputSelectors {
			if err := ctx.Err(); err != nil {
				return err
			}

			err := p.Locator(sel).Fill(f.fetchConfig.Search.Query)
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
	if len(f.fetchConfig.Results.DataSelectors) > 0 {
		for _, sel := range f.fetchConfig.Results.DataSelectors {
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

func (f *PlaywrightFetcher) shouldStopPagination(results []crawler.FetchResult) bool {
	if f.fetchConfig.MaxItems <= 0 {
		return false
	}
	return len(results) >= f.fetchConfig.MaxItems
}

func (f *PlaywrightFetcher) hasPaginationSection(p playwright.Page) bool {
	if len(f.fetchConfig.Pagination.ContainerSelectors) == 0 {
		return false
	}
	for _, sel := range f.fetchConfig.Pagination.ContainerSelectors {
		count, err := p.Locator(sel).Count()
		if err == nil && count > 0 {
			return true
		}
	}
	return false
}

func (f *PlaywrightFetcher) isNextDisabled(p playwright.Page) bool {
	if f.fetchConfig.Pagination.DisabledSelector == "" {
		return false
	}
	count, err := p.Locator(f.fetchConfig.Pagination.DisabledSelector).Count()
	return err == nil && count > 0
}

func (f *PlaywrightFetcher) clickNextPageWithRetry(ctx context.Context, p playwright.Page) error {
	maxRetries := f.fetchConfig.Pagination.MaxRetries
	if maxRetries <= 0 {
		maxRetries = defaultMaxRetries
	}

	strategy := f.fetchConfig.Pagination.Strategy
	if strategy == "" {
		strategy = paginationStrategyAuto
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		if attempt > 0 {
			if delayErr := randomDelay(ctx); delayErr != nil {
				return delayErr
			}
		}

		var clicked bool
		switch strategy {
		case paginationStrategyNext:
			clicked = f.clickNextButton(p)
		case paginationStrategyNumbers:
			clicked = f.clickNextPageNumber(p)
		default:
			clicked = f.clickNextButton(p)
			if !clicked {
				clicked = f.clickNextPageNumber(p)
			}
		}

		if clicked {
			return nil
		}

		lastErr = errors.New("no pagination control found to click")
	}

	return fmt.Errorf("click next page after %d retries: %w", maxRetries, lastErr)
}

func (f *PlaywrightFetcher) clickNextButton(p playwright.Page) bool {
	for _, sel := range f.fetchConfig.Pagination.NextSelectors {
		locator := p.Locator(sel)
		count, err := locator.Count()
		if err != nil || count == 0 {
			continue
		}
		err = f.clickWithTimeout(locator.First())
		if err == nil {
			return true
		}
	}
	return false
}

func (f *PlaywrightFetcher) clickNextPageNumber(p playwright.Page) bool {
	for _, sel := range f.fetchConfig.Pagination.PageNumberSelectors {
		entries, err := p.Locator(sel).All()
		if err != nil || len(entries) == 0 {
			continue
		}
		for _, entry := range entries {
			text, textErr := entry.TextContent()
			if textErr != nil {
				continue
			}
			if strings.TrimSpace(text) == "" {
				continue
			}
			err := f.clickWithTimeout(entry)
			if err == nil {
				return true
			}
		}
	}
	return false
}

func (f *PlaywrightFetcher) waitForNextPageLoad(ctx context.Context, p playwright.Page) error {
	if len(f.fetchConfig.Pagination.WaitForSelectors) == 0 {
		return nil
	}

	for _, sel := range f.fetchConfig.Pagination.WaitForSelectors {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		locator := p.Locator(sel)
		err := locator.First().WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(f.timeoutInMs()),
		})
		if err == nil {
			return nil
		}
	}

	return errors.New("no wait-for selectors matched on next page")
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
	bctx.SetDefaultTimeout(f.timeoutInMs())
	bctx.SetDefaultNavigationTimeout(f.timeoutInMs())

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
		Headless: false,
		Timeout:  defaultTimeout,
		Search: SearchConfig{
			InputSelectors: []string{
				seekKeywordsInputSelector,
				seekSearchPlaceholder,
			},
			Query: "Software Engineer Jobs",
			SubmitSelectors: []string{
				"button[type=submit]",
				"button[aria-label='Search']",
			},
		},
		Results: ResultsConfig{
			ListingSelectors: []string{
				seekJobTitleSelector,
				seekJobLinkSelector,
				seekJobTestIdSelector,
			},
			DataSelectors: []string{
				"a[id*='job-title']",
				seekJobTitleSelector,
				".job-title a",
				"article a[href*='/job/']",
			},
		},
		Canonicalization: CanonicalizationConfig{
			IgnoreQueryParams:    []string{seekTrackingParamSol, seekTrackingParamRef, seekTrackingParamOrigin},
			RootRelativePrefixes: []string{seekJobPathPrefix},
		},
		Pagination: PaginationConfig{
			ContainerSelectors: []string{seekPaginationContainer},
			NextSelectors:      []string{seekPaginationNext},
			DisabledSelector:   seekPaginationDisabled,
			WaitForSelectors:   []string{seekJobTitleSelector},
			Strategy:           paginationStrategyAuto,
			MaxRetries:         defaultMaxRetries,
		},
	}
}
