// Package env provides a reliable way to find the project root directory and load .env files.
// It works for unit tests, debugging (VS Code launch configs), and running the program.
package env

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
)

var projectRootOnce struct {
	root   string
	loaded bool
	err    error
}

func ProjectRoot() (string, error) {
	if projectRootOnce.loaded {
		return projectRootOnce.root, projectRootOnce.err
	}

	root, err := findProjectRoot()
	projectRootOnce.root = root
	projectRootOnce.loaded = true
	projectRootOnce.err = err

	return root, err
}

func findProjectRoot() (string, error) {
	// Strategy 1: Try current working directory first (works when running from project root)
	cwd, err := os.Getwd()
	if err == nil {
		if root := verifyProjectRoot(cwd); root != "" {
			return root, nil
		}
	}

	// Strategy 2: Search upward from current working directory
	if cwd != "" {
		if root := searchUpward(cwd); root != "" {
			return root, nil
		}
	}

	// Strategy 3: Search from the calling package's location
	// This handles tests run from within package directories
	if root := searchFromCaller(); root != "" {
		return root, nil
	}

	return "", errors.New("could not find project root: no .env or go.mod file found")
}

// verifyProjectRoot assumes that Project root will contain .env and .gomod file
// return empty string if neither .env or .gomod file are not there
// todo: think of a better way to verify the project root since .env and .gomod in child folders could cause bugs.
func verifyProjectRoot(dir string) string {
	if _, err := os.Stat(filepath.Join(dir, ".env")); err == nil {
		return dir
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return dir
	}
	return ""
}

func searchUpward(startDir string) string {
	dir := startDir
	for {
		if root := verifyProjectRoot(dir); root != "" {
			return root
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// searchFromCaller searches for project root from the location of this package.
func searchFromCaller() string {
	// Get the directory of this file (internal/env/env.go)
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	if filename == "" {
		return ""
	}

	pkgDir := filepath.Dir(filename)
	return searchUpward(pkgDir)
}

// LoadEnv loads the .env file from the project root into environment variables.
// It returns an error only if the project root cannot be found.
// If the .env file does not exist, it returns nil (no error) — this allows
// tests to run without a .env file by falling back to system environment variables.
func LoadEnv() error {
	root, err := ProjectRoot()
	if err != nil {
		return fmt.Errorf("env load failed: %w", err)
	}

	envPath := filepath.Join(root, ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		// .env file doesn't exist — this is not an error, just skip loading.
		return nil
	}

	return godotenv.Load(envPath)
}

// MustLoadEnv loads the .env file and panics if it fails.
// Use this during initialization or when loading is critical.
func MustLoadEnv() {
	if err := LoadEnv(); err != nil {
		panic(err)
	}
}

// GetEnv returns the value of an environment variable, falling back to system env.
func GetEnv(key string) string {
	return os.Getenv(key)
}

// GetEnvOrDefault returns the value of an environment variable or a default value.
func GetEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
