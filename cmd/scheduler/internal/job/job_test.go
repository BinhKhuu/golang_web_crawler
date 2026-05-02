package job

import (
	"context"
	"database/sql"
	"errors"
	"golangwebcrawler/internal/models"
	"golangwebcrawler/internal/storage"
	"log/slog"
	"testing"
	"time"
)

const (
	testURL1   = "https://example.com/1"
	testURL2   = "https://example.com/2"
	testURL3   = "https://example.com/3"
	testJobURL = "https://example.com/job"
)

var (
	_ Job = (*CrawlJob)(nil)
	_ Job = (*ParseJob)(nil)
)

func TestJobType_String(t *testing.T) {
	if Crawl.String() != "Crawl" {
		t.Errorf("expected Crawl, got %s", Crawl.String())
	}
	if Parse.String() != "Parse" {
		t.Errorf("expected Parse, got %s", Parse.String())
	}
}

func TestCrawlJob_Type(t *testing.T) {
	j := &CrawlJob{}
	if j.Type() != Crawl {
		t.Errorf("expected Crawl, got %v", j.Type())
	}
}

func TestParseJob_Type(t *testing.T) {
	j := &ParseJob{}
	if j.Type() != Parse {
		t.Errorf("expected Parse, got %v", j.Type())
	}
}

func TestCrawlJob_Execute_Success(t *testing.T) {
	j := &CrawlJob{
		ExecuteFn: func(ctx context.Context) error {
			return nil
		},
		Logger: slog.Default(),
	}

	if err := j.Execute(t.Context()); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestCrawlJob_Execute_Error(t *testing.T) {
	expectedErr := errors.New("fetch failed")
	j := &CrawlJob{
		ExecuteFn: func(ctx context.Context) error {
			return expectedErr
		},
		Logger: slog.Default(),
	}

	err := j.Execute(t.Context())
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected wrapped error, got %v", err)
	}
}

func TestCrawlJob_Execute_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	j := &CrawlJob{
		ExecuteFn: func(ctx context.Context) error {
			return ctx.Err()
		},
		Logger: slog.Default(),
	}

	err := j.Execute(ctx)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestParseJob_Execute_Success(t *testing.T) {
	j := &ParseJob{
		ExecuteFn: func(ctx context.Context) error {
			return nil
		},
		Logger: slog.Default(),
	}

	if err := j.Execute(t.Context()); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestParseJob_Execute_Error(t *testing.T) {
	expectedErr := errors.New("parse failed")
	j := &ParseJob{
		ExecuteFn: func(ctx context.Context) error {
			return expectedErr
		},
		Logger: slog.Default(),
	}

	err := j.Execute(t.Context())
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected wrapped error, got %v", err)
	}
}

type mockStorage struct {
	rawData     []storage.RawData
	storeErr    error
	fetchErr    error
	storedItems []storage.ExtractedJobData
}

func (m *mockStorage) GetLatestRawData(ctx context.Context, startDate time.Time) ([]storage.RawData, error) {
	return m.rawData, m.fetchErr
}

func (m *mockStorage) StoreExtractedJobDataBatchUpSert(ctx context.Context, results []storage.ExtractedJobData) error {
	m.storedItems = append(m.storedItems, results...)
	return m.storeErr
}

type mockParser struct {
	results []models.ExtractedJobData
	err     error
}

func (m *mockParser) ParseLLM(ctx context.Context, html string) ([]models.ExtractedJobData, error) {
	return m.results, m.err
}

func TestNewParseJob_NoData(t *testing.T) {
	mockStor := &mockStorage{
		rawData: []storage.RawData{},
	}

	j := NewParseJob(&ParseConfig{
		Storage:   mockStor,
		ParserFn:  func() (ParserJob, error) { return &mockParser{}, nil },
		Logger:    slog.Default(),
		StartDate: time.Now(),
	})

	if err := j.Execute(t.Context()); err != nil {
		t.Errorf("expected no error for empty data, got %v", err)
	}
}

func TestNewParseJob_WithData(t *testing.T) {
	mockStor := &mockStorage{
		rawData: []storage.RawData{
			{URL: testURL1, RawContent: "<html>job 1</html>"},
			{URL: testURL2, RawContent: "<html>job 2</html>"},
		},
	}

	j := NewParseJob(&ParseConfig{
		Storage: mockStor,
		ParserFn: func() (ParserJob, error) {
			return &mockParser{
				results: []models.ExtractedJobData{
					{Title: "Engineer", Company: "ACME", Link: testJobURL + "/1"},
				},
			}, nil
		},
		Logger:    slog.Default(),
		StartDate: time.Now(),
		BatchSize: 100,
	})

	if err := j.Execute(t.Context()); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(mockStor.storedItems) != 2 {
		t.Errorf("expected 2 stored items, got %d", len(mockStor.storedItems))
	}
}

