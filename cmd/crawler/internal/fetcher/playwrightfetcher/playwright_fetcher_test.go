package playwrightfetcher

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"unicode/utf8"
)

// todo prevent this from running in pipeline because playwright runs in headed mode for anti bot detection.
func Test_Fetch(t *testing.T) {
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

	res, err := fetcher.Fetch(ctx, url)
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
