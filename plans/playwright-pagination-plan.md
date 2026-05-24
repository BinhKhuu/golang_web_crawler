# Playwright Pagination Support Plan

## Status: Complete ✅

## Problem

`FetchSPAConfig()` in `playwright_fetcher.go` only scrapes the first page of search results. SPA applications typically have pagination sections with "Next" buttons or numbered page links. The scraper should navigate through pages until a configured item limit is reached or no more pages exist.

## Changes

### 1. Add `PaginationConfig` struct

**File:** `internal/fetcher/playwrightfetcher/playwright_fetcher.go`

```go
type PaginationConfig struct {
    // ContainerSelectors: selectors that indicate a pagination section exists
    ContainerSelectors []string `json:"containerSelectors"`
    // NextSelectors: selectors for "next page" button (tried in order)
    NextSelectors []string `json:"nextSelectors"`
    // PageNumberSelectors: selectors for numbered page buttons
    PageNumberSelectors []string `json:"pageNumberSelectors"`
    // DisabledSelector: attribute/selector indicating "next" is disabled
    DisabledSelector string `json:"disabledSelector"`
    // WaitForSelectors: after clicking next, wait for these to appear
    WaitForSelectors []string `json:"waitForSelectors"`
    // Strategy: "auto" (default) | "next-button" | "page-numbers"
    Strategy string `json:"strategy"`
    // MaxRetries: retries on next-page click failure (default: 2)
    MaxRetries int `json:"maxRetries"`
}
```

### 2. Add `MaxItems` to `PlaywrightFetcherConfig`

**File:** `internal/fetcher/playwrightfetcher/playwright_fetcher.go`

Add field:
```go
MaxItems int `json:"maxItems"` // 0 = unlimited
```

### 3. Add SEEK pagination selectors

**File:** `internal/fetcher/playwrightfetcher/seekplaywrightconfig.go`

Add constants and wire into `GetSeekConfiguration()`:
```go
seekPaginationContainer   = "[data-automation='pagination']"
seekPaginationNext        = "button[data-automation='pagination-next']"
seekPaginationDisabled    = "button[disabled]"
```

### 4. Update `DefaultConfig()`

**File:** `internal/fetcher/playwrightfetcher/playwright_fetcher.go`

Add pagination defaults with `Strategy: "auto"`, `MaxRetries: 2`.

### 5. Update `seek.json`

**File:** `cmd/binhcrawler/configs/seek.json`

Add pagination section:
```json
"pagination": {
  "containerSelectors": ["[data-automation='pagination']"],
  "nextSelectors": ["button[data-automation='pagination-next']"],
  "disabledSelector": "button[disabled]",
  "waitForSelectors": ["a[data-automation='jobTitle']"],
  "strategy": "auto",
  "maxRetries": 2
},
"maxItems": 0
```

### 6. Add `--max-items` CLI flag

**File:** `cmd/binhcrawler/commands/crawlcommand.go`

Add field to `CrawlCommand`:
```go
MaxItems int `default:"0" description:"Max items to scrape (0 = unlimited)" long:"max-items"`
```

Wire into `buildPlaywrightFetcherConfig()` merge logic.

### 7. Refactor `FetchSPAConfig()` with pagination loop

**File:** `internal/fetcher/playwrightfetcher/playwright_fetcher.go`

New flow:
```
FetchSPAConfig():
  1. Navigate to URL
  2. Fill search input
  3. Submit search
  4. Loop:
     a. waitAndCollectResults() → append to results
     b. if MaxItems > 0 && len(results) >= MaxItems → trim & break
     c. if !hasPaginationSection(p) → break
     d. if isNextDisabled(p) → break
     e. clickNextPageWithRetry(ctx, p) → on all retries exhausted, break
     f. waitForNextPageLoad(ctx, p)
     g. randomDelay()
  5. Return results
```

### 8. New helper methods on `PlaywrightFetcher`

**File:** `internal/fetcher/playwrightfetcher/playwright_fetcher.go`

| Method | Purpose |
|--------|---------|
| `hasPaginationSection(p playwright.Page) bool` | Check if any `ContainerSelectors` exist on page |
| `isNextDisabled(p playwright.Page) bool` | Check if next button matches `DisabledSelector` |
| `clickNextPageWithRetry(ctx context.Context, p playwright.Page) error` | Click next with retry logic (up to `MaxRetries`) |
| `waitForNextPageLoad(ctx context.Context, p playwright.Page) error` | Wait for `WaitForSelectors` to appear after navigation |
| `shouldStopPagination(results []crawler.FetchResult) bool` | Check `MaxItems` limit |

## Tests

**File:** `cmd/binhcrawler/commands/crawlcommand_test.go`

- `TestPaginationConfig_Defaults` — verify default pagination config values
- `TestPaginationConfig_JSONMerge` — verify JSON config merges correctly
- `TestBuildConfig_MaxItemsFlag` — verify CLI `--max-items` flag overrides

## Files Modified

| File | Change |
|------|--------|
| `internal/fetcher/playwrightfetcher/playwright_fetcher.go` | Add `PaginationConfig`, `MaxItems`, refactor `FetchSPAConfig()`, add 5 helper methods |
| `internal/fetcher/playwrightfetcher/seekplaywrightconfig.go` | Add SEEK pagination selector constants |
| `cmd/binhcrawler/configs/seek.json` | Add pagination section |
| `cmd/binhcrawler/commands/crawlcommand.go` | Add `--max-items` CLI flag |
| `cmd/binhcrawler/commands/crawlcommand_test.go` | Add tests for pagination config merge and `--max-items` flag |

## Verification

```bash
golangci-lint run --fix
golangci-lint run ./...
go test ./...
```
