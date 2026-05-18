package commands

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"golangwebcrawler/cmd/binhcrawler/internal/job"
	"golangwebcrawler/cmd/binhcrawler/internal/orchestrator"
	"golangwebcrawler/internal/crawler"
	"golangwebcrawler/internal/fetcher/playwrightfetcher"
	"golangwebcrawler/internal/storage"
	"log/slog"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func createTempJSON(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp(t.TempDir(), "config-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, writeErr := tmpFile.WriteString(content); writeErr != nil {
		t.Fatalf("failed to write temp file: %v", writeErr)
	}
	tmpFile.Close()
	return tmpFile.Name()
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

func Test_InitDB_SuccessfulConnection(t *testing.T) {
	db, err := InitDb()
	if err != nil {
		t.Errorf("expected no error but got %v", err)
	}
	defer db.Close()
}

func Test_InitDB_Error(t *testing.T) {
	setupDbFnMock := func() (*sql.DB, error) { return nil, errors.New("Mock Error") }
	setupDatabaseFn = setupDbFnMock

	_, err := InitDb()
	if err == nil {
		t.Errorf("Expected error but not nil")
	}
}

func Test_Execute_DBDeferredClose(t *testing.T) {
	mockDB, _, mockDbErr := sqlmock.New()
	if mockDbErr != nil {
		t.Errorf("unexpected error setting up mock db %v", mockDbErr)
	}

	mockDbSetupFn := func() (*sql.DB, error) {
		return mockDB, nil
	}
	setupDatabaseFn = mockDbSetupFn

	buff := &bytes.Buffer{}
	baseCmd, baseCmdErr := SetupBaseCommand(buff, LogLevelInfo)
	if baseCmdErr != nil {
		t.Errorf("SetupBaseCommand failed with error %v", baseCmdErr)
	}

	cmd := &CrawlCommand{
		BaseCommand: *baseCmd,
		GlobalOpts: GlobalOpts{
			LogLevel: "info",
		},
	}

	_ = cmd.Execute([]string{})

	ctx := t.Context()
	pingErr := mockDB.PingContext(ctx)
	if pingErr == nil {
		t.Error("expected error when pinging closed DB, but got nil - Close() was not called")
	}
}

func TestLoadJSONConfig_Valid(t *testing.T) {
	jsonContent := `{
		"url": "https://example.com",
		"headless": false,
		"timeout": 5000,
		"search": {
			"inputSelectors": ["#test-input"],
			"query": "Test Query",
			"submitSelectors": ["#test-submit"]
		},
		"results": {
			"listingSelectors": [".test-listing"],
			"dataSelectors": [".test-data"]
		},
		"canonicalization": {
			"ignoreQueryParams": ["test"],
			"rootRelativePrefixes": ["test/"]
		}
	}`
	path := createTempJSON(t, jsonContent)

	var config playwrightfetcher.PlaywrightFetcherConfig
	if err := loadJSONConfig(path, &config); err != nil {
		t.Fatalf("expected no error but got %v", err)
	}

	if config.URL != "https://example.com" {
		t.Errorf("expected URL https://example.com but got %s", config.URL)
	}
	if config.Headless {
		t.Error("expected Headless false but got true")
	}
	if config.Timeout != 5000 {
		t.Errorf("expected Timeout 5000 but got %d", config.Timeout)
	}
	if len(config.Search.InputSelectors) != 1 || config.Search.InputSelectors[0] != "#test-input" {
		t.Errorf("expected InputSelectors [#test-input] but got %v", config.Search.InputSelectors)
	}
	if config.Search.Query != "Test Query" {
		t.Errorf("expected Query 'Test Query' but got %s", config.Search.Query)
	}
}

func TestLoadJSONConfig_MissingFile(t *testing.T) {
	var config playwrightfetcher.PlaywrightFetcherConfig
	err := loadJSONConfig("/nonexistent/path/config.json", &config)
	if err == nil {
		t.Error("expected error for missing file but got nil")
	}
}

func TestBuildConfig_MergePriority(t *testing.T) {
	jsonContent := `{
		"url": "https://json-config.com",
		"timeout": 8000,
		"search": { "query": "JSON Query" }
	}`
	path := createTempJSON(t, jsonContent)

	cmd := &CrawlCommand{
		URL:        "https://cli-override.com",
		Timeout:    30000,
		Query:      "CLI Query",
		ConfigFile: path,
	}

	config, buildErr := buildPlaywrightFetcherConfig(cmd, newTestLogger())
	if buildErr != nil {
		t.Fatalf("expected no error but got %v", buildErr)
	}

	if config.URL != "https://cli-override.com" {
		t.Errorf("CLI URL should override JSON, expected https://cli-override.com but got %s", config.URL)
	}
	if config.Timeout != 30000 {
		t.Errorf("CLI Timeout should override JSON, expected 30000 but got %d", config.Timeout)
	}
	if config.Search.Query != "CLI Query" {
		t.Errorf("CLI Query should override JSON, expected 'CLI Query' but got %s", config.Search.Query)
	}

	defaultConfig := playwrightfetcher.DefaultConfig()
	if len(config.Results.ListingSelectors) != len(defaultConfig.Results.ListingSelectors) {
		t.Error("Results from default config should be preserved when not overridden by JSON or CLI")
	}
}

func TestBuildConfig_DefaultHeadlessIsFalse(t *testing.T) {
	cmd := &CrawlCommand{
		ConfigFile: DefaultConfigPath,
	}

	config, buildErr := buildPlaywrightFetcherConfig(cmd, newTestLogger())
	if buildErr != nil {
		t.Fatalf("expected no error but got %v", buildErr)
	}

	if config.Headless {
		t.Error("expected Headless false (headed mode) by default to avoid bot detection")
	}
}

func TestBuildConfig_HeadlessOverride(t *testing.T) {
	jsonContent := `{
		"url": "https://example.com",
		"headless": false
	}`
	path := createTempJSON(t, jsonContent)

	cmd := &CrawlCommand{
		Headless:   true,
		ConfigFile: path,
	}

	config, buildErr := buildPlaywrightFetcherConfig(cmd, newTestLogger())
	if buildErr != nil {
		t.Fatalf("expected no error but got %v", buildErr)
	}

	if !config.Headless {
		t.Error("expected CLI --headless flag to override config file")
	}
}

func TestBuildConfig_DefaultPathFallback(t *testing.T) {
	cmd := &CrawlCommand{
		ConfigFile: DefaultConfigPath,
	}

	config, buildErr := buildPlaywrightFetcherConfig(cmd, newTestLogger())
	if buildErr != nil {
		t.Fatalf("expected no error for missing default config but got %v", buildErr)
	}

	defaultConfig := playwrightfetcher.DefaultConfig()
	if config.URL != defaultConfig.URL {
		t.Errorf("expected fallback to default URL but got %s", config.URL)
	}
}

func TestBuildConfig_CustomPathError(t *testing.T) {
	cmd := &CrawlCommand{
		ConfigFile: "configs/nonexistent.json",
	}

	_, buildErr := buildPlaywrightFetcherConfig(cmd, newTestLogger())
	if buildErr == nil {
		t.Error("expected error for missing custom config but got nil")
	}
}

func TestExtractAllowedDomains(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected []string
	}{
		{
			name:     "subdomain strips first label",
			url:      "https://www.seek.com.au/jobs",
			expected: []string{"seek.com.au"},
		},
		{
			name:     "bare domain strips first label",
			url:      "https://example.com",
			expected: []string{"com"},
		},
		{
			name:     "no subdomain single label",
			url:      "http://localhost/path",
			expected: []string{"localhost"},
		},
		{
			name:     "invalid url returns empty",
			url:      "://not-a-url",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAllowedDomains(tt.url)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d domains, got %d: %v", len(tt.expected), len(result), result)
			}
			for i, domain := range tt.expected {
				if result[i] != domain {
					t.Errorf("domain[%d]: expected %q, got %q", i, domain, result[i])
				}
			}
		})
	}
}

