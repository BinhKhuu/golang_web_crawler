# Refactor: WaitForNextPageLoad Tests

## Current State

Four separate test functions (lines 718-775):
- `Test_WaitForNextPageLoad_ReturnWithNoResults` - empty selectors, expect nil
- `Test_WaitForNextPageLoad_CtxError` - expired context, expect error
- `Test_WaitForNextPageLoad_ReturnResults` - DefaultConfig selectors, expect nil
- `Test_WaitForNextPageLoad_NoMatchError` - non-matching selector, expect error

Each repeats the same setup/defer pattern.

## Proposed Plan

Consolidate into a single table-driven test `Test_WaitForNextPageLoad`:

```go
func Test_WaitForNextPageLoad(t *testing.T) {
    tc := []struct {
        name               string
        waitForSelectors   []string
        ctx                func(t *testing.T) context.Context
        expectErr          bool
    }{
        {
            name:             "returns nil when no selectors configured",
            waitForSelectors: []string{},
            ctx:              func(t *testing.T) context.Context { return t.Context() },
            expectErr:        false,
        },
        {
            name:             "returns context error when context is expired",
            waitForSelectors: []string{".next-button"},
            ctx: func(t *testing.T) context.Context {
                ctx, _ := context.WithTimeout(context.Background(), 0)
                return ctx
            },
            expectErr: true,
        },
        {
            name:             "returns nil when selectors match page content",
            waitForSelectors: DefaultConfig().Pagination.WaitForSelectors,
            ctx:              func(t *testing.T) context.Context { return t.Context() },
            expectErr:        false,
        },
        {
            name:             "returns error when no selectors match",
            waitForSelectors: []string{".next-button"},
            ctx:              func(t *testing.T) context.Context { return t.Context() },
            expectErr:        true,
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

            f.fetchConfig.Pagination.WaitForSelectors = tt.waitForSelectors

            err := f.waitForNextPageLoad(tt.ctx(t), p)
            if (err != nil) != tt.expectErr {
                t.Errorf("expected error=%t, got err=%v", tt.expectErr, err)
            }
        })
    }
}
```

## Benefits
- Eliminates ~40 lines of duplicated setup/defer code
- Follows Go table-driven test convention used elsewhere in the file
- Each case is self-contained and clearly describes its precondition
