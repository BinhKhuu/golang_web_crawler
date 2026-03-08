package fetcher

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"
)

type MockHTTPClient_Success struct {
	Response *http.Response
	Err      error
}

func (m *MockHTTPClient_Success) Get(url string) (*http.Response, error) {
	return m.Response, m.Err
}

type MockHTTPClient_Error struct {
	Err error
}

func (m *MockHTTPClient_Error) Get(url string) (*http.Response, error) {
	return nil, m.Err
}

func TestHTTPFetcher_Fetch(t *testing.T) {
	mockResponse := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString("mock body")),
	}
	mockClient := &MockHTTPClient_Success{Response: mockResponse, Err: nil}
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

func Test_HTTPFetcher_Error(t *testing.T) {
	mockError := errors.New("mock error")
	mockClient := &MockHTTPClient_Error{Err: mockError}
	fetcher := NewHTTPFetcher(mockClient)
	const url = "http://example.com"

	_, err := fetcher.Fetch(url)
	if err == nil {
		t.Fatalf("expected error")
	}

	if err != mockError {
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

func Test_HTTPFetcher_Fetch_Error(t *testing.T) {
	mockError := &MockHTTPClient_Error{Err: http.ErrHandlerTimeout}
	fetcher := NewHTTPFetcher(mockError)

	const url = "http://example.com"
	_, err := fetcher.Fetch(url)
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}

	if err != http.ErrHandlerTimeout {
		t.Errorf("expected error %v, got %v", http.ErrHandlerTimeout, err)
	}
}
