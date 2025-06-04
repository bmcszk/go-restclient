package restclient

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	requestSeparator = "###"
	commentPrefix    = "#"
)

// loadEnvironmentFile attempts to load a specific environment's variables from a JSON file.
// It returns the variables map or nil if the environment/file is not found or on error.
func loadEnvironmentFile(filePath string, selectedEnvName string) (map[string]string, error) {
	if selectedEnvName == "" {
		return nil, nil // No environment selected, nothing to load
	}

	if _, statErr := os.Stat(filePath); statErr != nil {
		if os.IsNotExist(statErr) {
			slog.Debug("Environment file not found.", "file", filePath)
			return nil, nil // File not found is not an error, just means no vars from this file
		}
		// Another error occurred trying to stat the file (e.g., permissions)
		slog.Warn("Error checking environment file", "error", statErr, "file", filePath)
		return nil, fmt.Errorf("checking environment file %s: %w", filePath, statErr)
	}

	envFileBytes, readErr := os.ReadFile(filePath)
	if readErr != nil {
		slog.Warn("Failed to read environment file", "error", readErr, "file", filePath)
		return nil, fmt.Errorf("reading environment file %s: %w", filePath, readErr)
	}

	var allEnvs map[string]map[string]string
	if unmarshalErr := json.Unmarshal(envFileBytes, &allEnvs); unmarshalErr != nil {
		slog.Warn("Failed to unmarshal environment file", "error", unmarshalErr, "file", filePath)
		return nil, fmt.Errorf("unmarshalling environment file %s: %w", filePath, unmarshalErr)
	}

	if selectedEnvVars, ok := allEnvs[selectedEnvName]; ok {
		return selectedEnvVars, nil
	}

	slog.Debug("Selected environment not found in environment file", "environment", selectedEnvName, "file", filePath)
	return nil, nil // Environment not found in this file
}

