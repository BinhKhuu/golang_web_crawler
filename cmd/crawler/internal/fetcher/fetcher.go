package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

type FetchResult struct {
	URL        string
	StatusCode int
	Body       []byte
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type HTTPFetcher struct {
	client HTTPClient
}

func NewHTTPFetcher(client HTTPClient) *HTTPFetcher {
	return &HTTPFetcher{client: client}
}

func (f *HTTPFetcher) Fetch(ctx context.Context, url string) (FetchResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return FetchResult{}, fmt.Errorf("creating request for %s: %w", url, err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return FetchResult{}, fmt.Errorf("fetching %s: %w", url, err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return FetchResult{}, fmt.Errorf("reading body for %s: %w", url, err)
	}

	return FetchResult{
		URL:        url,
		StatusCode: resp.StatusCode,
		Body:       body,
	}, nil
}
