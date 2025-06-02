package restclient

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

const (
	requestSeparator = "###"
	commentPrefix    = "#"
)

// ParseRequestFile reads a .rest or .http file and parses it into a ParsedFile struct
// containing one or more Request definitions. It requires a `client` instance to access
// programmatic and request-scoped system variables, which are used at parse time to resolve
// the values of file-level variables (e.g., `@host = {{programmatic_var}}` or `@api_key = {{$uuid}}`).
//
// The `filePath` is used for opening the file and for context in error messages.
// A .env file in the same directory as `filePath` will also be loaded and used for resolving
// `@variable` definitions if present.
func parseRequestFile(filePath string, client *Client) (*ParsedFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open request file %s: %w", filePath, err)
	}
	defer func() { _ = file.Close() }()

	// Load .env file from the same directory as the request file for @var resolution time
	dotEnvVarsForParser := make(map[string]string)
	envFilePath := filepath.Join(filepath.Dir(filePath), ".env")
	if _, statErr := os.Stat(envFilePath); statErr == nil {
		loadedVars, loadErr := godotenv.Read(envFilePath)
		if loadErr == nil {
			dotEnvVarsForParser = loadedVars
		}
	}
	osEnvGetter := func(key string) (string, bool) { return os.LookupEnv(key) }

	// Generate request-scoped system variables once for this file parsing pass
	var requestScopedSystemVarsForFileParse map[string]string
	if client != nil {
		requestScopedSystemVarsForFileParse = client.generateRequestScopedSystemVariables()
	} else {
		requestScopedSystemVarsForFileParse = make(map[string]string)
	}

	return parseRequests(file, filePath, client, requestScopedSystemVarsForFileParse, osEnvGetter, dotEnvVarsForParser)
}

// reqScopedSystemVarsForParser is generated once per file parsing pass for resolving @-vars consistently.
// var reqScopedSystemVarsForParser map[string]string // REMOVED GLOBAL

