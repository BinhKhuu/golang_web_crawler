package playwrightfetcher

import (
	"net/url"
	"path"
	"strings"
)

func canonicalizeFetchedURL(baseURL, href string, ignoreQueryParams, rootRelativePrefixes []string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}

	rawHref := strings.TrimSpace(href)
	if shouldTreatAsRootRelative(rawHref, rootRelativePrefixes) {
		rawHref = "/" + rawHref
	}

	u, err := url.Parse(rawHref)
	if err != nil {
		return ""
	}

	resolved := base.ResolveReference(u)
	resolved.Fragment = ""

	resolved.Scheme = strings.ToLower(resolved.Scheme)
	host := strings.ToLower(resolved.Hostname())
	port := resolved.Port()
	if (resolved.Scheme == "http" && port == "80") || (resolved.Scheme == "https" && port == "443") || port == "" {
		resolved.Host = host
	} else {
		resolved.Host = host + ":" + port
	}

	cleanPath := path.Clean(resolved.Path)
	if cleanPath == "." {
		cleanPath = "/"
	}
	if cleanPath != "/" && strings.HasSuffix(cleanPath, "/") {
		cleanPath = strings.TrimSuffix(cleanPath, "/")
	}
	resolved.Path = cleanPath

	query := resolved.Query()
	ignoreSet := make(map[string]struct{}, len(ignoreQueryParams))
	for _, k := range ignoreQueryParams {
		if key := strings.TrimSpace(strings.ToLower(k)); key != "" {
			ignoreSet[key] = struct{}{}
		}
	}

	for key := range query {
		lowerKey := strings.ToLower(key)
		if strings.HasPrefix(lowerKey, "utm_") {
			query.Del(key)
			continue
		}
		if _, shouldIgnore := ignoreSet[lowerKey]; shouldIgnore {
			query.Del(key)
		}
	}

	resolved.RawQuery = query.Encode()
	return resolved.String()
}

func shouldTreatAsRootRelative(href string, rootRelativePrefixes []string) bool {
	if href == "" || strings.HasPrefix(href, "/") || strings.HasPrefix(href, "./") || strings.HasPrefix(href, "../") {
		return false
	}

	for _, prefix := range rootRelativePrefixes {
		cleanPrefix := strings.TrimSpace(prefix)
		if cleanPrefix == "" {
			continue
		}
		if strings.HasPrefix(href, cleanPrefix) {
			return true
		}
	}

	return false
}
