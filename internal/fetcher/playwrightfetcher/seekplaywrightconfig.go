package playwrightfetcher

const (
	// Seek selectors.
	seekJobTitleSelector       = "a[data-automation='jobTitle']"
	seekJobLinkSelector        = "a.job-link"
	seekJobTestIdSelector      = "a[data-testid='job-result']"
	seekKeywordsInputSelector  = "input[name=keywords]"
	seekSearchPlaceholder      = "input[placeholder*='Search']"
	seekSubmitButton           = "button[type='submit']"
	seekAutomationSearchButton = "button[data-automation='searchButton']"

	// Seek canonicalization.
	seekTrackingParamSol    = "sol"
	seekTrackingParamRef    = "ref"
	seekTrackingParamOrigin = "origin"
	seekJobPathPrefix       = "job/"

	// Seek default URLs.
	seekSoftwareEngineerJobsURL = "https://www.seek.com.au/software-engineer-jobs"
)

func GetSeekConfiguration() PlaywrightFetcherConfig {
	return PlaywrightFetcherConfig{
		// Target
		URL:      seekSoftwareEngineerJobsURL,
		Headless: true,
		Timeout:  defaultTimeout,

		// Search interaction: fill input, submit, then wait for results
		Search: SearchConfig{
			InputSelectors: []string{
				seekKeywordsInputSelector,
				seekSearchPlaceholder,
			},
			Query: "Software Engineer Jobs",
			SubmitSelectors: []string{
				seekSubmitButton,
				seekAutomationSearchButton,
			},
		},

		// Result collection: selectors for job listing links, then detail content
		Results: ResultsConfig{
			ListingSelectors: []string{
				seekJobTitleSelector,
				seekJobLinkSelector,
				seekJobTestIdSelector,
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
			IgnoreQueryParams:    []string{seekTrackingParamSol, seekTrackingParamRef, seekTrackingParamOrigin},
			RootRelativePrefixes: []string{seekJobPathPrefix},
		},
	}
}
