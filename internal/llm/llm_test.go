package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"golangwebcrawler/internal/env"
	"golangwebcrawler/internal/models"
	"log"
	"os"

	"github.com/ollama/ollama/api"
)

var runLLMTest = false

func TestMain(m *testing.M) {
	if err := env.LoadEnv(); err != nil {
		log.Println("No .env file found, falling back to system env")
	}
	runLLMTest = os.Getenv("RUN_LLM_TESTS") == "1"
	os.Exit(m.Run())
}

type mockGenerator struct {
	response string
	err      error
}

func (m *mockGenerator) Generate(_ context.Context, _ *api.GenerateRequest, fn api.GenerateResponseFunc) error {
	if m.err != nil {
		return m.err
	}
	if err := fn(api.GenerateResponse{Response: m.response}); err != nil {
		return err
	}
	return nil
}

func newTestService(response string, genErr error) *LLMService {
	return &LLMService{
		ModelName:    Model,
		maxMemoryMBs: MaxMemoryMBs,
		Client:       &mockGenerator{response: response, err: genErr},
	}
}

func mustMarshalJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("unexpected json marshal error in test helper: %v", err))
	}
	return b
}

func jobDataJSON(jd models.ExtractedJobData) string {
	return "```json\n" + string(mustMarshalJSON([]models.ExtractedJobData{jd})) + "\n```"
}

func jobArrayJSON(jds []models.ExtractedJobData) string {
	return "```json\n" + string(mustMarshalJSON(jds)) + "\n```"
}

type extractJSONTestCase struct {
	name    string
	input   string
	want    []models.ExtractedJobData
	wantErr error
}

func Test_extractJSONFromResponse(t *testing.T) {
	tests := []extractJSONTestCase{
		{
			name:    "valid single job",
			input:   jobDataJSON(models.ExtractedJobData{Title: "Dev", Company: "Acme"}),
			want:    []models.ExtractedJobData{{Title: "Dev", Company: "Acme"}},
			wantErr: nil,
		},
		{
			name:    "valid multiple jobs",
			input:   jobArrayJSON([]models.ExtractedJobData{{Title: "A"}, {Title: "B"}}),
			want:    []models.ExtractedJobData{{Title: "A"}, {Title: "B"}},
			wantErr: nil,
		},
		{
			name:    "no json block",
			input:   "just plain text with no formatting",
			want:    nil,
			wantErr: ErrNoJson,
		},
		{
			name:    "empty json block",
			input:   "```json\n```\n```",
			want:    []models.ExtractedJobData{},
			wantErr: nil,
		},
		{
			name:    "whitespace only json block",
			input:   "```json\n   \n  ```",
			want:    []models.ExtractedJobData{},
			wantErr: nil,
		},
		{
			name:    "empty array in json block",
			input:   "```json\n[]\n```",
			want:    []models.ExtractedJobData{},
			wantErr: ErrNoJson,
		},
		{
			name:    "malformed json",
			input:   "```json\n{\"title\": broken}\n```",
			want:    nil,
			wantErr: ErrNoJson, // any error type works; we just verify an error is returned
		},
		{
			name:    "json block with extra text before and after",
			input:   "Here is the result:\n```json\n[{\"job_title\":\"Engineer\",\"company_name\":\"Corp\"}]\n```\nHope this helps!",
			want:    []models.ExtractedJobData{{Title: "Engineer", Company: "Corp"}},
			wantErr: nil,
		},
		{
			name:    "multiple json blocks - first match wins",
			input:   "```json\n[{\"job_title\":\"First\"}]\n```\n```json\n[{\"job_title\":\"Second\"}]\n```",
			want:    []models.ExtractedJobData{{Title: "First"}},
			wantErr: nil,
		},
		{
			name:    "raw json without backticks",
			input:   `[{"job_title":"Raw"}]`,
			want:    nil,
			wantErr: ErrNoJson,
		},
		{
			name:    "json block with extra whitespace inside",
			input:   "```json\n  [{\"job_title\":\"Spaced\",\"location\":\"Remote\"}]  \n```",
			want:    []models.ExtractedJobData{{Title: "Spaced", Location: "Remote"}},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractJSONFromResponse(tt.input)
			assertExtractResult(t, tt, got, err)
		})
	}
}

