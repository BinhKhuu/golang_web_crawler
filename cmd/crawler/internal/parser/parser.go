package parser

import (
	"bytes"

	"github.com/PuerkitoBio/goquery"
)

type HTTPParser struct{}

func NewHTTPParser() *HTTPParser {
	return &HTTPParser{}
}

func (p *HTTPParser) ParseLinks(body []byte) ([]string, error) {
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
