# Testing Philosophy

## Overview
This project separates tests into two categories: **unit tests** and **local integration tests**.
The goal is to keep the CI/CD pipeline fast and dependency-free, while still allowing
full integration testing locally when needed.

---

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
Skipping Integration Tests
Integration tests use t.Skip() to self-exclude when the environment variable is not set:

Package Setup (TestMain)
Each package that requires setup (e.g. loading .env) uses TestMain to run initialisation once before all tests in that package:

CI/CD
Only unit tests run in the pipeline via go test ./...
No external services are required in the pipeline
Integration tests are the developer's responsibility to run locally before merging



# Package Setup (TestMain)
Each package that requires setup (e.g. loading .env) uses TestMain to run initialisation once before all tests in that package:

``` go
func TestMain(m *testing.M) {
    if err := godotenv.Load("../../../../.env"); err != nil {
        log.Println("No .env file found, falling back to system env")
    }
    os.Exit(m.Run())
}
```

## TODO how to test locally with docker and ollama