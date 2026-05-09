package parser

import (
	"context"
	"errors"
	"golangwebcrawler/internal/models"
	"os"
	"strings"
	"testing"
)

type mockLLMService struct{}

const (
	testFilePathCard = "./test/testcard.txt"

	// Test data constants for expected results.
	seekBaseURL      = "https://www.seek.com.au"
	sydneyNSW        = "Sydney NSW"
	melbourneVIC     = "Melbourne VIC"
	brisbaneQLD      = "Brisbane QLD"
	softwareEngineer = "Software Engineer"

	// Test data constants for mock LLM responses.
	exampleBaseURL = "https://example.com"
)

func (m *mockLLMService) QueryLLM(ctx context.Context, prompt string) ([]models.ExtractedJobData, error) {
	mockDetails := getMockJobDetails()
	return []models.ExtractedJobData{mockDetails}, nil
}

func getTestData(t *testing.T) (string, []models.ExtractedJobData) {
	content, err := os.ReadFile("./test/testcard.txt") // Todo move testoutoput.txt to a testdata folder
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}
	html := string(content)
	return html, getExpectedResults()
}

func Test_ParseJobData(t *testing.T) {
	testData, expectedResults := getTestData(t)
	jobs, err := ParseJobDataQuery(testData)
	if err != nil {
		t.Fatalf("Error parsing job data: %v", err)
	}

	if len(jobs) == 0 {
		t.Fatal("Expected to find job listings, but found none")
	}

	for i, job := range jobs {
		if job.Title != expectedResults[i].Title {
			t.Errorf("Job #%d: Expected Title '%s', got '%s'", i+1, expectedResults[i].Title, job.Title)
		}
		if job.Company != expectedResults[i].Company {
			t.Errorf("Job #%d: Expected Company '%s', got '%s'", i+1, expectedResults[i].Company, job.Company)
		}
		if job.Location != expectedResults[i].Location {
			t.Errorf("Job #%d: Expected Location '%s', got '%s'", i+1, expectedResults[i].Location, job.Location)
		}
		if normalize(job.Salary) != normalize(expectedResults[i].Salary) {
			t.Errorf("Job #%d: Expected Salary '%s', got '%s'", i+1, expectedResults[i].Salary, job.Salary)
		}

		// dont care about the other properties only the main ones
	}
}

func normalize(s string) string {
	// Replaces the longer en-dash (–) with a standard hyphen (-)
	return strings.ReplaceAll(s, "–", "-")
}

