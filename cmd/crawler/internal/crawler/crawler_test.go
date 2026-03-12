package crawler

import (
	"fmt"
	"golangwebcrawler/cmd/crawler/internal/models"
	"sync"
	"testing"
)

// Compile-time assertions
var _ Fetcher = (*MockFetchResults)(nil)
var _ Parser = (*MockParserResults)(nil)

type MockFetchResults struct {
	URL        string
	StatusCode int
	Body       []byte
	Err        error
}

func (m *MockFetchResults) Fetch(url string) (models.FetchResult, error) {
	return models.FetchResult{
		URL:        m.URL,
		StatusCode: m.StatusCode,
		Body:       m.Body,
		Err:        m.Err,
	}, m.Err
}

type MockParserResults struct {
	Links []string
}

func (m *MockParserResults) ParseLinks(body []byte) ([]string, error) {
	return m.Links, nil
}

type MockStorageService struct {
	Stored []models.RawData
}

func (m *MockStorageService) Store(result models.RawData) error {
	m.Stored = append(m.Stored, result)
	return nil
}

func TestIsNavigated(t *testing.T) {
	c := NewCrawler()
	c.MarkVisited("https://example.com")

	if !c.IsNavigated("https://example.com") {
		t.Error("expected URL to be marked as visited")
	}
	if c.IsNavigated("https://other.com") {
		t.Error("expected URL to not be visited")
	}
}

func Test_MarkVisited(t *testing.T) {
	url := "http://example.com"
	c := NewCrawler()

	c.MarkVisited(url)

	if c.Visited[url] != true {
		t.Errorf("expected MarkVisited to mark URL as visited")
	}
}

func Test_ConcurrentMarkVisited(t *testing.T) {
	c := NewCrawler()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c.MarkVisited(fmt.Sprintf("https://example.com/%d", i))
		}(i)
	}

	wg.Wait()

	// concurrent map writes would cause a panic, so if we got here, it means the method is thread-safe. Now we check the results.
	// Only one goroutine should see false (not navigated yet)

	if len(c.Visited) != 100 {
		t.Errorf("expected 100 visited URLs, got %d", len(c.Visited))
	}

}

func Test_VisitedIsThreadSafe(t *testing.T) {
	c := NewCrawler()
	var wg sync.WaitGroup
	url := "http://example.com"

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.MarkVisited(url)
		}()
	}
	wg.Wait()

	// concurrent map writes would cause a panic, so if we got here, it means the method is thread-safe. Now we check the results.
	// Only one goroutine should see false (not navigated yet)
}

func createMockFetchResults(url string, statusCode int, body []byte, err error) *MockFetchResults {
	return &MockFetchResults{
		URL:        url,
		StatusCode: statusCode,
		Body:       body,
		Err:        err,
	}
}

func createMockParseResults(links []string) *MockParserResults {
	return &MockParserResults{
		Links: links,
	}
}

func createMockStoreageService() *MockStorageService {
	return &MockStorageService{
		Stored: []models.RawData{},
	}
}

func Test_CrawlAsync(t *testing.T) {
	// todo implement test for CrawlAsync, maybe using a mock fetcher that returns predefined results and checking if the crawler correctly marks URLs as visited and handles errors
	mockFetcher := createMockFetchResults("https://example.com", 200, []byte("mock body"), nil)
	mockParser := createMockParseResults([]string{"https://example.com/about", "https://example.com/contact"})
	mockStorage := createMockStoreageService()
	c := NewCrawler()

	var wg sync.WaitGroup
	wg.Add(1)
	err := c.CrawlAsync(&wg, "https://example.com", mockFetcher, mockParser, mockStorage)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	wg.Wait()

	if len(mockStorage.Stored) == 0 {
		t.Error("expected storage to have at least one result")
	}
	if mockStorage.Stored[0].URL != "https://example.com" {
		t.Errorf("unexpected stored URL: %s", mockStorage.Stored[0].URL)
	}
}
