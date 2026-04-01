package parser

import (
	"golangwebcrawler/cmd/parser/internal/storage"
	"golangwebcrawler/internal/models"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type LLMService interface {
	QueryLLM(html string, prompt string) (*models.ExtractedJobData, error)
}

type JobListingParser struct {
	JobListing     models.JobListing
	RawContent     models.RawData
	StorageService *storage.ParserStorageService
	LLMService     LLMService
}

func (j *JobListingParser) Parse(html string) (models.JobListing, error) {
	return j.JobListing, nil
}

func NewJobListingParser(storage *storage.ParserStorageService, llmSerivce LLMService) Parser[models.JobListing] {
	return &JobListingParser{
		StorageService: storage,
		LLMService:     llmSerivce,
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

		fullLink := link
		if strings.HasPrefix(link, "/") {
			fullLink = "https://www.seek.com.au" + link
		}
		listings = append(listings, models.ExtractedJobData{
			Title:       title,
			Company:     company,
			Location:    location,
			Salary:      salary,
			Description: description,
			Links:       []string{fullLink},
			Skills:      []string{}, // goquery can't extract these, leave empty or parse from description
		})
	})
	return listings, nil
}

// todo test this.
func cleanHTMLForLLM(rawHTML string) (string, error) {
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return "", err
	}

	// Remove noise tags
	doc.Find("script, style, nav, footer, iframe, noscript").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	// Normalize text content
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
func (j *JobListingParser) ParseJobDataLLM(html string) (models.ExtractedJobData, error) {
	cleanHTMLForLLM, err := cleanHTMLForLLM(html)
	if err != nil {
		return models.ExtractedJobData{}, err
	}

	prompt := "Extract the following fields in JSON format: \n\t- job_title\n\t- company_name\n\t- salary_range\n\t- location\n\t- description\n\t- links (as an array)(this is the job advertisement URL, not the company profile or search filter)\n\t- required_skills (as an array)\n\t\n\tText to process: " + html

	jobData, err := j.LLMService.QueryLLM(cleanHTMLForLLM, prompt)
	if err != nil {
		return models.ExtractedJobData{}, err
	}
	return *jobData, nil
}

// todo figure out if this is needed. It parses data from a html to find links, the goal is to ask the llm get me the links that are related to the domain and looks like a job listing url.
func getPotentialJobLinks(rawHTML string, baseURL string) []string {
	var links []string
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		text := strings.ToLower(s.Text())

		// Heuristic: Job links usually have "view", "details", or are long IDs
		// and they are almost never "privacy", "terms", or "contact"
		isNoise := strings.Contains(text, "privacy") || strings.Contains(text, "terms")

		if !isNoise && (len(href) > 10) {
			// Resolve relative URLs
			if strings.HasPrefix(href, "/") {
				href = baseURL + href
			}
			links = append(links, href)
		}
	})
	return links
}
