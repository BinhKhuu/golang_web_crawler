package crawler

import (
	"golangwebcrawler/cmd/crawler/internal/models"
	"log"
	"net/url"
	"strings"
	"sync"
)

type Fetcher interface {
	Fetch(url string) (models.FetchResult, error)
}

type CrawlerState struct {
	visited        map[string]bool
	lock           sync.Mutex
	wg             sync.WaitGroup
	maxDepth       int
	allowedDomains []string
}

type Crawler interface {
	CrawlAsync(startUrl string, currentDepth int, fetcher Fetcher, parser Parser, storage StorageService) error
	IsNavigated(url string) bool
	MarkVisited(url string)
}

type Parser interface {
	ParseLinks(body []byte) ([]string, error)
}

// StorageService defines how crawl results are persisted
type StorageService interface {
	StoreRawData(result models.RawData) error
}

func (c *CrawlerState) CrawlAsync(startUrl string, depth int, fetcher Fetcher, parser Parser, storage StorageService) error {
	if depth == 0 {
		// todo maybe add logging to indicate why we are stopping the crawl at this point
		return nil // Reached max depth, stop crawling further
	}
	if !isAllowedDomain(startUrl, c.allowedDomains) {
		return nil
	}
	c.lock.Lock()
	if c.visited == nil {
		c.visited = make(map[string]bool)
	}
	if c.visited[startUrl] {
		c.lock.Unlock()
		return nil
	}
	c.visited[startUrl] = true
	c.lock.Unlock()

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
		if err := storage.StoreRawData(crawlResult); err != nil {
			log.Printf("error storing result for URL %s: %v", startUrl, err)
		}
	}

	for _, link := range links {
		c.wg.Add(1)
		go func(url string) {
			defer c.wg.Done()
			depth := depth - 1
			if err := c.CrawlAsync(url, depth, fetcher, parser, storage); err != nil {
				log.Printf("error crawling URL %s: %v", url, err)
			}
		}(link)
	}

	return nil
}

func isAllowedDomain(url string, allowedDomains []string) bool {
	for _, domain := range allowedDomains {
		if containsDomain(url, domain) {
			return true
		}
	}
	return false
}

func containsDomain(rawURL, domain string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := parsed.Hostname() // strips port if present
	return host == domain || strings.HasSuffix(host, "."+domain)
}

func (c *CrawlerState) IsNavigated(url string) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.visited[url]
}

func (c *CrawlerState) MarkVisited(url string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.visited == nil {
		c.visited = make(map[string]bool)
	}
	c.visited[url] = true
}

func NewCrawler(maxDepth int, allowedDomains []string) *CrawlerState {
	return &CrawlerState{
		visited:        make(map[string]bool),
		wg:             sync.WaitGroup{},
		maxDepth:       maxDepth,
		allowedDomains: allowedDomains,
	}
}

func (c *CrawlerState) Wait() {
	c.wg.Wait()
}
