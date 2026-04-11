package playwrightfetcher

func GetSeekConfiguration() PlaywrightFetcherConfig {
	defaultTimeout := 10000
	return PlaywrightFetcherConfig{
		URL:      "https://www.seek.com.au",
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
			"a[id*='job-title']",
			"a[data-automation='jobTitle']",
			".job-title a",
			"article a[href*='/job/']",
		},
		SPAUpdateSelectors: []string{
			"#job-details",
			".JobDetail",
			"[data-automation='jobDetail']",
			".job-detail-content",
		},
		Timeout: defaultTimeout,
	}
}
