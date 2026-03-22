package main

import (
	"golangwebcrawler/cmd/parser/internal/parser"
	"golangwebcrawler/cmd/parser/models"
	"log"
)

// This is a simple test to ensure the parser can be created and used without panicking. It does not test the actual parsing logic, which should be covered by unit tests in the parser package.
func main() {
	j, err := ParseJobListing("<html><body><h1>Sample Job Listing</h1></body></html>")
	if err != nil {
		log.Fatalf("Failed to parse job listing: %v", err)
	}
	log.Printf("Parsed Job Listing: %+v", j)
}

func ParseJobListing(html string) (models.JobListing, error) {
	p, err := parser.NewParser[models.JobListing]()
	if err != nil {
		log.Fatalf("Failed to create parser: %v", err)
	}
	j, err := p.Parse(html)
	if err != nil {
		log.Fatalf("Failed to parse HTML: %v", err)
	}
	return j, nil
}
