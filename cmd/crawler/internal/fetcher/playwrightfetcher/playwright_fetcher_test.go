package playwrightfetcher

import (
	"context"
	"log"
	"log/slog"
	"os"
	"testing"
	"unicode/utf8"

	"github.com/joho/godotenv"
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

// todo fill in.
func Test_ConfigurePlaywrightBrowser(t *testing.T) {
}

// todo fill in.
func Test_Close(t *testing.T) {
}
