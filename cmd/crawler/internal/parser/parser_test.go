package parser

import (
	"reflect"
	"testing"
)

func Test_ParseLinks(t *testing.T) {
	body := []byte("<html><body><a href='http://example.com'>Example</a><a href='http://test.com'>Test</a></body></html>")
	parser := NewHttpParser()
	results, err := parser.ParseLinks(body)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := []string{"http://example.com", "http://test.com"}

	if len(results) != len(expected) {
		t.Fatalf("expected %d links, got %d", len(expected), len(results))
	}

	if !reflect.DeepEqual(results, expected) {
		t.Errorf("expected %v, got %v", expected, results)
	}
}
