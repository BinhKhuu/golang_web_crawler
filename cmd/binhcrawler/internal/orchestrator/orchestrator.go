package orchestrator

import (
	"context"
	"fmt"
	"golangwebcrawler/cmd/binhcrawler/internal/job"
	"log/slog"
	"sync"
)

// Mode defines how jobs are executed.
type Mode int

const (
	// Sequential executes jobs in the order they were added.
	Sequential Mode = iota
	// Concurrent starts all jobs but ensures crawl completes before parse.
	Concurrent
	// Independent runs each job without coordinating with others.
	Independent
)

func (m Mode) String() string {
	return [...]string{"Sequential", "Concurrent", "Independent"}[m]
}

// Orchestrator schedules and executes jobs according to the configured mode.
type Orchestrator struct {
	jobs   []job.Job
	mode   Mode
	logger *slog.Logger
}

// New creates a new Orchestrator with the given jobs and execution mode.
func New(jobs []job.Job, mode Mode, logger *slog.Logger) *Orchestrator {
	return &Orchestrator{
		jobs:   jobs,
		mode:   mode,
		logger: logger,
	}
}

// Run executes all scheduled jobs according to the orchestrator's mode.
func (o *Orchestrator) Run(ctx context.Context) error {
	o.logger.Info("orchestrator starting", "mode", o.mode, "jobs", len(o.jobs))

	switch o.mode {
	case Sequential:
		return o.runSequential(ctx)
	case Concurrent:
		return o.runConcurrent(ctx)
	case Independent:
		return o.runIndependent(ctx)
	default:
		return fmt.Errorf("unsupported execution mode: %s", o.mode)
	}
}

// runSequential executes jobs in order.
func (o *Orchestrator) runSequential(ctx context.Context) error {
	for i, j := range o.jobs {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("context cancelled before job %d: %w", i, err)
		}

		o.logger.Info("executing job", "index", i, "type", j.Type())
		if err := j.Execute(ctx); err != nil {
			return fmt.Errorf("job %d (%s) failed: %w", i, j.Type(), err)
		}
	}
	o.logger.Info("all sequential jobs completed")
	return nil
}

// runConcurrent starts all jobs but ensures crawl jobs complete before parse jobs.
func (o *Orchestrator) runConcurrent(ctx context.Context) error {
	var crawlJobs, parseJobs []job.Job
	for _, j := range o.jobs {
		switch j.Type() {
		case job.Crawl:
			crawlJobs = append(crawlJobs, j)
		case job.Parse:
			parseJobs = append(parseJobs, j)
		}
	}

	// Run all crawl jobs concurrently.
	if err := o.runJobBatch(ctx, crawlJobs, "crawl"); err != nil {
		return err
	}

	// Run all parse jobs concurrently after crawls complete.
	return o.runJobBatch(ctx, parseJobs, "parse")
}

// runJobBatch runs a batch of jobs concurrently and returns the first error.
func (o *Orchestrator) runJobBatch(ctx context.Context, jobs []job.Job, label string) error {
	if len(jobs) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(jobs))

	for _, j := range jobs {
		wg.Add(1)
		go func(j job.Job) {
			defer wg.Done()
			o.logger.Info("executing job", "label", label, "type", j.Type())
			errCh <- j.Execute(ctx)
		}(j)
	}

	wg.Wait()

	for range jobs {
		if err := <-errCh; err != nil {
			return fmt.Errorf("%s job failed: %w", label, err)
		}
	}

	return nil
}

// runIndependent runs each job independently, continuing on error.
func (o *Orchestrator) runIndependent(ctx context.Context) error {
	var mu sync.Mutex
	var firstErr error

	var wg sync.WaitGroup
	for i, j := range o.jobs {
		wg.Add(1)
		go func(idx int, j job.Job) {
			defer wg.Done()
			o.logger.Info("executing independent job", "index", idx, "type", j.Type())
			if err := j.Execute(ctx); err != nil {
				mu.Lock()
				defer mu.Unlock()
				if firstErr == nil {
					firstErr = fmt.Errorf("job %d (%s) failed: %w", idx, j.Type(), err)
				}
			}
		}(i, j)
	}

	wg.Wait()
	o.logger.Info("all independent jobs completed")
	return firstErr
}
