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

// isValidHTTPToken checks if a string is a valid HTTP token (method, header field name, etc.)
// as per RFC 7230, Section 3.2.6: 1*tchar.
// tchar = "!" / "#" / "$" / "%" / "&" / "'" / "*" / "+" / "-" / "." /
//
//	"^" / "_" / "`" / "|" / "~" / DIGIT / ALPHA
func isValidHTTPToken(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '!' || r == '#' || r == '$' || r == '%' || r == '&' || r == '\'' ||
			r == '*' || r == '+' || r == '-' || r == '.' || r == '^' || r == '_' ||
			r == '`' || r == '|' || r == '~' {
			continue
		}
		return false
	}
	return true
}
