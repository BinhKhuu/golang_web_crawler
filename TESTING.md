# Testing Philosophy

## Overview
This project separates tests into two categories: **unit tests** and **local integration tests**.
The goal is to keep the CI/CD pipeline fast and dependency-free, while still allowing
full integration testing locally when needed.

---

## Testing LLMS
assume LLM will return something because it likes to talk and make up data or give examples
e.g. 
"<html><body><h1>Sample Job Listing</h1></body></html>" will return no results because it can't find jobdata here but it will tell you about it
"Sample Job Listing<" will return results and make some assumptions like links = SampleJoblisting.com


## Test Categories

### Unit Tests
- Run on every `go test ./...` call
- No external dependencies (no Docker, no Ollama, no database)
- Fast and deterministic
- Use mock data from `./test/` directory

### Local Integration Tests
- Require external services (Docker, Ollama)
- Too expensive to run in CI/CD
- Controlled via the `RUN_LLM_TESTS` environment variable
    - update the .env file with RUN_LLM_TEST = true

---

## Running Tests
``` bash
RUN_LLM_TESTS=1 go test -v ./...
```
### Unit Tests (default)
```bash
go test ./...
```

# Conventions
## Test File Naming

*_test.go	Unit tests — always run
*_local_test.go	Integration tests — skipped unless RUN_LLM_TESTS=1

## Test Data
Test fixtures live in ./test/ directory
Each package manages its own test data

## Skipping Integration Tests
Integration tests use t.Skip() to self-exclude when the environment variable is not set:

### Package Setup (TestMain)
Each package that requires setup (e.g. loading .env) uses TestMain to run initialisation once before all tests in that package:

```go
func TestMain(m *testing.M) {
    if err := godotenv.Load("../../../../.env"); err != nil {
        log.Println("No .env file found, falling back to system env")
    }
    os.Exit(m.Run())
}
```

---

## Project Root Detection

The project root is detected by finding a `.project-root` marker file. This approach is explicit and avoids false positives from nested modules or missing configuration files.

### How It Works

The [`internal/env/env.go`](internal/env/env.go) package searches for the `.project-root` file by:

1. Checking the current working directory
2. Searching upward from the current directory to parent directories
3. Falling back to searching from the calling package's location

### Mocking the Marker File in Tests

Since the marker file is explicit, tests can easily mock the project root by creating a temporary `.project-root` file:

```go
func TestWithMockedRoot(t *testing.T) {
    // Create a temporary directory with the marker file
    tmpDir := t.TempDir()
    os.WriteFile(filepath.Join(tmpDir, ".project-root"), []byte(""), 0644)

    // Reset the cached project root (package-level variable reset)
    defer func() {
        // Reset env package cache by re-importing or using internal reset function
    }()

    // Change to temp dir so detection finds our mock marker
    oldWd, _ := os.Getwd()
    os.Chdir(tmpDir)
    defer os.Chdir(oldWd)

    root, err := env.ProjectRoot()
    assert.NoError(t, err)
    assert.Equal(t, tmpDir, root)
}
```

**Key Benefits:**
- Tests can create isolated environments with mock markers
- No reliance on `.env` or `go.mod` which may not exist in test environments
- Clear and explicit project boundary detection

---

## CI/CD

Only unit tests run in the pipeline via `go test ./...`.
No external services are required in the pipeline.
Integration tests are the developer's responsibility to run locally before merging.

---

## Testing Locally with Docker and Ollama

### Prerequisites

1. Start PostgreSQL via Docker Compose:
   ```bash
   docker-compose up -d postgres
   ```

2. Run database migrations:
   ```bash
   export $(grep -v '^#' .env | xargs)
   migrate -path infra/migrations/scripts \
     -database "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}" \
     up
   ```

3. Start Ollama:
   ```bash
   # Terminal 1: Start the Ollama server
   ollama serve

   # Terminal 2: Pull and verify model
   ollama pull gemma4:latest
   ollama list
   ```

### Running Full Integration Tests

Enable LLM and fetch tests via the environment variable:

```bash
RUN_LLM_TESTS=1 RUN_FETCH_TESTS=1 go test -v ./...
```

Or update the `.env` file:
```bash
RUN_LLM_TESTS=1
RUN_FETCH_TESTS=1
```

Then run:
```bash
go test -v ./...
```
