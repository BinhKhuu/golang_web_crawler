package fetcher

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

type SPAResult struct {
	URL           string
	HTML          string
	Links         []SPALink
	DetailsPanels []DetailsPanel
}

type SPALink struct {
	Text     string
	Href     string
	Selector string
}

type DetailsPanel struct {
	TriggerLink SPALink
	HTML        string
}

type BrowserFetcherConfig struct {
	Headless      bool
	WaitTimeout   time.Duration
	RenderTimeout time.Duration
	MaxLinks      int
	WaitSelectors []string
}

func DefaultBrowserFetcherConfig() BrowserFetcherConfig {
	return BrowserFetcherConfig{
		Headless:      true,
		WaitTimeout:   15 * time.Second,
		RenderTimeout: 3 * time.Second,
		MaxLinks:      20,
		WaitSelectors: []string{"body"},
	}
}

type BrowserFetcher struct {
	browser *rod.Browser
	config  BrowserFetcherConfig
}

func NewBrowserFetcher(config BrowserFetcherConfig) (*BrowserFetcher, error) {
	l := launcher.New().
		Headless(config.Headless).
		Set("disable-gpu", "").
		Set("no-sandbox", "").
		Set("disable-dev-shm-usage", "")

	url, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launching browser: %w", err)
	}

	browser := rod.New().ControlURL(url).MustConnect()

	return &BrowserFetcher{
		browser: browser,
		config:  config,
	}, nil
}

func (f *BrowserFetcher) FetchSPA(ctx context.Context, url string) (SPAResult, error) {
	page := f.browser.MustPage()
	defer page.MustClose()

	timeoutCtx, cancel := context.WithTimeout(ctx, f.config.WaitTimeout)
	defer cancel()

	if err := page.Navigate(url); err != nil {
		return SPAResult{}, fmt.Errorf("navigating to %s: %w", url, err)
	}

	for _, selector := range f.config.WaitSelectors {
		if _, err := page.Timeout(f.config.WaitTimeout).Element(selector); err != nil {
			return SPAResult{}, fmt.Errorf("waiting for selector %s: %w", selector, err)
		}
	}

	f.waitForNetworkIdle(page)

	time.Sleep(f.config.RenderTimeout)

	html := page.MustHTML()

	links, err := f.extractClickableLinks(page)
	if err != nil {
		return SPAResult{}, fmt.Errorf("extracting links: %w", err)
	}

	var panels []DetailsPanel
	for _, link := range links {
		select {
		case <-timeoutCtx.Done():
			return SPAResult{
				URL:           url,
				HTML:          html,
				Links:         links,
				DetailsPanels: panels,
			}, nil
		default:
		}

		panel, err := f.clickAndWaitForDetails(page, link)
		if err != nil {
			continue
		}
		panels = append(panels, panel)

		if err := f.closeDetailsPanel(page); err != nil {
			if navErr := page.Timeout(5 * time.Second).Navigate(url); navErr != nil {
				return SPAResult{
					URL:           url,
					HTML:          html,
					Links:         links,
					DetailsPanels: panels,
				}, fmt.Errorf("resetting page state: %w", navErr)
			}
			time.Sleep(f.config.RenderTimeout)
		}
	}

	return SPAResult{
		URL:           url,
		HTML:          html,
		Links:         links,
		DetailsPanels: panels,
	}, nil
}

func (f *BrowserFetcher) extractClickableLinks(page *rod.Page) ([]SPALink, error) {
	elements := page.MustElements("a[href]")

	var links []SPALink
	for _, el := range elements {
		href, _ := el.Attribute("href")
		if href == nil || *href == "" || *href == "#" {
			continue
		}

		text := el.MustText()
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}

		selector, err := f.getElementSelector(el)
		if err != nil {
			selector = ""
		}

		links = append(links, SPALink{
			Text:     text,
			Href:     *href,
			Selector: selector,
		})

		if len(links) >= f.config.MaxLinks {
			break
		}
	}

	if len(links) == 0 {
		elements := page.MustElements("button, [role='button'], [onclick]")
		for _, el := range elements {
			text := el.MustText()
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}

			selector, err := f.getElementSelector(el)
			if err != nil {
				selector = ""
			}

			links = append(links, SPALink{
				Text:     text,
				Selector: selector,
			})

			if len(links) >= f.config.MaxLinks {
				break
			}
		}
	}

	return links, nil
}

