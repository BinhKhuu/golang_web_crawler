package parser

import (
	"context"
	"golangwebcrawler/internal/models"
	"html"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type LLMService interface {
	QueryLLM(ctx context.Context, prompt string) ([]models.ExtractedJobData, error)
}

type JobListingParser struct {
	JobListing models.JobListing
	RawContent models.RawData
	LLMService LLMService
}

func (j *JobListingParser) ParseQuery(ctx context.Context, html string) ([]models.ExtractedJobData, error) {
	d, err := ParseJobDataQuery(html)
	return d, err
}

func (j *JobListingParser) ParseLLM(ctx context.Context, html string) ([]models.ExtractedJobData, error) {
	d, err := j.ParseJobDataLLM(ctx, html)
	return d, err
}

func NewJobListingParser(llmService LLMService) Parser[models.JobListing] {
	return &JobListingParser{
		LLMService: llmService,
	}
}

func ParseJobDataQuery(html string) ([]models.ExtractedJobData, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return []models.ExtractedJobData{}, err
	}

	listings := []models.ExtractedJobData{}
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
			Skills:      []string{},
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

func (j *JobListingParser) ParseJobDataLLM(ctx context.Context, html string) ([]models.ExtractedJobData, error) {
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

	jobData, err := j.LLMService.QueryLLM(ctx, prompt)
	if err != nil {
		return []models.ExtractedJobData{}, err
	}
	jobData = sanitiseExtractedData(jobData)
	return jobData, nil
}

func sanitiseExtractedData(jobData []models.ExtractedJobData) []models.ExtractedJobData {
	for i, data := range jobData {
		jobData[i].Link = html.UnescapeString(data.Link)
	}
	return jobData
}
