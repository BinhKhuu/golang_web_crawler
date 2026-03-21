package fetcher

import (
	"golangwebcrawler/cmd/crawler/internal/models"
	"io"
	"log"
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

func (f *HTTPFetcher) Fetch(url string) (models.FetchResult, error) {
	resp, err := f.client.Get(url)
	if err != nil {
		return models.FetchResult{URL: url, Err: err}, err
	}

	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			// log error don't want to override original error if it exists
			log.Printf("error closing response body for URL %s: %v", url, closeErr)
		}
	}()

	body, _ := io.ReadAll(resp.Body)
	return models.FetchResult{
		URL:        url,
		StatusCode: resp.StatusCode,
		Body:       body,
		Err:        nil,
	}, nil
}
