package fetcher

import (
	"context"
	"testing"
	"time"
)

func TestNewBrowserFetcher(t *testing.T) {
	config := BrowserFetcherConfig{
		Headless:      true,
		WaitTimeout:   10 * time.Second,
		RenderTimeout: 1 * time.Second,
		MaxLinks:      10,
		WaitSelectors: []string{"body"},
	}

	fetcher, err := NewBrowserFetcher(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer fetcher.Close()

	if fetcher.browser == nil {
		t.Fatal("expected browser to be initialized")
	}
}

func TestNewBrowserFetcherWithOptions(t *testing.T) {
	fetcher, err := NewBrowserFetcherWithOptions(
		WithHeadless(true),
		WithWaitTimeout(10*time.Second),
		WithRenderTimeout(1*time.Second),
		WithMaxLinks(15),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer fetcher.Close()

	if fetcher.config.Headless != true {
		t.Errorf("expected headless to be true")
	}
	if fetcher.config.WaitTimeout != 10*time.Second {
		t.Errorf("expected wait timeout to be 10s, got %v", fetcher.config.WaitTimeout)
	}
	if fetcher.config.MaxLinks != 15 {
		t.Errorf("expected max links to be 15, got %d", fetcher.config.MaxLinks)
	}
}

func TestDefaultBrowserFetcherConfig(t *testing.T) {
	config := DefaultBrowserFetcherConfig()

	if config.Headless != true {
		t.Errorf("expected default headless to be true")
	}
	if config.WaitTimeout != 15*time.Second {
		t.Errorf("expected default wait timeout to be 15s, got %v", config.WaitTimeout)
	}
	if config.RenderTimeout != 3*time.Second {
		t.Errorf("expected default render timeout to be 3s, got %v", config.RenderTimeout)
	}
	if config.MaxLinks != 20 {
		t.Errorf("expected default max links to be 20, got %d", config.MaxLinks)
	}
}

func TestBrowserFetcher_FetchSPA_ContextCancelled(t *testing.T) {
	fetcher, err := NewBrowserFetcherWithOptions(
		WithHeadless(true),
		WithWaitTimeout(10*time.Second),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer fetcher.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = fetcher.FetchSPA(ctx, "https://example.com")
	if err == nil {
		t.Fatal("expected error due to cancelled context")
	}
}

func TestBrowserFetcher_ClickAndWait_ContextCancelled(t *testing.T) {
	fetcher, err := NewBrowserFetcherWithOptions(
		WithHeadless(true),
		WithWaitTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer fetcher.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = fetcher.ClickAndWait(ctx, "https://example.com", "a", "body")
	if err == nil {
		t.Fatal("expected error due to cancelled context")
	}
}

func TestBrowserFetcher_Fetch(t *testing.T) {
	fetcher, err := NewBrowserFetcherWithOptions(
		WithHeadless(true),
		WithWaitTimeout(10*time.Second),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer fetcher.Close()

	result, err := fetcher.Fetch(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.URL != "https://example.com" {
		t.Errorf("expected URL to be https://example.com, got %s", result.URL)
	}

	if result.StatusCode != 200 {
		t.Errorf("expected status code 200, got %d", result.StatusCode)
	}

	if len(result.Body) == 0 {
		t.Error("expected non-empty body")
	}
}

func TestBrowserFetcher_FetchWithDetails(t *testing.T) {
	fetcher, err := NewBrowserFetcherWithOptions(
		WithHeadless(true),
		WithWaitTimeout(10*time.Second),
		WithRenderTimeout(1*time.Second),
		WithMaxLinks(5),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer fetcher.Close()

	result, err := fetcher.FetchWithDetails(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.URL != "https://example.com" {
		t.Errorf("expected URL to be https://example.com, got %s", result.URL)
	}

	if len(result.HTML) == 0 {
		t.Error("expected non-empty HTML")
	}
}

func TestBrowserFetcher_Close(t *testing.T) {
	fetcher, err := NewBrowserFetcherWithOptions(
		WithHeadless(true),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err = fetcher.Close()
	if err != nil {
		t.Fatalf("expected no error on close, got %v", err)
	}
}