func TestNewCrawlJob(t *testing.T) {
	mockFetcher := &mockFetcher{}
	mockDB, _, mockErr := sqlmock.New()
	if mockErr != nil {
		t.Fatalf("failed to create sqlmock: %v", mockErr)
	}
	defer mockDB.Close()

	stor := storage.NewService(mockDB, newTestLogger())
	logger := newTestLogger()

	j := newCrawlJob("https://example.com", mockFetcher, stor, 3, []string{"example.com"}, 5, logger)

	if j == nil {
		t.Fatal("expected non-nil CrawlJob")
	}
	if j.ExecuteFn == nil {
		t.Error("expected ExecuteFn to be set")
	}
	if j.Logger != logger {
		t.Error("expected Logger to match provided logger")
	}
}

type mockFetcher struct{}

func (m *mockFetcher) Fetch(_ context.Context, _ string) ([]crawler.FetchResult, error) {
	return nil, nil
}

func TestNewCrawlJob_JobType(t *testing.T) {
	mockFetcher := &mockFetcher{}
	mockDB, _, mockErr := sqlmock.New()
	if mockErr != nil {
		t.Fatalf("failed to create sqlmock: %v", mockErr)
	}

	stor := storage.NewService(mockDB, newTestLogger())

	j := newCrawlJob("https://example.com", mockFetcher, stor, 3, []string{"example.com"}, 5, newTestLogger())

	if j.Type() != job.Crawl {
		t.Errorf("expected JobType Crawl, got %v", j.Type())
	}
}

