# Feature: [Feature Name] - Agent Blueprint

## 1. Context & Objective
- **Goal**: Orchestrator interacts with the crawler and the parser to schedule the crawling and parsing process. each crawl and parse is a type of job that targets a specific type of data or website.
- **User Story**: As a user, I want to call the orchestrator to schedule a crawl and parse job, so that the data is stored in the database and can be accessed later on.
- **Priority**: High

## 2. Technical Environment
- **Stack**: golang
- **Key Files**: 
  - Source: `cmd/scheduler`
  - Data: `internal/models`, `cmd/crawler/internal/models`, `cmd/parser/internal/models`
- **Constraints**: use golang, `golangci-lint` to check code style, any changes to crawler or parser must pass all tests.

## 3. Implementation Plan
### Phase 1: Orchestrator Crawler
- [ ] schedule a crawl for `seek.com.au` using playwrightfetcher functionality should describe what its doing, pattern should allow extension (other crawl types)
- [ ] Write unit tests for logic.

### Phase 2: Orchestrator Parser
- [ ] schedule a parse on crawled data, pattern should allow extension (other parse types). 
- [ ] allow crawler and parser to run in any order (sequential, concurrent, independently, etc.)
- [ ] Wrie unit test for logic.

## 4. Success Criteria (Verification)
- [ ] Schedule a crawl and parse to site sucessfully
- [ ] schedule a parse and crawl to site sucessfully
- [ ] schedule a parse and crawl to site in parallel

## 5. Agent Instructions
- **Coding Style**: `golangci-lint` to check code style
- **Safety**: `golangci-lint` to check code style, do not change .env settings unless adding new settings for new unit tests
- **Reporting**: Provide a summary of modified files after each task completion.
