package parser

import (
	"golangwebcrawler/cmd/parser/internal/storage"
	"golangwebcrawler/internal/models"
)

type JobListingParser struct {
	JobListing     models.JobListing
	RawContent     models.RawData
	storageService storage.ParserStorageService
}

func (j *JobListingParser) Parse(html string) (models.JobListing, error) {
	return j.JobListing, nil
}

func NewJobListingParser(storage storage.ParserStorageService) Parser[models.JobListing] {
	return &JobListingParser{
		storageService: storage,
	}
}