func getExpectedResults() []models.ExtractedJobData {
	return []models.ExtractedJobData{
		{
			Title:    "Software Developer",
			Company:  "Girraphic Park Pty Ltd",
			Location: sydneyNSW,
			Salary:   "$95,000 - $115,000 per year",
			Link:     seekBaseURL,
		},
		{
			Title:    "Junior Full Stack Developer (Graduate / 1–2 Years Experience)",
			Company:  "LeasePLUS Team",
			Location: melbourneVIC,
			Salary:   "",
			Link:     seekBaseURL,
		},
		{
			Title:    "Software Developer",
			Company:  "Girraphic Park Pty Ltd",
			Location: sydneyNSW,
			Salary:   "$95,000 – $115,000 per year",
			Link:     seekBaseURL,
		},
		{
			Title:    "Junior Full Stack Developer (Graduate / 1–2 Years Experience)",
			Company:  "LeasePLUS Team",
			Location: melbourneVIC,
			Salary:   "",
			Link:     seekBaseURL,
		},
		{
			Title:    "Junior-Mid Developers - Open to different tech stacks - C++/TypeScript",
			Company:  "Round Table Recruitment",
			Location: brisbaneQLD,
			Salary:   "$60,000 – $90,000 per year",
			Link:     seekBaseURL,
		},
		{
			Title:    softwareEngineer,
			Company:  "AJQ",
			Location: melbourneVIC,
			Salary:   "",
			Link:     seekBaseURL,
		},
		{
			Title:    softwareEngineer,
			Company:  "Geoscape Australia",
			Location: sydneyNSW,
			Salary:   "$125,000 – $145,000 per year",
			Link:     seekBaseURL,
		},
		{
			Title:    softwareEngineer,
			Company:  "Wymac Gaming Solutions",
			Location: "ClaytonMelbourne VIC",
			Salary:   "",
			Link:     seekBaseURL,
		},
		{
			Title:    softwareEngineer,
			Company:  "Boeing Defence Australia",
			Location: brisbaneQLD,
			Salary:   "Permanent role, annual bonus, employee benefits.",
			Link:     seekBaseURL,
		},
		{
			Title:    softwareEngineer,
			Company:  "Real Time",
			Location: melbourneVIC,
			Salary:   "📍 $180k-$200k base + 25% Bonus",
			Link:     seekBaseURL,
		},
		{
			Title:    "Junior Software Engineer (Full-Stack) — Python/PHP + React",
			Company:  "DMA Global",
			Location: sydneyNSW,
			Salary:   "$50,000 – $70,000 per year",
			Link:     seekBaseURL,
		},
		{
			Title:    softwareEngineer,
			Company:  "Veracross",
			Location: "MiamiGold Coast QLD",
			Salary:   "",
			Link:     seekBaseURL,
		},
		{
			Title:    softwareEngineer,
			Company:  "Netbay Internet Pty Ltd",
			Location: "Box HillMelbourne VIC",
			Salary:   "$115,000 – $125,000 per year",
			Link:     seekBaseURL,
		},
		{
			Title:    softwareEngineer,
			Company:  "Boeing Defence Australia",
			Location: brisbaneQLD,
			Salary:   "Permanent role, generous allowances, annual bonus.",
			Link:     seekBaseURL,
		},
		{
			Title:    softwareEngineer,
			Company:  "Maptek Pty Ltd",
			Location: "GlensideAdelaide SA",
			Salary:   "",
			Link:     seekBaseURL,
		},
		{
			Title:    softwareEngineer,
			Company:  "Chubb Fire and Security Pty Ltd",
			Location: "ParramattaSydney NSW",
			Salary:   "",
			Link:     seekBaseURL,
		},
		{
			Title:    softwareEngineer,
			Company:  "Energetica",
			Location: melbourneVIC,
			Salary:   "$100,000 – $125,000 per year",
			Link:     seekBaseURL,
		},
		{
			Title:    "Software Engineer (Java Backend)",
			Company:  "P&C Partners Pty Ltd",
			Location: brisbaneQLD,
			Salary:   "Up to $120,000 plus super",
			Link:     seekBaseURL,
		},
		{
			Title:    softwareEngineer,
			Company:  "PRA",
			Location: sydneyNSW,
			Salary:   "Up to $650 pd",
			Link:     seekBaseURL,
		},
		{
			Title:    softwareEngineer,
			Company:  "Geoscience Australia",
			Location: "SymonstonCanberra ACT",
			Salary:   "",
			Link:     seekBaseURL,
		},
		{
			Title:    "Jnr Full Stack Software Engineer (Vue.js & C#.Net)",
			Company:  "Task Recruitment",
			Location: "West EndBrisbane QLD",
			Salary:   "$70,000 to $90,000 base salary",
			Link:     seekBaseURL,
		},
		{
			Title:    softwareEngineer,
			Company:  "GENUS INFRASTRUCTURE PTY LTD",
			Location: "Perth WA",
			Salary:   "",
			Link:     seekBaseURL,
		},
	}
}

func getMockJobDetails() models.ExtractedJobData {
	return models.ExtractedJobData{
		Title:    "Software Engineer",
		Company:  "Tech Company",
		Location: "San Francisco, CA",
		Salary:   "$120,000 - $150,000",
	}
}

func getJobListingService() *JobListingParser {
	mockLLMService := &mockLLMService{}
	return &JobListingParser{
		LLMService: mockLLMService, // You can mock this if needed
	}
}

func getTestLLMData(t *testing.T, filepath string) (string, models.ExtractedJobData) {
	content, err := os.ReadFile(filepath) // Todo move testoutoput.txt to a testdata folder
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}
	return string(content), getMockJobDetails()
}

func Test_ParseJobDataLLM(t *testing.T) {
	joblistingParser := getJobListingService()
	testData, expected := getTestLLMData(t, testFilePathCard)
	jobDetails, err := joblistingParser.ParseJobDataLLM(context.Background(), testData)
	if err != nil {
		t.Fatalf("Error parsing job data: %v", err)
	}

	if len(jobDetails) == 0 {
		t.Fatal("expected job details from LLM, got empty slice")
	}

	if jobDetails[0].Title != expected.Title {
		t.Errorf("Expected Title '%s', got '%s'", expected.Title, jobDetails[0].Title)
	}
	if jobDetails[0].Company != expected.Company {
		t.Errorf("Expected Company '%s', got '%s'", expected.Company, jobDetails[0].Company)
	}
	if jobDetails[0].Location != expected.Location {
		t.Errorf("Expected Location '%s', got '%s'", expected.Location, jobDetails[0].Location)
	}
	if normalize(jobDetails[0].Salary) != normalize(expected.Salary) {
		t.Errorf("Expected Salary '%s', got '%s'", expected.Salary, jobDetails[0].Salary)
	}
}

