package main

import (
	"fmt"
	"golangwebcrawler/cmd/binhcrawler/commands"
	"golangwebcrawler/internal/env"
	"log"
	"os"

	"github.com/jessevdk/go-flags"
)

func main() {
	if err := env.LoadEnv(); err != nil {
		log.Printf("Warning: failed to load .env file: %v", err)
	}

	parser, err := BuildParser()
	if err != nil {
		log.Fatalf("Error building parser %s", err)
	}
	_, pErr := parser.Parse()
	if pErr != nil {
		log.Fatalf("Error parsing command line arguments: %v\n", pErr)
		os.Exit(1)
		return
	}
}

func BuildParser() (*flags.Parser, error) {
	baseCommand := &commands.BaseCommand{
		Out: os.Stdout,
	}

	parser := flags.NewParser(nil, flags.Default)

	_, err := parser.AddCommand("crawl", "Crawl a website", "", &commands.CrawlCommand{
		BaseCommand: *baseCommand,
	})
	if err != nil {
		return nil, fmt.Errorf("error adding crawl command to parser: %w", err)
	}

	return parser, nil
}
