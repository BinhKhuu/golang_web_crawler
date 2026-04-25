package models

type RawData struct {
	URL         string
	ContentType string
	RawContent  string
	FetchedAt   string
}

type RawDataItem struct {
	URL         string
	ContentType string
	RawContent  string
}