func Test_ParseJobDataLLM_ErrorFromLLM(t *testing.T) {
	mockErr := errors.New("llm query failed")
	mockLLMWithError := &mockLLMServiceError{err: mockErr}
	parser := &JobListingParser{
		LLMService: mockLLMWithError,
	}

	_, err := parser.ParseJobDataLLM(context.Background(), "<article>test</article>")
	if err == nil {
		t.Fatal("expected error from LLM, got nil")
	}
	if !errors.Is(err, mockErr) {
		t.Errorf("expected error %v, got %v", mockErr, err)
	}
}

func Test_ParseJobDataLLM_MultipleJobs(t *testing.T) {
	mockMultipleJobs := &mockLLMServiceMulti{}
	parser := &JobListingParser{
		LLMService: mockMultipleJobs,
	}

	jobs, err := parser.ParseJobDataLLM(context.Background(), "<article>test</article>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(jobs) != 3 {
		t.Errorf("expected 3 jobs, got %d", len(jobs))
	}

	expectedTitles := []string{"Job A", "Job B", "Job C"}
	for i, job := range jobs {
		if job.Title != expectedTitles[i] {
			t.Errorf("job %d: expected title '%s', got '%s'", i, expectedTitles[i], job.Title)
		}
	}
}

type mockLLMServiceError struct {
	err error
}

func (m *mockLLMServiceError) QueryLLM(ctx context.Context, prompt string) ([]models.ExtractedJobData, error) {
	return nil, m.err
}

type mockLLMServiceMulti struct{}

func (m *mockLLMServiceMulti) QueryLLM(ctx context.Context, prompt string) ([]models.ExtractedJobData, error) {
	return []models.ExtractedJobData{
		{Title: "Job A", Company: "Company A", Link: exampleBaseURL + "/1"},
		{Title: "Job B", Company: "Company B", Link: exampleBaseURL + "/2"},
		{Title: "Job C", Company: "Company C", Link: exampleBaseURL + "/3"},
	}, nil
}

func Test_CleanDataForLLM(t *testing.T) {
	testData, _ := getTestLLMData(t, testFilePathCard)
	cleanedData, err := cleanHTMLForLLM(testData)
	if err != nil {
		t.Fatalf("Error cleaning HTML for LLM: %v", err)
	}

	if len(cleanedData) == 0 {
		t.Error("expected non-empty cleaned data")
	}

	if len(cleanedData) >= len(testData) {
		t.Error("cleaned data should be shorter than original data")
	}

	// Tabs should be removed since we trim each line.
	if strings.Contains(cleanedData, "\t") {
		t.Error("cleaned data should not contain tabs")
	}

	// HTML tags should be stripped.
	if strings.Contains(cleanedData, "<script>") || strings.Contains(cleanedData, "<style>") ||
		strings.Contains(cleanedData, "<nav>") || strings.Contains(cleanedData, "<footer>") ||
		strings.Contains(cleanedData, "<iframe>") || strings.Contains(cleanedData, "<noscript>") {
		t.Error("cleaned data should not contain script or style tags")
	}

	// Lines are joined with double newlines, so triple newlines should not appear.
	if strings.Contains(cleanedData, "\n\n\n") {
		t.Error("cleaned data should not contain triple newlines")
	}
}

func Test_NewJobListingParser(t *testing.T) {
	mockLLM := &mockLLMService{}
	parser := NewJobListingParser(mockLLM)

	if parser == nil {
		t.Fatal("expected non-nil parser")
	}

	jobParser, ok := parser.(*JobListingParser)
	if !ok {
		t.Fatalf("expected *JobListingParser, got %T", parser)
	}

	if jobParser.LLMService != mockLLM {
		t.Error("expected LLMService to be set")
	}
}

func Test_NewJobListingParser_Interface(t *testing.T) {
	mockLLM := &mockLLMService{}
	parser := NewJobListingParser(mockLLM)

	if _, ok := any(parser).(Parser[models.JobListing]); !ok {
		t.Fatal("expected parser to implement Parser[models.JobListing] interface")
	}
}

