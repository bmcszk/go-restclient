package restclient

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	requestSeparator = "###"
	commentPrefix    = "#"
)

// ParseRequestFile reads a .rest or .http file and parses it into a ParsedFile struct
// containing one or more Request definitions.
func ParseRequestFile(filePath string) (*ParsedFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open request file %s: %w", filePath, err)
	}
	defer func() { _ = file.Close() }() // Correctly ignore error for defer file.Close()

	return parseRequests(file, filePath)
}

func parseRequests(reader io.Reader, filePath string) (*ParsedFile, error) {
	scanner := bufio.NewScanner(reader)
	parsedFile := &ParsedFile{
		FilePath: filePath,
		Requests: []*Request{},
	}

	var currentRequest *Request
	var bodyLines []string
	parsingBody := false
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		originalLine := scanner.Text() // Keep original line for body
		processedLine := strings.TrimSpace(originalLine)

		if strings.HasPrefix(processedLine, commentPrefix) {
			// Handle special comments like request name, e.g., "### My Request Name"
			if strings.HasPrefix(processedLine, requestSeparator) {
				if currentRequest != nil && (currentRequest.Method != "" || len(bodyLines) > 0) {
					currentRequest.RawBody = strings.Join(bodyLines, "\n")
					currentRequest.RawBody = strings.TrimRight(currentRequest.RawBody, " \t\n")
					currentRequest.Body = strings.NewReader(currentRequest.RawBody)
					parsedFile.Requests = append(parsedFile.Requests, currentRequest)
				}
				bodyLines = []string{}
				parsingBody = false
				currentRequest = &Request{Headers: make(http.Header), FilePath: filePath, LineNumber: lineNumber}
				name := strings.TrimSpace(strings.TrimPrefix(processedLine, requestSeparator))
				if name != "" {
					currentRequest.Name = name
				}
			}
			continue // Skip comment lines from direct parsing as request parts
		}

		// If we are parsing the body, empty lines are significant.
		// Otherwise, an empty line signifies end of headers or separates requests.
		if processedLine == "" && !parsingBody {
			if currentRequest != nil && currentRequest.Method != "" && !parsingBody {
				parsingBody = true
			}
			continue
		}

		if currentRequest == nil {
			// This might happen if the file doesn't start with ### or has content before the first separator.
			// Decide on strictness: error out, or try to find the first valid request start.
			// For now, let's assume files are well-formed or the first non-comment line is the method line.
			currentRequest = &Request{Headers: make(http.Header), FilePath: filePath, LineNumber: lineNumber}
			// No separator found yet, so no name.
		}

		if parsingBody {
			bodyLines = append(bodyLines, originalLine) // Use original line for body
		} else {
			// Parsing request line or headers
			if currentRequest.Method == "" { // First non-comment, non-empty line is the request line
				parts := strings.Fields(processedLine)
				if len(parts) < 2 { // Must have at least METHOD URL
					return nil, fmt.Errorf("line %d: invalid request line: %s. Expected METHOD URL [HTTP_VERSION]", lineNumber, processedLine)
				}
				currentRequest.Method = strings.ToUpper(parts[0])
				parsedURL, err := url.Parse(parts[1])
				if err != nil {
					return nil, fmt.Errorf("line %d: invalid URL %s: %w", lineNumber, parts[1], err)
				}
				currentRequest.URL = parsedURL
				if len(parts) > 2 {
					currentRequest.HTTPVersion = parts[2]
				} else {
					currentRequest.HTTPVersion = "HTTP/1.1" // Default
				}
			} else { // Parsing headers
				parts := strings.SplitN(processedLine, ":", 2)
				if len(parts) != 2 {
					return nil, fmt.Errorf("line %d: invalid header line: %s. Expected 'Key: Value'", lineNumber, processedLine)
				}
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				currentRequest.Headers.Add(key, value)
			}
		}
	}

	// Add the last request if any
	if currentRequest != nil && (currentRequest.Method != "" || len(bodyLines) > 0) {
		rawJoinedBody := strings.Join(bodyLines, "\n")
		currentRequest.RawBody = strings.TrimRight(rawJoinedBody, " \t\n")
		currentRequest.Body = strings.NewReader(currentRequest.RawBody)
		parsedFile.Requests = append(parsedFile.Requests, currentRequest)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading request file %s: %w", filePath, err)
	}

	if len(parsedFile.Requests) == 0 && filePath != "" { // filePath check to avoid error on empty reader in tests
		return nil, fmt.Errorf("no valid requests found in file %s", filePath)
	}

	return parsedFile, nil
}

// ParseExpectedResponseFile reads a file and parses it into a slice of ExpectedResponse definitions.
func ParseExpectedResponseFile(filePath string) ([]*ExpectedResponse, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open expected response file %s: %w", filePath, err)
	}
	defer func() { _ = file.Close() }()

	return parseExpectedResponses(file, filePath)
}

