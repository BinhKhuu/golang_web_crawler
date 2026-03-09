package crawler

import "sync"

type FetchResult struct {
	URL        string
	StatusCode int
	Body       []byte
	Err        error
}

type Fetcher interface {
	Fetch(url string) (FetchResult, error)
}

type CrawlerState struct {
	Visited map[string]bool
	Lock    sync.Mutex
}

type Crawler interface {
	Crawl(startUrl string, fetcher Fetcher) ([]string, error)
	IsNavigated(url string) bool
	MarkVisited(url string)
}

// Compile-time assertion to ensure CrawlerState implements Crawler interface
var _ Crawler = (*CrawlerState)(nil)

func (c *CrawlerState) Crawl(startUrl string, fetcher Fetcher) ([]string, error) {
	c.Lock.Lock()
	if c.Visited == nil {
		c.Visited = make(map[string]bool)
	}
	c.Visited[startUrl] = true
	c.Lock.Unlock()

	result, err := fetcher.Fetch(startUrl)
	if err != nil {
		return nil, err
	}

	// todo extract links from result.Body and crawl them recursively

	return []string{result.URL}, nil
}

func (c *CrawlerState) IsNavigated(url string) bool {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	return c.Visited[url]
}

func (c *CrawlerState) MarkVisited(url string) {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	if c.Visited == nil {
		c.Visited = make(map[string]bool)
	}
	c.Visited[url] = true
}

func NewCrawler() *CrawlerState {
	return &CrawlerState{
		Visited: make(map[string]bool),
	}
}
