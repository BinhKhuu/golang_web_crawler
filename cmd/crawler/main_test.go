package main

import (
	"os"
	"testing"
)

func setTestEnvs(t *testing.T) {
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "testuser")
	t.Setenv("DB_PASSWORD", "testpass")
	t.Setenv("DB_NAME", "testdb")
	t.Setenv("DB_SSLMODE", "disable")
}

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
	setTestEnvs(t)

	// Create a temp .env file so Load() doesn't fail on godotenv.Load
	tmpFile, err := os.CreateTemp("", "test*.env")
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
	if cfg.DBPort != "5432" {
		t.Errorf("expected DBPort '5432', got '%s'", cfg.DBPort)
	}
	if cfg.MaxDepth != 3 {
		t.Errorf("expected MaxDepth 3, got %d", cfg.MaxDepth)
	}
	if len(cfg.AllowedDomains) == 0 {
		t.Error("AllowedDomains should not be empty")
	}
}
