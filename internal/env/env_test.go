package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectRoot_FindsFromPackage(t *testing.T) {
	// Reset the cache to test fresh lookup
	projectRootOnce = struct {
		root   string
		loaded bool
		err    error
	}{}

	root, err := ProjectRoot()
	if err != nil {
		t.Fatalf("expected to find project root, got error: %v", err)
	}
	if root == "" {
		t.Fatal("expected non-empty project root")
	}

	// Verify the root contains expected files
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("expected go.mod to exist in project root: %v", err)
	}
}

func TestLoadEnv_NoErrorWhenMissing(t *testing.T) {
	// Reset the cache
	projectRootOnce = struct {
		root   string
		loaded bool
		err    error
	}{}

	t.Setenv("DB_USER", "test_value")

	// LoadEnv should not error even if .env doesn't exist (it gracefully skips)
	err := LoadEnv()
	if err != nil {
		t.Fatalf("expected no error when .env is missing, got: %v", err)
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	t.Setenv("TEST_VAR_DEFAULT", "custom_value")

	result := GetEnvOrDefault("TEST_VAR_DEFAULT", "default_value")
	if result != "custom_value" {
		t.Errorf("expected 'custom_value', got '%s'", result)
	}

	os.Unsetenv("TEST_VAR_DEFAULT")
	result = GetEnvOrDefault("TEST_VAR_DEFAULT", "default_value")
	if result != "default_value" {
		t.Errorf("expected 'default_value', got '%s'", result)
	}
}

func TestGetEnv(t *testing.T) {
	t.Setenv("TEST_VAR_GET", "get_test_value")

	result := GetEnv("TEST_VAR_GET")
	if result != "get_test_value" {
		t.Errorf("expected 'get_test_value', got '%s'", result)
	}

	result = GetEnv("NONEXISTENT_VAR")
	if result != "" {
		t.Errorf("expected empty string for nonexistent var, got '%s'", result)
	}
}
