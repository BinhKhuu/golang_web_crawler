# Agent Instructions

## Before Starting Any Task

1. **Read `README.md`** — it contains the project overview, required tools, environment setup, database configuration, and how to run the application.
2. **Read any relevant documentation** in the project root: `MIGRATIONS.md`, `OLLAMA.md`, `DOCKER.md`, `TESTING.md`.
3. **Scan the codebase** to understand current structure before making changes.

## Project Context (from README.md)

- **What**: A Go web crawler with scheduler, parser, and LLM-based job data extraction.
- **Stack**: Go 1.25+, PostgreSQL, Playwright (headless browser), Ollama (LLM).
- **Entry points**: `cmd/scheduler` (main orchestrator), `cmd/parser` (standalone parser CLI), `cmd/api` (stub).
- **Run**: `go run cmd/scheduler/main.go` after setting up DB and migrations.
- **DB**: PostgreSQL via Docker at `localhost:5433`, database name `jobs_webcrawler`.
- **LLM model**: Configured in `internal/llm/llm.go` (default: `mistral:latest`).

## Coding Standards

- Run `golangci-lint run ./...` before finishing — all lint checks must pass.
- Run `go test ./...` — all tests must pass.
- Use `atomic.Int32` instead of `int32` + manual atomic functions.
- Extract repeated string literals into constants (goconst).
- Format imports with standard library, then third-party, then local packages.

## Constraints

- Do not change `.env` settings unless required for new unit tests.
- Changes to crawler/parser logic must pass all existing tests.
- Do not couple scheduler to command-private packages under `cmd/*/internal`.
