package crawler

import (
	"errors"
	"fmt"
	"golangwebcrawler/cmd/crawler/internal/models"
	"reflect"
	"sync"
	"testing"
)

// Compile-time assertions
var _ Fetcher = (*MockFetchResults)(nil)
var _ Parser = (*MockParserResults)(nil)
var _ StorageService = (*MockStorageService)(nil)

const (
	testBaseURL = "https://example.com"
	maxDepth    = 4
	testDomain  = "example.com"
)

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
	Err   error
}

func (m *MockParserResults) ParseLinks(body []byte) ([]string, error) {
	return m.Links, m.Err
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
	c.MarkVisited(testBaseURL)

	if !c.IsNavigated(testBaseURL) {
		t.Error("expected URL to be marked as visited")
	}
	if c.IsNavigated("https://other.com") {
		t.Error("expected URL to not be visited")
	}
}

func Test_MarkVisited(t *testing.T) {
	url := testBaseURL
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
			c.MarkVisited(fmt.Sprintf(testBaseURL+"/%d", i))
		}(i)
	}

	wg.Wait()

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
}

func createMockFetchResults(url string, statusCode int, body []byte, err error) *MockFetchResults {
	return &MockFetchResults{
		URL:        url,
		StatusCode: statusCode,
		Body:       body,
		Err:        err,
	}
}

func createMockParseResults(links []string, err error) *MockParserResults {
	return &MockParserResults{
		Links: links,
		Err:   err,
	}
}

func createMockStoreageService() *MockStorageService {
	return &MockStorageService{
		Stored: []models.RawData{},
	}
}

func Test_CrawlAsync(t *testing.T) {
	mockFetcher := createMockFetchResults(testBaseURL, 200, []byte("mock body"), nil)
	mockParser := createMockParseResults([]string{testBaseURL + "/about", testBaseURL + "/contact"}, nil)
	mockStorage := createMockStoreageService()
	c := createTestCrawler()

	err := c.CrawlAsync(testBaseURL, maxDepth, mockFetcher, mockParser, mockStorage)
	c.Wait()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	c.Wait()

	if len(mockStorage.Stored) == 0 {
		t.Error("expected storage to have at least one result")
	}
	if mockStorage.Stored[0].URL != testBaseURL {
		t.Errorf("unexpected stored URL: %s", mockStorage.Stored[0].URL)
	}
}

func createTestCrawler(allowedDomains ...string) *CrawlerState {
	if len(allowedDomains) == 0 {
		allowedDomains = []string{testDomain}
	}
	return NewCrawler(maxDepth, allowedDomains)
}

