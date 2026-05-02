# Feature: Orchestrator Crawl and Parse Scheduling - Agent Blueprint

## 1. Context & Objective
- Goal: Orchestrator interacts with crawler and parser workflows to schedule crawling and parsing jobs. Each crawl/parse is a job type targeting a specific website or data type.
- User Story: As a user, I want to call the orchestrator to schedule crawl and parse jobs so data is stored in the database and can be accessed later.
- Priority: High

## 2. Technical Environment
- Stack: golang
- Key Files:
  - Scheduler orchestration: [cmd/scheduler](cmd/scheduler)
  - Shared reusable crawl and parse libraries: [internal](internal)
  - Command entrypoints (thin wiring only): [cmd/parser/main.go](cmd/parser/main.go), [cmd/scheduler/main.go](cmd/scheduler/main.go)
  - Shared models and storage: [internal/models](internal/models), [internal/storage/storage.go](internal/storage/storage.go)
- Constraints:
  - Use golang
  - Use golangci-lint to check code style
  - Changes to crawler/parser logic must pass tests
  - Do not couple scheduler to command-private packages under cmd/*/internal

## 3. Implementation Plan
### Phase 0: Architecture Refactor for Reuse
- [ ] Crawler logic is already in shared `internal/crawler` package (cmd/crawler removed).
- [ ] Extract reusable parser logic from cmd/parser/internal into shared root internal packages.
- [ ] Keep command mains as thin composition layers.
- [ ] Ensure scheduler imports only shared packages/interfaces, not cmd/*/internal.

### Phase 1: Orchestrator Task Model
- [ ] Define task contracts for crawl and parse jobs.
- [ ] Implement strategy-based concrete tasks for seek crawl and seek parse.
- [ ] Add dependency metadata to support DAG-based ordering.
- [ ] Write unit tests for task construction and execution contracts.

### Phase 2: Orchestration Modes
- [ ] Implement crawl then parse sequential mode.
- [ ] Implement parse then crawl sequential mode.
- [ ] Implement parallel mode for independent crawl and parse execution.
- [ ] Add DAG validation tests for dependency handling and cycle rejection.

### Phase 3: Integration and Safety
- [ ] Wire scheduler to shared crawl and parse runners.
- [ ] Verify data flow from raw crawled content to parsed job data persistence.
- [ ] Add integration tests for orchestrated runs and failure propagation.
- [ ] Run golangci-lint and related test suites.

## 4. Success Criteria (Verification)
- [ ] Schedule crawl and parse successfully.
- [ ] Schedule parse and crawl successfully.
- [ ] Schedule parse and crawl in parallel successfully.
- [ ] Scheduler uses shared root internal packages and does not import cmd/*/internal.
  - [ ] Existing parser/scheduler command entrypoints remain functional.
- [ ] All related tests and lint checks pass.

## 5. Agent Instructions
- Coding Style: Use golangci-lint to check code style.
- Safety: Do not change .env settings unless adding new settings required by new unit tests.
- Reporting: Provide a summary of modified files after each task completion.