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
		currentRequest.RawBody = strings.Join(bodyLines, "\n")
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

	var currentExpectedResponse *ExpectedResponse
	var bodyLines []string
	parsingBody := false
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		originalLine := scanner.Text()
		processedLine := strings.TrimSpace(originalLine)

		if strings.HasPrefix(processedLine, commentPrefix) {
			continue // Skip comment lines
		}

		if strings.HasPrefix(processedLine, requestSeparator) { // Using same separator for now
			if currentExpectedResponse != nil && ((currentExpectedResponse.Status != nil && *currentExpectedResponse.Status != "") || currentExpectedResponse.StatusCode != nil || len(bodyLines) > 0) {
				bodyStr := strings.Join(bodyLines, "\n")
				currentExpectedResponse.Body = &bodyStr
				expectedResponses = append(expectedResponses, currentExpectedResponse)
			}
			bodyLines = []string{}
			parsingBody = false
			currentExpectedResponse = &ExpectedResponse{Headers: make(http.Header)}
			continue
		}

		if processedLine == "" && !parsingBody {
			if currentExpectedResponse != nil && ((currentExpectedResponse.Status != nil && *currentExpectedResponse.Status != "") || currentExpectedResponse.StatusCode != nil) && !parsingBody {
				parsingBody = true
			}
			continue
		}

		if currentExpectedResponse == nil {
			currentExpectedResponse = &ExpectedResponse{Headers: make(http.Header)}
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
				statusCodeInt, err := parseInt(parts[1])
				if err != nil {
					return nil, fmt.Errorf("line %d: invalid status code '%s': %w", lineNumber, parts[1], err)
				}
				currentExpectedResponse.StatusCode = &statusCodeInt

				var finalStatusString string
				if len(parts) > 2 { // e.g. HTTP/1.1 200 OK
					statusText := strings.Join(parts[2:], " ")      // "OK"
					finalStatusString = parts[1] + " " + statusText // "200 OK"
				} else { // e.g. HTTP/1.1 200
					statusText := http.StatusText(statusCodeInt)
					if statusText == "" {
						statusText = "Unknown Status"
					}
					finalStatusString = fmt.Sprintf("%d %s", statusCodeInt, statusText) // "200 OK"
				}
				currentExpectedResponse.Status = &finalStatusString

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

	if currentExpectedResponse != nil && ((currentExpectedResponse.Status != nil && *currentExpectedResponse.Status != "") || currentExpectedResponse.StatusCode != nil || len(bodyLines) > 0) {
		bodyStr := strings.Join(bodyLines, "\n")
		currentExpectedResponse.Body = &bodyStr
		expectedResponses = append(expectedResponses, currentExpectedResponse)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading expected response file %s: %w", filePath, err)
	}

	if len(expectedResponses) == 0 && filePath != "" { // filePath check for tests
		return nil, fmt.Errorf("no valid expected responses found in file %s", filePath)
	}
	return expectedResponses, nil
}

// parseInt is a helper, like strconv.Atoi but for this limited context.
func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscan(s, &n)
	return n, err
}
