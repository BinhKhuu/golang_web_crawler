# Test Plan: `shouldStopPagination`

## Function Under Test

```go
func (f *PlaywrightFetcher) shouldStopPagination(results []crawler.FetchResult) bool {
    if f.fetchConfig.MaxItems <= 0 {
        return false
    }
    return len(results) >= f.fetchConfig.MaxItems
}
```

## Test Cases (Table-Driven)

| # | Scenario | MaxItems | Results Count | Expected |
|---|----------|----------|---------------|----------|
| 1 | MaxItems is 0 (unlimited) | 0 | any | `false` |
| 2 | MaxItems is negative (treated as unlimited) | -1 | any | `false` |
| 3 | MaxItems is negative (treated as unlimited) | -100 | any | `false` |
| 4 | Results below limit | 5 | 3 | `false` |
| 5 | Results exactly at limit | 5 | 5 | `true` |
| 6 | Results exceed limit | 5 | 10 | `true` |
| 7 | Empty results with limit set | 5 | 0 | `false` |
| 8 | Single result at limit of 1 | 1 | 1 | `true` |
| 9 | Single result below limit of 5 | 5 | 1 | `false` |

## Proposed Test Code Structure

```go
func Test_ShouldStopPagination(t *testing.T) {
    tc := []struct {
        name          string
        maxItems      int
        resultsCount  int
        expectedStop  bool
    }{
        {
            name:         "returns false when MaxItems is zero",
            maxItems:     0,
            resultsCount: 5,
            expectedStop: false,
        },
        {
            name:         "returns false when MaxItems is negative",
            maxItems:     -1,
            resultsCount: 5,
            expectedStop: false,
        },
        {
            name:         "returns false when results below limit",
            maxItems:     5,
            resultsCount: 3,
            expectedStop: false,
        },
        {
            name:         "returns true when results equal limit",
            maxItems:     5,
            resultsCount: 5,
            expectedStop: true,
        },
        {
            name:         "returns true when results exceed limit",
            maxItems:     5,
            resultsCount: 10,
            expectedStop: true,
        },
        {
            name:         "returns false when results empty and limit set",
            maxItems:     5,
            resultsCount: 0,
            expectedStop: false,
        },
        {
            name:         "returns true when single result meets limit of 1",
            maxItems:     1,
            resultsCount: 1,
            expectedStop: true,
        },
    }

    for _, tt := range tc {
        t.Run(tt.name, func(t *testing.T) {
            f := createMockFetcher()
            f.fetchConfig.MaxItems = tt.maxItems

            results := make([]crawler.FetchResult, tt.resultsCount)
            for i := range results {
                results[i] = crawler.FetchResult{
                    URL:        fmt.Sprintf("https://example.com/job/%d", i+1),
                    StatusCode: 200,
                    Body:       []byte(""),
                }
            }

            stop := f.shouldStopPagination(results)
            if stop != tt.expectedStop {
                t.Fatalf("expected %t, got %t", tt.expectedStop, stop)
            }
        })
    }
}
```

## Notes

- The original test only covered case #4 (results below limit) with `MaxItems=3` and 2 results.
- All new tests use mock data (no Playwright browser required).
- Tests are independent and can run without `RUN_FETCH_TESTS=1`.
