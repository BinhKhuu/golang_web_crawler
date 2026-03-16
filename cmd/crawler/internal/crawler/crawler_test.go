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
var _ StorageService = (*MockStorageService)(nil)
var maxDepth = 4

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

func (m *MockStorageService) StoreRawData(result models.RawData) error {
	m.Stored = append(m.Stored, result)
	return nil
}

func TestIsNavigated(t *testing.T) {
	c := createTestCrawler()
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
	c := createTestCrawler()

	c.MarkVisited(url)

	if c.visited[url] != true {
		t.Errorf("expected MarkVisited to mark URL as visited")
	}
}

func Test_ConcurrentMarkVisited(t *testing.T) {
	c := createTestCrawler()
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

	if len(c.visited) != 100 {
		t.Errorf("expected 100 visited URLs, got %d", len(c.visited))
	}

}

func Test_VisitedIsThreadSafe(t *testing.T) {

	c := createTestCrawler()
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

// todo this test does not assert if the goroutines have been run corectly
func Test_CrawlAsync(t *testing.T) {
	// todo implement test for CrawlAsync, maybe using a mock fetcher that returns predefined results and checking if the crawler correctly marks URLs as visited and handles errors
	mockFetcher := createMockFetchResults("https://example.com", 200, []byte("mock body"), nil)
	mockParser := createMockParseResults([]string{"https://example.com/about", "https://example.com/contact"})
	mockStorage := createMockStoreageService()
	c := createTestCrawler()

	err := c.CrawlAsync("https://example.com", maxDepth, mockFetcher, mockParser, mockStorage)
	c.Wait()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	c.Wait()

	if len(mockStorage.Stored) == 0 {
		t.Error("expected storage to have at least one result")
	}
	if mockStorage.Stored[0].URL != "https://example.com" {
		t.Errorf("unexpected stored URL: %s", mockStorage.Stored[0].URL)
	}
}

func createTestCrawler(allowedDomains ...string) *CrawlerState {
	if len(allowedDomains) == 0 {
		allowedDomains = []string{"example.com"}
	}
	return NewCrawler(maxDepth, allowedDomains)
}

func Test_IsAllowedDomain(t *testing.T) {
	tc := []struct {
		testName       string
		url            string
		allowedDomains []string
		expectedResult bool
	}{
		{
			testName:       "URL is in allowed domains",
			url:            "https://example.com/page",
			allowedDomains: []string{"example.com", "other.com"},
			expectedResult: true,
		},
		{
			testName:       "URL is not in allowed domains",
			url:            "https://bad.com/page",
			allowedDomains: []string{"example.com", "other.com", "bads.com"},
			expectedResult: false,
		},
		{
			testName:       "URL is in allowed domains as subdomain",
			url:            "https://sub.example.com/page",
			allowedDomains: []string{"example.com"},
			expectedResult: true,
		},
	}

	for _, tc := range tc {
		t.Run(tc.testName, func(t *testing.T) {
			result := isAllowedDomain(tc.url, tc.allowedDomains)
			if result != tc.expectedResult {
				t.Errorf("unexpected result for URL %s and allowed domains %v: got %v, want %v", tc.url, tc.allowedDomains, result, tc.expectedResult)
			}
		})
	}
}

func Test_containsDomain(t *testing.T) {
	tc := []struct {
		testName       string
		url            string
		domain         string
		expectedResult bool
	}{
		{
			testName:       "URL contains domain",
			url:            "https://example.com/page",
			domain:         "example.com",
			expectedResult: true,
		},
		{
			testName:       "URL does not contain domain",
			url:            "https://other.com/page",
			domain:         "example.com",
			expectedResult: false,
		},
		{
			testName:       "URL contains domain as subdomain",
			url:            "https://sub.example.com/page",
			domain:         "example.com",
			expectedResult: true,
		},
	}

	for _, tc := range tc {
		t.Run(tc.testName, func(t *testing.T) {
			result := containsDomain(tc.url, tc.domain)
			if result != tc.expectedResult {
				t.Errorf("unexpected result for URL %s and domain %s: got %v, want %v", tc.url, tc.domain, result, tc.expectedResult)

			}
		})
	}
}

func Test_MaxDepthIsRespected(t *testing.T) {
	allowedDomains := []string{"example.com"}
	c := createTestCrawler(allowedDomains...)
	mockFetcher := createMockFetchResults("https://example.com", 200, []byte("mock body"), nil)
	mockParser := createMockParseResults([]string{"https://example.com/about", "https://example.com/contact"})
	mockStorage := createMockStoreageService()

	if err := c.CrawlAsync("https://example.com", 1, mockFetcher, mockParser, mockStorage); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(c.visited) != 1 {
		t.Errorf("expected depth traversal of 1 but got %d", len(c.visited))
	}
}

func Test_ShouldCrawl(t *testing.T) {
	tc := []struct {
		testName       string
		url            string
		allowedDomains []string
		depth          int
		markVisited    bool
		expectedResult bool
	}{
		{
			allowedDomains: []string{"example.com"},
			testName:       "URL is allowed and not visited",
			url:            "https://example.com",
			depth:          1,
			markVisited:    false,
			expectedResult: true,
		},
		{
			allowedDomains: []string{"example.com"},
			testName:       "Depth is 0 so should not crawl",
			url:            "https://example.com",
			depth:          0,
			markVisited:    false,
			expectedResult: false,
		},
		{
			allowedDomains: []string{"example.com"},
			testName:       "URL is not in allowed domains",
			url:            "https://bad.com",
			depth:          1,
			markVisited:    false,
			expectedResult: false,
		},
		{
			allowedDomains: []string{"example.com"},
			testName:       "URL is already visited",
			url:            "https://example.com",
			depth:          1,
			markVisited:    true,
			expectedResult: false,
		},
	}

	for _, tc := range tc {
		t.Run(tc.testName, func(t *testing.T) {
			c := createTestCrawler(tc.allowedDomains...)

			if tc.markVisited {
				c.MarkVisited(tc.url)
			}

			result := shouldCrawl(tc.depth, tc.url, c)

			if result != tc.expectedResult {
				t.Errorf("unexpected result for URL %s and allowed domains %v: got %v, want %v", tc.url, tc.allowedDomains, result, tc.expectedResult)
			}

		})
	}

}