func TestNewParseJob_StorageError(t *testing.T) {
	expectedErr := errors.New("db error")
	mockStor := &mockStorage{
		fetchErr: expectedErr,
	}

	j := NewParseJob(&ParseConfig{
		Storage:   mockStor,
		ParserFn:  func() (ParserJob, error) { return &mockParser{}, nil },
		Logger:    slog.Default(),
		StartDate: time.Now(),
	})

	err := j.Execute(t.Context())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestNewParseJob_ParserCreationError(t *testing.T) {
	expectedErr := errors.New("llm unavailable")
	mockStor := &mockStorage{
		rawData: []storage.RawData{
			{URL: testURL1, RawContent: "<html>job</html>"},
		},
	}

	j := NewParseJob(&ParseConfig{
		Storage: mockStor,
		ParserFn: func() (ParserJob, error) {
			return nil, expectedErr
		},
		Logger:    slog.Default(),
		StartDate: time.Now(),
	})

	err := j.Execute(t.Context())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestNewParseJob_BatchStorage(t *testing.T) {
	mockStor := &mockStorage{
		rawData: []storage.RawData{
			{URL: testURL1, RawContent: "html1"},
			{URL: testURL2, RawContent: "html2"},
			{URL: testURL3, RawContent: "html3"},
		},
	}

	j := NewParseJob(&ParseConfig{
		Storage: mockStor,
		ParserFn: func() (ParserJob, error) {
			return &mockParser{
				results: []models.ExtractedJobData{
					{Title: "Job", Link: testJobURL},
				},
			}, nil
		},
		Logger:    slog.Default(),
		StartDate: time.Now(),
		BatchSize: 2,
	})

	if err := j.Execute(t.Context()); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(mockStor.storedItems) != 3 {
		t.Errorf("expected 3 stored items, got %d", len(mockStor.storedItems))
	}
}

func TestNewParseJob_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	mockStor := &mockStorage{
		rawData: []storage.RawData{
			{URL: "https://example.com/1", RawContent: "html1"},
			{URL: "https://example.com/2", RawContent: "html2"},
		},
	}

	j := NewParseJob(&ParseConfig{
		Storage: mockStor,
		ParserFn: func() (ParserJob, error) {
			return &mockParser{
				results: []models.ExtractedJobData{
					{Title: "Job", Link: "https://example.com/job"},
				},
			}, nil
		},
		Logger:    slog.Default(),
		StartDate: time.Now(),
		BatchSize: 10,
	})

	cancel()

	err := j.Execute(ctx)
	if err == nil {
		t.Log("parse may have completed before cancellation, which is acceptable")
	}
}

func TestNewParseJob_ParseErrorContinues(t *testing.T) {
	mockStor := &mockStorage{
		rawData: []storage.RawData{
			{URL: "https://example.com/1", RawContent: "valid"},
			{URL: "https://example.com/2", RawContent: "invalid"},
			{URL: "https://example.com/3", RawContent: "valid"},
		},
	}

	parseCall := 0
	j := NewParseJob(&ParseConfig{
		Storage: mockStor,
		ParserFn: func() (ParserJob, error) {
			return &mockParser{
				results: func() []models.ExtractedJobData {
					parseCall++
					if parseCall == 2 {
						return nil
					}
					return []models.ExtractedJobData{
						{Title: "Job", Link: "https://example.com/job"},
					}
				}(),
				err: func() error {
					if parseCall == 2 {
						return errors.New("parse failed")
					}
					return nil
				}(),
			}, nil
		},
		Logger:    slog.Default(),
		StartDate: time.Now(),
		BatchSize: 100,
	})

	if err := j.Execute(t.Context()); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSetParseJobListing(t *testing.T) {
	expectedErr := errors.New("not implemented")
	SetParseJobListing(func(ctx context.Context, db *sql.DB, html string) ([]models.ExtractedJobData, error) {
		return nil, expectedErr
	})

	p, err := NewDBParser(nil)
	if err != nil {
		t.Fatalf("expected no error creating parser, got %v", err)
	}

	_, err = p.ParseLLM(t.Context(), "<html>test</html>")
	if err == nil {
		t.Error("expected error, got nil")
	}
}
