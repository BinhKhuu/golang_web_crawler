package parser

import (
	"golangwebcrawler/internal/models"
	"os"
	"strings"
	"testing"
)

type mockLLMService struct{}

func (m *mockLLMService) QueryLLM(html string, prompt string) (*models.ExtractedJobData, error) {
	mockDetails := getMockJobDetails()
	return &mockDetails, nil
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
			Location: "Sydney NSW",
			Salary:   "$95,000 - $115,000 per year",
			Links:    []string{"https://www.seek.com.au"},
		},
		{
			Title:    "Junior Full Stack Developer (Graduate / 1–2 Years Experience)",
			Company:  "LeasePLUS Team",
			Location: "Melbourne VIC",
			Salary:   "",
			Links:    []string{"https://www.seek.com.au"},
		},
		{
			Title:    "Software Developer",
			Company:  "Girraphic Park Pty Ltd",
			Location: "Sydney NSW",
			Salary:   "$95,000 – $115,000 per year",
			Links:    []string{"https://www.seek.com.au"},
		},
		{
			Title:    "Junior Full Stack Developer (Graduate / 1–2 Years Experience)",
			Company:  "LeasePLUS Team",
			Location: "Melbourne VIC",
			Salary:   "",
			Links:    []string{"https://www.seek.com.au"},
		},
		{
			Title:    "Junior-Mid Developers - Open to different tech stacks - C++/TypeScript",
			Company:  "Round Table Recruitment",
			Location: "Brisbane QLD",
			Salary:   "$60,000 – $90,000 per year",
			Links:    []string{"https://www.seek.com.au"},
		},
		{
			Title:    "Software Engineer",
			Company:  "AJQ",
			Location: "Melbourne VIC",
			Salary:   "",
			Links:    []string{"https://www.seek.com.au"},
		},
		{
			Title:    "Software Engineer",
			Company:  "Geoscape Australia",
			Location: "Sydney NSW",
			Salary:   "$125,000 – $145,000 per year",
			Links:    []string{"https://www.seek.com.au"},
		},
		{
			Title:    "Software Engineer",
			Company:  "Wymac Gaming Solutions",
			Location: "ClaytonMelbourne VIC",
			Salary:   "",
			Links:    []string{"https://www.seek.com.au"},
		},
		{
			Title:    "Software Engineer",
			Company:  "Boeing Defence Australia",
			Location: "Brisbane QLD",
			Salary:   "Permanent role, annual bonus, employee benefits.",
			Links:    []string{"https://www.seek.com.au"},
		},
		{
			Title:    "Software Engineer",
			Company:  "Real Time",
			Location: "Melbourne VIC",
			Salary:   "📍 $180k-$200k base + 25% Bonus",
			Links:    []string{"https://www.seek.com.au"},
		},
		{
			Title:    "Junior Software Engineer (Full-Stack) — Python/PHP + React",
			Company:  "DMA Global",
			Location: "Sydney NSW",
			Salary:   "$50,000 – $70,000 per year",
			Links:    []string{"https://www.seek.com.au"},
		},
		{
			Title:    "Software Engineer",
			Company:  "Veracross",
			Location: "MiamiGold Coast QLD",
			Salary:   "",
			Links:    []string{"https://www.seek.com.au"},
		},
		{
			Title:    "Software Engineer",
			Company:  "Netbay Internet Pty Ltd",
			Location: "Box HillMelbourne VIC",
			Salary:   "$115,000 – $125,000 per year",
			Links:    []string{"https://www.seek.com.au"},
		},
		{
			Title:    "Software Engineer",
			Company:  "Boeing Defence Australia",
			Location: "Brisbane QLD",
			Salary:   "Permanent role, generous allowances, annual bonus.",
			Links:    []string{"https://www.seek.com.au"},
		},
		{
			Title:    "Software Engineer",
			Company:  "Maptek Pty Ltd",
			Location: "GlensideAdelaide SA",
			Salary:   "",
			Links:    []string{"https://www.seek.com.au"},
		},
		{
			Title:    "Software Engineer",
			Company:  "Chubb Fire and Security Pty Ltd",
			Location: "ParramattaSydney NSW",
			Salary:   "",
			Links:    []string{"https://www.seek.com.au"},
		},
		{
			Title:    "Software Engineer",
			Company:  "Energetica",
			Location: "Melbourne VIC",
			Salary:   "$100,000 – $125,000 per year",
			Links:    []string{"https://www.seek.com.au"},
		},
		{
			Title:    "Software Engineer (Java Backend)",
			Company:  "P&C Partners Pty Ltd",
			Location: "Brisbane QLD",
			Salary:   "Up to $120,000 plus super",
			Links:    []string{"https://www.seek.com.au"},
		},
		{
			Title:    "Software Engineer",
			Company:  "PRA",
			Location: "Sydney NSW",
			Salary:   "Up to $650 pd",
			Links:    []string{"https://www.seek.com.au"},
		},
		{
			Title:    "Software Engineer",
			Company:  "Geoscience Australia",
			Location: "SymonstonCanberra ACT",
			Salary:   "",
			Links:    []string{"https://www.seek.com.au"},
		},
		{
			Title:    "Jnr Full Stack Software Engineer (Vue.js & C#.Net)",
			Company:  "Task Recruitment",
			Location: "West EndBrisbane QLD",
			Salary:   "$70,000 to $90,000 base salary",
			Links:    []string{"https://www.seek.com.au"},
		},
		{
			Title:    "Software Engineer",
			Company:  "GENUS INFRASTRUCTURE PTY LTD",
			Location: "Perth WA",
			Salary:   "",
			Links:    []string{"https://www.seek.com.au"},
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
		StorageService: nil,            // You can mock this if needed
		LLMService:     mockLLMService, // You can mock this if needed
	}
}

func getTestLLMData(t *testing.T) (string, models.ExtractedJobData) {
	content, err := os.ReadFile("./test/testcard.txt") // Todo move testoutoput.txt to a testdata folder
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}
	return string(content), getMockJobDetails()
}

func Test_ParseJobDataLLM(t *testing.T) {
	joblistingParser := getJobListingService()
	testData, expected := getTestLLMData(t)
	jobDetails, err := joblistingParser.ParseJobDataLLM(testData)
	if err != nil {
		t.Fatalf("Error parsing job data: %v", err)
	}

	if jobDetails.Title != expected.Title {
		t.Errorf("Expected Title '%s', got '%s'", expected.Title, jobDetails.Title)
	}
	if jobDetails.Company != expected.Company {
		t.Errorf("Expected Company '%s', got '%s'", expected.Company, jobDetails.Company)
	}
	if jobDetails.Location != expected.Location {
		t.Errorf("Expected Location '%s', got '%s'", expected.Location, jobDetails.Location)
	}
	if normalize(jobDetails.Salary) != normalize(expected.Salary) {
		t.Errorf("Expected Salary '%s', got '%s'", expected.Salary, jobDetails.Salary)
	}
}