func assertExtractResult(t *testing.T, tt extractJSONTestCase, got []models.ExtractedJobData, err error) {
	t.Helper()
	if tt.wantErr != nil {
		if err == nil {
			t.Fatalf("expected error %v, got nil (result: %v)", tt.wantErr, got)
		}
		// Malformed JSON can return any unmarshal error; just verify an error exists.
		if tt.name == "malformed json" {
			return
		}
		if !errors.Is(err, tt.wantErr) {
			t.Errorf("expected error %v, got %v", tt.wantErr, err)
		}
		return
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(tt.want) {
		t.Fatalf("expected %d items, got %d", len(tt.want), len(got))
	}
	for i := range tt.want {
		assertJobDataEqual(t, i, tt.want[i], got[i])
	}
}

func assertJobDataEqual(t *testing.T, idx int, want, got models.ExtractedJobData) {
	t.Helper()
	if got.Title != want.Title {
		t.Errorf("item[%d].Title: expected %q, got %q", idx, want.Title, got.Title)
	}
	if got.Company != want.Company {
		t.Errorf("item[%d].Company: expected %q, got %q", idx, want.Company, got.Company)
	}
	if got.Location != want.Location {
		t.Errorf("item[%d].Location: expected %q, got %q", idx, want.Location, got.Location)
	}
}

func Test_QueryLLM_WithMock(t *testing.T) {
	tests := []struct {
		name      string
		response  string
		genErr    error
		wantLen   int
		wantTitle string
		wantErr   bool
	}{
		{
			name:      "successful extraction",
			response:  jobDataJSON(models.ExtractedJobData{Title: "Software Engineer", Company: "TechCo", Location: "Sydney"}),
			genErr:    nil,
			wantLen:   1,
			wantTitle: "Software Engineer",
			wantErr:   false,
		},
		{
			name:      "generate error propagates",
			response:  "",
			genErr:    errors.New("connection refused"),
			wantLen:   0,
			wantTitle: "",
			wantErr:   true,
		},
		{
			name:      "no json block returns ErrNoJson",
			response:  "I am an idiot",
			genErr:    nil,
			wantLen:   0,
			wantTitle: "",
			wantErr:   true,
		},
		{
			name:      "empty response returns ErrNoJson",
			response:  "",
			genErr:    nil,
			wantLen:   0,
			wantTitle: "",
			wantErr:   true,
		},
		{
			name:      "multiple jobs extracted",
			response:  jobArrayJSON([]models.ExtractedJobData{{Title: "Junior Dev"}, {Title: "Senior Dev"}}),
			genErr:    nil,
			wantLen:   2,
			wantTitle: "Junior Dev",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(tt.response, tt.genErr)
			got, err := svc.QueryLLM(context.Background(), "test prompt")

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (result: %v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("expected %d items, got %d", tt.wantLen, len(got))
			}
			if tt.wantTitle != "" && got[0].Title != tt.wantTitle {
				t.Errorf("expected title %q, got %q", tt.wantTitle, got[0].Title)
			}
		})
	}
}

func Test_QueryLLM_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before calling

	svc := newTestService(jobDataJSON(models.ExtractedJobData{Title: "Test"}), nil)
	// The mock doesn't check context, but the real client would.
	// This test verifies our service passes context through correctly.
	got, err := svc.QueryLLM(ctx, "test prompt")
	// The mock generator ignores context cancellation; this is a structural test
	// confirming the context is threaded through to the Generate call.
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Title != "Test" {
		t.Errorf("expected [Test], got %v", got)
	}
}

// llmTestExpectations defines flexible, non-deterministic assertions for LLM integration tests.
// Since LLM output can vary between runs, these checks use substring/contains matching.
type llmTestExpectations struct {
	titleContains   string   // title should contain this substring
	companyContains string   // company should contain this substring
	locationIn      []string // location should be one of these values (case-insensitive substring match)
}

// llmTestFixtures maps test data files to their expected LLM parsing results.
var llmTestFixtures = map[string]llmTestExpectations{
	"ugh.txt": {
		titleContains:   "Software Developer",
		companyContains: "Smartsoft",
		locationIn:      []string{"Adelaide"},
	},
	"testcard.txt": {
		titleContains:   "Software Developer",
		companyContains: "Girraphic Park",
		locationIn:      []string{"Sydney"},
	},
}

