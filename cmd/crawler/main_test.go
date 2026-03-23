package main

import (
	"golangwebcrawler/internal/testhelpers"
	"os"
	"testing"
)

func Test_Load(t *testing.T) {
	cfg, err := Load("./.testenv")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.DBHost == "" {
		t.Error("DBHost should not be empty")
	}
	if cfg.DBPort == "" {
		t.Error("DBPort should not be empty")
	}
	if cfg.MaxDepth <= 0 {
		t.Error("MaxDepth should be greater than 0")
	}
	if len(cfg.AllowedDomains) == 0 {
		t.Error("AllowedDomains should not be empty")
	}
}

func Test_Load_FileNotFound(t *testing.T) {
	_, err := Load("nonexistent.env")
	if err == nil {
		t.Error("expected error when loading nonexistent file")
	}
}

func Test_Load_WithEnvVars(t *testing.T) {
	testhelpers.SetTestEnvs(t)

	// Create a temp .env file so Load() doesn't fail on godotenv.Load
	tmpFile, err := os.CreateTemp(t.TempDir(), "test*.env")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.DBHost != "localhost" {
		t.Errorf("expected DBHost 'localhost', got '%s'", cfg.DBHost)
	}
	if cfg.DBPort != "5433" {
		t.Errorf("expected DBPort '5433', got '%s'", cfg.DBPort)
	}
	if cfg.MaxDepth != 3 {
		t.Errorf("expected MaxDepth 3, got %d", cfg.MaxDepth)
	}
	if len(cfg.AllowedDomains) == 0 {
		t.Error("AllowedDomains should not be empty")
	}
}
