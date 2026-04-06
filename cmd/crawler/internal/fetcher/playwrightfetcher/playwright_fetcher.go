package playwrightfetcher

import (
	"context"
	"golangwebcrawler/cmd/crawler/internal/crawler"
	"log/slog"
	"math/rand"
	"time"

	"github.com/playwright-community/playwright-go"
)

type PlaywrightFetcher struct {
	logger *slog.Logger
}

// todo decide if logger is needed
func NewPlaywrightFetcher(logger *slog.Logger) *PlaywrightFetcher {
	return &PlaywrightFetcher{
		logger: logger,
	}
}

// todo add channel and defer ctx close logic
func (f *PlaywrightFetcher) Fetch(ctx context.Context, url string) (crawler.FetchResult, error) {
	b, err := configurePlaywrightBrowser()
	if err != nil {
		return crawler.FetchResult{}, err
	}

	p, err := b.NewPage()
	// 4. Add random delays and human-like behavior
	p.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(30000),
	})

	// Random delay
	time.Sleep(time.Duration(rand.Intn(2000)+1000) * time.Millisecond)

	// Simulate mouse movement
	p.Mouse().Move(float64(rand.Intn(100)), float64(rand.Intn(100)))

	if _, err = p.WaitForSelector(`a[id*='job-title']`); err != nil {
		return crawler.FetchResult{}, err
	}
	entries, err := p.Locator(`a[id*='job-title']`).All()
	if err != nil {
		return crawler.FetchResult{}, err
	}

	for _, entry := range entries {
		var msg string
		text, err := entry.TextContent()
		if err != nil {
			msg = "error getting text content"
		}
		msg = text
		f.logger.Info("playwright fetcher", "url", url, "content", msg)
	}

	return crawler.FetchResult{}, nil
}

// configurePlaywrightBrowser sets up a Playwright browser instance with enhanced stealth options to better mimic human behavior and avoid detection by anti-bot measures.
// will launch a browser in headed mode
func configurePlaywrightBrowser() (playwright.Browser, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, err
	}

	b, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false), // Run in headed mode if possible
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
		return nil, err
	}

	ops := playwright.BrowserNewContextOptions{
		UserAgent:         playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"),
		Viewport:          &playwright.Size{Width: 1920, Height: 1080}, // Use ViewportSize pointer
		Locale:            playwright.String("en-US"),
		TimezoneId:        playwright.String("America/New_York"),
		Permissions:       []string{"geolocation", "notifications"}, // Add notifications to look more "human"
		JavaScriptEnabled: playwright.Bool(true),

		// Add these to prevent 'nil' errors in the driver's internal logic:
		IgnoreHttpsErrors: playwright.Bool(true),
		HasTouch:          playwright.Bool(false),
	}

	// 2. More comprehensive script injection
	bctx, err := b.NewContext(ops)
	if err != nil {
		return nil, err
	}

	// 3. Enhanced stealth scripts
	bctx.AddInitScript(playwright.Script{
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
	return b, nil
}