func Test_ParseQuery(t *testing.T) {
	testData, _ := getTestData(t)
	mockLLM := &mockLLMService{}
	parser := NewJobListingParser(mockLLM)

	jobs, err := parser.ParseQuery(context.Background(), testData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(jobs) == 0 {
		t.Fatal("expected job listings, got empty slice")
	}

	if jobs[0].Title == "" {
		t.Error("expected non-empty title")
	}
}

func Test_ParseLLM(t *testing.T) {
	testData, _ := getTestData(t)
	mockLLM := &mockLLMService{}
	parser := NewJobListingParser(mockLLM)

	jobs, err := parser.ParseLLM(context.Background(), testData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(jobs) == 0 {
		t.Fatal("expected job details, got empty slice")
	}

	if jobs[0].Title != "Software Engineer" {
		t.Errorf("expected 'Software Engineer', got '%s'", jobs[0].Title)
	}
}

func Test_ParseJobDataQuery_EmptyHTML(t *testing.T) {
	jobs, err := ParseJobDataQuery("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(jobs) != 0 {
		t.Errorf("expected empty slice for empty HTML, got %d jobs", len(jobs))
	}
}

func Test_ParseJobDataQuery_NoJobCards(t *testing.T) {
	html := `<div><p>No job cards here</p></div>`
	jobs, err := ParseJobDataQuery(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(jobs) != 0 {
		t.Errorf("expected empty slice for HTML without job cards, got %d jobs", len(jobs))
	}
}

func Test_SanitiseExtractedData(t *testing.T) {
	t.Run("unescapes HTML entities", func(t *testing.T) {
		data := []models.ExtractedJobData{
			{Link: "https://example.com/job?foo=bar&baz=qux"},
		}

		result := sanitiseExtractedData(data)

		expected := "https://example.com/job?foo=bar&baz=qux"
		if result[0].Link != expected {
			t.Errorf("expected '%s', got '%s'", expected, result[0].Link)
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		var data []models.ExtractedJobData

		result := sanitiseExtractedData(data)

		// Function returns nil for nil input (in-place modification).
		if result != nil {
			t.Errorf("expected nil for nil input, got %d items", len(result))
		}
	})

	t.Run("empty initialized slice", func(t *testing.T) {
		data := []models.ExtractedJobData{}

		result := sanitiseExtractedData(data)

		if len(result) != 0 {
			t.Errorf("expected empty slice, got %d items", len(result))
		}
	})

	t.Run("multiple jobs", func(t *testing.T) {
		data := []models.ExtractedJobData{
			{Link: "https://example.com/1&page=2"},
			{Link: "https://example.com/2<test>"},
			{Link: "https://example.com/3"},
		}

		result := sanitiseExtractedData(data)

		expecteds := []string{
			"https://example.com/1&page=2",
			"https://example.com/2<test>",
			"https://example.com/3",
		}

		for i, expected := range expecteds {
			if result[i].Link != expected {
				t.Errorf("job %d: expected '%s', got '%s'", i, expected, result[i].Link)
			}
		}
	})

	t.Run("empty link", func(t *testing.T) {
		data := []models.ExtractedJobData{
			{Link: ""},
		}

		result := sanitiseExtractedData(data)

		if result[0].Link != "" {
			t.Errorf("expected empty link, got '%s'", result[0].Link)
		}
	})
}

func Test_CleanHTMLForLLM_RemoveScriptTags(t *testing.T) {
	html := `<html><body><script>var x = 1;</script><p>Hello</p></body></html>`
	result, err := cleanHTMLForLLM(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result, "var x") {
		t.Error("expected script content to be removed")
	}

	if !strings.Contains(result, "Hello") {
		t.Error("expected non-script content to remain")
	}
}

func Test_CleanHTMLForLLM_RemoveStyleTags(t *testing.T) {
	html := `<html><body><style>.foo{color:red;}</style><p>Hello</p></body></html>`
	result, err := cleanHTMLForLLM(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result, "color") {
		t.Error("expected style content to be removed")
	}

	if !strings.Contains(result, "Hello") {
		t.Error("expected non-style content to remain")
	}
}

func Test_CleanHTMLForLLM_NewlineNormalization(t *testing.T) {
	html := `<html><body><p>Line 1</p>\r\n<p>Line 2</p>\n<p>Line 3</p></body></html>`
	result, err := cleanHTMLForLLM(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result, "\r") {
		t.Error("expected no carriage returns in result")
	}

	if strings.Contains(result, "\n\n\n") {
		t.Error("expected no triple newlines in result")
	}

	if !strings.Contains(result, "Line 1") || !strings.Contains(result, "Line 2") || !strings.Contains(result, "Line 3") {
		t.Error("expected all lines to be present in result")
	}
}

func Test_CleanHTMLForLLM_EmptyInput(t *testing.T) {
	result, err := cleanHTMLForLLM("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "" {
		t.Errorf("expected empty string, got '%s'", result)
	}
}

func Test_CleanHTMLForLLM_OnlyWhitespace(t *testing.T) {
	html := "   \n\t  \r\n   "
	result, err := cleanHTMLForLLM(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "" {
		t.Errorf("expected empty string for whitespace-only input, got '%s'", result)
	}
}
