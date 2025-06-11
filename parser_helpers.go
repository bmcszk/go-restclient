package restclient

import (
	"strings"
)

// Helper functions extracted from parser.go to reduce file size

// extractURLAndVersion splits a URL string that may contain HTTP version.
// Returns the URL and HTTP version (if present).
// extractURLAndVersion splits a URL string that may contain HTTP version.
// Returns the URL and HTTP version (if present).
func (*requestParserState) extractURLAndVersion(urlAndVersion string) (urlStr, httpVersion string) {
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
	return allValidHTTPTokenChars(s)
}

// allValidHTTPTokenChars checks if all characters in string are valid HTTP token chars
func allValidHTTPTokenChars(s string) bool {
	for _, r := range s {
		if !isValidHTTPTokenChar(r) {
			return false
		}
	}
	return true
}

// isValidHTTPTokenChar checks if a rune is a valid HTTP token character
func isValidHTTPTokenChar(r rune) bool {
	return isAlphaNumeric(r) || isSpecialHTTPTokenChar(r)
}

// isAlphaNumeric checks if character is alphanumeric
func isAlphaNumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// isSpecialHTTPTokenChar checks if character is a special HTTP token character
func isSpecialHTTPTokenChar(r rune) bool {
	switch r {
	case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
		return true
	default:
		return false
	}
}