func parseExpectedResponses(reader io.Reader, filePath string) ([]*ExpectedResponse, error) {
	scanner := bufio.NewScanner(reader)
	var expectedResponses []*ExpectedResponse

	currentExpectedResponse := &ExpectedResponse{Headers: make(http.Header)}
	var bodyLines []string
	parsingBody := false
	lineNumber := 0
	processedAnyLine := false

	for scanner.Scan() {
		lineNumber++
		originalLine := scanner.Text()
		processedLine := strings.TrimSpace(originalLine)

		if processedLine == requestSeparator { // Exact match for "###"
			processedAnyLine = true // Mark that we processed the separator itself
			// Current response (before separator) is complete. Add it if it has content.
			if (currentExpectedResponse.Status != nil && *currentExpectedResponse.Status != "") ||
				currentExpectedResponse.StatusCode != nil ||
				len(currentExpectedResponse.Headers) > 0 || // Also check headers for empty responses
				len(bodyLines) > 0 {
				bodyStr := strings.Join(bodyLines, "\n")
				currentExpectedResponse.Body = &bodyStr
				expectedResponses = append(expectedResponses, currentExpectedResponse)
			}
			// Reset for the new response that starts *after* this separator.
			currentExpectedResponse = &ExpectedResponse{Headers: make(http.Header)}
			bodyLines = []string{}
			parsingBody = false
			continue
		}

		processedAnyLine = true // Mark that we are processing content (if not separator)

		if strings.HasPrefix(processedLine, commentPrefix) {
			// If it's just a comment line like "#" with nothing else, or only whitespace after #,
			// and we're in parsingBody mode, it should be part of the body.
			if parsingBody && strings.TrimSpace(strings.TrimPrefix(processedLine, commentPrefix)) == "" {
				bodyLines = append(bodyLines, originalLine) // Add original line if it's a body comment
			}
			// Otherwise, always skip normal comment lines from direct parsing as response parts
			continue
		}

		if processedLine == "" && !parsingBody {
			// This condition signifies the start of the body or an ignored blank line between responses
			if (currentExpectedResponse.Status != nil && *currentExpectedResponse.Status != "") || currentExpectedResponse.StatusCode != nil {
				parsingBody = true
			}
			continue
		}

		// No need for: if currentExpectedResponse == nil { currentExpectedResponse = &ExpectedResponse{Headers: make(http.Header)} }
		// because it's initialized before the loop and reset after each separator.

		if parsingBody {
			bodyLines = append(bodyLines, originalLine)
		} else {
			// Parsing status line or headers
			if (currentExpectedResponse.Status == nil || *currentExpectedResponse.Status == "") && currentExpectedResponse.StatusCode == nil { // First non-comment, non-empty line is the status line
				parts := strings.Fields(processedLine)
				if len(parts) < 2 { // Must have at least HTTP_VERSION STATUS_CODE [STATUS_TEXT]
					return nil, fmt.Errorf("line %d: invalid status line: '%s'. Expected HTTP_VERSION STATUS_CODE [STATUS_TEXT]", lineNumber, processedLine)
				}
				// parts[0] is HTTP Version, parts[1] is StatusCode, rest is StatusText
				statusCodeInt, err := parseInt(parts[1])
				if err != nil {
					return nil, fmt.Errorf("line %d: invalid status code '%s': %w", lineNumber, parts[1], err)
				}
				currentExpectedResponse.StatusCode = &statusCodeInt

				var finalStatusString string
				if len(parts) > 2 {
					finalStatusString = strings.Join(parts[1:], " ") // e.g. "200 OK"
				} else {
					finalStatusString = parts[1] // Just code, e.g. "200"
				}
				currentExpectedResponse.Status = &finalStatusString // Store the combined status code and text or just code
			} else { // Parsing headers
				parts := strings.SplitN(processedLine, ":", 2)
				if len(parts) != 2 {
					return nil, fmt.Errorf("line %d: invalid header line: '%s'. Expected 'Key: Value'", lineNumber, processedLine)
				}
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				currentExpectedResponse.Headers.Add(key, value)
			}
		}
	}

	// Add the last expected response pending in currentExpectedResponse, if it has content
	if processedAnyLine && ((currentExpectedResponse.Status != nil && *currentExpectedResponse.Status != "") ||
		currentExpectedResponse.StatusCode != nil ||
		len(currentExpectedResponse.Headers) > 0 || // Also check headers for empty responses
		len(bodyLines) > 0) {
		bodyStr := strings.Join(bodyLines, "\n")
		currentExpectedResponse.Body = &bodyStr
		expectedResponses = append(expectedResponses, currentExpectedResponse)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading expected response file %s (last processed line %d): %w", filePath, lineNumber, err)
	}

	// If no responses were parsed and it's a file (not an empty test reader)
	if len(expectedResponses) == 0 && filePath != "" {
		// If we didn't even process any non-comment lines, it's a specific kind of "not found"
		if !processedAnyLine {
			// TestParseExpectedResponses_EmptyFile expects this specific wording
			return nil, fmt.Errorf("no valid expected responses found in file %s", filePath)
		}
		// Otherwise, content was there but nothing valid was parsed from it
		return nil, fmt.Errorf("no valid expected responses found in file %s despite processing content", filePath)
	}

	return expectedResponses, nil
}

// parseInt is a helper function to convert string to int.
func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscan(s, &n)
	return n, err
}
