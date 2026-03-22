package parser

import "golangwebcrawler/cmd/parser/models"

type JobListingParser struct {
	JobListing models.JobListing
}

func (j *JobListingParser) Parse(html string) (models.JobListing, error) {
	return j.JobListing, nil
}

func NewJobListingParser() Parser[models.JobListing] {
	return &JobListingParser{}
}