func Test_IsAllowedDomain(t *testing.T) {
	const otherDomain = "other.com"
	tc := []struct {
		testName       string
		url            string
		allowedDomains []string
		expectedResult bool
	}{
		{
			testName:       "URL is in allowed domains",
			url:            testBaseURL + "/page",
			allowedDomains: []string{testDomain, otherDomain},
			expectedResult: true,
		},
		{
			testName:       "URL is not in allowed domains",
			url:            "https://bad.com/page",
			allowedDomains: []string{testDomain, otherDomain, "bads.com"},
			expectedResult: false,
		},
		{
			testName:       "URL is in allowed domains as subdomain",
			url:            "https://sub.example.com/page",
			allowedDomains: []string{testDomain},
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

func Test_ContainsDomain(t *testing.T) {
	tc := []struct {
		testName       string
		url            string
		domain         string
		expectedResult bool
	}{
		{
			testName:       "URL contains domain",
			url:            testBaseURL + "/page",
			domain:         testDomain,
			expectedResult: true,
		},
		{
			testName:       "URL does not contain domain",
			url:            "https://other.com/page",
			domain:         testDomain,
			expectedResult: false,
		},
		{
			testName:       "URL contains domain as subdomain",
			url:            "https://sub.example.com/page",
			domain:         testDomain,
			expectedResult: true,
		},
		{
			testName:       "URL link path with just / should not be considered as containing the domain",
			url:            "/about",
			domain:         testDomain,
			expectedResult: false,
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
	allowedDomains := []string{testDomain}
	c := createTestCrawler(allowedDomains...)
	mockFetcher := createMockFetchResults(testBaseURL, 200, []byte("mock body"), nil)
	mockParser := createMockParseResults([]string{testBaseURL + "/about", testBaseURL + "/contact"}, nil)
	mockStorage := createMockStoreageService()

	if err := c.CrawlAsync(testBaseURL, 1, mockFetcher, mockParser, mockStorage); err != nil {
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
			allowedDomains: []string{testDomain},
			testName:       "URL is allowed and not visited",
			url:            testBaseURL,
			depth:          1,
			markVisited:    false,
			expectedResult: true,
		},
		{
			allowedDomains: []string{testDomain},
			testName:       "Depth is 0 so should not crawl",
			url:            testBaseURL,
			depth:          0,
			markVisited:    false,
			expectedResult: false,
		},
		{
			allowedDomains: []string{testDomain},
			testName:       "URL is not in allowed domains",
			url:            "https://bad.com",
			depth:          1,
			markVisited:    false,
			expectedResult: false,
		},
		{
			allowedDomains: []string{testDomain},
			testName:       "URL is already visited",
			url:            testBaseURL,
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

func Test_ProcessUrl(t *testing.T) {
	t.Run("successful fetch and parse", func(t *testing.T) {
		mockFetcher := createMockFetchResults(testBaseURL, 200, []byte("mock body"), nil)
		mockParser := createMockParseResults([]string{testBaseURL + "/about", testBaseURL + "/contact"}, nil)
		mockStorage := createMockStoreageService()

		links, err := processUrl(mockFetcher, testBaseURL, mockParser, mockStorage)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(links) != 2 {
			t.Errorf("expected 2 links, got %d", len(links))
		}
	})

	t.Run("fetcher returns error", func(t *testing.T) {
		mockFetcher := createMockFetchResults(testBaseURL, 500, nil, errors.New("fetch error"))
		mockParser := createMockParseResults(nil, nil)
		mockStorage := createMockStoreageService()

		links, err := processUrl(mockFetcher, testBaseURL, mockParser, mockStorage)
		if err == nil {
			t.Error("expected error, got nil")
		}
		if links != nil {
			t.Error("expected nil links on error")
		}
	})

	t.Run("parser returns error", func(t *testing.T) {
		mockFetcher := createMockFetchResults(testBaseURL, 200, []byte("mock body"), nil)
		mockParser := createMockParseResults(nil, errors.New("parse error"))
		mockStorage := createMockStoreageService()

		links, err := processUrl(mockFetcher, testBaseURL, mockParser, mockStorage)
		if err == nil {
			t.Error("expected error, got nil")
		}
		if links != nil {
			t.Error("expected nil links on error")
		}
	})

	t.Run("storage is nil", func(t *testing.T) {
		mockFetcher := createMockFetchResults(testBaseURL, 200, []byte("mock body"), nil)
		mockParser := createMockParseResults([]string{testBaseURL + "/about"}, nil)

		links, err := processUrl(mockFetcher, testBaseURL, mockParser, nil)
		if err != nil {
			t.Fatalf("expected no error with nil storage, got %v", err)
		}
		if len(links) != 1 {
			t.Errorf("expected 1 link, got %d", len(links))
		}
	})
}

func Test_FormatLinks(t *testing.T) {
	links := []string{
		"/about",
		"https://example.com/contact",
		"//example.com/blog",
		"test",
		"/about/me",
	}

	baseUrl := "https://example.com"

	expected := []string{
		"https://example.com/about",
		"https://example.com/contact",
		"https://example.com/blog",
		"https://example.com/test",
		"https://example.com/about/me",
	}

	result, err := formatLinks(links, baseUrl)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result) != len(expected) {
		t.Fatalf("expected %d links, got %d", len(expected), len(result))
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("unexpected formatted links: got %v, want %v", result, expected)
	}
}