func TestParseMode_CaseInsensitive_Mode(t *testing.T) {
	tests := []struct {
		name     string
		modeStr  string
		expected orchestrator.Mode
		wantErr  bool
	}{
		{
			name:     "sequential lowercase",
			modeStr:  "sequential",
			expected: orchestrator.Sequential,
		},
		{
			name:     "sequential uppercase",
			modeStr:  "SEQUENTIAL",
			expected: orchestrator.Sequential,
		},
		{
			name:     "concurrent lowercase",
			modeStr:  "concurrent",
			expected: orchestrator.Concurrent,
		},
		{
			name:     "concurrent uppercase",
			modeStr:  "CONCURRENT",
			expected: orchestrator.Concurrent,
		},
		{
			name:     "independent lowercase",
			modeStr:  "independent",
			expected: orchestrator.Independent,
		},
		{
			name:     "independent uppercase",
			modeStr:  "INDEPENDENT",
			expected: orchestrator.Independent,
		},
		{
			name:     "empty string defaults to sequential",
			modeStr:  "",
			expected: orchestrator.Sequential,
		},
		{
			name:    "invalid mode returns error",
			modeStr: "parallel",
			wantErr: true,
		},
		{
			name:    "random string returns error",
			modeStr: "foo",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, err := parseMode(tt.modeStr)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if mode != tt.expected {
				t.Errorf("expected mode %v (%s), got %v (%s)", tt.expected, tt.expected, mode, mode)
			}
		})
	}
}

func TestNewParseJob(t *testing.T) {
	mockDB, _, mockErr := sqlmock.New()
	if mockErr != nil {
		t.Fatalf("failed to create sqlmock: %v", mockErr)
	}
	defer mockDB.Close()

	stor := storage.NewService(mockDB, newTestLogger())
	logger := newTestLogger()

	j := newParseJob(stor, mockDB, logger)

	if j == nil {
		t.Fatal("expected non-nil ParseJob")
	}
	if j.ExecuteFn == nil {
		t.Error("expected ExecuteFn to be set")
	}
	if j.Logger != logger {
		t.Error("expected Logger to match provided logger")
	}
	if j.Type() != job.Parse {
		t.Errorf("expected JobType Parse, got %v", j.Type())
	}
}
