package playwrightfetcher

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"unicode/utf8"

	"github.com/joho/godotenv"
	"github.com/playwright-community/playwright-go"
)

var runFetchTest = false

func TestMain(m *testing.M) {
	if err := godotenv.Load("../../../../../.env"); err != nil {
		log.Println("No .env file found, falling back to system env")
	}
	runFetchTest = os.Getenv("RUN_FETCH_TESTS") == "1"
	os.Exit(m.Run())
}

func Test_FetchSPAConfig(t *testing.T) {
	if !runFetchTest {
		t.Skip("Skipping: set RUN_LLM_TESTS=1 to run")
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
	_, err = fetcher.Fetch(ctx, config.URL)
	if err != nil {
		fetcher.Close()
		t.Fatalf("fetching url %s: %v", config.URL, err)
	}
}

// todo prevent this from running in pipeline because playwright runs in headed mode for anti bot detection.
func Test_FetchDefault(t *testing.T) {
	if !runFetchTest {
		t.Skip("Skipping: set RUN_LLM_TESTS=1 to run")
	}
	url := "https://www.seek.com.au/software-engineer-jobs"
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	config := DefaultConfig()
	fetcher, err := NewPlaywrightFetcher(logger, &config)
	if err != nil {
		t.Fatalf("creating playwright fetcher: %v", err)
	}

	results, err := fetcher.Fetch(ctx, url)
	res := results[0]
	if err != nil {
		t.Fatalf("fetching url %s: %v", url, err)
	}

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

func Test_WaitAndCollectResults_ShouldReturnSlice(t *testing.T) {
	if !runFetchTest {
		t.Skip("Skipping: set RUN_FETCH_TESTS=1 to run")
	}

	// serve local html with known structure
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body>
            <a data-automation="jobTitle" href="/job/1">Software Engineer</a>
            <a data-automation="jobTitle" href="/job/2">Go Developer</a>
        </body></html>`)
	}))
	defer ts.Close()

	f := createMockFetcher()
	f.fetchConfig.ResultsSelectors = []string{"a[data-automation='jobTitle']"}
	f.fetchConfig.DataSelectors = []string{"a[data-automation='jobTitle']"}
	f.fetchConfig.Timeout = defaultTimeout

	err := f.configurePlaywrightBrowser()
	if err != nil {
		t.Fatalf("configurePlaywrightBrowser() error = %v", err)
	}
	defer f.Close()

	p, _ := f.browserCtx.NewPage()
	pOpts := playwright.PageGotoOptions{}
	defer p.Close()
	p.Goto(ts.URL, pOpts)

	results := f.waitAndCollectResults(p)
	if len(results) == 0 {
		t.Error("expected results, got empty slice")
	}
}
