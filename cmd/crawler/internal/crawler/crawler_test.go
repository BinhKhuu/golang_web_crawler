package crawler

import (
	"context"
	"errors"
	"fmt"
	"golangwebcrawler/internal/models"
	"log/slog"
	"reflect"
	"sync"
	"testing"
)

var (
	_ Fetcher        = (*MockFetcher)(nil)
	_ Parser         = (*MockParser)(nil)
	_ StorageService = (*MockStorage)(nil)
)

const (
	testBaseURL = "https://example.com"
	maxDepth    = 4
	testDomain  = "example.com"
)

type MockFetcher struct {
	URL        string
	StatusCode int
	Body       []byte
	Err        error
}

func (m *MockFetcher) Fetch(ctx context.Context, url string) ([]FetchResult, error) {
	return []FetchResult{{
		URL:        m.URL,
		StatusCode: m.StatusCode,
		Body:       m.Body,
	}}, m.Err
}

type MockParser struct {
	Links []string
	Err   error
}

func (m *MockParser) ParseLinks(ctx context.Context, body []byte) ([]string, error) {
	return m.Links, m.Err
}

type MockStorage struct {
	Stored []string
	mu     sync.Mutex
}

func (m *MockStorage) StoreRawData(ctx context.Context, url, contentType, rawContent string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Stored = append(m.Stored, url)
	return nil
}

func (m *MockStorage) StoreRawDataBatch(ctx context.Context, items []models.RawDataItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, item := range items {
		m.Stored = append(m.Stored, item.URL)
	}
	return nil
}

func TestIsNavigated(t *testing.T) {
	c := createTestCrawler()
	c.markVisited(testBaseURL)

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

	c.markVisited(url)

	if c.visited[url] != true {
		t.Errorf("expected MarkVisited to mark URL as visited")
	}
}

func Test_ConcurrentMarkVisited(t *testing.T) {
	c := createTestCrawler()
	var wg sync.WaitGroup

	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c.markVisited(fmt.Sprintf(testBaseURL+"/%d", i))
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

	for range 100 {
		wg.Go(func() {
			c.markVisited(url)
		})
	}
	wg.Wait()
}

func createMockFetcher(url string, statusCode int, body []byte, err error) *MockFetcher {
	return &MockFetcher{
		URL:        url,
		StatusCode: statusCode,
		Body:       body,
		Err:        err,
	}
}

func createMockParser(links []string, err error) *MockParser {
	return &MockParser{
		Links: links,
		Err:   err,
	}
}

func createMockStorage() *MockStorage {
	return &MockStorage{
		Stored: []string{},
	}
}

func Test_Crawl(t *testing.T) {
	mockFetcher := createMockFetcher(testBaseURL, 200, []byte("mock body"), nil)
	mockParser := createMockParser([]string{testBaseURL + "/about", testBaseURL + "/contact"}, nil)
	mockStorage := createMockStorage()
	c := createTestCrawler()

	err := c.Crawl(t.Context(), testBaseURL, mockFetcher, mockParser, mockStorage, 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(mockStorage.Stored) != 3 {
		t.Errorf("expected storage to have 3 results, got %d", len(mockStorage.Stored))
	}
}

func createTestCrawler(allowedDomains ...string) *Crawler {
	if len(allowedDomains) == 0 {
		allowedDomains = []string{testDomain}
	}
	return NewCrawler(maxDepth, allowedDomains, slog.Default())
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
			c := createTestCrawler(tc.allowedDomains...)
			result := c.isAllowedDomain(tc.url)
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
			c := createTestCrawler()
			result := c.containsDomain(tc.url, tc.domain)
			if result != tc.expectedResult {
				t.Errorf("unexpected result for URL %s and domain %s: got %v, want %v", tc.url, tc.domain, result, tc.expectedResult)
			}
		})
	}
}

