package playwrightfetcher

import (
	"context"
	"fmt"
	"golangwebcrawler/internal/crawler"
	"golangwebcrawler/internal/env"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/playwright-community/playwright-go"
)

const paginationSelectors = "[data-automation='pagination-next']"

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

func Test_WaitAndCollectResults_AndfetchSPAConfigPageSelectors(t *testing.T) {
	tc := []struct {
		name                string
		pageSelectors       *PaginationConfig
		expectedResultCount int
	}{
		{
			name: "should navigate pages",
			pageSelectors: &PaginationConfig{
				ContainerSelectors: []string{paginationSelectors},
				NextSelectors:      []string{paginationSelectors},
				DisabledSelector:   "button[disabled]",
				WaitForSelectors:   []string{"a[data-automation='jobTitle']"},
				Strategy:           "auto",
				MaxRetries:         2,
				MaxPages:           3,
			},
			expectedResultCount: 2,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			f, p := setup_paginationTest(t)
			f.fetchConfig.Pagination = *tt.pageSelectors
			defer f.Close()

			mockFn := func(ctx context.Context, p playwright.Page) ([]crawler.FetchResult, error) {
				return []crawler.FetchResult{}, nil
			}

			_, err := collectPageResults(t.Context(), f, p, mockFn)
			if err != nil {
				t.Fatalf("collected page results error: %v", err)
			}

			// Page URL changed to last page
			finalUrl := p.URL()
			if !strings.Contains(finalUrl, "#page=3") {
				t.Errorf("expected final URL to contain #page=3, got %s", finalUrl)
			}

			// Page info text updated
			pageInfo, _ := p.Locator(".page-info").TextContent()
			if pageInfo != "Page 3 of 3" {
				t.Errorf("expected page info 'Page 3 of 3', got %q", pageInfo)
			}

			// Job titles changed to page 3 content
			jobTitles, _ := p.Locator("a[data-automation='jobTitle']").AllTextContents()
			expectedJobs := []string{"Frontend Architect"}
			if len(jobTitles) != len(expectedJobs) {
				t.Errorf("expected %d jobs, got %d", len(expectedJobs), len(jobTitles))
			}

			// Next button is now disabled (DOM element changed)
			nextDisabled, _ := p.Locator("span.disabled").Count()
			if nextDisabled == 0 {
				t.Error("expected disabled next button on last page")
			}
		})
	}
}

func createPaginationTestServer(t *testing.T) (*httptest.Server, error) {
	htmlBytes, err := os.ReadFile("./testdata/pagination_html")
	if err != nil {
		t.Fatalf("failed to read HTML file: %v", err)
	}

	ts := createTestHttpServer(string(htmlBytes))
	return ts, err
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

func Test_ShouldStopPagination(t *testing.T) {
	tc := []struct {
		name         string
		maxItems     int
		resultsCount int
		expectedStop bool
	}{
		{
			name:         "returns false when MaxItems is zero",
			maxItems:     0,
			resultsCount: 5,
			expectedStop: false,
		},
		{
			name:         "returns false when MaxItems is negative",
			maxItems:     -1,
			resultsCount: 5,
			expectedStop: false,
		},
		{
			name:         "returns false when results below limit",
			maxItems:     5,
			resultsCount: 3,
			expectedStop: false,
		},
		{
			name:         "returns true when results equal limit",
			maxItems:     5,
			resultsCount: 5,
			expectedStop: true,
		},
		{
			name:         "returns true when results exceed limit",
			maxItems:     5,
			resultsCount: 10,
			expectedStop: true,
		},
		{
			name:         "returns false when results empty and limit set",
			maxItems:     5,
			resultsCount: 0,
			expectedStop: false,
		},
		{
			name:         "returns true when single result meets limit of 1",
			maxItems:     1,
			resultsCount: 1,
			expectedStop: true,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			f := createMockFetcher()
			f.fetchConfig.MaxItems = tt.maxItems

			results := make([]crawler.FetchResult, tt.resultsCount)
			for i := range results {
				results[i] = crawler.FetchResult{
					URL:        fmt.Sprintf("https://example.com/job/%d", i+1),
					StatusCode: http.StatusOK,
					Body:       []byte(""),
				}
			}

			stop := f.shouldStopPagination(results)
			if stop != tt.expectedStop {
				t.Fatalf("expected %t, got %t", tt.expectedStop, stop)
			}
		})
	}
}

