package parser

import (
	"bytes"
	"context"

	"github.com/PuerkitoBio/goquery"
)

type HTTPParser struct{}

func NewHTTPParser() *HTTPParser {
	return &HTTPParser{}
}

func (p *HTTPParser) ParseLinks(ctx context.Context, body []byte) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	links := make([]string, 0)
	r := bytes.NewReader(body)

	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		links = append(links, href)
	})
	return links, nil
}
