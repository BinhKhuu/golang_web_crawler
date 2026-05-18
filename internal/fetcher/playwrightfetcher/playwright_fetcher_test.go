package playwrightfetcher

import (
	"context"
	"fmt"
	"golangwebcrawler/internal/env"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"unicode/utf8"

	"github.com/playwright-community/playwright-go"
)

var runFetchTest = false

func TestMain(m *testing.M) {
	if err := env.LoadEnv(); err != nil {
		log.Println("No .env file found, falling back to system env")
	}
	runFetchTest = os.Getenv("RUN_FETCH_TESTS") == "1"
	os.Exit(m.Run())
}

func Test_FetchSPAConfig(t *testing.T) {
	if !runFetchTest {
		t.Skip("Skipping: set RUN_FETCH_TESTS=1 to run")
	}

	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	config := GetSeekConfiguration()
	fetcher, err := NewConfiguredPlaywrightFetcher(logger, &config)
	if err != nil {
		fetcher.Close()
		t.Fatalf("creating playwright fetcher: %v", err)
	}
	defer func() {
		if closeErr := fetcher.Close(); closeErr != nil {
			log.Printf("error closing fetcher: %v", closeErr)
		}
	}()
	results, err := fetcher.Fetch(ctx, config.URL)
	if err != nil {
		fetcher.Close()
		t.Fatalf("fetching url %s: %v", config.URL, err)
	}
	if len(results) == 0 {
		fetcher.Close()
		t.Fatalf("expected non-empty data, got empty")
	}
}

func Test_FetchDefault(t *testing.T) {
	if !runFetchTest {
		t.Skip("Skipping: set RUN_FETCH_TESTS=1 to run")
	}
	url := seekSoftwareEngineerJobsURL
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	config := DefaultConfig()
	fetcher, err := NewPlaywrightFetcher(logger, &config)
	if err != nil {
		t.Fatalf("creating playwright fetcher: %v", err)
	}
	defer func() {
		if closeErr := fetcher.Close(); closeErr != nil {
			log.Printf("error closing fetcher: %v", closeErr)
		}
	}()
	results, err := fetcher.Fetch(ctx, url)
	if err != nil {
		fetcher.Close()
		t.Fatalf("fetching url %s: %v", url, err)
	}
	res := results[0]

	if !utf8.Valid(res.Body) {
		t.Fatalf("expected valid UTF-8 body, got invalid data")
	}

	if len(res.Body) == 0 {
		t.Fatalf("expected non-empty body, got empty")
	}
}

func Test_DefaultConfiguration(t *testing.T) {
	config := DefaultConfig()
	if config.URL == "" {
		t.Errorf("expected default URL to be empty, got %s", config.URL)
	}
	if config.Timeout == 0 {
		t.Errorf("expected default timeout to be %d, got %d", defaultTimeout, config.Timeout)
	}

	// rest of configuration can be empty as they are optional and depend on the target website.
}

func Test_ConfigurePlaywrightBrowser(t *testing.T) {
	if !runFetchTest {
		t.Skip("Skipping: set RUN_FETCH_TESTS=1 to run")
	}
	f := createMockFetcher()
	err := f.configurePlaywrightBrowser()
	if err != nil {
		t.Fatalf("configurePlaywrightBrowser() error = %v", err)
	}
	defer f.Close()

	if f.pw == nil {
		t.Error("expected pw to be set, got nil")
	}
	if f.browser == nil {
		t.Error("expected browser to be set, got nil")
	}
	if f.browserCtx == nil {
		t.Error("expected browserCtx to be set, got nil")
	}
}

func Test_Close(t *testing.T) {
	if !runFetchTest {
		t.Skip("Skipping: set RUN_FETCH_TESTS=1 to run")
	}

	f := createMockFetcher()
	err := f.configurePlaywrightBrowser()
	if err != nil {
		t.Fatalf("configurePlaywrightBrowser() error = %v", err)
	}

	if !f.browser.IsConnected() {
		t.Error("expected browser to be connected before close")
	}

	err = f.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if f.browser.IsConnected() {
		t.Error("expected browser to be disconnected after close")
	}
}

// createMockFetcher will skip the test if RUN_FETCH_TESTS is not set, otherwise it will create a PlaywrightFetcher with a logger and empty config for testing purposes. This allows us to test internal methods without needing to set up a full configuration or environment.
func createMockFetcher() *PlaywrightFetcher {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	f := &PlaywrightFetcher{
		logger:      logger,
		fetchConfig: &PlaywrightFetcherConfig{},
	}
	return f
}

