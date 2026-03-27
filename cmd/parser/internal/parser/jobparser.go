package parser

import (
	"fmt"
	"golangwebcrawler/cmd/parser/internal/storage"
	"golangwebcrawler/internal/models"
	"strings"

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

func PraseJobData(html string) (models.JobListing, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return models.JobListing{}, err
	}

	var titles []string
	doc.Find(`[data-automation="jobTitle"]`).Each(func(i int, s *goquery.Selection) {
		titles = append(titles, s.Text())
	})
	fmt.Printf("Job titles: %v\n", titles)
	fmt.Printf("Document title: %s\n", doc.Find("title").Text())
	return models.JobListing{}, err
}
