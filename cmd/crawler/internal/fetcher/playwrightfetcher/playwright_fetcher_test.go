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
	fetcher := NewPlaywrightFetcher(logger)
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
