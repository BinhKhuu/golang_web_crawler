package models

import "time"

type JobListing struct {
	ID              int64
	Title           string
	Company         string
	Location        string
	RemoteFlag      bool
	SalaryMin       *float64 // nullable
	SalaryMax       *float64 // nullable
	Currency        string
	DescriptionHTML string
	DescriptionText string
	PostedDate      *time.Time // nullable
	ExpiresAt       *time.Time // nullable
	Source          string
	SourceID        string
	URL             string
	Tags            []string
	RawJSON         []byte
	CrawlTimestamp  time.Time
}