func Test_WaitAndCollectResults_AndfetchSPAConfigDataSelectors(t *testing.T) {
	if !runFetchTest {
		t.Skip("Skipping: set RUN_FETCH_TESTS=1 to run")
	}

	tc := []struct {
		name                string
		html                string
		resultsSelectors    []string
		dataSelectors       []string
		expectedResultCount int
	}{
		{
			name: "should use fallback selector when first selector fails",
			html: `
				<html>
				<body>
					<a class="job-link" href="/job/1">Software Engineer</a>
					<div id="job-details">Job details content</div>
				</body>
				</html>`,
			resultsSelectors: []string{
				seekJobTitleSelector,
				seekJobLinkSelector,
			},
			dataSelectors: []string{
				"a[data-automation='jobDetailsPage']",
				"#job-details",
			},
			expectedResultCount: 1,
		},
		{
			name: "should return empty when ResultsSelectors is empty",
			html: `
				<html>
					<body>
						<a data-automation="jobTitle" href="/job/1">Software Engineer</a>
						<div id="job-details">Job details content</div>
					</body>
				</html>`,
			resultsSelectors:    []string{},
			dataSelectors:       []string{},
			expectedResultCount: 0,
		},
		{
			name: "should return empty when DataSelectors dont match page content",
			html: `
			<html>
				<body>
					<a data-automation="jobTitle" href="/job/1">Software Engineer</a>
					<div id="job-details">Job details content</div>
				</body>
			</html>`,
			resultsSelectors:    []string{seekJobTitleSelector},
			dataSelectors:       []string{"a.no-match"},
			expectedResultCount: 0,
		},
		{
			name: "should handle multiple entries under one selector",
			html: `<html><body>
				<a data-automation="jobTitle" href="/job/1">Job 1</a>
				<a data-automation="jobTitle" href="/job/2">Job 2</a>
				<a data-automation="jobTitle" href="/job/3">Job 3</a>
				<div data-automation="jobDetailsPage">Job details content</div>
				</body></html>
				<div data-automation="jobDetailsPage2">Job details content2</div>
				</body></html>
				<div data-automation="jobDetailsPage3">Job details content3</div>
				</body></html>`,
			resultsSelectors: []string{seekJobTitleSelector},
			dataSelectors: []string{
				"div[data-automation='jobDetailsPage']",
			},
			expectedResultCount: 3,
		},
		{
			name: "should handle multiple entries under one selector with multiple data selectors",
			html: `<html><body>
				<a data-automation="jobTitle" href="/job/1">Job 1</a>
				<a data-automation="jobTitle" href="/job/2">Job 2</a>
				<a data-automation="jobTitle" href="/job/3">Job 3</a>
				<div data-automation="jobDetailsPage">Job details content</div>
				</body></html>
				<div data-automation="jobDetailsPage2">Job details content2</div>
				</body></html>
				<div data-automation="jobDetailsPage3">Job details content3</div>
				</body></html>`,
			resultsSelectors: []string{seekJobTitleSelector},
			dataSelectors: []string{
				"div[data-automation='jobDetailsPage']",
				"div[data-automation='jobDetailsPage2']",
				"div[data-automation='jobDetailsPage3']",
			},
			expectedResultCount: 9,
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			ts := createTestHttpServer(tt.html)

			f := createMockFetcher()
			f.fetchConfig.Results.ListingSelectors = tt.resultsSelectors
			f.fetchConfig.Results.DataSelectors = tt.dataSelectors
			f.fetchConfig.Timeout = defaultTimeout

			err := f.configurePlaywrightBrowser()
			if err != nil {
				t.Fatalf("configurePlaywrightBrowser() error = %v", err)
			}
			defer f.Close()

			p, _ := f.browserCtx.NewPage()
			pOpts := playwright.PageGotoOptions{}
			defer func() {
				closeErr := p.Close()
				if closeErr != nil {
					t.Logf("error closing page: %v", closeErr)
				}
			}()
			_, err = p.Goto(ts.URL, pOpts)
			if err != nil {
				t.Fatalf("page.Goto() error = %v", err)
			}

			results, err := f.waitAndCollectResults(context.Background(), p)
			if err != nil {
				t.Fatalf("waitAndCollectResults() error = %v", err)
			}
			if len(results) != tt.expectedResultCount {
				t.Errorf("error expected to return %d results, got %d", tt.expectedResultCount, len(results))
			}
		})
	}
}

func createTestHttpServer(html string) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, html)
	}))
	return ts
}

func Test_CanonicalizeFetchedURL(t *testing.T) {
	tc := []struct {
		name              string
		baseURL           string
		href              string
		ignoreQueryParams []string
		rootPrefixes      []string
		expected          string
	}{
		{
			name:    "removes fragment and ignored params",
			baseURL: seekSoftwareEngineerJobsURL,
			href:    "/job/91318081?type=standard&ref=search-standalone&origin=cardTitle#sol=2ecb52bdcb0bfb96f8160ca64024c28215a0a063",
			ignoreQueryParams: []string{
				seekTrackingParamSol,
				seekTrackingParamRef,
				seekTrackingParamOrigin,
			},
			expected: "https://www.seek.com.au/job/91318081?type=standard",
		},
		{
			name:    "treats configured bare prefix as root-relative",
			baseURL: seekSoftwareEngineerJobsURL + "/in-All-Australia",
			href:    "job/91318081?type=standard&ref=search-standalone&origin=cardTitle#sol=383e9b9d93f39fc67d84d3223d264c7c94ae6961",
			ignoreQueryParams: []string{
				seekTrackingParamSol,
				seekTrackingParamRef,
				seekTrackingParamOrigin,
			},
			rootPrefixes: []string{seekJobPathPrefix},
			expected:     "https://www.seek.com.au/job/91318081?type=standard",
		},
		{
			name:     "strips utm params generically",
			baseURL:  "https://example.com/jobs",
			href:     "https://example.com/job/1?utm_source=x&utm_medium=y&id=1#abc",
			expected: "https://example.com/job/1?id=1",
		},
		{
			name:     "resolves relative links",
			baseURL:  "https://example.com/jobs/search",
			href:     "../job/100?foo=bar#frag",
			expected: "https://example.com/job/100?foo=bar",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			got := canonicalizeFetchedURL(tt.baseURL, tt.href, tt.ignoreQueryParams, tt.rootPrefixes)
			if got != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}