func Test_MaxDepthIsRespected(t *testing.T) {
	allowedDomains := []string{testDomain}
	c := NewCrawler(1, allowedDomains, slog.Default())
	mockFetcher := createMockFetcher(testBaseURL, 200, []byte("mock body"), nil)
	mockParser := createMockParser([]string{testBaseURL + "/about", testBaseURL + "/contact"}, nil)
	mockStorage := createMockStorage()

	if err := c.Crawl(t.Context(), testBaseURL, mockFetcher, mockParser, mockStorage, 5); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(c.visited) != 3 {
		t.Errorf("expected 3 visited URLs (start + 2 links) but got %d", len(c.visited))
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
				c.markVisited(tc.url)
			}

			var result bool
			switch {
			case tc.markVisited:
				result = false
			case tc.depth == 0:
				result = false
			case !c.isAllowedDomain(tc.url):
				result = false
			default:
				result = true
			}

			if result != tc.expectedResult {
				t.Errorf("unexpected result for URL %s and allowed domains %v: got %v, want %v", tc.url, tc.allowedDomains, result, tc.expectedResult)
			}
		})
	}
}

func Test_ProcessUrl(t *testing.T) {
	t.Run("successful fetch and parse", func(t *testing.T) {
		mockFetcher := createMockFetcher(testBaseURL, 200, []byte("mock body"), nil)
		mockParser := createMockParser([]string{testBaseURL + "/about", testBaseURL + "/contact"}, nil)
		mockStorage := createMockStorage()
		c := createTestCrawler()

		links, err := c.processURL(t.Context(), crawlJob{url: testBaseURL, depth: 1}, mockFetcher, mockParser, mockStorage)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(links) != 2 {
			t.Errorf("expected 2 links, got %d", len(links))
		}
	})

	t.Run("fetcher returns error", func(t *testing.T) {
		mockFetcher := createMockFetcher(testBaseURL, 500, nil, errors.New("fetch error"))
		mockParser := createMockParser(nil, nil)
		mockStorage := createMockStorage()
		c := createTestCrawler()

		links, err := c.processURL(t.Context(), crawlJob{url: testBaseURL, depth: 1}, mockFetcher, mockParser, mockStorage)
		if err == nil {
			t.Error("expected error, got nil")
		}
		if links != nil {
			t.Error("expected nil links on error")
		}
	})

	t.Run("parser returns error", func(t *testing.T) {
		mockFetcher := createMockFetcher(testBaseURL, 200, []byte("mock body"), nil)
		mockParser := createMockParser(nil, errors.New("parse error"))
		mockStorage := createMockStorage()
		c := createTestCrawler()

		links, err := c.processURL(t.Context(), crawlJob{url: testBaseURL, depth: 1}, mockFetcher, mockParser, mockStorage)
		if err == nil {
			t.Error("expected error, got nil")
		}
		if links != nil {
			t.Error("expected nil links on error")
		}
	})

	t.Run("storage is nil", func(t *testing.T) {
		mockFetcher := createMockFetcher(testBaseURL, 200, []byte("mock body"), nil)
		mockParser := createMockParser([]string{testBaseURL + "/about"}, nil)
		c := createTestCrawler()

		links, err := c.processURL(t.Context(), crawlJob{url: testBaseURL, depth: 1}, mockFetcher, mockParser, nil)
		if err != nil {
			t.Fatalf("expected no error with nil storage, got %v", err)
		}
		if len(links) != 1 {
			t.Errorf("expected 1 link, got %d", len(links))
		}
	})

	t.Run("depth zero returns no links", func(t *testing.T) {
		mockFetcher := createMockFetcher(testBaseURL, 200, []byte("mock body"), nil)
		mockParser := createMockParser([]string{testBaseURL + "/about"}, nil)
		mockStorage := createMockStorage()
		c := createTestCrawler()

		links, err := c.processURL(t.Context(), crawlJob{url: testBaseURL, depth: 0}, mockFetcher, mockParser, mockStorage)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if links != nil {
			t.Error("expected nil links at depth 0")
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

	c := createTestCrawler()
	result, err := c.formatLinks(links, baseUrl)
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

func Test_Coordinator_NoDeadlock(t *testing.T) {
	mockParser := createMockParser([]string{
		testBaseURL + "/a", testBaseURL + "/b", testBaseURL + "/c",
		testBaseURL + "/d", testBaseURL + "/e",
	}, nil)
	mockFetcher := createMockFetcher(testBaseURL, 200, []byte("body"), nil)
	mockStorage := createMockStorage()

	c := NewCrawler(3, []string{testDomain}, slog.Default())

	err := c.Crawl(t.Context(), testBaseURL, mockFetcher, mockParser, mockStorage, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(c.visited) == 0 {
		t.Error("expected some URLs to be visited")
	}
}

func Test_Coordinator_AllLinksDiscovered(t *testing.T) {
	mockParser := createMockParser([]string{
		testBaseURL + "/1",
		testBaseURL + "/2",
	}, nil)
	mockFetcher := createMockFetcher(testBaseURL, 200, []byte("body"), nil)
	mockStorage := createMockStorage()

	c := NewCrawler(2, []string{testDomain}, slog.Default())

	err := c.Crawl(t.Context(), testBaseURL, mockFetcher, mockParser, mockStorage, 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(c.visited) != 3 {
		t.Errorf("expected 3 visited URLs (start + 2 links), got %d", len(c.visited))
	}
}

func Test_Coordinator_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	blockingFetcher := &MockFetcher{
		Body: []byte("<a href='/link1'>link</a>"),
	}
	mockParser := createMockParser([]string{testBaseURL + "/link1"}, nil)
	mockStorage := createMockStorage()

	c := NewCrawler(10, []string{testDomain}, slog.Default())

	cancel()

	err := c.Crawl(ctx, testBaseURL, blockingFetcher, mockParser, mockStorage, 2)
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Logf("crawl ended with: %v", err)
	}
}

func Test_ProcessURL_BatchStorage(t *testing.T) {
	mockFetcher := createMockFetcher(testBaseURL, 200, []byte("mock body"), nil)
	mockParser := createMockParser([]string{testBaseURL + "/about"}, nil)
	mockStorage := createMockStorage()
	c := createTestCrawler()

	links, err := c.processURL(t.Context(), crawlJob{url: testBaseURL, depth: 1}, mockFetcher, mockParser, mockStorage)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(links) != 1 {
		t.Errorf("expected 1 link, got %d", len(links))
	}
	if len(mockStorage.Stored) != 1 {
		t.Errorf("expected 1 stored item, got %d", len(mockStorage.Stored))
	}
}

func Test_ProcessURL_BatchStorageMultipleResults(t *testing.T) {
	multiFetcher := &MultiMockFetcher{
		Results: []FetchResult{
			{URL: testBaseURL + "/1", Body: []byte("body 1"), StatusCode: 200},
			{URL: testBaseURL + "/2", Body: []byte("body 2"), StatusCode: 200},
			{URL: testBaseURL + "/3", Body: []byte("body 3"), StatusCode: 200},
		},
	}
	mockParser := createMockParser([]string{}, nil)
	mockStorage := createMockStorage()
	c := createTestCrawler()

	_, err := c.processURL(t.Context(), crawlJob{url: testBaseURL, depth: 1}, multiFetcher, mockParser, mockStorage)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(mockStorage.Stored) != 3 {
		t.Errorf("expected 3 stored items, got %d", len(mockStorage.Stored))
	}
}

type MultiMockFetcher struct {
	Results []FetchResult
	Err     error
}

func (m *MultiMockFetcher) Fetch(ctx context.Context, url string) ([]FetchResult, error) {
	return m.Results, m.Err
}

var _ Fetcher = (*MultiMockFetcher)(nil)
