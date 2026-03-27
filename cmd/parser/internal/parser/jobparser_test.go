package parser

import (
	"os"
	"testing"
)

func getTestData(t *testing.T) string {
	content, err := os.ReadFile("testoutput.txt") // Todo move testoutoput.txt to a testdata folder
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}
	html := string(content)
	return html
}

func Test_ParseJobData(t *testing.T) {
	testData := getTestData(t)

	_, err := PraseJobData(testData)
	if err != nil {
		t.Fatalf("Error parsing job data: %v", err)
	}

}