func Test_HasPaginationSection(t *testing.T) {
	tc := []struct {
		name               string
		containerSelectors []string
		expected           bool
	}{
		{
			name:               "returns false when no selectors configured",
			containerSelectors: nil,
			expected:           false,
		},
		{
			name:               "returns false when selectors do not match",
			containerSelectors: []string{".missing-pagination"},
			expected:           false,
		},
		{
			name:               "returns true when any selector matches",
			containerSelectors: []string{".missing-pagination", paginationSelectors},
			expected:           true,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			f, p := setup_paginationTest(t)
			defer f.Close()
			defer func() {
				if closeErr := p.Close(); closeErr != nil {
					t.Logf("error closing page: %v", closeErr)
				}
			}()

			f.fetchConfig.Pagination.ContainerSelectors = tt.containerSelectors

			hasSelectors := f.hasPaginationSection(p)
			if hasSelectors != tt.expected {
				t.Fatalf("expected %t, got %t", tt.expected, hasSelectors)
			}
		})
	}
}

func setup_paginationTest(t *testing.T) (*PlaywrightFetcher, playwright.Page) {
	f := createMockFetcher()
	f.fetchConfig.Timeout = 5000
	f.fetchConfig.Pagination.ContainerSelectors = []string{paginationSelectors}
	ts, err := createPaginationTestServer(t)
	if err != nil {
		t.Fatalf("failed to start test server: % v", err)
	}

	err = f.configurePlaywrightBrowser()
	if err != nil {
		t.Fatalf("configurePlaywrightBrowser() error: %v", err)
	}
	p, _ := f.browserCtx.NewPage()

	pOpts := playwright.PageGotoOptions{}
	_, err = p.Goto(ts.URL, pOpts)
	if err != nil {
		t.Fatalf("failed to navigate: %v", err)
	}
	return f, p
}

func Test_IsNextDisabled(t *testing.T) {
	tc := []struct {
		name             string
		disabledSelector string
		expected         bool
	}{
		{
			name:             "returns false when disabled selector is empty",
			disabledSelector: "",
			expected:         false,
		},
		{
			name:             "returns false when disabled selector does not match",
			disabledSelector: "button[disabled]",
			expected:         false,
		},
		{
			name:             "returns true when disabled selector matches page",
			disabledSelector: paginationSelectors,
			expected:         true,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			f, p := setup_paginationTest(t)
			defer f.Close()
			defer func() {
				if closeErr := p.Close(); closeErr != nil {
					t.Logf("error closing page: %v", closeErr)
				}
			}()

			f.fetchConfig.Pagination.DisabledSelector = tt.disabledSelector

			res := f.isNextDisabled(p)
			if res != tt.expected {
				t.Errorf("isNextDisabled() returned %v expected %v", res, tt.expected)
			}
		})
	}
}

