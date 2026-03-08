package fetcher

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"
)

type MockHTTPClient struct {
	Response *http.Response
	Err      error
}

func (m *MockHTTPClient) Get(url string) (*http.Response, error) {
	return m.Response, m.Err
}

func TestHTTPFetcher_Fetch(t *testing.T) {
	mockResponse := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString("mock body")),
	}
	mockClient := &MockHTTPClient{Response: mockResponse, Err: nil}
	fetcher := NewHTTPFetcher(mockClient)

	const url = "http://example.com"
	result, err := fetcher.Fetch(url)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.StatusCode != 200 {
		t.Errorf("expected status code 200, got %d", result.StatusCode)
	}

	if string(result.Body) != "mock body" {
		t.Errorf("expected body 'mock body', got %s", string(result.Body))
	}

	if result.URL != url {
		t.Errorf("expected URL %s, got %s", url, result.URL)
	}
}

func TestHTTPFetcher_Fetch_Http_404(t *testing.T) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	fetcher := NewHTTPFetcher(client)

	const url = "https://tools-httpstatus.pickup-services.com/404"
	result, err := fetcher.Fetch(url)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.StatusCode != http.StatusNotFound {
		t.Errorf("expected return 404 but got %v", result.StatusCode)
	}

	if result.URL != url {
		t.Errorf("expected URL %s, got %s", url, result.URL)
	}
}
