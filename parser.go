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
func parseRequestFile(filePath string) (*ParsedFile, error) {
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
	currentFileVariables := make(map[string]string) // Variables accumulated in the file scope

	var currentRequest *Request
	var bodyLines []string
	parsingBody := false
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		originalLine := scanner.Text() // Keep original line for body
		processedLine := strings.TrimSpace(originalLine)

		// Handle variable definitions like @name = value
		if strings.HasPrefix(processedLine, "@") {
			parts := strings.SplitN(processedLine[1:], "=", 2)
			if len(parts) == 2 {
				varName := strings.TrimSpace(parts[0])
				varValue := strings.TrimSpace(parts[1])
				if varName != "" {
					currentFileVariables[varName] = varValue
				}
			}
			continue // Variable definition line, skip further processing for this line
		}

		if strings.HasPrefix(processedLine, commentPrefix) {
			// Handle special comments like request name, e.g., "### My Request Name"
			if strings.HasPrefix(processedLine, requestSeparator) {
				if currentRequest != nil && (currentRequest.Method != "" || len(bodyLines) > 0) {
					currentRequest.RawBody = strings.Join(bodyLines, "\n")
					currentRequest.RawBody = strings.TrimRight(currentRequest.RawBody, " \t\n")
					currentRequest.Body = strings.NewReader(currentRequest.RawBody)
					// Assign a copy of the current file variables to this request
					currentRequest.ActiveVariables = make(map[string]string)
					for k, v := range currentFileVariables {
						currentRequest.ActiveVariables[k] = v
					}
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
		}

		if parsingBody {
			bodyLines = append(bodyLines, originalLine) // Use original line for body
		} else {
			// Parsing request line or headers
			if currentRequest.Method == "" { // First non-comment, non-empty line is the request line
				// Attempt to parse as a request line (METHOD URL [HTTP_VERSION])
				// Correctly parse the request line: first word is method, the rest is URL + version
				parts := strings.SplitN(processedLine, " ", 2) // Split into method and the rest
				if len(parts) == 2 {
					method := strings.ToUpper(strings.TrimSpace(parts[0]))
					urlAndVersionStr := strings.TrimSpace(parts[1])

					var urlStr string
					var httpVersion string

					// Check if the last part of urlAndVersionStr is an HTTP version
					lastSpaceIdx := strings.LastIndex(urlAndVersionStr, " ")
					if lastSpaceIdx != -1 {
						potentialVersion := strings.TrimSpace(urlAndVersionStr[lastSpaceIdx+1:])
						if strings.HasPrefix(strings.ToUpper(potentialVersion), "HTTP/") {
							httpVersion = potentialVersion
							urlStr = strings.TrimSpace(urlAndVersionStr[:lastSpaceIdx])
						} else {
							// No HTTP version found at the end after a space
							urlStr = urlAndVersionStr
						}
					} else {
						// No spaces in urlAndVersionStr, so the whole thing is the URL
						urlStr = urlAndVersionStr
					}

					currentRequest.Method = method
					currentRequest.RawURLString = urlStr // Store the raw URL string before any parsing
					currentRequest.HTTPVersion = httpVersion

					// Best-effort initial parse. This URL might contain variables.
					parsedURL, _ := url.Parse(urlStr) // Best effort, ignore error here
					currentRequest.URL = parsedURL
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
		// Assign a copy of the current file variables to this last request
		currentRequest.ActiveVariables = make(map[string]string)
		for k, v := range currentFileVariables {
			currentRequest.ActiveVariables[k] = v
		}
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
func parseExpectedResponseFile(filePath string) ([]*ExpectedResponse, error) {
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

		if processedLine == requestSeparator {
			processedAnyLine = true
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

		processedAnyLine = true

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