// parseRequests performs the core parsing logic for an HTTP request file (e.g., .http, .rest).
// It takes an io.Reader for the content, the original filePath for context, a client instance,
// pre-generated requestScopedSystemVars for resolving @-variables, an osEnvGetter, and dotEnvVars.
//
// Inside this function:
//   - It scans the input line by line.
//   - It identifies request separators (`###`), comments (`#`), and variable definitions (`@name = value`).
//   - When an `@variable = value` is encountered, its `value` is immediately resolved using the provided
//     `client` (for its programmatic variables), `requestScopedSystemVars`, `osEnvGetter`, and `dotEnvVars`.
//     The resolved value is stored in `currentFileVariables`.
//   - For each request parsed, a copy of `currentFileVariables` is assigned to `request.ActiveVariables`.
//     These active variables are later used by `client.ExecuteFile` when resolving placeholders in the
//     request URL, headers, and body.
//
// Returns a `ParsedFile` struct containing all parsed requests or an error if issues occur.
func parseRequests(reader io.Reader, filePath string, client *Client,
	requestScopedSystemVars map[string]string,
	osEnvGetter func(string) (string, bool),
	dotEnvVars map[string]string) (*ParsedFile, error) {
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

		// Request Separator (###)
		if strings.HasPrefix(trimmedLine, requestSeparator) {
			// Finalize the previous request, if any and if it's substantial
			if currentRequest != nil && (currentRequest.Method != "" || currentRequest.RawURLString != "" || len(bodyLines) > 0) {
				currentRequest.RawBody = strings.Join(bodyLines, "\n")
				currentRequest.RawBody = strings.TrimRight(currentRequest.RawBody, " \t\r\n") // Trim trailing whitespace/newlines
				currentRequest.Body = strings.NewReader(currentRequest.RawBody)

				// Ensure ActiveVariables is initialized and populated with current file-level variables
				if currentRequest.ActiveVariables == nil { // Should usually be pre-initialized if request was created
					currentRequest.ActiveVariables = make(map[string]string)
				}
				for k, v := range currentFileVariables {
					currentRequest.ActiveVariables[k] = v
				}
				parsedFile.Requests = append(parsedFile.Requests, currentRequest)
			}

			// Start a new request
			bodyLines = []string{}
			parsingBody = false
			requestName := strings.TrimSpace(trimmedLine[len(requestSeparator):])
			currentRequest = &Request{
				Name:            requestName,
				Headers:         make(http.Header),
				FilePath:        filePath,
				LineNumber:      lineNumber,              // Line number where ### was found
				ActiveVariables: make(map[string]string), // Initialize, will be populated before adding to parsedFile.Requests or here if needed immediately
			}
			continue
		}

		// Variable Definitions (@name = value)
		if strings.HasPrefix(trimmedLine, "@") {
			parts := strings.SplitN(trimmedLine[1:], "=", 2)
			if len(parts) == 2 {
				varName := strings.TrimSpace(parts[0])
				varValue := strings.TrimSpace(parts[1])
				if varName != "" {
					// Resolve varValue immediately
					resolvedVarValue := varValue
					if client != nil {
						// For resolving @-vars, file-scoped vars (currentFileVariables) are not used as a source for themselves.
						// EnvironmentVars and GlobalVars are passed as nil here because they are not applicable
						// during the initial parsing of @-variables. They apply during request execution.
						resolvedVarValue = client.resolveVariablesInText(varValue, client.programmaticVars, nil, nil, nil, requestScopedSystemVars, osEnvGetter, dotEnvVars)
						resolvedVarValue = client.substituteDynamicSystemVariables(resolvedVarValue)
					}
					currentFileVariables[varName] = resolvedVarValue
				}
			}
			continue // Variable definition line, skip further processing
		}

		// Comments (# or //)
		isHashComment := strings.HasPrefix(trimmedLine, commentPrefix) // commentPrefix is "#"
		isSlashComment := strings.HasPrefix(trimmedLine, "//")

		if isHashComment || isSlashComment {
			var commentContent string
			if isHashComment {
				commentContent = strings.TrimSpace(trimmedLine[len(commentPrefix):])
			} else { // isSlashComment
				commentContent = strings.TrimSpace(trimmedLine[len("//"):])
			}

			if strings.HasPrefix(commentContent, "@name ") {
				requestNameFromComment := strings.TrimSpace(commentContent[len("@name "):])
				if requestNameFromComment != "" {
					if currentRequest == nil {
						// This @name appears before any other request content (like METHOD URL or ### separator for the current block).
						// Initialize a request placeholder. LineNumber points to this @name comment.
						// If a METHOD URL line follows, it will populate this currentRequest.
						currentRequest = &Request{
							Name:            requestNameFromComment,
							Headers:         make(http.Header),
							FilePath:        filePath,
							LineNumber:      lineNumber, // Line number of the @name comment itself
							ActiveVariables: make(map[string]string),
						}
					} else {
						// @name applies to the current request, potentially overriding a name set by ###
						// or setting the name if it's the first request in a file (implicitly defined).
						currentRequest.Name = requestNameFromComment
					}
				}
			}
			// After processing a potential @name directive, or if it's just a regular comment,
			// always skip the comment line for other parsing rules (like METHOD/URL, header, body).
			continue
		}

		// If none of the above, then it's part of the request (method/URL, header, or body line)
		processedLine := trimmedLine

		if currentRequest == nil { // This is the first request in the file or after initial comments/vars
			currentRequest = &Request{
				Headers:         make(http.Header),
				FilePath:        filePath,
				LineNumber:      lineNumber,              // Line number of the first significant line of this request
				ActiveVariables: make(map[string]string), // Initialize, will be populated before adding
			}
			// Name for the first request (if not starting with ###) will be empty by default.
			// It can be set by `// @name RequestName` (handled in Task T2).
		}

		if processedLine == "" && !parsingBody {
			// Empty line: signifies end of headers and start of body,
			// or just an empty line between headers.
			if currentRequest != nil && currentRequest.Method != "" {
				parsingBody = true
			}
			continue
		}

		if parsingBody {
			bodyLines = append(bodyLines, originalLine)
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
					parsedURL, _ := url.Parse(urlStr)
					currentRequest.URL = parsedURL

					if finalizeAfterThisLine {
						// Request complete due to same-line ### separator comment
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

	// After the loop, add the last pending request, if any and if it's substantial
	if currentRequest != nil && (currentRequest.Method != "" || currentRequest.RawURLString != "" || len(bodyLines) > 0) {
		currentRequest.RawBody = strings.Join(bodyLines, "\n")
		currentRequest.RawBody = strings.TrimRight(currentRequest.RawBody, " \t\r\n") // Trim trailing whitespace/newlines
		currentRequest.Body = strings.NewReader(currentRequest.RawBody)

		// Ensure ActiveVariables is initialized and populated with current file-level variables
		if currentRequest.ActiveVariables == nil {
			currentRequest.ActiveVariables = make(map[string]string)
		}
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
//
// DEPRECATED: This function is deprecated and will be removed or made internal in a future version.
// It does not support variable substitution. Users should migrate to using `ValidateResponses` from the
// `validator.go` file, which handles .hresp file parsing, variable extraction, and substitution
// before validation. For direct parsing of already substituted .hresp content, one can read the file
// into an `io.Reader` and use `parseExpectedResponses` directly.
func parseExpectedResponseFile(filePath string) ([]*ExpectedResponse, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open expected response file %s: %w", filePath, err)
	}
	defer func() { _ = file.Close() }() // Correctly ignore error for defer file.Close()

	return parseExpectedResponses(file, filePath) // filePath is used for error reporting within parseExpectedResponses
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

		// Request Separator (###)
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

		// Regular Comments (#)
		if strings.HasPrefix(trimmedLine, commentPrefix) || strings.HasPrefix(trimmedLine, "@") { // commentPrefix is "#"
			continue // Move to next line
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
