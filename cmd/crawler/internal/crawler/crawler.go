package crawler

import (
	"golangwebcrawler/cmd/crawler/internal/models"
	"log"
	"sync"
)

type Fetcher interface {
	Fetch(url string) (models.FetchResult, error)
}

type CrawlerState struct {
	Visited map[string]bool
	Lock    sync.Mutex
}

type Crawler interface {
	CrawlAsync(wg *sync.WaitGroup, startUrl string, fetcher Fetcher, parser Parser) error
	IsNavigated(url string) bool
	MarkVisited(url string)
}

type Parser interface {
	ParseLinks(body []byte) ([]string, error)
}

// Compile-time assertion to ensure CrawlerState implements Crawler interface
var _ Crawler = (*CrawlerState)(nil)

func (c *CrawlerState) CrawlAsync(wg *sync.WaitGroup, startUrl string, fetcher Fetcher, parser Parser) error {
	defer wg.Done()
	c.Lock.Lock()
	if c.Visited == nil {
		c.Visited = make(map[string]bool)
	}
	if c.Visited[startUrl] {
		c.Lock.Unlock()
		return nil
	}
	c.Visited[startUrl] = true
	c.Lock.Unlock()

	result, err := fetcher.Fetch(startUrl)
	if err != nil {
		return err
	}

	links, err := parser.ParseLinks(result.Body)
	if err != nil {
		return err
	}

	for _, link := range links {
		wg.Add(1)
		go func(url string) {
			if err := c.CrawlAsync(wg, url, fetcher, parser); err != nil {
				log.Printf("error crawling URL %s: %v", url, err)
			}
		}(link)
	}

	// todo store results in some way, maybe in a channel or a shared data structure with proper synchronization
	return nil
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
