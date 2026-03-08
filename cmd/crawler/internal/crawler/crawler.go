package crawler

type FetchResult struct {
	URL        string
	StatusCode int
	Body       []byte
	Err        error
}

type Fetcher interface {
	Fetch(url string) (FetchResult, error)
}

func Crawl(startUrl string, fetcher Fetcher) ([]string, error) {
	return []string{}, nil
}
