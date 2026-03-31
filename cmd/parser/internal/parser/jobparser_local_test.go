package parser

import (
	"golangwebcrawler/internal/models"
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	// Runs once before all tests
	if err := godotenv.Load("../../../../.env"); err != nil {
		log.Println("No .env file found, falling back to system env")
	}

	// Run all tests
	os.Exit(m.Run())
}

func getTestLLMData(t *testing.T) (string, []models.JobCard) {
	content, err := os.ReadFile("./test/testcard.txt") // Todo move testoutoput.txt to a testdata folder
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}
	return string(content), getExpectedLLMResults()
}

func getExpectedLLMResults() []models.JobCard {
	return []models.JobCard{
		{
			Title:    "Software Developer",
			Company:  "Girraphic Park Pty Ltd",
			Location: "Sydney NSW",
			Salary:   "$95,000 - $115,000 per year",
			Link:     "https://www.seek.com.au",
		},
	}
}

// this is actually testing the LLM which requires docker and ollama running, its too expsensive to set this up so we will comment it out for now, but it can be used to test the LLM parsing logic when needed.
func Test_ParseJobDataLLM(t *testing.T) {
	if os.Getenv("RUN_LLM_TESTS") == "" {
		t.Skip("Skipping: set RUN_LLM_TESTS=1 to run")
	}
	testData, expected := getTestLLMData(t)
	jobDetails, err := ParseJobDataLLM(testData)
	if err != nil {
		t.Fatalf("Error parsing job data: %v", err)
	}

	if jobDetails.Title != expected[0].Title {
		t.Errorf("Expected Title '%s', got '%s'", expected[0].Title, jobDetails.Title)
	}
	if jobDetails.Company != expected[0].Company {
		t.Errorf("Expected Company '%s', got '%s'", expected[0].Company, jobDetails.Company)
	}
	if jobDetails.Location != expected[0].Location {
		t.Errorf("Expected Location '%s', got '%s'", expected[0].Location, jobDetails.Location)
	}
	if normalize(jobDetails.Salary) != normalize(expected[0].Salary) {
		t.Errorf("Expected Salary '%s', got '%s'", expected[0].Salary, jobDetails.Salary)
	}
}
