# Fix Orchestrator Context Deadline Exceeded

## Problem

```
orchestrator run: context cancelled before job 1: context deadline exceeded
```

The orchestrator finishes and doesn't wait for the crawler because the global context expires mid-crawl.

## Root Cause

**`cmd/binhcrawler/commands/crawlcommand.go:104-109`**

```go
ctx := context.Background()
if pwConfig.Timeout > 0 {
    var cancel context.CancelFunc
    ctx, cancel = context.WithTimeout(ctx, time.Duration(pwConfig.Timeout)*time.Millisecond)
    defer cancel()
}
```

`pwConfig.Timeout` (default: 10,000ms = 10s) is used as a **global context deadline** for the entire orchestrator run (crawl + parse). But it was designed as a **per-operation Playwright timeout** — used in `playwright_fetcher.go:266` for `waitForElementVisibility`.

10 seconds is far too short for crawling a site with:
- Max depth of 3
- Multiple pages per level
- 1-3 second random delays between requests
- Playwright page load and interaction operations

### Execution Flow

```
CrawlCommand.Execute()
  ctx = context.WithTimeout(10s)
  orch.Run(ctx)
    runSequential(ctx)
      job 0 (Crawl): j.Execute(ctx)  <-- starts crawling
        [10 seconds pass while crawler is still working]
        ctx expires -> context.DeadlineExceeded
      job 1 (Parse): ctx.Err() != nil  <-- "context cancelled before job 1"
        return error
```

### Per-Operation Timeouts Already in Place

These will continue to protect against hangs:
- `playwright_fetcher.go:115,173` — 30s hardcoded page Goto timeout
- `playwright_fetcher.go:266` — `pwConfig.Timeout` for element wait
- `ctx.Err()` checks throughout crawler and fetcher

## Changes

### 1. `cmd/binhcrawler/commands/crawlcommand.go:104-111`

Remove the context timeout wrapper. Use `context.Background()` directly for the orchestrator.

**Replace:**
```go
ctx := context.Background()
if pwConfig.Timeout > 0 {
    var cancel context.CancelFunc
    ctx, cancel = context.WithTimeout(ctx, time.Duration(pwConfig.Timeout)*time.Millisecond)
    defer cancel()
}

if err := orch.Run(ctx); err != nil {
```

**With:**
```go
if err := orch.Run(context.Background()); err != nil {
```

### 2. Verify `"time"` import

Check if `"time"` is still needed after this change. It is — `typeutil.UTCTimeNow().Add(-1 * time.Minute)` at line 151 still uses it. No import change needed.

## Behavior After Fix

| Scenario | Before | After |
|----------|--------|-------|
| Crawl completes in < 10s | Works | Works |
| Crawl takes > 10s | Fails with "context cancelled before job 1" | Completes normally |
| Parse after crawl | Never reaches parse | Parse runs after crawl completes |
| Per-operation timeouts | 10s global + 30s page | 30s page + pwConfig.Timeout for waits |