// ParseRequestFile reads a .rest or .http file and parses it into a ParsedFile struct
// containing one or more Request definitions. It requires a `client` instance to access
// programmatic and request-scoped system variables, which are used at parse time to resolve
// the values of file-level variables (e.g., `@host = {{programmatic_var}}` or `@api_key = {{$uuid}}`).
//
// The `filePath` is used for opening the file and for context in error messages.
// A .env file in the same directory as `filePath` will also be loaded and used for resolving
// `@variable` definitions if present.
func ParseRequestFile(filePath string, client *Client, importStack []string) (*ParsedFile, error) {
	content, err := parseFileContent(filePath)
	if err != nil {
		// For the initial call, parseFileContent's error is fine.
		// For recursive calls, this error will be wrapped by the caller with import context.
		return nil, err
	}

	// Prepare arguments for parseRequests
	reader := strings.NewReader(content)
	// client.generateRequestScopedSystemVariables() is available based on client.go
	var requestScopedSystemVars map[string]string
	if client != nil {
		requestScopedSystemVars = client.generateRequestScopedSystemVariables()
	} else {
		requestScopedSystemVars = make(map[string]string) // Provide empty map if client is nil
	}
	// osEnvGetter is a standard library function
	osEnvGetter := os.LookupEnv
	dotEnvVars := make(map[string]string)
	// Load .env file specific to the current filePath directory
	localEnvFilePath := filepath.Join(filepath.Dir(filePath), ".env")
	if _, err := os.Stat(localEnvFilePath); err == nil {
		loadedDotEnvVars, loadErr := godotenv.Read(localEnvFilePath)
		if loadErr == nil {
			for k, v := range loadedDotEnvVars {
				dotEnvVars[k] = v
			}
		} else {
			slog.Warn("Error loading .env file", "file", localEnvFilePath, "error", loadErr)
		}
	}

	// If a client is provided and an environment is selected, load and merge environment-specific JSON vars
	if client != nil && client.selectedEnvironmentName != "" {
		// Load public env vars (http-client.env.json)
		publicEnvJsonFilePath := filepath.Join(filepath.Dir(filePath), "http-client.env.json")
		publicEnvSpecificVars, publicEnvLoadErr := loadEnvironmentFile(publicEnvJsonFilePath, client.selectedEnvironmentName)
		if publicEnvLoadErr != nil {
			slog.Warn("Error loading public environment-specific variables from JSON", "file", publicEnvJsonFilePath, "environment", client.selectedEnvironmentName, "error", publicEnvLoadErr)
		} else if publicEnvSpecificVars != nil {
			for k, v := range publicEnvSpecificVars {
				dotEnvVars[k] = v // Override or add, public JSON vars take precedence over .env
			}
			slog.Debug("Successfully loaded and merged public environment-specific variables from JSON", "file", publicEnvJsonFilePath, "environment", client.selectedEnvironmentName)
		}

		// Load private env vars (http-client.private.env.json)
		// These will override public and .env variables
		privateEnvJsonFilePath := filepath.Join(filepath.Dir(filePath), "http-client.private.env.json")
		privateEnvSpecificVars, privateEnvLoadErr := loadEnvironmentFile(privateEnvJsonFilePath, client.selectedEnvironmentName)
		if privateEnvLoadErr != nil {
			slog.Warn("Error loading private environment-specific variables from JSON", "file", privateEnvJsonFilePath, "environment", client.selectedEnvironmentName, "error", privateEnvLoadErr)
		} else if privateEnvSpecificVars != nil {
			for k, v := range privateEnvSpecificVars {
				dotEnvVars[k] = v // Override or add, private JSON vars take precedence over public JSON and .env
			}
			slog.Debug("Successfully loaded and merged private environment-specific variables from JSON", "file", privateEnvJsonFilePath, "environment", client.selectedEnvironmentName)
		}
	}

	currentParsedFile, err := ParseRequests(reader, filePath, client, requestScopedSystemVars, osEnvGetter, dotEnvVars, importStack)
	if err != nil {
		return nil, fmt.Errorf("error parsing initial requests from %s: %w", filePath, err)
	}
	// currentParsedFile is guaranteed non-nil if err is nil by ParseRequests contract.
	currentParsedFile.EffectiveEnvironmentVariables = dotEnvVars // Store the merged env vars

	// Store direct imports to return, as tests expect only direct imports in the final ParsedFile.ImportedFiles field.
	directImports := make([]string, len(currentParsedFile.ImportedFiles))
	copy(directImports, currentParsedFile.ImportedFiles)

	mergedRequests := []*Request{}
	finalMergedFileVariables := make(map[string]string)

	// Process imports first to establish base variables and requests
	// Variables from later imports override earlier ones.
	for _, importedRawPath := range currentParsedFile.ImportedFiles { // Iterate over the direct imports from current file
		resolvedImportPath := importedRawPath
		if !filepath.IsAbs(importedRawPath) {
			resolvedImportPath = filepath.Join(filepath.Dir(filePath), importedRawPath)
		}

		// Check for circular imports
		isCircular := false
		for _, stackedPath := range importStack {
			if stackedPath == resolvedImportPath {
				isCircular = true
				break
			}
		}
		if isCircular {
			return nil, fmt.Errorf("circular import detected: %s trying to import %s (stack: %v)", filePath, resolvedImportPath, importStack)
		}

		newImportStack := append(importStack, filePath)                                         // Add current file to stack before diving deeper
		importedParsedFile, err := ParseRequestFile(resolvedImportPath, client, newImportStack) // Pass client recursively
		if err != nil {
			// Check if the error is due to file not found
			if os.IsNotExist(errors.Unwrap(err)) { // errors.Unwrap requires 'errors' package
				return nil, fmt.Errorf("imported file not found: %s (imported by %s): %w", resolvedImportPath, filePath, err)
			}
			return nil, fmt.Errorf("error parsing imported file %s (imported by %s): %w", resolvedImportPath, filePath, err)
		}

		// Merge Requests (imported requests come first)
		mergedRequests = append(mergedRequests, importedParsedFile.Requests...)

		// Merge file variables from the imported file.
		// Variables from this `importedParsedFile` will override those from previous imports in the same loop (earlier directives).
		for k, v := range importedParsedFile.FileVariables {
			finalMergedFileVariables[k] = v
		}
		// Note: currentParsedFile.ImportedFiles should retain only direct imports of *this* file.
		// The recursive calls manage their own nested imports internally.
	}

	// Add requests from the current file (they come after all imported requests)
	mergedRequests = append(mergedRequests, currentParsedFile.Requests...)
	currentParsedFile.Requests = mergedRequests

	// Merge file variables from the current file, overriding any imported ones
	for k, v := range currentParsedFile.FileVariables { // These are variables defined directly in filePath
		finalMergedFileVariables[k] = v
	}
	currentParsedFile.FileVariables = finalMergedFileVariables

	// Restore direct imports for the final ParsedFile object, as per test expectations.
	currentParsedFile.ImportedFiles = directImports

	return currentParsedFile, nil
}

