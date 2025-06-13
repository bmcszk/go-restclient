package restclient

import (
	"net/url"
)

// joinURLPaths joins base and request paths, handling query strings and fragments
// using the standard library's url.JoinPath for proper path joining
func joinURLPaths(base *url.URL, requestURL *url.URL) *url.URL {
	// Join the paths using the standard url.JoinPath
	targetPath, err := url.JoinPath(base.Path, requestURL.Path)
	if err != nil {
		// Path join failed, return nil
		return nil
	}

	targetQuery := requestURL.RawQuery
	targetFragment := requestURL.Fragment

	// Create a URL struct from base parts and new path/query/fragment
	tempURL := url.URL{
		Scheme:   base.Scheme,
		Opaque:   base.Opaque,
		User:     base.User,
		Host:     base.Host,
		Path:     targetPath,
		RawQuery: targetQuery,
		Fragment: targetFragment,
	}

	// Get the string representation of this assembled URL
	finalURLStr := tempURL.String()
	// URL constructed successfully

	// Parse this string to get a fully validated *url.URL object
	finalResolvedURL, err := url.Parse(finalURLStr)
	if err != nil {
		// Failed to parse constructed URL
		return nil
	}

	return finalResolvedURL
}
