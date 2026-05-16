package commands

import (
	"bytes"
	"database/sql"
	"errors"
	"golangwebcrawler/internal/fetcher/playwrightfetcher"
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
