# Playwright Configuration Reference

This document describes all configurable options for the `PlaywrightFetcher` and how they affect page loading, navigation, and data extraction.

## WaitUntilState Options

`WaitUntilState` controls when `page.Goto()` resolves. The Go Playwright binding (`github.com/playwright-community/playwright-go`) exposes four constants:

| Constant | Value | Behavior |
|---|---|---|
| `playwright.WaitUntilStateLoad` | `"load"` | Waits for the `load` event — all resources (CSS, JS, images) fully loaded. Safe default but slower. |
| `playwright.WaitUntilStateDomContentLoaded` | `"domcontentloaded"` | Waits for `DOMContentLoaded` — HTML parsed, before images/CSS finish. Faster but may miss JS-rendered content. |
| `playwright.WaitUntilStateNetworkIdle` | `"networkidle"` | Waits for no network connections for 500ms. **Avoid** — persistent connections (WebSockets, polling, trackers) cause hangs until timeout. |
| `playwright.WaitUntilStateCommit` | `"commit"` | Waits for response to be committed — fastest but may return incomplete content. |

**Current usage**: `FetchDefault` uses `WaitUntilStateLoad`; `FetchSPAConfig` omits it (defaults to `"auto"`). The recommended pattern is fast navigation + explicit `WaitForSelector` for content readiness.

## Configuration Structs

### PlaywrightFetcherConfig

| Field | Type | Description |
|---|---|---|
| `URL` | `string` | Base URL to crawl |
| `Headless` | `bool` | Run browser in headless mode |
| `Timeout` | `int` | Timeout in ms for navigation, waits, clicks (default: 10000) |
| `MaxItems` | `int` | Max items to scrape (0 = unlimited) |
| `Search` | `SearchConfig` | Search interaction settings |
| `Results` | `ResultsConfig` | Result extraction selectors |
| `Canonicalization` | `CanonicalizationConfig` | URL normalization rules |
| `Pagination` | `PaginationConfig` | Pagination navigation settings |

### SearchConfig

Controls the search interaction performed after page load. Leave fields empty to skip.

| Field | Type | Description |
|---|---|---|
| `InputSelectors` | `[]string` | Selectors for the search input field (first match used) |
| `Query` | `string` | Search query text to fill into the input |
| `SubmitSelectors` | `[]string` | Selectors for the submit button (first match used) |

### ResultsConfig

Controls how results are extracted from search results and detail pages.

| Field | Type | Description |
|---|---|---|
| `ListingSelectors` | `[]string` | Selectors for job listing links on search page (first match used to locate listings) |
| `DataSelectors` | `[]string` | Selectors for content on detail pages (all matches used to extract data) |

### CanonicalizationConfig

Controls URL normalization for deduplication.

| Field | Type | Description |
|---|---|---|
| `IgnoreQueryParams` | `[]string` | Query params to strip (e.g. tracking params like `ref`, `sol`) |
| `RootRelativePrefixes` | `[]string` | Hrefs starting with these prefixes treated as root-relative (e.g. `"job/"` → `"/job/"`) |

### PaginationConfig

Controls navigation through paginated results.

| Field | Type | Description |
|---|---|---|
| `ContainerSelectors` | `[]string` | Selectors that indicate a pagination section exists (first match) |
| `NextSelectors` | `[]string` | Selectors for "next page" button (tried in order) |
| `PageNumberSelectors` | `[]string` | Selectors for numbered page buttons (tried in order) |
| `DisabledSelector` | `string` | Selector indicating "next" is disabled (e.g. `button[disabled]`) |
| `WaitForSelectors` | `[]string` | After clicking next, wait for these selectors to appear |
| `Strategy` | `string` | `"auto"` (default) \| `"next-button"` \| `"page-numbers"` |
| `MaxRetries` | `int` | Retries on next-page click failure (default: 2) |
| `MaxPages` | `int` | Maximum pages to crawl |

## Fetch Modes

### FetchDefault — Static Page Scraping

Uses `WaitUntilStateLoad` and extracts links from a hardcoded selector (`a[id*='job-title']`). Simple, no configuration needed.

### FetchSPAConfig — Configurable SPA Crawling

Uses default wait (`"auto"`), performs search interaction, then iterates through pages using the pagination config. Flow:

```
1. Navigate to URL (default WaitUntilState)
2. Fill search input (if Search.InputSelectors configured)
3. Submit search (if Search.SubmitSelectors configured)
4. Loop per page:
   a. waitAndCollectResults() → extract listings via ListingSelectors
   b. if MaxItems reached → stop
   c. if no pagination container → stop
   d. if next button disabled → stop
   e. clickNextPageWithRetry() → try NextSelectors or PageNumberSelectors
   f. waitForNextPageLoad() → wait for WaitForSelectors
   g. randomDelay() → 1-3 second jitter
5. Return all results
```

## Random Delay

A random delay of 1-3 seconds is applied between pagination clicks and item fetches to avoid rate limiting. Implemented in [`randomDelay()`](internal/fetcher/playwrightfetcher/playwright_fetcher.go:274).

## Example Configuration (SEEK)

[`cmd/binhcrawler/configs/seek.json`](cmd/binhcrawler/configs/seek.json):

```json
{
  "url": "https://www.seek.com.au",
  "headless": true,
  "timeout": 10000,
  "maxItems": 0,
  "search": {
    "inputSelectors": ["input[data-automation='search-input-text']"],
    "query": "software engineer",
    "submitSelectors": ["button[data-automation='search-input-submit']"]
  },
  "results": {
    "listingSelectors": ["a[data-automation='jobTitle']"],
    "dataSelectors": [
      "[data-automation='job-detail-title']",
      "[data-automation='job-detail-company-name']",
      "[data-automation='job-detail-salary-text']",
      "[data-automation='job-detail-location-text']",
      "[data-automation='job-detail-description']"
    ]
  },
  "canonicalization": {
    "ignoreQueryParams": ["icmlk", "ref"],
    "rootRelativePrefixes": ["job/"]
  },
  "pagination": {
    "containerSelectors": ["[data-automation='pagination']"],
    "nextSelectors": ["button[data-automation='pagination-next']"],
    "disabledSelector": "button[disabled]",
    "waitForSelectors": ["a[data-automation='jobTitle']"],
    "strategy": "auto",
    "maxRetries": 2,
    "maxPages": 10
  }
}
```
