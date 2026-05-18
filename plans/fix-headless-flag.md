# Fix Headless Flag Default and Wire It Up

## Problem
1. `go-flags` crashes on `default:"true"` for boolean flags
2. `CrawlCommand.Headless` is never wired to `buildPlaywrightFetcherConfig` (dead code)
3. Default should be headed mode (`headless=false`) to avoid bot detection

## Changes

### 1. `cmd/binhcrawler/commands/crawlcommand.go`
- **Done:** Removed `default:"true"` from `Headless` tag
- **Done:** Added documentation comment explaining why headed mode is the default
- **TODO:** Wire `c.Headless` into config in `buildPlaywrightFetcherConfig`:
  ```go
  config.Headless = config.Headless || c.Headless
  ```
  (CLI `--headless` enables it; config file can also set it)

### 2. `cmd/binhcrawler/commands/crawlcommand_test.go`
- **Done:** Added `TestBuildConfig_DefaultHeadlessIsFalse`
- **Done:** Added `TestBuildConfig_HeadlessOverride`

### 3. `internal/fetcher/playwrightfetcher/playwright_fetcher.go`
- Change `DefaultConfig()` `Headless: true` → `Headless: false`

### 4. `internal/fetcher/playwrightfetcher/seekplaywrightconfig.go`
- Change `GetSeekConfiguration()` `Headless: true` → `Headless: false`

### 5. `cmd/binhcrawler/configs/seek.json`
- Change `"headless": true` → `"headless": false`

### 6. `plans/SCHEDULER_CLI_PLAN.md`
- Update Headless row: Default `true` → `false`
- Update code example: remove `default:"true"` from tag

## Behavior After Fix

| Scenario | Headless |
|----------|----------|
| `go run ... crawl` (no flags) | `false` (headed) |
| `go run ... crawl --headless` | `true` |
| `go run ... crawl --config seek.json` (json says false) | `false` |
| `go run ... crawl --config seek.json --headless` | `true` (CLI wins) |
