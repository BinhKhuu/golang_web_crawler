package parser

import (
	"context"
	"fmt"
	"golangwebcrawler/cmd/parser/internal/storage"
	"golangwebcrawler/internal/models"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/ollama/ollama/api"
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

func cleanHTMLForLLM(rawHTML string) (string, error) {
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return "", err
	}

	// 1. Remove the "noise" tags that waste LLM tokens
	doc.Find("script, style, nav, footer, iframe, noscript").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})
	// 1. Remove the absolute junk (scripts/styles) but keep the body structure
	//doc.Find("script, style, nav, footer").Remove()

	var articles []string
	// 2. Find every individual job container
	// Note: You may need to inspect the page to find the exact selector (e.g., "article")
	// doc.Find("article[data-testid='job-card']").Each(func(i int, s *goquery.Selection) {

	// 	// 3. Get the HTML of JUST this one job card
	// 	// This keeps the internal structure (like <h3> for title) for the LLM
	// 	jobHtml, _ := s.Html()

	// 	// 4. Now send ONLY this small, structured snippet to Ollama
	// 	articles = append(articles, jobHtml)
	// })

	// var articles []string
	doc.Find("article[data-testid='job-card']").Each(func(i int, s *goquery.Selection) {
		articles = append(articles, s.Text())
	})

	// // 2. Get the text from the body (or the whole doc if no body)
	// var bodyText string
	// if body := doc.Find("body"); body.Length() > 0 {
	// 	bodyText = body.Text()
	// } else {
	// 	bodyText = doc.Text()
	// }

	// 3. Clean up excessive whitespace/newlines
	var cleanedLines []string
	for _, line := range articles {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleanedLines = append(cleanedLines, trimmed)
		}
	}

	return strings.Join(cleanedLines, "########\n\n"), nil
	//return strings.Join(articles, "\n\n"), nil
}

// ParseJobDataLLM use LLM to parse job data from HTML content. This is a placeholder function and should be implemented with actual LLM logic.
func ParseJobDataLLM(html string) (models.JobListing, error) {
	cleanHTMLForLLM, err := cleanHTMLForLLM(html)
	if err != nil {
		return models.JobListing{}, err
	}

	InitLLMConnection(cleanHTMLForLLM)
	// Placeholder for LLM-based parsing logic
	return models.JobListing{}, nil
}

func InitLLMConnection(html string) {
	// Connect to the Ollama container's API
	client, _ := api.ClientFromEnvironment()

	//links := getPotentialJobLinks(html, "https://www.seek.com.au")
	//linksPrompt := fmt.Sprintf("I will provide a list of URLs from a job search page. Return a JSON array of only the URLs that appear to be individual job advertisements. Ignore links to search filters, company profiles, or legal pages: %s", strings.Join(links, "\n"))

	// linksReq := &api.GenerateRequest{
	// 	Model:  "mistral:latest",
	// 	Prompt: linksPrompt,
	// 	Options: map[string]interface{}{
	// 		"num_ctx": 16384, // This is temporary for THIS specific call only
	// 	},
	// 	Stream: new(bool), // Set to false for a single complete response
	// }

	// err := client.Generate(context.Background(), linksReq, func(resp api.GenerateResponse) error {
	// 	fmt.Print(resp.Response)
	// 	return nil
	// })

	prompt := fmt.Sprintf(`Extract the following fields in JSON format: 
	- job_title
	- company_name
	- salary_range
	- location
	- links (this is the job advertisement URL, not the company profile or search filter)
	- required_skills (as an array)
	
	Text to process: %s`, html)

	// dataFromDB := "Job: Backend Dev. Pay: 100k. Needs Go and Docker."
	//prompt := fmt.Sprintf("Extract job title from the following input its HTML: %s", html)
	//prompt := fmt.Sprint("how many planets are there?")
	req := &api.GenerateRequest{
		Model:  "mistral:latest",
		Prompt: prompt,
		Options: map[string]interface{}{
			"num_ctx": 16384, // This is temporary for THIS specific call only
		},
		Stream: new(bool), // Set to false for a single complete response
	}

	err := client.Generate(context.Background(), req, func(resp api.GenerateResponse) error {
		fmt.Print(resp.Response)
		return nil
	})
	if err != nil {
		fmt.Printf("Error generating response: %v\n", err)
	}
}

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
