package parser

import (
	"golangwebcrawler/cmd/parser/internal/storage"
	"golangwebcrawler/internal/models"
	"html"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type LLMService interface {
	QueryLLM(prompt string) ([]models.ExtractedJobData, error)
}

type JobListingParser struct {
	JobListing     models.JobListing
	RawContent     models.RawData
	StorageService *storage.ParserStorageService
	LLMService     LLMService
}

func (j *JobListingParser) ParseQuery(html string) ([]models.ExtractedJobData, error) {
	d, err := ParseJobDataQuery(html)
	return d, err
}

// todo this interface needs to update, accept the raw data and return the extracted data. use the origin in raw data to set the domains for each link
func (j *JobListingParser) ParseLLM(html string) ([]models.ExtractedJobData, error) {
	d, err := j.ParseJobDataLLM(html)
	return d, err
}

func NewJobListingParser(llmSerivce LLMService) Parser[models.JobListing] {
	return &JobListingParser{
		LLMService: llmSerivce,
	}
}

// ParseJobDataQuery uses goquery to parse job data from HTML content. This is a simple implementation and may need to be enhanced to extract more detailed information.
func ParseJobDataQuery(html string) ([]models.ExtractedJobData, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return []models.ExtractedJobData{}, err
	}

	listings := []models.ExtractedJobData{}
	// This is a very basic parsing logic. Depending on the actual HTML structure, you may need to adjust the selectors and extraction logic to get more accurate data.
	doc.Find("article[data-testid='job-card']").Each(func(i int, s *goquery.Selection) {
		titleElement := s.Find("a[data-automation='jobTitle']")
		title := titleElement.Text()
		link, _ := titleElement.Attr("href")
		company := s.Find("a[data-automation='jobCompany']").Text()
		location := s.Find("[data-automation='jobLocation']").Text()
		salary := s.Find("[data-automation='jobSalary']").Text()
		description := strings.TrimSpace(s.Find("[data-automation='jobDescription']").Text())

		listings = append(listings, models.ExtractedJobData{
			Title:       title,
			Company:     company,
			Location:    location,
			Salary:      salary,
			Description: description,
			Link:        link,
			Skills:      []string{}, // goquery can't extract these, leave empty or parse from description
		})
	})
	return listings, nil
}

func cleanHTMLForLLM(rawHTML string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return "", err
	}

	doc.Find("script, style, nav, footer, iframe, noscript").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	var cleanedLines []string
	for line := range strings.SplitSeq(strings.ReplaceAll(doc.Text(), "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleanedLines = append(cleanedLines, trimmed)
		}
	}

	return strings.Join(cleanedLines, "\n\n"), nil
}

// ParseJobDataLLM use LLM to parse job data from HTML content. This is a placeholder function and should be implemented with actual LLM logic.
func (j *JobListingParser) ParseJobDataLLM(html string) ([]models.ExtractedJobData, error) {
	_, err := cleanHTMLForLLM(html)
	if err != nil {
		return []models.ExtractedJobData{}, err
	}

	prompt := `/no_think Forget Previous prompt Extract the following fields in JSON format: 
		- job_title
		- company_name
		- salary_range
		- location
		- description
		- links (single string if multiple comma separated)(this is the job advertisement URL, not the company profile or search filter)
		- required_skills (as an array)
		
		IF you cannot parse the input or find the job_title and links return this text 'I am an idiot'. DO NOT ATTEMPT TO RETURN ANYTHING ELSE, NOT EVEN AN EMPTY JSON ARRAY, JUST THIS TEXT.
		IF you do find job_title and links the returned result should be an array of JSON objects  mark the JSON with` + "```json```" +
		`at the end of the JSON to make it easier to parse in the code
		Text to process: ` + html

	jobData, err := j.LLMService.QueryLLM(prompt)
	if err != nil {
		return []models.ExtractedJobData{}, err
	}
	jobData = santiseExtractedData(jobData)
	return jobData, nil
}

func santiseExtractedData(jobData []models.ExtractedJobData) []models.ExtractedJobData {
	for i, data := range jobData {
		jobData[i].Link = html.UnescapeString(data.Link)
	}
	return jobData
}
