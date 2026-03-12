package parser

import (
	"bytes"

	"github.com/PuerkitoBio/goquery"
)

type HttpParser struct {
}

func NewHttpParser() *HttpParser {
	return &HttpParser{}
}

func (p *HttpParser) ParseLinks(body []byte) ([]string, error) {
	links := make([]string, 0)
	r := bytes.NewReader(body)

	doc, err := goquery.NewDocumentFromReader(r)

	if err != nil {
		return nil, err
	}

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		links = append(links, href)
	})
	return links, nil
}
