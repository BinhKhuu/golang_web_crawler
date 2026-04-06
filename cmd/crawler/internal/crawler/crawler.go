package crawler

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Fetcher interface {
	Fetch(ctx context.Context, url string) (FetchResult, error)
}

type FetchResult struct {
	URL        string
	StatusCode int
	Body       []byte
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

type discoveredLinks struct {
	links []string
	depth int
}

func NewCrawler(maxDepth int, allowedDomains []string, logger *slog.Logger) *Crawler {
	return &Crawler{
		maxDepth:       maxDepth,
		allowedDomains: allowedDomains,
		visited:        make(map[string]bool),
		logger:         logger,
	}
}

/*
Crawl performs concurrent web crawling with the following architecture:

Coordinator manages crawl state, coordinates workers, and prevents deadlocks:
  - Tracks pending jobs and queued discoveries
  - Enforces graceful shutdown with a 5-second grace period
  - Terminates when no work remains or timeout expires

Channels:
  - jobs: URLs ready to be crawled by worker goroutines
  - discovered: Newly found URLs waiting to be queued (backpressure buffer)
  - done: Signals crawl completion to prevent early return

Termination conditions:
 1. Context is cancelled (external cancellation)
 2. No pending jobs AND discovery queue empty for 5 seconds (natural completion)

Workers process URLs concurrently (limited by concurrency parameter) and send
discovered links back to the coordinator for scheduling.
*/
func (c *Crawler) Crawl(ctx context.Context, startURL string, fetcher Fetcher, parser Parser, storage StorageService, concurrency int) error {
	if concurrency <= 0 {
		concurrency = 10
	}

	jobs := make(chan crawlJob)
	discovered := make(chan discoveredLinks, concurrency)
	var workerWG sync.WaitGroup
	done := make(chan struct{})

	if !c.markVisited(startURL) {
		return nil
	}

	go func() {
		c.coordinator(ctx, jobs, discovered, &workerWG, done)
	}()

	for range concurrency {
		workerWG.Go(func() {
			for job := range jobs {
				links, err := c.processURL(ctx, job, fetcher, parser, storage)
				if err != nil {
					c.logger.Warn("error processing URL", "url", job.url, "error", err)
				}
				discovered <- discoveredLinks{links: links, depth: job.depth}
			}
		})
	}

	jobs <- crawlJob{url: startURL, depth: c.maxDepth}

	<-done

	return nil
}

func (c *Crawler) IsNavigated(u string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.visited[u]
}

func (c *Crawler) coordinator(ctx context.Context, jobs chan<- crawlJob, discovered <-chan discoveredLinks, workerWG *sync.WaitGroup, done chan<- struct{}) {
	pending := 1
	var queue []crawlJob
	const resetSeconds = 5
	idleTimer := time.NewTimer(resetSeconds * time.Second)
	if !idleTimer.Stop() {
		<-idleTimer.C
	}
	for {
		var sendCh chan<- crawlJob
		var jobToSend crawlJob

		if len(queue) > 0 {
			sendCh = jobs
			jobToSend = queue[0]
			idleTimer.Stop()
		}

		select {
		case <-ctx.Done(): // interrupt signal release all resources and return
			close(jobs)
			workerWG.Wait()
			close(done)
			return
		case sendCh <- jobToSend: // is sendCh ready for a job
			queue = queue[1:]
		case batch := <-discovered: // goroutine discovered new links and pushed to discovered queue
			pending--
			for _, link := range batch.links {
				if c.markVisited(link) {
					pending++
					queue = append(queue, crawlJob{url: link, depth: batch.depth - 1})
				}
			}
			if pending == 0 && len(queue) == 0 {
				idleTimer.Reset(resetSeconds * time.Second)
			}
		case <-idleTimer.C:
			if pending == 0 && len(queue) == 0 {
				close(jobs)
				workerWG.Wait()
				close(done)
				return
			}
		}
	}
}

func (c *Crawler) processURL(ctx context.Context, job crawlJob, fetch Fetcher, parse Parser, store StorageService) ([]string, error) {
	if job.depth <= 0 {
		return nil, nil
	}

	if !c.isAllowedDomain(job.url) {
		return nil, nil
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	fetchResult, err := fetch.Fetch(ctx, job.url)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", job.url, err)
	}

	links, err := parse.ParseLinks(ctx, fetchResult.Body)
	if err != nil {
		return nil, fmt.Errorf("parsing links from %s: %w", job.url, err)
	}

	formattedLinks, err := c.formatLinks(links, job.url)
	if err != nil {
		return nil, fmt.Errorf("formatting links from %s: %w", job.url, err)
	}

	if store != nil {
		if err := store.StoreRawData(ctx, job.url, "", string(fetchResult.Body)); err != nil {
			c.logger.Warn("error storing raw data", "url", job.url, "error", err)
		}
	}

	return formattedLinks, nil
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
