package main

import (
	"context"
	"database/sql"
	"fmt"
	"golangwebcrawler/internal/dbstore"
	"golangwebcrawler/internal/models"
	"golangwebcrawler/internal/parser"
	"golangwebcrawler/internal/storage"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	err := Load("../../.env")
	if err != nil {
		logger.Error("Error loading .env file", "error", err)
		return
	}

	db, err := InitDb()
	if err != nil {
		logger.Error("Failed to initialize database", "error", err)
		return
	}
	defer db.Close()

	htmlContent, err := os.ReadFile("./internal/parser/test/testcard.txt")
	if err != nil {
		logger.Error("Failed to read testcard.txt", "error", err)
		return
	}

	j, err := ParseJobListing(context.Background(), string(htmlContent), db, logger)
	if err != nil {
		logger.Error("Failed to parse job listing", "error", err)
	}
	logger.Info("Parsed Job Listing", "job", j)
}

func ParseJobListing(ctx context.Context, html string, db *sql.DB, logger *slog.Logger) (models.JobListing, error) {
	storageService := storage.NewService(db, logger)
	p, err := parser.NewParser[models.JobListing](db)
	if err != nil {
		logger.Error("Failed to create parser", "error", err)
		return models.JobListing{
			Title: "Error",
		}, err
	}
	j, err := p.ParseLLM(ctx, html)
	if err != nil {
		logger.Error("Failed to parse HTML", "error", err)
		return models.JobListing{}, err
	}

	extracted := make([]storage.ExtractedJobData, len(j))
	for i, item := range j {
		extracted[i] = storage.ExtractedJobData{
			Title:       item.Title,
			Company:     item.Company,
			Location:    item.Location,
			Salary:      item.Salary,
			Description: item.Description,
			Skills:      item.Skills,
			Link:        item.Link,
		}
	}

	err = storageService.StoreExtractedJobDataBatchUpSert(ctx, extracted)
	if err != nil {
		logger.Error("Failed to store extracted job data", "error", err)
		return models.JobListing{}, err
	}
	logger.Info("Parsed Job Data", "data", j)
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
		return nil, fmt.Errorf("error setting up database: %w", err)
	}

	return database, nil
}
