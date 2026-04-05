package fetcher

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"
)

type MockHTTPClient struct {
	Response *http.Response
	Err      error
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.Response, m.Err
}

func TestHTTPFetcher_Fetch(t *testing.T) {
	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString("mock body")),
	}
	mockClient := &MockHTTPClient{Response: mockResponse, Err: nil}
	fetcher := NewHTTPFetcher(mockClient)

	const url = "http://example.com"
	result, err := fetcher.Fetch(t.Context(), url)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.StatusCode != http.StatusOK {
		t.Errorf("expected status code 200, got %d", result.StatusCode)
	}

	if string(result.Body) != "mock body" {
		t.Errorf("expected body 'mock body', got %s", string(result.Body))
	}

	if result.URL != url {
		t.Errorf("expected URL %s, got %s", url, result.URL)
	}
}

func Test_HTTPFetcher_Error(t *testing.T) {
	mockError := errors.New("mock error")
	mockClient := &MockHTTPClient{Err: mockError}
	fetcher := NewHTTPFetcher(mockClient)
	const url = "http://example.com"

	_, err := fetcher.Fetch(t.Context(), url)
	if err == nil {
		t.Fatalf("expected error")
	}

	if !errors.Is(err, mockError) {
		t.Errorf("expected error %v, got %v", mockError, err)
	}
}

func Test_HTTPFetcher_Fetch_Http_404(t *testing.T) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	fetcher := NewHTTPFetcher(client)

	const url = "https://httpbin.org/404"
	result, err := fetcher.Fetch(t.Context(), url)
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

func Test_HTTPFetcher_Fetch_ContextCancelled(t *testing.T) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	fetcher := NewHTTPFetcher(client)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	const url = "http://example.com"
	_, err := fetcher.Fetch(ctx, url)
	if err == nil {
		t.Fatalf("expected an error due to cancelled context, got nil")
	}
}
