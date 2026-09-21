package restclient

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
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

// resolveRequestURL returns the request URL resolved against BaseURL, or an error.
func (*Client) resolveRequestURL(
	baseURLStr string,
	initialRequestURL *url.URL,
	rawRequestURLStr string,
) (*url.URL, error) {
	currentRequestURL, err := determineCurrentRequestURL(initialRequestURL, rawRequestURLStr)
	if err != nil {
		return nil, err
	}

	freshRequestURL, err := sanitizeRequestURL(currentRequestURL)
	if err != nil {
		return nil, err
	}

	return resolveWithBaseURL(freshRequestURL, baseURLStr)
}

// determineCurrentRequestURL determines which URL to use for processing
func determineCurrentRequestURL(initialRequestURL *url.URL, rawRequestURLStr string) (*url.URL, error) {
	if initialRequestURL != nil {
		return initialRequestURL, nil
	}
	if rawRequestURLStr != "" {
		parsedRawURL, err := url.Parse(rawRequestURLStr)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to parse rawRequestURLString '%s' after variable expansion: %w",
				rawRequestURLStr, err)
		}
		return parsedRawURL, nil
	}
	return nil, errors.New("request URL is unexpectedly nil and rawRequestURLString is empty")
}

// sanitizeRequestURL re-parses a URL to ensure it's valid
func sanitizeRequestURL(currentRequestURL *url.URL) (*url.URL, error) {
	currentRequestURLStr := currentRequestURL.String()
	freshRequestURL, err := url.Parse(currentRequestURLStr)
	if err != nil {
		return nil, fmt.Errorf("failed to re-parse current requestURL string '%s': %w", currentRequestURLStr, err)
	}
	return freshRequestURL, nil
}

// resolveWithBaseURL resolves a request URL against a base URL
func resolveWithBaseURL(freshRequestURL *url.URL, baseURLStr string) (*url.URL, error) {
	if freshRequestURL.IsAbs() {
		return freshRequestURL, nil
	}
	if baseURLStr == "" {
		return freshRequestURL, nil
	}

	freshBase, err := parseAndSanitizeBaseURL(baseURLStr)
	if err != nil {
		return nil, err
	}

	return handleSpecialPathJoining(freshRequestURL, freshBase)
}

// parseAndSanitizeBaseURL parses and sanitizes a base URL
func parseAndSanitizeBaseURL(baseURLStr string) (*url.URL, error) {
	base, err := url.Parse(baseURLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid BaseURL %s: %w", baseURLStr, err)
	}

	baseStr := base.String()
	freshBase, err := url.Parse(baseStr)
	if err != nil {
		return nil, fmt.Errorf("failed to re-parse base URL string '%s': %w", baseStr, err)
	}
	return freshBase, nil
}

// handleSpecialPathJoining handles special cases for URL path joining
func handleSpecialPathJoining(freshRequestURL, freshBase *url.URL) (*url.URL, error) {
	if strings.HasPrefix(freshRequestURL.Path, "/") && freshBase.Path != "" && freshBase.Path != "/" {
		finalResolvedURL := joinURLPaths(freshBase, freshRequestURL)
		if finalResolvedURL == nil {
			return nil, fmt.Errorf("failed to join URL paths: %s and %s", freshBase.Path, freshRequestURL.Path)
		}
		return finalResolvedURL, nil
	}
	return freshBase.ResolveReference(freshRequestURL), nil
}
