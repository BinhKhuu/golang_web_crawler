package parser

import (
	"golangwebcrawler/cmd/parser/internal/storage"
	"golangwebcrawler/internal/models"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
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

// ParseJobDataQuery uses goquery to parse job data from HTML content. This is a simple implementation and may need to be enhanced to extract more detailed information.
func ParseJobDataQuery(html string) ([]models.JobCard, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return []models.JobCard{}, err
	}

	listings := []models.JobCard{}
	// This is a very basic parsing logic. Depending on the actual HTML structure, you may need to adjust the selectors and extraction logic to get more accurate data.
	doc.Find("article[data-testid='job-card']").Each(func(i int, s *goquery.Selection) {
		titleElement := s.Find("a[data-automation='jobTitle']")
		title := titleElement.Text()
		link, _ := titleElement.Attr("href")
		company := s.Find("a[data-automation='jobCompany']").Text()
		location := s.Find("[data-automation='jobLocation']").Text()
		salary := s.Find("[data-automation='jobSalary']").Text()
		listings = append(listings, models.JobCard{
			Title:      title,
			Company:    company,
			Location:   location,
			Salary:     salary,
			Link:       "https://www.seek.com.au" + link,
			ScrapeDate: time.Now(),
		})
	})
	return listings, nil
}

// ParseJobDataLLM use LLM to parse job data from HTML content. This is a placeholder function and should be implemented with actual LLM logic.
func ParseJobDataLLM(html string) (models.JobListing, error) {
	// Placeholder for LLM-based parsing logic
	return models.JobListing{}, nil
}