func parseFileContent(filePath string) (string, error) {
	// Ensure filePath is absolute for consistent error reporting and file operations.
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for %s: %w", filePath, err)
	}

	contentBytes, err := os.ReadFile(absFilePath) // Use os.ReadFile for simplicity
	if err != nil {
		// Return the error directly. The caller (parseRequestFile) will handle os.IsNotExist if needed.
		// This also makes parseFileContent more general, as it doesn't need to know about import stacks.
		return "", fmt.Errorf("failed to read file %s: %w", absFilePath, err)
	}
	return string(contentBytes), nil
}

// reqScopedSystemVarsForParser is generated once per file parsing pass for resolving @-vars consistently.
// var reqScopedSystemVarsForParser map[string]string // REMOVED GLOBAL

// ParseRequests performs the core parsing logic for an HTTP request file (e.g., .http, .rest).
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
func ParseRequests(reader io.Reader, filePath string, client *Client,
	requestScopedSystemVars map[string]string,
	osEnvGetter func(string) (string, bool),
	dotEnvVars map[string]string,
	importStack []string) (*ParsedFile, error) {
	scanner := bufio.NewScanner(reader)
	parsedFile := &ParsedFile{
		FilePath:      filePath,
		Requests:      []*Request{},
		FileVariables: make(map[string]string), // Initialize FileVariables
	}
	currentFileVariables := make(map[string]string) // Variables accumulated in the file scope
	justSawEmptyLineSeparator := false              // Flag to indicate the previous line was an empty separator

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
				currentRequest.Body = strings.NewReader(currentRequest.RawBody)               // For single read if needed directly by parser consumers
				// GetBody allows the body to be read multiple times, as required by http.Request
				rawBodyCopy := currentRequest.RawBody // Capture rawBody for the closure
				currentRequest.GetBody = func() (io.ReadCloser, error) {
					return io.NopCloser(strings.NewReader(rawBodyCopy)), nil
				}

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
			justSawEmptyLineSeparator = false // Reset flag when a new request block starts
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

		// Import Directives (e.g., // @import "path/to/file.http" or # @import "path/to/file.http")
		// This block only identifies the import directive and records the path.
		// The actual parsing and merging of imported files is handled by the ParseRequestFile function's loop.
		if strings.HasPrefix(trimmedLine, "// @import") || strings.HasPrefix(trimmedLine, "# @import") {
			importPrefix := ""
			if strings.HasPrefix(trimmedLine, "// @import") {
				importPrefix = "// @import"
			} else { // Must be # @import
				importPrefix = "# @import"
			}

			// Ensure there's a space after the directive before the quoted path
			if len(trimmedLine) > len(importPrefix) && trimmedLine[len(importPrefix)] == ' ' {
				pathPart := strings.TrimSpace(trimmedLine[len(importPrefix)+1:]) // +1 for the space
				if len(pathPart) > 1 && pathPart[0] == '"' && pathPart[len(pathPart)-1] == '"' {
					importedFilePath := pathPart[1 : len(pathPart)-1]
					if importedFilePath != "" {
						parsedFile.ImportedFiles = append(parsedFile.ImportedFiles, importedFilePath)
						slog.Debug("Identified import directive", "path", importedFilePath, "line", lineNumber, "file", filePath)
					} else {
						slog.Warn("Empty path in import directive", "line", lineNumber, "content", originalLine, "filePath", filePath)
					}
				} else {
					slog.Warn("Malformed import directive: path not correctly quoted", "line", lineNumber, "content", originalLine, "filePath", filePath)
				}
			} else {
				slog.Warn("Malformed import directive: missing space after directive or path", "line", lineNumber, "content", originalLine, "filePath", filePath)
			}
			continue // Skip further processing for this line
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
						resolvedVarValue = client.substituteDynamicSystemVariables(resolvedVarValue, dotEnvVars)
					}
					currentFileVariables[varName] = resolvedVarValue
				} else {
					return nil, fmt.Errorf("line %d: variable name cannot be empty in definition: %s", lineNumber, originalLine)
				}
			} else {
				// This means no '=' was found, or it was the first character after '@'
				return nil, fmt.Errorf("line %d: malformed in-place variable definition, missing '=' or name: %s", lineNumber, originalLine)
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
			// A comment line is not an empty separator, so reset the flag.
			justSawEmptyLineSeparator = false
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

		// "[DEBUG_PARSER_LINE_STATE]",
		// 	"lineNumber", lineNumber,
		// 	"trimmedLine", trimmedLine,
		// 	"processedLine", processedLine,
		// 	"currentRequestMethod", currentRequest.Method,
		// 	"parsingBody", parsingBody,
		// 	"numHeaders", len(currentRequest.Headers),
		// 	"filePath", filePath,
		// )

		// Handle empty lines explicitly first
		if processedLine == "" {
			if currentRequest.Method != "" && !parsingBody {
				// This empty line *could* be the separator before the body,
				// or just an empty line between headers, or between request-line and first header.
				// "[DEBUG_PARSER_ENCOUNTERED_EMPTY_LINE_POTENTIAL_SEPARATOR]", "lineNumber", lineNumber)
				justSawEmptyLineSeparator = true // Set flag: next non-empty line should be body
			} else if parsingBody {
				// Empty line while already parsing body, append to body
				bodyLines = append(bodyLines, originalLine) // Append original line to preserve formatting
			}
			// In all cases of an empty line, we continue to the next line.
			continue
		}

		// If we reach here, processedLine is NOT empty.

		// If the *previous* line was an empty line acting as a potential separator,
		// then this current non-empty line MUST be the start of the body.
		if justSawEmptyLineSeparator {
			// "[DEBUG_PARSER_IMMEDIATE_BODY_TRANSITION_DUE_TO_FLAG]", "lineNumber", lineNumber, "processedLine", processedLine)
			parsingBody = true
			bodyLines = append(bodyLines, originalLine) // Add this line to body as it's the first body line
			justSawEmptyLineSeparator = false           // Reset the flag as it has served its purpose
			continue                                    // Move to the next line, now in body parsing mode
		}

		if parsingBody {
			bodyLines = append(bodyLines, originalLine) // Add original line to preserve whitespace
		} else {
			// Not parsing body, and line is not empty. Must be a request line or a header.
			if currentRequest.Method == "" { // Expecting a request line
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
				// "[DEBUG_PARSER_HEADER_INPUT]", "lineNumber", lineNumber, "processedLine", processedLine)
				// "[DEBUG_CONTAINS_COLON_CHECK]", "lineNumber", lineNumber, "processedLine", processedLine, "containsColon", strings.Contains(processedLine, ":"))
				parts := strings.SplitN(processedLine, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					currentRequest.Headers.Add(key, value) // Add to current request's headers
				} else {
					// This line is not a valid header (no colon, or malformed).
					// Assume this is the start of the body.
					// "[DEBUG_PARSER_NOT_HEADER_SWITCH_TO_BODY]", "lineNumber", lineNumber, "processedLine", processedLine, "partsLen", len(parts))
					parsingBody = true
					bodyLines = append(bodyLines, originalLine) // Add this line to body as it's the first body line
				}
			}
		}
	}

	// After the loop, add the last pending request, if any and if it's substantial
	if currentRequest != nil && (currentRequest.Method != "" || currentRequest.RawURLString != "" || len(bodyLines) > 0) {
		currentRequest.RawBody = strings.Join(bodyLines, "\n")
		currentRequest.RawBody = strings.TrimRight(currentRequest.RawBody, " \t\r\n") // Trim trailing whitespace/newlines
		currentRequest.Body = strings.NewReader(currentRequest.RawBody)               // For single read if needed directly by parser consumers
		// GetBody allows the body to be read multiple times, as required by http.Request
		rawBodyCopy := currentRequest.RawBody // Capture rawBody for the closure
		currentRequest.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(rawBodyCopy)), nil
		}

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

	// Assign accumulated file-level variables to the ParsedFile struct
	// Ensure FileVariables is initialized (it should be by the struct instantiation)
	if parsedFile.FileVariables == nil {
		parsedFile.FileVariables = make(map[string]string)
	}
	for k, v := range currentFileVariables {
		parsedFile.FileVariables[k] = v
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
// Use `Client.ValidateResponse` or `Client.Validate` which handle variable substitution internally
// before validation. For direct parsing of already substituted .hresp content, one can read the file
// into an `io.Reader` and use `ParseExpectedResponses` directly.
func ParseExpectedResponseFile(filePath string) ([]*ExpectedResponse, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening expected response file %s: %w", filePath, err)
	}
	defer func() { _ = file.Close() }()
	return ParseExpectedResponses(file, filePath)
}

// ParseExpectedResponses parses expected HTTP response definitions from an io.Reader.
// It expects the content provided by the reader to be the raw .hresp format, typically after
// any necessary variable substitutions have been applied by the caller if the .hresp content
// itself contained variables (though .hresp files are not typically expected to have variables
// that need substitution at this stage of parsing).
//
// The `filePath` argument is used for context in error messages.
//
// Returns a slice of `ExpectedResponse` structs or an error if parsing fails (e.g., due to
// malformed status lines or headers).
func ParseExpectedResponses(reader io.Reader, filePath string) ([]*ExpectedResponse, error) {
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
