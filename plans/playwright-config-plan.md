# Playwright Fetcher Configuration Plan

## Problem Statement

The `CrawlCommand` in [`cmd/binhcrawler/commands/crawlcommand.go`](cmd/binhcrawler/commands/crawlcommand.go:69) needs to build a `PlaywrightFetcherConfig` with nested structs containing slice fields (selectors, tracking params). CLI flags alone are impractical for array-based settings like CSS selectors.

## Configuration Structure

The [`PlaywrightFetcherConfig`](internal/fetcher/playwrightfetcher/playwright_fetcher.go:58) contains:

```go
type PlaywrightFetcherConfig struct {
    URL              string           // scalar - good for CLI flag
    Headless         bool             // scalar - good for CLI flag  
    Timeout          int              // scalar - good for CLI flag
    Search           SearchConfig     // contains []string fields - better in JSON
    Results          ResultsConfig    // contains []string fields - better in JSON
    Canonicalization CanonicalizationConfig // contains []string fields - better in JSON
}
```

The nested configs contain arrays of selectors:
- `Search.InputSelectors`, `Search.SubmitSelectors` — CSS/XPath selectors for form interaction
- `Results.ListingSelectors`, `Results.DataSelectors` — CSS selectors for content extraction
- `Canonicalization.IgnoreQueryParams`, `Canonicalization.RootRelativePrefixes` — URL normalization rules

---

## Selected Approach: Hybrid Configuration

**JSON config file for complex settings + CLI flags for simple scalar overrides**

### Rationale
- Selectors are site-specific and numerous — JSON handles arrays naturally
- CLI flags stay clean for quick overrides (`--url`, `--timeout`)
- Multiple site configs: `seek.json`, `linkedin.json`, etc.
- Backwards compatible with existing [`GetSeekConfiguration()`](internal/fetcher/playwrightfetcher/seekplaywrightconfig.go:23)

### Decision Points (Resolved)

| Decision | Choice |
|----------|--------|
| Config file location | Relative to current working directory (CWD) |
| Default config file | Built-in default path: `configs/seek.json` relative to CWD |
| Config validation | Return error on invalid JSON or missing file; no silent fallback |

---

## Proposed File Structure

```
cmd/binhcrawler/
  configs/
    seek.json          # Default Seek.com.au configuration
```

### Proposed JSON Structure (`cmd/binhcrawler/configs/seek.json`)

```json
{
  "url": "https://www.seek.com.au/software-engineer-jobs",
  "headless": true,
  "timeout": 10000,
  "search": {
    "inputSelectors": ["input[name=keywords]", "input[placeholder*='Search']"],
    "query": "Software Engineer Jobs",
    "submitSelectors": ["button[type='submit']", "button[data-automation='searchButton']"]
  },
  "results": {
    "listingSelectors": ["a[data-automation='jobTitle']", "a.job-link", "a[data-testid='job-result']"],
    "dataSelectors": ["#job-details", ".JobDetail", "[data-automation='jobDetailsPage']"]
  },
  "canonicalization": {
    "ignoreQueryParams": ["sol", "ref", "origin"],
    "rootRelativePrefixes": ["job/"]
  }
}
```

---

## Modified CrawlCommand Struct

```go
type CrawlCommand struct {
    BaseCommand
    GlobalOpts `group:"Global Options"`

    URL         string `default:"" description:"Target URL to crawl (overrides config file)" long:"url" short:"u"`
    MaxDepth    int    `default:"3" description:"Maximum crawl depth" long:"max-depth" short:"D"`
    Concurrency int    `default:"10" description:"Number of concurrent crawls" long:"concurrency" short:"c"`
    Mode        string `default:"sequential" description:"Execution mode (sequential, concurrent, independent)" long:"mode" short:"m"`
    Headless    bool   `default:"true" description:"Run browser in headless mode" long:"headless"`
    Query       string `default:"" description:"Search query (overrides config file)" long:"query" short:"q"`
    Timeout     int    `default:"0" description:"Playwright timeout in ms (overrides config file)" long:"timeout" short:"t"`
    ParseAfter  bool   `description:"Automatically run parse after crawl completes" long:"parse"`
    ConfigFile  string `default:"configs/seek.json" description:"Path to site configuration JSON file" long:"config" short:"f"`
}
```

---

## Merge Priority (Highest to Lowest)

1. **CLI flags** (`--url`, `--timeout` when explicitly set)
2. **JSON config file values** (from `ConfigFile` path)
3. **[`GetSeekConfiguration()`](internal/fetcher/playwrightfetcher/seekplaywrightconfig.go:23) defaults** (built-in fallback)

---

## buildPlaywrightFetcherConfig Logic

```
1. config := GetSeekConfiguration()          // Start with built-in defaults
2. if ConfigFile != "" {                     // Load JSON config if specified
       loadJSONConfig(ConfigFile, &config)   // Override with file values (returns error on failure)
   }
3. Apply CLI overrides:                      // Highest priority
   - if URL != "" { config.URL = URL }
   - if Timeout > 0 { config.Timeout = Timeout }
4. Return final config
```

---

## Implementation Steps

1. **Add `--config` flag to CrawlCommand**
   - File: [`cmd/binhcrawler/commands/crawlcommand.go`](cmd/binhcrawler/commands/crawlcommand.go:19)
   - Add `ConfigFile string` field with `default:"configs/seek.json" long:"config" short:"f"`

2. **Create config loader function**
   - File: `cmd/binhcrawler/commands/crawlcommand.go` (inline) or new file `cmd/binhcrawler/commands/config.go`
   - Function signature: `func loadJSONConfig(path string, target *playwrightfetcher.PlaywrightFetcherConfig) error`
   - Uses `os.ReadFile(path)` + `json.Unmarshal(data, target)`

3. **Implement merge logic in buildPlaywrightFetcherConfig**
   - Replace current stub with the 3-step merge logic described above

4. **Add default config file to project**
   - Create `cmd/binhcrawler/configs/seek.json` with Seek.com.au selectors from [`GetSeekConfiguration()`](internal/fetcher/playwrightfetcher/seekplaywrightconfig.go:23)

5. **Write unit tests**
   - Test JSON loading with valid config file
   - Test JSON loading with non-existent file path (expect error)
   - Test CLI override priority over config file values

---

## Files to Modify/Create

| File | Action | Description |
|------|--------|-------------|
| `cmd/binhcrawler/commands/crawlcommand.go` | Modify | Add `ConfigFile` field, implement `loadJSONConfig` and merge logic in `buildPlaywrightFetcherConfig` |
| `cmd/binhcrawler/configs/seek.json` | Create | Default Seek.com.au configuration file |
| `cmd/binhcrawler/commands/crawlcommand_test.go` | Modify | Add tests for config loading and merge priority |

---

## Example Usage

```bash
# Use default config (configs/seek.json)
./binhcrawler crawl

# Override URL only
./binhcrawler crawl --url "https://www.seek.com.au/developer-jobs"

# Use custom config file
./binhcrawler crawl --config configs/linkedin.json

# Override multiple settings
./binhcrawler crawl --url "https://example.com" --timeout 30000 --headless
```
