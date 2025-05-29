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
		originalLine := scanner.Text()
		trimmedLine := strings.TrimSpace(originalLine)

		// 1. Check for Request Separator (###)
		// If a line starts with ###, treat any string after ### on that line as a comment.
		// Finalize previous request, start new one. No name from this line.
		if strings.HasPrefix(trimmedLine, requestSeparator) {
			if currentRequest != nil && (currentRequest.Method != "" || len(bodyLines) > 0) {
				currentRequest.RawBody = strings.Join(bodyLines, "\n")
				currentRequest.RawBody = strings.TrimRight(currentRequest.RawBody, " \t\n")
				currentRequest.Body = strings.NewReader(currentRequest.RawBody)
				currentRequest.ActiveVariables = make(map[string]string)
				for k, v := range currentFileVariables {
					currentRequest.ActiveVariables[k] = v
				}
				parsedFile.Requests = append(parsedFile.Requests, currentRequest)
			}
			bodyLines = []string{}
			parsingBody = false
			currentRequest = &Request{Headers: make(http.Header), FilePath: filePath, LineNumber: lineNumber}
			// Request name is NOT extracted from the separator line itself as per REQ-LIB-027.
			continue // Move to next line
		}

		// 2. Check for Variable Definitions (@name = value)
		// Must be checked before general comments, as variable lines are not comments.
		if strings.HasPrefix(trimmedLine, "@") {
			parts := strings.SplitN(trimmedLine[1:], "=", 2)
			if len(parts) == 2 {
				varName := strings.TrimSpace(parts[0])
				varValue := strings.TrimSpace(parts[1])
				if varName != "" {
					currentFileVariables[varName] = varValue
				}
			}
			continue // Variable definition line, skip further processing
		}

		// 3. Check for Regular Comments (#)
		// If a line starts with # (but not ###, already handled), it's a comment.
		if strings.HasPrefix(trimmedLine, commentPrefix) { // commentPrefix is "#"
			continue // Move to next line
		}

		// If none of the above, then it's part of the request (method/URL, header, or body line)
		processedLine := trimmedLine // For method/header parsing, body uses originalLine

		// Ensure currentRequest is initialized if this is the first meaningful line.
		if currentRequest == nil {
			currentRequest = &Request{Headers: make(http.Header), FilePath: filePath, LineNumber: lineNumber}
		}

		if processedLine == "" && !parsingBody {
			// This signifies the end of headers and start of the body,
			// or it's just an empty line between headers (which is fine).
			// It only transitions to parsingBody if a method has been identified.
			if currentRequest != nil && currentRequest.Method != "" {
				parsingBody = true
			}
			continue
		}

		if parsingBody {
			bodyLines = append(bodyLines, originalLine) // Use original line for body
		} else {
			// Parsing request line (METHOD URL [VERSION]) or headers (Key: Value)
			if currentRequest.Method == "" { // First non-comment, non-empty, non-variable line is the request line
				parts := strings.SplitN(processedLine, " ", 2)
				if len(parts) >= 2 { // Need at least METHOD and URL
					method := strings.ToUpper(strings.TrimSpace(parts[0]))
					urlAndVersionStr := strings.TrimSpace(parts[1])
					finalizeAfterThisLine := false

					// Handle same-line ### comments for request line
					if sepIndex := strings.Index(urlAndVersionStr, requestSeparator); sepIndex != -1 {
						urlAndVersionStr = strings.TrimSpace(urlAndVersionStr[:sepIndex])
						finalizeAfterThisLine = true
					}

					var urlStr string
					var httpVersion string

					lastSpaceIdx := strings.LastIndex(urlAndVersionStr, " ")
					if lastSpaceIdx != -1 {
						potentialVersion := strings.TrimSpace(urlAndVersionStr[lastSpaceIdx+1:])
						if strings.HasPrefix(strings.ToUpper(potentialVersion), "HTTP/") {
							httpVersion = potentialVersion
							urlStr = strings.TrimSpace(urlAndVersionStr[:lastSpaceIdx])
						} else {
							urlStr = urlAndVersionStr
						}
					} else {
						urlStr = urlAndVersionStr
					}

					currentRequest.Method = method
					currentRequest.RawURLString = urlStr
					currentRequest.HTTPVersion = httpVersion
					// Best-effort initial parse. This URL might contain variables.
					parsedURL, _ := url.Parse(urlStr) // Best effort, ignore error here
					currentRequest.URL = parsedURL

					if finalizeAfterThisLine {
						// This request is now complete due to same-line ### separator comment
						currentRequest.RawBody = strings.Join(bodyLines, "\n") // Should be empty here
						currentRequest.RawBody = strings.TrimRight(currentRequest.RawBody, " \t\n")
						currentRequest.Body = strings.NewReader(currentRequest.RawBody)
						currentRequest.ActiveVariables = make(map[string]string)
						for k, v := range currentFileVariables {
							currentRequest.ActiveVariables[k] = v
						}
						parsedFile.Requests = append(parsedFile.Requests, currentRequest)

						bodyLines = []string{}
						parsingBody = false
						// Prepare for the next line to be a new request
						currentRequest = &Request{Headers: make(http.Header), FilePath: filePath} // LineNumber will be set at the start of the next relevant line processing
						continue
					}
				}
			} else { // Parsing headers
				parts := strings.SplitN(processedLine, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					currentRequest.Headers.Add(key, value)
				}
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

	// REQ-LIB-028: Ignore any block between separators that doesn't have a request.
	// This implies that if a file results in zero valid requests after parsing (e.g., it only contained
	// comments, separators, or variable definitions but no actual request data), returning an empty
	// list of requests is acceptable and not an error, unless the file path itself was an issue (handled by os.Open).
	// So, we no longer error out if parsedFile.Requests is empty, allowing for files that correctly parse to zero requests.

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
		trimmedLine := strings.TrimSpace(originalLine)

		// 1. Check for Response Separator (###)
		// If a line starts with ###, treat any string after ### on that line as a comment.
		// Finalize previous response, start new one.
		if strings.HasPrefix(trimmedLine, requestSeparator) {
			processedAnyLine = true // Mark that we've processed a significant line
			// Current response (before separator) is complete. Add it if it has content.
			if (currentExpectedResponse.Status != nil && *currentExpectedResponse.Status != "") ||
				currentExpectedResponse.StatusCode != nil ||
				len(currentExpectedResponse.Headers) > 0 ||
				len(bodyLines) > 0 {
				bodyStr := strings.Join(bodyLines, "\n") // Corrected to \n
				currentExpectedResponse.Body = &bodyStr
				expectedResponses = append(expectedResponses, currentExpectedResponse)
			}
			// Reset for the new response that starts *after* this separator.
			currentExpectedResponse = &ExpectedResponse{Headers: make(http.Header)}
			bodyLines = []string{}
			parsingBody = false
			continue // Consumed separator line (and its comment), move to next line.
		}

		// 2. Check for Regular Comments (#)
		// If a line starts with # (but not ###, already handled), it's a comment.
		if strings.HasPrefix(trimmedLine, commentPrefix) { // commentPrefix is "#"
			// If it's a comment within the body (e.g. # within a JSON string), it should be kept.
			// The original logic for comments in bodies was:
			// if parsingBody && strings.TrimSpace(strings.TrimPrefix(processedLine, commentPrefix)) == ""
			// This is too specific (only blank comments). Let's simplify: if parsingBody, comments are part of body.
			if parsingBody {
				// Body lines are added with originalLine later, so this comment will be included if it's part of body text.
				// No explicit action needed here if we let body parsing handle it.
			} else {
				// If not parsing body, this # comment is a structural comment, so skip the line.
				continue
			}
			// If we reached here, it means it was a comment in the body. Let it fall through to body processing.
		}

		// If we are here, the line is not a separator and not a non-body comment.
		processedAnyLine = true      // Mark that we are processing a potentially contentful line.
		processedLine := trimmedLine // For status/header parsing, body uses originalLine

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

	// REQ-LIB-028: Ignore any block between separators that doesn't have a response.
	// If parsing completed without error but yielded no responses (e.g., file only contained comments or empty blocks),
	// return an empty list, not an error. This aligns with SCENARIO-LIB-028-006.

	return expectedResponses, nil
}

// parseInt is a helper function to convert string to int.
func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscan(s, &n)
	return n, err
}