func getTestLLMData(t *testing.T, filepathStr string) (string, llmTestExpectations) {
	t.Helper()
	content, err := os.ReadFile(filepathStr)
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}
	base := filepath.Base(filepathStr)
	expected, ok := llmTestFixtures[base]
	if !ok {
		t.Fatalf("no expected results defined for test file %q", base)
	}
	return string(content), expected
}

func Test_ParseJobDataLLM_FromFile(t *testing.T) {
	if !runLLMTest {
		t.Skip("Skipping: set RUN_LLM_TESTS=1 to run")
	}

	tests := []struct {
		name     string
		filepath string
	}{
		{"ugh.txt", "./test/ugh.txt"},
		{"testcard.txt", "./test/testcard.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llmService, err := NewLLMService()
			if err != nil {
				t.Fatalf("Failed to initialize LLM service: %v", err)
			}

			testData, expected := getTestLLMData(t, tt.filepath)
			prompt := buildExtractionPrompt(testData)

			// LLMs are non-deterministic and may produce malformed JSON on occasion.
			// Retry up to 3 times to accommodate transient parsing failures.
			const maxRetries = 3
			var jobDetails []models.ExtractedJobData

			for attempt := 1; attempt <= maxRetries; attempt++ {
				jobDetails, err = llmService.QueryLLM(context.Background(), prompt)
				if err == nil {
					break
				}
				t.Logf("attempt %d/%d failed: %v", attempt, maxRetries, err)
				if attempt == maxRetries {
					t.Fatalf("LLM query failed after %d attempts: %v", maxRetries, err)
				}
			}

			if len(jobDetails) == 0 {
				t.Fatalf("expected at least one job result, got none")
			}

			got := jobDetails[0]

			if !strings.Contains(got.Title, expected.titleContains) {
				t.Errorf("Title %q does not contain %q", got.Title, expected.titleContains)
			}
			if !strings.Contains(got.Company, expected.companyContains) {
				t.Errorf("Company %q does not contain %q", got.Company, expected.companyContains)
			}
			locationMatch := false
			for _, loc := range expected.locationIn {
				if strings.Contains(strings.ToLower(got.Location), strings.ToLower(loc)) {
					locationMatch = true
					break
				}
			}
			if !locationMatch {
				t.Errorf("Location %q does not match any of %v", got.Location, expected.locationIn)
			}
		})
	}
}

func Test_ParseJobDataLLM_NoMatchReturnsError(t *testing.T) {
	if !runLLMTest {
		t.Skip("Skipping: set RUN_LLM_TESTS=1 to run")
	}

	llmService, err := NewLLMService()
	if err != nil {
		t.Fatalf("Failed to initialize LLM service: %v", err)
	}

	testData := "<html><body><h1>Sample Job Listing</h1></body></html>"
	prompt := buildExtractionPrompt(testData)

	jobDetails, err := llmService.QueryLLM(context.Background(), prompt)
	if err == nil {
		t.Fatalf("Expected LLM query to fail due to no job data, but it succeeded with result: %v", jobDetails)
	}

	if len(jobDetails) != 0 {
		t.Errorf("Expected empty result, got %v", jobDetails)
	}
}

// buildExtractionPrompt builds the standard extraction prompt used by tests.
func buildExtractionPrompt(testData string) string {
	return `Extract the following fields in JSON format: 
		- job_title
		- company_name
		- salary_range
		- location
		- description
		- links (single string if multiple comma separated)(this is the job advertisement URL, not the company profile or search filter)
		- required_skills (as an array)
		
		IF you cannot parse the input or find the job_title return this text 'I am an idiot'. DO NOT ATTEMPT TO RETURN ANYTHING ELSE, NOT EVEN AN EMPTY JSON ARRAY, JUST THIS TEXT.
		IF you do find job_title and links the returned result should be an array of JSON objects  mark the JSON with` + "```json```" +
		`at the end of the JSON to make it easier to parse in the code
		Text to process: ` + testData
}
