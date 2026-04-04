package crawler

import (
	"context"
	"fmt"
	"golangwebcrawler/cmd/crawler/internal/fetcher"
	"log/slog"
	"net/url"
	"strings"
	"sync"
)

type Fetcher interface {
	Fetch(ctx context.Context, url string) (fetcher.FetchResult, error)
}

type Parser interface {
	ParseLinks(ctx context.Context, body []byte) ([]string, error)
}

type StorageService interface {
	StoreRawData(ctx context.Context, url, contentType, rawContent string) error
}

type Crawler struct {
	maxDepth       int
	allowedDomains []string
	visited        map[string]bool
	mu             sync.Mutex
	logger         *slog.Logger
}

type crawlJob struct {
	url   string
	depth int
}

const channelBufferLimit = 100

func NewCrawler(maxDepth int, allowedDomains []string, logger *slog.Logger) *Crawler {
	return &Crawler{
		maxDepth:       maxDepth,
		allowedDomains: allowedDomains,
		visited:        make(map[string]bool),
		logger:         logger,
	}
}

func (c *Crawler) Crawl(ctx context.Context, startURL string, fetcher Fetcher, parser Parser, storage StorageService, concurrency int) error {
	if concurrency <= 0 {
		concurrency = 10
	}

	jobs := make(chan crawlJob, channelBufferLimit)
	var pending sync.WaitGroup

	if !c.markVisited(startURL) {
		return nil
	}

	for range concurrency {
		go func() {
			for job := range jobs {
				if err := c.processURL(ctx, job, fetcher, parser, storage, jobs, &pending); err != nil {
					c.logger.Warn("error processing URL", "url", job.url, "error", err)
				}
				pending.Done()
			}
		}()
	}

	pending.Add(1)
	jobs <- crawlJob{url: startURL, depth: c.maxDepth}

	pending.Wait()
	close(jobs)

	return nil
}

func (c *Crawler) IsNavigated(u string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.visited[u]
}

// processURL will drop crawls when channel is blocked to prevent deadlocks and keep workers moving. This means some links may be skipped if the queue is full, but it ensures the crawler continues to operate smoothly under load.
func (c *Crawler) processURL(ctx context.Context, job crawlJob, fetch Fetcher, parse Parser, store StorageService, jobs chan<- crawlJob, pending *sync.WaitGroup) error {
	if job.depth <= 0 {
		return nil
	}

	if !c.isAllowedDomain(job.url) {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	fetchResult, err := fetch.Fetch(ctx, job.url)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", job.url, err)
	}

	links, err := parse.ParseLinks(ctx, fetchResult.Body)
	if err != nil {
		return fmt.Errorf("parsing links from %s: %w", job.url, err)
	}

	formattedLinks, err := c.formatLinks(links, job.url)
	if err != nil {
		return fmt.Errorf("formatting links from %s: %w", job.url, err)
	}

	if store != nil {
		if err := store.StoreRawData(ctx, job.url, "", string(fetchResult.Body)); err != nil {
			c.logger.Warn("error storing raw data", "url", job.url, "error", err)
		}
	}

	for _, link := range formattedLinks {
		if c.markVisited(link) {
			pending.Add(1)
			select {
			case jobs <- crawlJob{url: link, depth: job.depth - 1}:
				// channel accepted the job, continue
			default:
				// Channel is FULL. Drop the link and move on to unblock the worker.
				c.logger.Warn("Queue full, dropping link", "url", link)
				pending.Done()
			}
		}
	}

	return nil
}

func (c *Crawler) formatLinks(links []string, baseURL string) ([]string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	for i, link := range links {
		parsed, err := url.Parse(link)
		if err != nil {
			continue
		}
		if !parsed.IsAbs() {
			links[i] = base.ResolveReference(parsed).String()
		}
	}
	return links, nil
}

func (c *Crawler) markVisited(u string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.visited[u] {
		return false
	}
	c.visited[u] = true
	return true
}

func (c *Crawler) isAllowedDomain(rawURL string) bool {
	for _, domain := range c.allowedDomains {
		if c.containsDomain(rawURL, domain) {
			return true
		}
	}
	return false
}

func (c *Crawler) containsDomain(rawURL, domain string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	host := parsed.Hostname()
	return host == domain || strings.HasSuffix(host, "."+domain)
}
