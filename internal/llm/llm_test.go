package llm

import (
	"golangwebcrawler/internal/models"
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Println("No .env file found, falling back to system env")
	}

	os.Exit(m.Run())
}

func getTestLLMData(t *testing.T) (string, []models.ExtractedJobData) {
	content, err := os.ReadFile("./test/testcard.txt") // Todo move testoutoput.txt to a testdata folder
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}
	return string(content), getExpectedLLMResults()
}

func getExpectedLLMResults() []models.ExtractedJobData {
	return []models.ExtractedJobData{
		{
			Title:    "Software Developer",
			Company:  "Girraphic Park Pty Ltd",
			Location: "Sydney NSW",
			Salary:   "$95,000 - $115,000 per year",
			Link:     "https://www.seek.com.au",
		},
	}
}

func Test_ParseJobDataLLM(t *testing.T) {
	if os.Getenv("RUN_LLM_TESTS") == "" {
		t.Skip("Skipping: set RUN_LLM_TESTS=1 to run")
	}

	llmService, err := NewLLMService()
	if err != nil {
		t.Fatalf("Failed to initialize LLM service: %v", err)
	}

	testData, expected := getTestLLMData(t)
	// todo fix this prompt there is a + testData at the end its a copy of the query in the jobParser which also needs fixing
	prompt := `Extract the following fields in JSON format: 
		- job_title
		- company_name
		- salary_range
		- location
		- description
		- links (single string if multiple comma separated)(this is the job advertisement URL, not the company profile or search filter)
		- required_skills (as an array)
		
		Text to process: ` + testData

	jobDetails, err := llmService.QueryLLM(prompt)
	if err != nil {
		t.Fatalf("LLM query failed: %v", err)
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

	// todo add more testing for the other properties
}
