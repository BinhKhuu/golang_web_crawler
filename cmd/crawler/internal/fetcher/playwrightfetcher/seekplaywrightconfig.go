package playwrightfetcher

func GetSeekConfiguration() PlaywrightFetcherConfig {
	return PlaywrightFetcherConfig{
		// Target
		URL:      "https://www.seek.com.au/software-engineer-jobs",
		Headless: true,
		Timeout:  10000,

		// Search interaction: fill input, submit, then wait for results
		Search: SearchConfig{
			InputSelectors: []string{
				"input[name=keywords]",
				"input[placeholder*='Search']",
			},
			Query: "Software Engineer Jobs",
			SubmitSelectors: []string{
				"button[type='submit']",
				"button[data-automation='searchButton']",
			},
		},

		// Result collection: selectors for job listing links, then detail content
		Results: ResultsConfig{
			ListingSelectors: []string{
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
		},

		// URL canonicalization: strip tracking params and resolve root-relative hrefs
		Canonicalization: CanonicalizationConfig{
			IgnoreQueryParams:    []string{"sol", "ref", "origin"},
			RootRelativePrefixes: []string{"job/"},
		},
	}
}
