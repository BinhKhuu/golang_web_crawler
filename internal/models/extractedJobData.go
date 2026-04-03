package models

// ExtractedJobData return from llm transformed to another model for storage in db.
type ExtractedJobData struct {
	Title       string   `json:"job_title"`
	Company     string   `json:"company_name"`
	Location    string   `json:"location"`
	Salary      string   `json:"salary_range"`
	Description string   `json:"description"`
	Skills      []string `json:"required_skills"`
	Link        string   `json:"links"`
}
