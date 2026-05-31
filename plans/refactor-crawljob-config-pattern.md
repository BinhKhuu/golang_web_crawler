# Refactor CrawlJob to Config Struct + Factory Pattern

Make `CrawlJob` consistent with `ParseJob` by introducing `CrawlConfig` and `NewCrawlJob`.

## Steps

1. **`job.go`** — Add interfaces after `ParserJob`:
   - `CrawlFetcherJob`: `Fetch(ctx context.Context, url string) ([]crawler.FetchResult, error)`
   - `CrawlStorageJob`: `StoreRawDataBatch(ctx context.Context, items []models.RawDataItem) error`
   - New imports: `golangwebcrawler/internal/crawler`, `golangwebcrawler/internal/crawlerparser`

2. **`job.go`** — Add `CrawlConfig` struct: `StartURL`, `Fetcher CrawlFetcherJob`, `Storage CrawlStorageJob`, `MaxDepth`, `AllowedDomains`, `Concurrency`, `Logger`

3. **`job.go`** — Add `NewCrawlJob(cfg *CrawlConfig) *CrawlJob` factory — closure calls `crawler.NewCrawler` + `crawlerparser.NewHTTPParser` + `c.Crawl(...)`, returns `&CrawlJob{ExecuteFn: executeFn, Logger: cfg.Logger}`

4. **`crawlcommand.go`** — Delete private `newCrawlJob()`. Replace call site with `job.NewCrawlJob(&job.CrawlConfig{...})`.

5. **`job_test.go`** — No changes. Tests construct `&CrawlJob{ExecuteFn: mockFn}` directly to test `Execute()` behaviour; factory is not involved.

## Verification
- `go build ./...`
- `go test ./cmd/binhcrawler/...`
- `golangci-lint run ./...`

## Notes
- `CrawlStorageJob` is narrower than `crawler.StorageService` — only `StoreRawDataBatch` is actually called by the crawler.
- `CrawlFetcherJob` mirrors `crawler.Fetcher` exactly; the import is unavoidable due to `FetchResult` type.