func Test_ClickNextPageWithRetry(t *testing.T) {
	tc := []struct {
		name                string
		nextSelectors       []string
		strategy            string
		clicks              int
		expectedErr         bool
		assertLastPageState bool
	}{
		{
			name:                "navigates to last page with valid selector",
			nextSelectors:       []string{paginationSelectors},
			strategy:            "",
			clicks:              2,
			expectedErr:         false,
			assertLastPageState: true,
		},
		{
			name:                "returns error when selectors are not configured",
			nextSelectors:       nil,
			strategy:            paginationStrategyNext,
			clicks:              1,
			expectedErr:         true,
			assertLastPageState: false,
		},
		{
			name:                "returns error when selectors are invalid",
			nextSelectors:       []string{".missing-next-button"},
			strategy:            paginationStrategyNext,
			clicks:              1,
			expectedErr:         true,
			assertLastPageState: false,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			f, p := setup_paginationTest(t)
			defer f.Close()
			defer func() {
				if closeErr := p.Close(); closeErr != nil {
					t.Logf("error closing page: %v", closeErr)
				}
			}()

			f.fetchConfig.Pagination.MaxRetries = 3
			f.fetchConfig.Pagination.NextSelectors = tt.nextSelectors
			f.fetchConfig.Pagination.Strategy = tt.strategy

			var err error
			for range tt.clicks {
				err = f.clickNextPageWithRetry(t.Context(), p)
				if err != nil {
					break
				}
			}

			if (err != nil) != tt.expectedErr {
				t.Fatalf("expected error=%t, got err=%v", tt.expectedErr, err)
			}

			if !tt.assertLastPageState {
				return
			}

			if waitErr := p.Locator("span.disabled").First().WaitFor(playwright.LocatorWaitForOptions{
				State:   playwright.WaitForSelectorStateVisible,
				Timeout: playwright.Float(5000),
			}); waitErr != nil {
				t.Fatalf("waiting for last page state: %v", waitErr)
			}

			finalURL := p.URL()
			if !strings.Contains(finalURL, "#page=3") {
				t.Errorf("expected final URL to contain #page=3, got %s", finalURL)
			}

			pageInfo, _ := p.Locator(".page-info").TextContent()
			if pageInfo != "Page 3 of 3" {
				t.Errorf("expected page info 'Page 3 of 3', got %q", pageInfo)
			}

			jobTitles, _ := p.Locator("a[data-automation='jobTitle']").AllTextContents()
			expectedJobs := []string{"Frontend Architect"}
			if len(jobTitles) != len(expectedJobs) {
				t.Errorf("expected %d jobs, got %d", len(expectedJobs), len(jobTitles))
			}

			nextDisabled, _ := p.Locator("span.disabled").Count()
			if nextDisabled == 0 {
				t.Error("expected disabled next button on last page")
			}
		})
	}
}

func Test_WaitForNextPageLoad(t *testing.T) {
	tc := []struct {
		name             string
		waitForSelectors []string
		ctx              func(t *testing.T) context.Context
		expectErr        bool
	}{
		{
			name:             "returns nil when no selectors configured",
			waitForSelectors: []string{},
			ctx:              func(t *testing.T) context.Context { return t.Context() },
			expectErr:        false,
		},
		{
			name:             "returns context error when context is expired",
			waitForSelectors: []string{".next-button"},
			ctx: func(t *testing.T) context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 0)
				defer cancel()
				return ctx
			},
			expectErr: true,
		},
		{
			name:             "returns nil when selectors match page content",
			waitForSelectors: DefaultConfig().Pagination.WaitForSelectors,
			ctx:              func(t *testing.T) context.Context { return t.Context() },
			expectErr:        false,
		},
		{
			name:             "returns error when no selectors match",
			waitForSelectors: []string{".next-button"},
			ctx:              func(t *testing.T) context.Context { return t.Context() },
			expectErr:        true,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			f, p := setup_paginationTest(t)
			defer f.Close()
			defer func() {
				if closeErr := p.Close(); closeErr != nil {
					t.Logf("error closing page: %v", closeErr)
				}
			}()

			f.fetchConfig.Pagination.WaitForSelectors = tt.waitForSelectors

			err := f.waitForNextPageLoad(tt.ctx(t), p)
			if (err != nil) != tt.expectErr {
				t.Errorf("expected error=%t, got err=%v", tt.expectErr, err)
			}
		})
	}
}
