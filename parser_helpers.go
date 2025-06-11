package restclient

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
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

// Expected response parsing functions moved from parser.go

// parseExpectedStatusLine parses a line as an HTTP status line (HTTP_VERSION STATUS_CODE [STATUS_TEXT]).
// It updates the provided ExpectedResponse with the parsed status code and status string.
func parseExpectedStatusLine(line string, lineNumber int, resp *ExpectedResponse) error {
	parts := strings.Fields(line)
	if len(parts) < 2 { // Must have at least HTTP_VERSION STATUS_CODE [STATUS_TEXT]
		return fmt.Errorf(
			"line %d: invalid status line: '%s'. Expected HTTP_VERSION STATUS_CODE [STATUS_TEXT]",
			lineNumber, line)
	}
	// parts[0] is HTTP Version, parts[1] is StatusCode, rest is StatusText
	statusCodeInt, err := parseInt(parts[1])
	if err != nil {
		return fmt.Errorf("line %d: invalid status code '%s': %w", lineNumber, parts[1], err)
	}
	resp.StatusCode = &statusCodeInt

	var finalStatusString string
	if len(parts) > 2 {
		finalStatusString = strings.Join(parts[1:], " ") // e.g. "200 OK"
	} else {
		finalStatusString = parts[1] // Just code, e.g. "200"
	}
	resp.Status = &finalStatusString // Store the combined status code and text or just code
	return nil
}

// parseExpectedHeaderLine parses a line as an HTTP header (Key: Value).
// It updates the provided ExpectedResponse by adding the parsed header.
func parseExpectedHeaderLine(line string, lineNumber int, resp *ExpectedResponse) error {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("line %d: invalid header line: '%s'. Expected 'Key: Value'", lineNumber, line)
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	resp.Headers.Add(key, value)
	return nil
}

// processExpectedStatusOrHeaderLine determines if a line is an expected status or header and parses it accordingly.
func processExpectedStatusOrHeaderLine(line string, lineNumber int, resp *ExpectedResponse) error {
	if (resp.Status == nil || *resp.Status == "") && resp.StatusCode == nil {
		// Expecting a status line
		return parseExpectedStatusLine(line, lineNumber, resp)
	}
	// Expecting a header line
	return parseExpectedHeaderLine(line, lineNumber, resp)
}

// parseExpectedResponses parses expected HTTP response definitions from an io.Reader.
// It expects the content provided by the reader to be the raw .hresp format, typically after
// any variable substitutions have already been performed (e.g., by `resolveAndSubstitute`).
//
// The `filePath` argument is used for context in error messages only and does not imply that this
// function reads from the file system directly. It processes content line by line, interpreting
// status lines, headers, and body sections, separated by "###". Comments (#) and lines starting
// with "@" (which should have been processed prior to calling this function if they were variable
// definitions) are ignored.
//
// Returns a slice of `ExpectedResponse` structs or an error if parsing fails (e.g., due to
// malformed status lines or headers).
// responseParserState holds the state for parsing expected responses
type responseParserState struct {
	expectedResponses       []*ExpectedResponse
	currentExpectedResponse *ExpectedResponse
	bodyLines               []string
	parsingBody             bool
	lineNumber              int
	processedAnyLine        bool
}

func parseExpectedResponses(reader io.Reader, filePath string) ([]*ExpectedResponse, error) {
	scanner := bufio.NewScanner(reader)
	state := &responseParserState{
		expectedResponses:       []*ExpectedResponse{},
		currentExpectedResponse: &ExpectedResponse{Headers: make(http.Header)},
		bodyLines:               []string{},
		parsingBody:             false,
		lineNumber:              0,
		processedAnyLine:        false,
	}

	for scanner.Scan() {
		state.lineNumber++
		originalLine := scanner.Text()
		trimmedLine := strings.TrimSpace(originalLine)

		if err := state.processLine(originalLine, trimmedLine); err != nil {
			return nil, err
		}
	}

	state.finalizeLastResponse()

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf(
			"error reading expected response file %s (last processed line %d): %w",
			filePath, state.lineNumber, err)
	}

	return state.expectedResponses, nil
}

// processLine processes a single line during expected response parsing
func (s *responseParserState) processLine(originalLine, trimmedLine string) error {
	if s.isRequestSeparator(trimmedLine) {
		s.handleRequestSeparator()
		return nil
	}

	if s.isComment(trimmedLine) {
		return nil
	}

	return s.processContentLine(originalLine, trimmedLine)
}

// isRequestSeparator checks if the line is a request separator
func (*responseParserState) isRequestSeparator(trimmedLine string) bool {
	return strings.HasPrefix(trimmedLine, requestSeparator)
}

// isComment checks if the line is a comment
func (*responseParserState) isComment(trimmedLine string) bool {
	return strings.HasPrefix(trimmedLine, commentPrefix) || strings.HasPrefix(trimmedLine, "@")
}

// handleRequestSeparator processes request separator lines
func (s *responseParserState) handleRequestSeparator() {
	s.processedAnyLine = true
	
	if s.hasResponseContent() {
		s.finalizeCurrentResponse()
	}
	
	s.resetForNewResponse()
}

// hasResponseContent checks if current response has any content
func (s *responseParserState) hasResponseContent() bool {
	return (s.currentExpectedResponse.Status != nil && *s.currentExpectedResponse.Status != "") ||
		s.currentExpectedResponse.StatusCode != nil ||
		len(s.currentExpectedResponse.Headers) > 0 ||
		len(s.bodyLines) > 0
}

// finalizeCurrentResponse adds the current response to the list
func (s *responseParserState) finalizeCurrentResponse() {
	bodyStr := strings.Join(s.bodyLines, "\n")
	s.currentExpectedResponse.Body = &bodyStr
	s.expectedResponses = append(s.expectedResponses, s.currentExpectedResponse)
}

// resetForNewResponse resets state for parsing a new response
func (s *responseParserState) resetForNewResponse() {
	s.currentExpectedResponse = &ExpectedResponse{Headers: make(http.Header)}
	s.bodyLines = []string{}
	s.parsingBody = false
}

// processContentLine processes non-comment, non-separator lines
func (s *responseParserState) processContentLine(originalLine, trimmedLine string) error {
	s.processedAnyLine = true

	if s.shouldStartBodyParsing(trimmedLine) {
		s.parsingBody = true
		return nil
	}

	if s.parsingBody {
		s.bodyLines = append(s.bodyLines, originalLine)
		return nil
	}

	// Skip empty lines when we haven't parsed a status line yet
	if trimmedLine == "" && (s.currentExpectedResponse.Status == nil || *s.currentExpectedResponse.Status == "") && s.currentExpectedResponse.StatusCode == nil {
		return nil
	}

	return processExpectedStatusOrHeaderLine(trimmedLine, s.lineNumber, s.currentExpectedResponse)
}

// shouldStartBodyParsing determines if we should start parsing the body
func (s *responseParserState) shouldStartBodyParsing(trimmedLine string) bool {
	return trimmedLine == "" && !s.parsingBody &&
		((s.currentExpectedResponse.Status != nil && *s.currentExpectedResponse.Status != "") ||
			s.currentExpectedResponse.StatusCode != nil)
}

// finalizeLastResponse handles the final response at end of parsing
func (s *responseParserState) finalizeLastResponse() {
	if s.processedAnyLine && s.hasResponseContent() {
		s.finalizeCurrentResponse()
	}
}