func (f *BrowserFetcher) getElementSelector(el *rod.Element) (string, error) {
	res, err := el.Eval(`() => {
		const el = this;
		if (el.id) return '#' + el.id;
		const parts = [];
		let current = el;
		while (current && current.nodeType === 1) {
			let selector = current.tagName.toLowerCase();
			if (current.id) { selector += '#' + current.id; parts.unshift(selector); break; }
			let sibling = current;
			let nth = 1;
			while (sibling.previousElementSibling) {
				sibling = sibling.previousElementSibling;
				if (sibling.tagName === current.tagName) nth++;
			}
			if (nth > 1 || current.nextElementSibling) selector += ':nth-of-type(' + nth + ')';
			parts.unshift(selector);
			current = current.parentElement;
		}
		return parts.join(' > ');
	}`)
	if err != nil {
		return "", err
	}
	return res.Value.Str(), nil
}

func (f *BrowserFetcher) clickAndWaitForDetails(page *rod.Page, link SPALink) (DetailsPanel, error) {
	elements := page.MustElements("a")

	var target *rod.Element
	for _, el := range elements {
		href, _ := el.Attribute("href")
		if href != nil && *href == link.Href {
			target = el
			break
		}
	}

	if target == nil {
		return DetailsPanel{}, fmt.Errorf("element not found for link: %s", link.Text)
	}

	target.MustClick()

	time.Sleep(f.config.RenderTimeout)

	f.waitForNetworkIdle(page)

	html := page.MustHTML()

	return DetailsPanel{
		TriggerLink: link,
		HTML:        html,
	}, nil
}

func (f *BrowserFetcher) closeDetailsPanel(page *rod.Page) error {
	closeSelectors := []string{
		"button[aria-label='Close']",
		"button.close",
		".modal-close",
		"[data-testid='close']",
		".dismiss",
	}

	for _, selector := range closeSelectors {
		els, err := page.Elements(selector)
		if err == nil && len(els) > 0 {
			els[0].MustClick()
			time.Sleep(500 * time.Millisecond)
			return nil
		}
	}

	return fmt.Errorf("no close button found")
}

func (f *BrowserFetcher) waitForNetworkIdle(page *rod.Page) {
	wait := page.WaitRequestIdle(500*time.Millisecond, nil, nil, nil)
	wait()
}

func (f *BrowserFetcher) Close() error {
	return f.browser.Close()
}

func (f *BrowserFetcher) Fetch(ctx context.Context, url string) (FetchResult, error) {
	spaResult, err := f.FetchSPA(ctx, url)
	if err != nil {
		return FetchResult{}, err
	}

	return FetchResult{
		URL:        spaResult.URL,
		StatusCode: 200,
		Body:       []byte(spaResult.HTML),
	}, nil
}

func (f *BrowserFetcher) FetchWithDetails(ctx context.Context, url string) (SPAResult, error) {
	return f.FetchSPA(ctx, url)
}

type BrowserFetcherOption func(*BrowserFetcherConfig)

func WithHeadless(headless bool) BrowserFetcherOption {
	return func(c *BrowserFetcherConfig) {
		c.Headless = headless
	}
}

func WithWaitTimeout(timeout time.Duration) BrowserFetcherOption {
	return func(c *BrowserFetcherConfig) {
		c.WaitTimeout = timeout
	}
}

func WithRenderTimeout(timeout time.Duration) BrowserFetcherOption {
	return func(c *BrowserFetcherConfig) {
		c.RenderTimeout = timeout
	}
}

func WithMaxLinks(max int) BrowserFetcherOption {
	return func(c *BrowserFetcherConfig) {
		c.MaxLinks = max
	}
}

func NewBrowserFetcherWithOptions(opts ...BrowserFetcherOption) (*BrowserFetcher, error) {
	config := DefaultBrowserFetcherConfig()
	for _, opt := range opts {
		opt(&config)
	}

	return NewBrowserFetcher(config)
}

func (f *BrowserFetcher) ClickAndWait(ctx context.Context, url string, clickSelector string, waitSelector string) (string, error) {
	page := f.browser.MustPage()
	defer page.MustClose()

	timeoutCtx, cancel := context.WithTimeout(ctx, f.config.WaitTimeout)
	defer cancel()

	if err := page.Navigate(url); err != nil {
		return "", fmt.Errorf("navigating to %s: %w", url, err)
	}

	if _, err := page.Timeout(f.config.WaitTimeout).Element("body"); err != nil {
		return "", fmt.Errorf("waiting for body: %w", err)
	}

	f.waitForNetworkIdle(page)

	el, err := page.Timeout(f.config.WaitTimeout).Element(clickSelector)
	if err != nil {
		return "", fmt.Errorf("finding element %s: %w", clickSelector, err)
	}

	el.MustClick()

	if waitSelector != "" {
		if _, err := page.Timeout(f.config.WaitTimeout).Element(waitSelector); err != nil {
			return page.MustHTML(), fmt.Errorf("waiting for selector %s after click: %w", waitSelector, err)
		}
	}

	select {
	case <-timeoutCtx.Done():
		return page.MustHTML(), nil
	default:
		time.Sleep(f.config.RenderTimeout)
	}

	return page.MustHTML(), nil
}
