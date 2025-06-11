package restclient

import (
	"fmt"
	"strings"
	"unicode"
)

// Helper functions extracted from parser.go to reduce file size

// isPotentialRequestLine checks if a line could be a request line
func isPotentialRequestLine(line string) bool {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return false
	}
	methodToken := strings.ToUpper(parts[0])
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true,
		"HEAD": true, "OPTIONS": true, "TRACE": true, "CONNECT": true,
	}
	return validMethods[methodToken]
}

// parseNameFromAtNameDirective checks if the commentContent is a well-formed @name directive
// and extracts the name value if present.
// It returns the extracted name (trimmed, or empty if no value) and a boolean indicating
// if the commentContent was indeed a recognized @name directive pattern.
func parseNameFromAtNameDirective(commentContent string) (nameValue string, isAtNamePattern bool) {
	if !strings.HasPrefix(commentContent, "@name") {
		return "", false // Not an @name pattern at all
	}

	// It starts with "@name". Now check if it's a valid form.
	// Valid forms: "@name" (no value), or "@name<whitespace>value"

	// Case 1: Exactly "@name"
	if len(commentContent) == len("@name") {
		return "", true // It's the @name pattern, value is empty.
	}

	// Case 2: Must be "@name" followed by whitespace to be our pattern.
	if !unicode.IsSpace(rune(commentContent[len("@name")])) {
		// e.g., "@nametag". This is not the "@name <value>" pattern.
		return "", false
	}

	// It's "@name" followed by whitespace. This is a recognized @name pattern.
	// Extract the potential value.
	// commentContent[len("@name"):] will get the part after "@name", including leading spaces.
	valuePart := commentContent[len("@name"):]
	// First, trim leading/trailing whitespace from the raw value part.
	trimmedValue := strings.TrimSpace(valuePart)
	// Then, normalize internal whitespace sequences (tabs, multiple spaces) to single spaces.
	// strings.Fields splits by any whitespace and removes empty strings resulting from multiple spaces.
	// strings.Join then puts them back with single spaces.
	normalizedName := strings.Join(strings.Fields(trimmedValue), " ")
	return normalizedName, true
}

// parseInt is a helper function to convert string to int.
func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscan(s, &n)
	return n, err
}

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
