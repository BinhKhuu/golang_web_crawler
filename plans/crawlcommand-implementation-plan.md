# CrawlCommand TODO Steps 4-7 Implementation Plan

## Overview
Wire up the remaining steps in `cmd/binhcrawler/commands/crawlcommand.go:Execute()` to create jobs, configure the orchestrator, and run the crawl.

## Steps

### Step 4 — Create CrawlJob
1. Create `playwrightfetcher.PlaywrightFetcher` using `pwConfig` via `playwrightfetcher.NewConfiguredPlaywrightFetcher(logger, &pwConfig)`
2. Defer `fetcher.Close()` after creation
3. Initialize storage service (`storage.Service`) — check existing constructor pattern from `main.go`
4. Build crawl function using `crawler.NewCrawler()` with:
   - `c.MaxDepth` for max depth
   - URL from config (CLI override takes priority)
   - `c.Concurrency` for concurrent requests
5. Return `&job.CrawlJob{ExecuteFn: crawlFn, Logger: logger}`

**Tests:**
- `TestExtractAllowedDomains` — table-driven tests for subdomain extraction (e.g. `www.seek.com.au` → `.com.au`, bare domain, invalid URL)
- `TestNewCrawlJob` — verify CrawlJob struct fields (ExecuteFn set, Logger passed through)

### Step 5 — Conditionally create ParseJob
- If `c.ParseAfter` is true:
  - Use `job.NewParseJob()` with `ParseConfig`:
    - `Storage`: storage service instance
    - `ParserFn`: `job.NewDBParserCreator(db)` factory
    - `Logger`: command logger
    - `StartDate`: current time minus 1 minute (match main.go pattern)
    - `BatchSize`: use default from existing codebase
  - Append to jobs slice alongside CrawlJob

**Tests:**
- `TestExecute_ParseAfterTrue` — verify ParseJob created when flag set (mock DB + storage)
- `TestExecute_ParseAfterFalse` — verify ParseJob not created when flag unset

### Step 6 — Parse Mode string to orchestrator.Mode
- Add helper function `parseMode(modeStr string) (orchestrator.Mode, error)`
- Map: `"sequential"` → `Sequential`, `"concurrent"` → `Concurrent`, `"independent"` → `Independent`
- Default to `Sequential` if empty
- Return error for invalid values

**Tests:**
- `TestParseMode` — table-driven tests for all valid modes, empty string default, and invalid input error

### Step 7 — Run orchestrator
1. Collect all jobs into `[]job.Job` (at minimum contains CrawlJob)
2. Call `orchestrator.New(jobs, mode, logger)`
3. Create context with optional timeout from `pwConfig.Timeout`
4. Call `orch.Run(ctx)` and return its error

**Tests:**
- Integration test for full Execute flow with mocked dependencies (fetcher, storage, orchestrator)

## Imports to add
- `context` — for orchestrator.Run(ctx)
- `golangwebcrawler/cmd/binhcrawler/internal/job` — CrawlJob, ParseJob, NewParseJob
- `golangwebcrawler/cmd/binhcrawler/internal/orchestrator` — Orchestrator, Mode
- `golangwebcrawler/internal/storage` — storage.Service
- Possibly `time`, `typeutil` (matching main.go patterns)

## Key references
- Existing job factory pattern: `cmd/binhcrawler/main.go` (`newCrawlJob`, `newParseJob`)
- Orchestrator: `cmd/binhcrawler/internal/orchestrator/orchestrator.go`
- Job types: `cmd/binhcrawler/internal/job/job.go`
