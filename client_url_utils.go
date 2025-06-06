package restclient

import (
	"log/slog"
	"net/url"
)

// joinURLPaths joins base and request paths, handling query strings and fragments
// using the standard library's url.JoinPath for proper path joining
func joinURLPaths(base *url.URL, requestURL *url.URL) *url.URL {
	// Join the paths using the standard url.JoinPath
	targetPath, err := url.JoinPath(base.Path, requestURL.Path)
	if err != nil {
		slog.Error("joinURLPaths: failed to join paths", "basePath", base.Path,
			"requestPath", requestURL.Path, "error", err)
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
	slog.Debug("joinURLPaths: constructed finalURLStr", "url", finalURLStr)

	// Parse this string to get a fully validated *url.URL object
	finalResolvedURL, err := url.Parse(finalURLStr)
	if err != nil {
		slog.Error("joinURLPaths: failed to parse self-constructed URL string", "url", finalURLStr, "error", err)
		return nil
	}

	return finalResolvedURL
}
