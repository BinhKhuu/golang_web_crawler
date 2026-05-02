package orchestrator

import (
	"context"
	"errors"
	"golangwebcrawler/cmd/scheduler/internal/job"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type mockJob struct {
	jobType job.JobType
	execute func(ctx context.Context) error
	called  int32
}

func (m *mockJob) Type() job.JobType {
	return m.jobType
}

func (m *mockJob) Execute(ctx context.Context) error {
	atomic.AddInt32(&m.called, 1)
	if m.execute != nil {
		return m.execute(ctx)
	}
	return nil
}

var _ job.Job = (*mockJob)(nil)

func TestMode_String(t *testing.T) {
	if Sequential.String() != "Sequential" {
		t.Errorf("expected Sequential, got %s", Sequential.String())
	}
	if Concurrent.String() != "Concurrent" {
		t.Errorf("expected Concurrent, got %s", Concurrent.String())
	}
	if Independent.String() != "Independent" {
		t.Errorf("expected Independent, got %s", Independent.String())
	}
}

func TestOrchestrator_Sequential_Success(t *testing.T) {
	var order []string
	mu := sync.Mutex{}

	job1 := &mockJob{
		jobType: job.Crawl,
		execute: func(ctx context.Context) error {
			mu.Lock()
			order = append(order, "crawl")
			mu.Unlock()
			return nil
		},
	}
	job2 := &mockJob{
		jobType: job.Parse,
		execute: func(ctx context.Context) error {
			mu.Lock()
			order = append(order, "parse")
			mu.Unlock()
			return nil
		},
	}

	orch := New([]job.Job{job1, job2}, Sequential, slog.Default())
	if err := orch.Run(t.Context()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(order) != 2 {
		t.Fatalf("expected 2 jobs to run, got %d", len(order))
	}
	if order[0] != "crawl" || order[1] != "parse" {
		t.Errorf("expected [crawl, parse], got %v", order)
	}
}

func TestOrchestrator_Sequential_ParseThenCrawl(t *testing.T) {
	var order []string
	mu := sync.Mutex{}

	job1 := &mockJob{
		jobType: job.Parse,
		execute: func(ctx context.Context) error {
			mu.Lock()
			order = append(order, "parse")
			mu.Unlock()
			return nil
		},
	}
	job2 := &mockJob{
		jobType: job.Crawl,
		execute: func(ctx context.Context) error {
			mu.Lock()
			order = append(order, "crawl")
			mu.Unlock()
			return nil
		},
	}

	orch := New([]job.Job{job1, job2}, Sequential, slog.Default())
	if err := orch.Run(t.Context()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(order) != 2 {
		t.Fatalf("expected 2 jobs to run, got %d", len(order))
	}
	if order[0] != "parse" || order[1] != "crawl" {
		t.Errorf("expected [parse, crawl], got %v", order)
	}
}

func TestOrchestrator_Sequential_JobFailure(t *testing.T) {
	expectedErr := errors.New("job failed")
	job1 := &mockJob{
		jobType: job.Crawl,
		execute: func(ctx context.Context) error {
			return expectedErr
		},
	}
	job2 := &mockJob{
		jobType: job.Parse,
	}

	orch := New([]job.Job{job1, job2}, Sequential, slog.Default())
	err := orch.Run(t.Context())
	if err == nil {
		t.Error("expected error, got nil")
	}

	if atomic.LoadInt32(&job2.called) != 0 {
		t.Error("second job should not have been called after first job failed")
	}
}

func TestOrchestrator_Sequential_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	job1 := &mockJob{jobType: job.Crawl}

	orch := New([]job.Job{job1}, Sequential, slog.Default())
	err := orch.Run(ctx)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestOrchestrator_Concurrent_CrawlBeforeParse(t *testing.T) {
	var order []string
	mu := sync.Mutex{}

	crawlJob := &mockJob{
		jobType: job.Crawl,
		execute: func(ctx context.Context) error {
			mu.Lock()
			order = append(order, "crawl")
			mu.Unlock()
			return nil
		},
	}

	parseJob := &mockJob{
		jobType: job.Parse,
		execute: func(ctx context.Context) error {
			mu.Lock()
			order = append(order, "parse")
			mu.Unlock()
			return nil
		},
	}

	orch := New([]job.Job{crawlJob, parseJob}, Concurrent, slog.Default())
	if err := orch.Run(t.Context()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(order) != 2 {
		t.Fatalf("expected 2 jobs to run, got %d", len(order))
	}
	if order[0] != "crawl" {
		t.Errorf("crawl should complete first, got order %v", order)
	}
}

func TestOrchestrator_Concurrent_MultipleCrawls(t *testing.T) {
	var crawlCount int32
	var parseCount int32

	crawl1 := &mockJob{
		jobType: job.Crawl,
		execute: func(ctx context.Context) error {
			atomic.AddInt32(&crawlCount, 1)
			return nil
		},
	}
	crawl2 := &mockJob{
		jobType: job.Crawl,
		execute: func(ctx context.Context) error {
			atomic.AddInt32(&crawlCount, 1)
			return nil
		},
	}
	parse1 := &mockJob{
		jobType: job.Parse,
		execute: func(ctx context.Context) error {
			atomic.AddInt32(&parseCount, 1)
			return nil
		},
	}

	orch := New([]job.Job{crawl1, crawl2, parse1}, Concurrent, slog.Default())
	if err := orch.Run(t.Context()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if atomic.LoadInt32(&crawlCount) != 2 {
		t.Errorf("expected 2 crawls, got %d", atomic.LoadInt32(&crawlCount))
	}
	if atomic.LoadInt32(&parseCount) != 1 {
		t.Errorf("expected 1 parse, got %d", atomic.LoadInt32(&parseCount))
	}
}

func TestOrchestrator_Concurrent_CrawlFailure(t *testing.T) {
	crawlJob := &mockJob{
		jobType: job.Crawl,
		execute: func(ctx context.Context) error {
			return errors.New("crawl failed")
		},
	}
	parseJob := &mockJob{
		jobType: job.Parse,
	}

	orch := New([]job.Job{crawlJob, parseJob}, Concurrent, slog.Default())
	err := orch.Run(t.Context())
	if err == nil {
		t.Error("expected error, got nil")
	}

	if atomic.LoadInt32(&parseJob.called) != 0 {
		t.Error("parse job should not run when crawl fails")
	}
}

func TestOrchestrator_Independent_AllRun(t *testing.T) {
	var runCount int32

	jobs := make([]job.Job, 3)
	for i := range jobs {
		jobs[i] = &mockJob{
			jobType: job.Crawl,
			execute: func(ctx context.Context) error {
				atomic.AddInt32(&runCount, 1)
				return nil
			},
		}
	}

	orch := New(jobs, Independent, slog.Default())
	if err := orch.Run(t.Context()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if atomic.LoadInt32(&runCount) != 3 {
		t.Errorf("expected 3 jobs to run, got %d", atomic.LoadInt32(&runCount))
	}
}

func TestOrchestrator_Independent_ContinuesOnError(t *testing.T) {
	var runCount int32

	jobs := make([]job.Job, 3)
	for i := range jobs {
		jobs[i] = &mockJob{
			jobType: job.Crawl,
			execute: func(ctx context.Context) error {
				atomic.AddInt32(&runCount, 1)
				return errors.New("job failed")
			},
		}
	}

	orch := New(jobs, Independent, slog.Default())
	err := orch.Run(t.Context())
	if err == nil {
		t.Error("expected error, got nil")
	}

	if atomic.LoadInt32(&runCount) != 3 {
		t.Errorf("expected all 3 jobs to run despite errors, got %d", atomic.LoadInt32(&runCount))
	}
}

func TestOrchestrator_Independent_MixedJobs(t *testing.T) {
	var crawlCount int32
	var parseCount int32

	crawlJob := &mockJob{
		jobType: job.Crawl,
		execute: func(ctx context.Context) error {
			atomic.AddInt32(&crawlCount, 1)
			return nil
		},
	}
	parseJob := &mockJob{
		jobType: job.Parse,
		execute: func(ctx context.Context) error {
			atomic.AddInt32(&parseCount, 1)
			return nil
		},
	}

	orch := New([]job.Job{crawlJob, parseJob}, Independent, slog.Default())
	if err := orch.Run(t.Context()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if atomic.LoadInt32(&crawlCount) != 1 {
		t.Errorf("expected 1 crawl, got %d", atomic.LoadInt32(&crawlCount))
	}
	if atomic.LoadInt32(&parseCount) != 1 {
		t.Errorf("expected 1 parse, got %d", atomic.LoadInt32(&parseCount))
	}
}

func TestOrchestrator_EmptyJobs(t *testing.T) {
	orch := New([]job.Job{}, Sequential, slog.Default())
	if err := orch.Run(t.Context()); err != nil {
		t.Fatalf("expected no error for empty jobs, got %v", err)
	}
}

func TestOrchestrator_Sequential_SingleJob(t *testing.T) {
	called := int32(0)
	j := &mockJob{
		jobType: job.Crawl,
		execute: func(ctx context.Context) error {
			atomic.AddInt32(&called, 1)
			return nil
		},
	}

	orch := New([]job.Job{j}, Sequential, slog.Default())
	if err := orch.Run(t.Context()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("expected job to be called once, got %d", atomic.LoadInt32(&called))
	}
}

func TestOrchestrator_Concurrent_NoParseJobs(t *testing.T) {
	var crawlCount int32

	jobs := make([]job.Job, 2)
	for i := range jobs {
		jobs[i] = &mockJob{
			jobType: job.Crawl,
			execute: func(ctx context.Context) error {
				atomic.AddInt32(&crawlCount, 1)
				return nil
			},
		}
	}

	orch := New(jobs, Concurrent, slog.Default())
	if err := orch.Run(t.Context()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if atomic.LoadInt32(&crawlCount) != 2 {
		t.Errorf("expected 2 crawls, got %d", atomic.LoadInt32(&crawlCount))
	}
}

func TestOrchestrator_Concurrent_NoCrawlJobs(t *testing.T) {
	var parseCount int32

	jobs := make([]job.Job, 2)
	for i := range jobs {
		jobs[i] = &mockJob{
			jobType: job.Parse,
			execute: func(ctx context.Context) error {
				atomic.AddInt32(&parseCount, 1)
				return nil
			},
		}
	}

	orch := New(jobs, Concurrent, slog.Default())
	if err := orch.Run(t.Context()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if atomic.LoadInt32(&parseCount) != 2 {
		t.Errorf("expected 2 parses, got %d", atomic.LoadInt32(&parseCount))
	}
}

func TestOrchestrator_Sequential_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	blockingJob := &mockJob{
		jobType: job.Crawl,
		execute: func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(10 * time.Second):
				return nil
			}
		},
	}

	orch := New([]job.Job{blockingJob}, Sequential, slog.Default())

	cancel()
	err := orch.Run(ctx)
	if err == nil {
		t.Error("expected error on cancelled context")
	}
}
