# Playwright Config Implementation Plan

## Status: Complete ✅ (including improvements)

### Completed

**Step 1:** Created `cmd/binhcrawler/configs/seek.json` — selectors from `GetSeekConfiguration()`.

**Step 2:** Added `ConfigFile string` field to `CrawlCommand` (`crawlcommand.go:35`). Cleared hardcoded defaults from `URL`, `Query`, `Timeout`.

**Step 3:** Added `loadJSONConfig` function (`crawlcommand.go:109`) — `os.ReadFile` + `json.Unmarshal`. Added imports: `encoding/json`, `fmt`, `log/slog`, `os`.

**Step 4:** Implemented merge logic in `buildPlaywrightFetcherConfig` (`crawlcommand.go:87`) — `DefaultConfig()` → JSON file → CLI overrides.

**Step 5:** Wired return value in `Execute()` (`crawlcommand.go:56`) — capture error, log and return on failure.

**Step 6:** Added tests in `crawlcommand_test.go`:
- `TestLoadJSONConfig_Valid` — temp JSON file, assert fields overridden ✅
- `TestLoadJSONConfig_MissingFile` — nonexistent path, assert error ✅
- `TestBuildConfig_MergePriority` — CLI flags win over JSON which wins over defaults ✅

### Improvements Implemented

**1. Test helper for temp JSON:** Added `createTempJSON(t, content string) string` (`crawlcommand_test.go:15`) — uses `t.TempDir()` for auto-cleanup, reduces duplication across tests.

**2. Config file fallback:** `buildPlaywrightFetcherConfig` (`crawlcommand.go:87`) now accepts `*slog.Logger`:
- Default path (`configs/seek.json`): warns and falls back to built-in defaults if file missing
- Custom paths: returns error (no silent fallback)
- Tests: `TestBuildConfig_DefaultPathFallback`, `TestBuildConfig_CustomPathError`

**3. golangci-lint fixes:** Resolved all issues introduced by changes:
- gofumpt formatting — fixed via `golangci-lint run --fix`
- usetesting (os.CreateTemp) — fixed by using `t.TempDir()` in helper
- perfsprint (fmt.Errorf → errors.New) — fixed by golangci-lint --fix
- Pre-existing issues (noctx, Ping) — not touched

### Verification
- `go test ./cmd/binhcrawler/commands/` — 12 tests pass
- `golangci-lint run ./cmd/binhcrawler/commands/...` — only 1 pre-existing issue (noctx)
