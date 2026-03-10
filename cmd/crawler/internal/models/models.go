package models

type FetchResult struct {
	URL        string
	StatusCode int
	Body       []byte
	Err        error
}
