# Refactor: Test_ClickNextButton Tests

## Current State

Two separate test functions (lines 775-802):
- `Test_ClickNextButton_Success` - Tests successful click with valid selector, missing page close defer
- `Test_ClickNextButton_NoMatch` - Tests when selector doesn't match, missing fetcher close defer

Each repeats the same setup/defer pattern. The second test has a resource leak (missing `f.Close()`).

## Loop Coverage Gaps

The [`clickNextButton`](internal/fetcher/playwrightfetcher/playwright_fetcher.go:563) method loops over selectors with these paths:

| Loop Path | Description | Current Coverage |
|-----------|-------------|------------------|
| Selector matches + click succeeds | First matching selector clicks successfully | Partial (single selector) |
| Selector no match → continue | Skip to next selector when count == 0 | **Missing** |
| Empty selectors slice | Loop never executes, returns false | **Missing** |

## Proposed Plan

Consolidate into a single table-driven test `Test_ClickNextButton` with comprehensive coverage:

```go
func Test_ClickNextButton(t *testing.T) {
	tc := []struct {
		name          string
		nextSelectors []string
		expectedClick bool
	}{
		{
			name:          "returns true when first selector matches",
			nextSelectors: []string{paginationSelectors},
			expectedClick: true,
		},
		{
			name:          "returns false when no selectors match",
			nextSelectors: []string{"#nonexistent"},
			expectedClick: false,
		},
		{
			name:          "returns true when fallback selector matches",
			nextSelectors: []string{"#nonexistent", paginationSelectors},
			expectedClick: true,
		},
		{
			name:          "returns false when selectors slice is empty",
			nextSelectors: []string{},
			expectedClick: false,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			f, p := setup_paginationTest(t)
			defer f.Close()
			defer func() {
				if closeErr := p.Close(); closeErr != nil {
					t.Logf("error closing page: %v", closeErr)
				}
			}()

			f.fetchConfig.Pagination.NextSelectors = tt.nextSelectors

			click := f.clickNextButton(p)
			if click != tt.expectedClick {
				t.Errorf("clickNextButton() returned %v, expected %v", click, tt.expectedClick)
			}
		})
	}
}
```

## Test Coverage Matrix

| Test Case | Loop Behavior | What It Tests |
|-----------|---------------|---------------|
| "returns true when first selector matches" | First iteration: count > 0, click succeeds | Basic success path |
| "returns false when no selectors match" | First iteration: count == 0, continue; loop ends | Single selector no-match |
| "returns true when fallback selector matches" | First: count == 0 → continue; Second: count > 0, click succeeds | Fallback selector loop |
| "returns false when selectors slice is empty" | Loop never executes | Empty slice edge case |

## Benefits
- Eliminates duplicated setup/defer code (~15 lines reduced)
- Fixes resource leak in `Test_ClickNextButton_NoMatch` (missing `f.Close()`)
- Follows Go table-driven test convention used elsewhere in the file
- Achieves 100% branch coverage of the `clickNextButton` loop
- Each case is self-contained and clearly describes its precondition
