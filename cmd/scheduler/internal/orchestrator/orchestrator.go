package orchestrator

import "log/slog"

type CrawlTask interface {
	Run() error
}

type ParseTask interface {
	Run() error
}

type SeekStrategy struct {
	logger *slog.Logger
}

type Orchestrator struct {
	crawlTasks []CrawlTask
	parseTasks []ParseTask
}

func NewOrchestrator(crawlTasks []CrawlTask, parseTasks []ParseTask) *Orchestrator {
	return &Orchestrator{
		crawlTasks: crawlTasks,
		parseTasks: parseTasks,
	}
}
