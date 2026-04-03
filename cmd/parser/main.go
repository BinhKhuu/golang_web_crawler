package main

import (
	"database/sql"
	"fmt"
	"golangwebcrawler/cmd/parser/internal/parser"
	"golangwebcrawler/internal/dbstore"
	"golangwebcrawler/internal/models"
	"log"

	"github.com/joho/godotenv"
)

// This is a simple test to ensure the parser can be created and used without panicking. It does not test the actual parsing logic, which should be covered by unit tests in the parser package.
func main() {
	err := Load("../../.env")
	if err != nil {
		log.Printf("Error loading .env file: %v\n", err)
		return
	}

	db, err := InitDb()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	j, err := ParseJobListing("<html><body><h1>Sample Job Listing</h1></body></html>", db)
	if err != nil {
		log.Fatalf("Failed to parse job listing: %v", err)
	}
	log.Printf("Parsed Job Listing: %+v", j)
}

func ParseJobListing(html string, db *sql.DB) (models.JobListing, error) {
	p, err := parser.NewParser[models.JobListing](db)
	if err != nil {
		log.Printf("Failed to create parser: %v", err)
		// testing data not finished
		return models.JobListing{
			Title: "Error",
		}, err
	}
	j, err := p.ParseLLM(html)
	if err != nil {
		log.Printf("Failed to parse HTML: %v", err)
		return models.JobListing{}, err
	}
	log.Printf("Parsed Job Data: %+v", j)
	return models.JobListing{}, nil
}

func Load(envFile string) error {
	if err := godotenv.Load(envFile); err != nil {
		return fmt.Errorf("failed to load env file: %w", err)
	}
	return nil
}

func InitDb() (*sql.DB, error) {
	database, err := dbstore.SetupDatabase()
	if err != nil {
		log.Printf("error setting up database: %v\n", err)
		return nil, err
	}

	return database, nil
}
