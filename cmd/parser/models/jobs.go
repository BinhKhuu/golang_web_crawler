package models

import "time"

// Todo remove this if its not used - replaced by the internal/models/jobs.go JobListing struct which is more comprehensive and used across the entire codebase. This one is just a placeholder for testing the parser and should be removed to avoid confusion.
// JobListing represents a full job record scraped from a job site like Seek.
type JobListing struct {
	// JobID             string     `json:"job_id"`
	// URL               string     `json:"url"`
	// Title             string     `json:"title"`
	// Company           string     `json:"company"`
	// Recruiter         string     `json:"recruiter,omitempty"`
	// Location          string     `json:"location"`
	// Suburb            string     `json:"suburb,omitempty"`
	// State             string     `json:"state,omitempty"`
	// Classification    string     `json:"classification"`
	// SubClassification string     `json:"sub_classification"`
	// WorkType          string     `json:"work_type"`
	// Salary            string     `json:"salary,omitempty"`
	// SalaryNotes       string     `json:"salary_notes,omitempty"`
	// RemoteOption      string     `json:"remote_option,omitempty"`
	// Summary           string     `json:"summary"`
	// Description       string     `json:"description"`
	// Responsibilities  []string   `json:"responsibilities,omitempty"`
	// Requirements      []string   `json:"requirements,omitempty"`
	// Benefits          []string   `json:"benefits,omitempty"`
	PostedAt  *time.Time `json:"posted_at,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	// Tags              []string   `json:"tags,omitempty"`
	// CompanyLogo       string     `json:"company_logo,omitempty"`
	// SearchKeywords    []string   `json:"search_keywords,omitempty"`
}
