package restclient

import (
	"strings"
)

// Helper functions extracted from parser.go to reduce file size

// extractURLAndVersion splits a URL string that may contain HTTP version.
// Returns the URL and HTTP version (if present).
// extractURLAndVersion splits a URL string that may contain HTTP version.
// Returns the URL and HTTP version (if present).
func (p *requestParserState) extractURLAndVersion(urlAndVersion string) (string, string) {
	// Check for HTTP version (e.g., "http://example.com HTTP/1.1")
	parts := strings.Split(urlAndVersion, " ")

	if len(parts) > 1 && strings.HasPrefix(parts[len(parts)-1], "HTTP/") {
		// Last part is HTTP version
		httpVersion := parts[len(parts)-1]
		// URL is everything else
		urlStr := strings.Join(parts[:len(parts)-1], " ")
		return strings.TrimSpace(urlStr), httpVersion
	}

	// No HTTP version, whole string is URL
	return urlAndVersion, ""
}
