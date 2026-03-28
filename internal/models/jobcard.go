package models

import "time"

type JobCard struct {
	ID             string    `json:"id"`             // From data-job-id="91188569"
	Title          string    `json:"title"`          // From data-automation="jobTitle"
	Company        string    `json:"company"`        // From data-automation="jobCompany"
	Location       string    `json:"location"`       // From data-automation="jobLocation"
	Salary         string    `json:"salary"`         // From data-automation="jobSalary"
	Description    string    `json:"description"`    // From data-automation="jobShortDescription"
	URL            string    `json:"url"`            // The href from the title link
	Classification string    `json:"classification"` // From data-automation="jobClassification"
	Link           string    `json:"link"`           // The href from the title link
	UpdateDate     time.Time `json:"update_date"`    // From data-automation="jobUpdateDate"
	ScrapeDate     time.Time `json:"scrape_date"`    // From data-automation="jobListingDate"
}
