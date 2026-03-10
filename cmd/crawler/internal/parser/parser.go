package parser

type HttpParser struct {
}

func NewHttpParser() *HttpParser {
	return &HttpParser{}
}

func (p *HttpParser) ParseLinks(body []byte) ([]string, error) {
	return []string{}, nil
}
