package playwrightfetcher

func GetSeekConfiguration() PlaywrightFetcherConfig {
	defaultTimeout := 10000
	return PlaywrightFetcherConfig{
		URL:      "https://www.seek.com.au/software-engineer-jobs",
		Headless: true,
		SearchInputSelectors: []string{
			"input[name=keywords]",
			"input[placeholder*='Search']",
		},
		SearchQuery: "Software Engineer Jobs",
		SearchSubmitSelectors: []string{
			"button[type='submit']",
			"button[data-automation='searchButton']",
		},
		ResultsSelectors: []string{
			"a[data-automation='jobTitle']",
			"a.job-link",
			"a[data-testid='job-result']",
		},
		DataSelectors: []string{
			"#job-details",
			".JobDetail",
			"[data-automation='jobDetail']",
			"[data-automation='jobDetailsPage']",
			".job-detail-content",
		},
		Timeout: defaultTimeout,
	}
}
