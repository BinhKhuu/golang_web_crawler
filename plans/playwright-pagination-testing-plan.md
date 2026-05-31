# Playwright Pagination Testing Plan

## Goal
Enable independent unit testing of pagination logic by decoupling result collection from the pagination loop.

## Pattern: Callback Injection (Strategy Pattern)
Pass `waitAndCollectResults` as a callback parameter to `collectPageResults`, allowing tests to inject mock collectors while production code uses the real method.

## Changes Required

### 1. Modify `collectPageResults` — Add Callback Parameter
**File**: [`internal/fetcher/playwrightfetcher/playwright_fetcher.go`](internal/fetcher/playwrightfetcher/playwright_fetcher.go:227)

**Current signature**:
```go
func collectPageResults(ctx context.Context, f *PlaywrightFetcher, p playwright.Page) ([]crawler.FetchResult, error)
```

**New signature**:
```go
func collectPageResults(ctx context.Context, f *PlaywrightFetcher, p playwright.Page, collectFn func(ctx context.Context, p playwright.Page) ([]crawler.FetchResult, error)) ([]crawler.FetchResult, error)
```

**Change**: Replace the internal call `f.waitAndCollectResults(ctx, p)` with `collectFn(ctx, p)`.

### 2. Update Callers of `collectPageResults`
**File**: [`internal/fetcher/playwrightfetcher/playwright_fetcher.go`](internal/fetcher/playwrightfetcher/playwright_fetcher.go:219)

In [`FetchSPAConfig`](internal/fetcher/playwrightfetcher/playwright_fetcher.go:182), update the call to pass `f.waitAndCollectResults` as the callback:

```go
res, err := collectPageResults(ctx, f, p, f.waitAndCollectResults)
```

### 3. Add Pagination Unit Tests
**File**: [`internal/fetcher/playwrightfetcher/playwright_fetcher_test.go`](internal/fetcher/playwrightfetcher/playwright_fetcher_test.go)

#### Test 3a: `Test_HasPaginationSection`
- **Given**: HTML with/without pagination container selectors
- **When**: `f.hasPaginationSection(p)` is called
- **Then**: Returns `true` when container exists, `false` otherwise

#### Test 3b: `Test_IsNextDisabled`
- **Given**: HTML with next button having/missing disabled attribute
- **When**: `f.isNextDisabled(p)` is called
- **Then**: Returns `true` when disabled selector matches, `false` otherwise

#### Test 3c: `Test_CollectPageResults_PaginatesMultiplePages`
- **Given**: Mock `collectFn` returning controlled results per page
- **When**: `collectPageResults(ctx, f, p, collectFn)` is called with real Playwright page
- **Then**: Verifies pagination loop navigates correct number of pages

#### Test 3d: `Test_CollectPageResults_StopsAtMaxItems`
- **Given**: Config with `MaxItems = 4`, mock returning 2 results per page
- **When**: Pagination loop runs
- **Then**: Stops after collecting 4 results (not all available)

#### Test 3e: `Test_CollectPageResults_StopsAtLastPage`
- **Given**: HTML where last page has no next button
- **When**: Pagination loop runs
- **Then**: Stops gracefully without error

### 4. Update Existing Tests
**File**: [`internal/fetcher/playwrightfetcher/playwright_fetcher_test.go`](internal/fetcher/playwrightfetcher/playwright_fetcher_test.go)

Update any existing calls to `collectPageResults` (if called directly in tests) to pass the callback.

## Test Matrix for Pagination

| Method | HTML Setup | Assertion |
|---|---|---|
| `hasPaginationSection` | Container present/absent | true/false |
| `isNextDisabled` | Disabled attr on next btn | true/false |
| `clickNextButton` | Next button clickable | Navigation occurs |
| `shouldStopPagination` | MaxItems config set | Stops at limit |
| `collectPageResults` (full loop) | Multi-page HTML with mock collectFn | Correct total results |

## Files Modified
1. `internal/fetcher/playwrightfetcher/playwright_fetcher.go` — 2 changes (function signature + caller)
2. `internal/fetcher/playwrightfetcher/playwright_fetcher_test.go` — new tests

## Verification Steps
1. Run `golangci-lint run ./...` — all checks pass
2. Run `go test ./...` — all tests pass
3. Verify new pagination tests cover all branch conditions
