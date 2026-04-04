package llm

import (
	"context"
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
		Text to process: ` + testData

	jobDetails, err := llmService.QueryLLM(context.Background(), prompt)
	if err != nil {
		t.Fatalf("LLM query failed: %v", err)
	}

	if jobDetails[0].Title != expected[0].Title {
		t.Errorf("Expected Title '%s', got '%s'", expected[0].Title, jobDetails[0].Title)
	}
	if jobDetails[0].Company != expected[0].Company {
		t.Errorf("Expected Company '%s', got '%s'", expected[0].Company, jobDetails[0].Company)
	}
	if jobDetails[0].Location != expected[0].Location {
		t.Errorf("Expected Location '%s', got '%s'", expected[0].Location, jobDetails[0].Location)
	}

	// todo add more testing for the other properties
}

func Test_ParseJobDataLLM_ReturnEmptyJsonWhenNoMatch(t *testing.T) {
	if os.Getenv("RUN_LLM_TESTS") == "" {
		t.Skip("Skipping: set RUN_LLM_TESTS=1 to run")
	}

	llmService, err := NewLLMService()
	if err != nil {
		t.Fatalf("Failed to initialize LLM service: %v", err)
	}

	testData := "<html><body><h1>Sample Job Listing</h1></body></html>"
	// todo fix this prompt there is a + testData at the end its a copy of the query in the jobParser which also needs fixing
	prompt := `/no_think Forget Previous prompt Extract the following fields in JSON format: 
		- job_title
		- company_name
		- salary_range
		- location
		- description
		- links (single string if multiple comma separated)(this is the job advertisement URL, not the company profile or search filter)
		- required_skills (as an array)
		
		IF you cannot find the required job data (fill in 3 fields) return this text 'I am an idiot'. DO NOT ATTEMPT TO RETURN ANYTHING ELSE, NOT EVEN AN EMPTY JSON ARRAY, JUST THIS TEXT.
		IF you do find job_title and links the returned result should be an array of JSON objects  mark the JSON with` + "```json```" +
		`at the end of the JSON to make it easier to parse in the code
		Text to process: ` + testData

	jobDetails, err := llmService.QueryLLM(context.Background(), prompt)
	if err == nil {
		t.Fatalf("Expected LLM query to fail due to no job data, but it succeeded with result: %v", jobDetails)
	}

	if len(jobDetails) != 0 {
		t.Errorf("Expected empty JSON array, got %v", jobDetails)
	}
}

func Test_ParseJobDataLLM_AttemptsToFillInJsonObject(t *testing.T) {
	if os.Getenv("RUN_LLM_TESTS") == "" {
		t.Skip("Skipping: set RUN_LLM_TESTS=1 to run")
	}

	llmService, err := NewLLMService()
	if err != nil {
		t.Fatalf("Failed to initialize LLM service: %v", err)
	}

	testData := "Sample Job Listing"
	// todo fix this prompt there is a + testData at the end its a copy of the query in the jobParser which also needs fixing
	prompt := `/no_think Forget Previous prompt Extract the following fields in JSON format: 
		- job_title
		- company_name
		- salary_range
		- location
		- description
		- links (single string if multiple comma separated)(this is the job advertisement URL, not the company profile or search filter)
		- required_skills (as an array)
		
		IF you cannot find the required job data (fill in 3 fields) return this text 'I am an idiot'. DO NOT ATTEMPT TO RETURN ANYTHING ELSE, NOT EVEN AN EMPTY JSON ARRAY, JUST THIS TEXT.
		IF you do find job_title and links the returned result should be an array of JSON objects  mark the JSON with` + "```json```" +
		`at the end of the JSON to make it easier to parse in the code
		Text to process: ` + testData

	jobDetails, err := llmService.QueryLLM(context.Background(), prompt)
	if err == nil {
		t.Fatalf("Expected LLM query to fail due to no job data, but it succeeded with result: %v", jobDetails)
	}

	if len(jobDetails) != 0 {
		t.Errorf("Expected empty JSON array, got %v", jobDetails)
	}
}
