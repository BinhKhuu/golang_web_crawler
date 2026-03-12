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
	CrawlAsync(wg *sync.WaitGroup, startUrl string, fetcher Fetcher, parser Parser, storage StorageService) error
	IsNavigated(url string) bool
	MarkVisited(url string)
}

type Parser interface {
	ParseLinks(body []byte) ([]string, error)
}

// StorageService defines how crawl results are persisted
type StorageService interface {
	Store(result models.RawData) error
}

// Compile-time assertion to ensure CrawlerState implements Crawler interface
var _ Crawler = (*CrawlerState)(nil)

func (c *CrawlerState) CrawlAsync(wg *sync.WaitGroup, startUrl string, fetcher Fetcher, parser Parser, storage StorageService) error {
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

	fetchResult, err := fetcher.Fetch(startUrl)
	if err != nil {
		return err
	}

	links, err := parser.ParseLinks(fetchResult.Body)
	if err != nil {
		return err
	}

	// todo  this should be storage data model not crawl result datamodel
	// todo parser should attempt to clean up the raw_content? or should that be a seperate process not related to crawl
	if storage != nil {
		crawlResult := models.RawData{
			URL:         startUrl,
			ContentType: "", // This can be set based on fetchResult if needed
			Raw_content: string(fetchResult.Body),
			Fetched_at:  "", // This can be set to current timestamp if needed
		}
		if err := storage.Store(crawlResult); err != nil {
			log.Printf("error storing result for URL %s: %v", startUrl, err)
		}
	}

	for _, link := range links {
		wg.Add(1)
		go func(url string) {
			if err := c.CrawlAsync(wg, url, fetcher, parser, storage); err != nil {
				log.Printf("error crawling URL %s: %v", url, err)
			}
		}(link)
	}

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
