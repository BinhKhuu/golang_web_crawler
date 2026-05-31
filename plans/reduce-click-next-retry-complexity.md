# Reduce Test_ClickNextPageWithRetry Cognitive Complexity

## Problem

[`Test_ClickNextPageWithRetry`](internal/fetcher/playwrightfetcher/playwright_fetcher_test.go:621) has cognitive complexity of 33 due to:
- Nested conditionals inside the table-driven test loop
- A large inline assertion block (27 lines) with multiple Playwright API calls
- Combined click loop with error handling

## Solution

Extract two helper functions to reduce the main test function complexity from ~33 to ~8.

### 1. Extract `assertLastPageState` helper

Move the 27-line inline assertion block (lines 686-712) into a dedicated helper:

```go
// assertLastPageState verifies the fetcher navigated to the final paginated page.
func assertLastPageState(t *testing.T, p playwright.Page) {
    t.Helper()
    // ... assertions for URL, page info, job titles, disabled button
}
```

### 2. Extract `executePaginationClicks` helper

Move the click loop with error handling into a dedicated helper:

```go
// executePaginationClicks clicks the next-page button up to maxClicks times,
// returning the first error encountered.
func executePaginationClicks(ctx context.Context, f *PlaywrightFetcher, p playwright.Page, maxClicks int) error {
    for range maxClicks {
        if err := f.clickNextPageWithRetry(ctx, p); err != nil {
            return err
        }
    }
    return nil
}
```

### 3. Refactored test function

Replace inline logic with helper calls:

```go
err := executePaginationClicks(t.Context(), f, p, tt.clicks)

if (err != nil) != tt.expectedErr {
    t.Fatalf("expected error=%t, got err=%v", tt.expectedErr, err)
}

if tt.assertLastPageState {
    assertLastPageState(t, p)
}
```

## Files Changed

- `internal/fetcher/playwrightfetcher/playwright_fetcher_test.go` — refactored test + 2 new helpers

## Verification

- Run `golangci-lint run ./...` — all lint checks must pass
- Run `go test ./...` — all tests must pass
