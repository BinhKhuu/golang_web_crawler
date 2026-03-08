package fetcher

import (
	"golangwebcrawler/cmd/crawler/internal/crawler"
	"io"
	"net/http"
)

type HTTPClient interface {
	Get(url string) (*http.Response, error)
}

type HTTPFetcher struct {
	client HTTPClient
}

func NewHTTPFetcher(client HTTPClient) *HTTPFetcher {
	return &HTTPFetcher{client: client}
}

func (f *HTTPFetcher) Fetch(url string) (crawler.FetchResult, error) {
	resp, err := f.client.Get(url)
	if err != nil {
		return crawler.FetchResult{URL: url, Err: err}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return crawler.FetchResult{
		URL:        url,
		StatusCode: resp.StatusCode,
		Body:       body,
		Err:        nil,
	}, nil
}
