package crawler

import (
	"fmt"
	"sync"
	"testing"
)

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
