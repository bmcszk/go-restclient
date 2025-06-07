package restclient

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"time"
)

const (
	requestSeparator   = "###"
	commentPrefix      = "#"
	slashCommentPrefix = "//"
)

// requestParserState holds the state during the parsing of a request file.
type requestParserState struct {
	filePath                string
	client                  *Client
	requestScopedSystemVars map[string]string
	osEnvGetter             func(string) (string, bool)
	dotEnvVars              map[string]string
	importStack             []string

	parsedFile                *ParsedFile
	currentRequest            *Request
	nextRequestName           string // Stores the name for the *next* request, captured from '### name' or 'METHOD URL ### name'
	bodyLines                 []string
	parsingBody               bool
	lineNumber                int
	currentFileVariables      map[string]string // Variables accumulated at the file scope
	justSawEmptyLineSeparator bool              // Flag to indicate the previous line was an empty separator
}

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
func parseRequestFile(filePath string, client *Client, importStack []string) (*ParsedFile, error) {
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %s: %w", filePath, err)
	}

	// Check for circular imports to prevent infinite recursion
	for _, importedPath := range importStack {
		if importedPath == absFilePath {
			return nil, fmt.Errorf("circular import detected: '%s' already in import stack %v", absFilePath, importStack)
		}
	}

	// Add current file to import stack to track import hierarchy
	newImportStack := append(importStack, absFilePath)

	file, err := os.Open(absFilePath) // Use absolute path
	if err != nil {
		return nil, fmt.Errorf("failed to open request file %s: %w", absFilePath, err)
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

	// Pass absFilePath for context, and newImportStack for recursion tracking
	reader := bufio.NewReader(file)
	parsedFile, err := parseRequests(reader, absFilePath, client, requestScopedSystemVarsForFileParse, osEnvGetter, dotEnvVarsForParser, newImportStack)
	if err != nil {
		return nil, err // Error already wrapped by parseRequests or is a direct parsing error
	}

	// Load http-client.env.json for environment-specific variables (Task T4)
	loadEnvironmentSpecificVariables(filePath, client, parsedFile) // Pass original filePath

	// Ensure programmatic variables are included in the file variables
	if client != nil && client.programmaticVars != nil {
		for key, val := range client.programmaticVars {
			// Convert value to string representation
			strVal := fmt.Sprintf("%v", val)
			parsedFile.FileVariables[key] = strVal
		}
	}

	// Second pass - resolve variables that reference other variables
	// This ensures that references like {{test_server_url}} in base_url get fully resolved
	for key, val := range parsedFile.FileVariables {
		// Only try to resolve if the value contains a variable reference
		if strings.Contains(val, "{{") && strings.Contains(val, "}}") {
			resolvedVal := resolveVariablesInValue(val, parsedFile.FileVariables)
			parsedFile.FileVariables[key] = resolvedVal
		}
	}

	return parsedFile, nil
}

// resolveVariablesInValue resolves variables within a string value using
// the provided file variables map
func resolveVariablesInValue(value string, variables map[string]string) string {
	// Use the same variable substitution logic that's used elsewhere in the codebase
	result := value
	for varName, varValue := range variables {
		placeholder := "{{" + varName + "}}"
		result = strings.ReplaceAll(result, placeholder, varValue)
	}
	return result
}

// loadEnvironmentSpecificVariables loads environment-specific variables from
// http-client.env.json and http-client.private.env.json based on the client's
// selected environment. It updates parsedFile.EnvironmentVariables.
// originalFilePath is the path originally passed to parseRequestFile, used for resolving .env.json files.
func loadEnvironmentSpecificVariables(originalFilePath string, client *Client, parsedFile *ParsedFile) {
	if client == nil || client.selectedEnvironmentName == "" || parsedFile == nil {
		return // Nothing to do if client, environment, or parsedFile is not set
	}

	mergedEnvVars := make(map[string]string)
	// Use the originalFilePath to determine the directory for http-client.env.json files,
	// consistent with original logic.
	fileDir := filepath.Dir(originalFilePath)

	// Load public environment file: http-client.env.json
	publicEnvFilePath := filepath.Join(fileDir, "http-client.env.json")
	publicEnvVars, err := loadEnvironmentFile(publicEnvFilePath, client.selectedEnvironmentName)
	if err != nil {
		slog.Warn("Error loading public environment file", "file", publicEnvFilePath, "environment", client.selectedEnvironmentName, "error", err)
	}
	for k, v := range publicEnvVars {
		mergedEnvVars[k] = v
	}

	// Load private environment file: http-client.private.env.json
	privateEnvFilePath := filepath.Join(fileDir, "http-client.private.env.json")
	privateEnvVars, err := loadEnvironmentFile(privateEnvFilePath, client.selectedEnvironmentName)
	if err != nil {
		slog.Warn("Error loading private environment file", "file", privateEnvFilePath, "environment", client.selectedEnvironmentName, "error", err)
	}
	for k, v := range privateEnvVars { // Override with private vars
		mergedEnvVars[k] = v
	}

	if len(mergedEnvVars) > 0 {
		parsedFile.EnvironmentVariables = mergedEnvVars
	} else {
		if parsedFile.EnvironmentVariables == nil { // Ensure it's initialized
			parsedFile.EnvironmentVariables = make(map[string]string)
		}
		slog.Debug("No environment variables loaded for selected environment", "environment", client.selectedEnvironmentName, "public_file", publicEnvFilePath, "private_file", privateEnvFilePath)
	}
}

// reqScopedSystemVarsForParser is generated once per file parsing pass for resolving @-vars consistently.
// var reqScopedSystemVarsForParser map[string]string // REMOVED GLOBAL

// lineType represents the different types of lines in an HTTP request file
type lineType int

const (
	lineTypeSeparator = iota
	lineTypeVariableDefinition
	lineTypeImportDirective
	lineTypeComment
	lineTypeContent // For request lines, headers, or body
)

// parseRequests reads HTTP requests from a reader and parses them into a ParsedFile struct.
// It's used by parseRequestFile to process individual HTTP request files.
func parseRequests(reader *bufio.Reader, filePath string, client *Client,
	requestScopedSystemVars map[string]string, osEnvGetter func(string) (string, bool),
	dotEnvVars map[string]string, importStack []string) (*ParsedFile, error) {

	// Initialize parser state
	parserState := &requestParserState{
		filePath:                filePath,
		client:                  client,
		requestScopedSystemVars: requestScopedSystemVars,
		osEnvGetter:             osEnvGetter,
		dotEnvVars:              dotEnvVars,
		importStack:             importStack,
		parsedFile:              &ParsedFile{Requests: make([]*Request, 0), FileVariables: make(map[string]string), FilePath: filePath},
		currentFileVariables:    make(map[string]string),
		lineNumber:              0,
	}

	// Process each line in the file
	for {
		line, err := reader.ReadString('\n')

		// Handle errors except EOF
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("error reading request file: %w", err)
		}

		// Process this line if not empty
		if line != "" {
			parserState.lineNumber++
			if err := processFileLine(parserState, line); err != nil {
				return nil, err
			}
		}

		// Break at EOF
		if err == io.EOF {
			break
		}
	}

	// Finalize the last request if there is one
	if parserState.currentRequest != nil {
		parserState.finalizeCurrentRequest()
	}

	// Ensure all file variables are copied to the parsed file
	for k, v := range parserState.currentFileVariables {
		parserState.parsedFile.FileVariables[k] = v
	}

	// Return the ParsedFile
	return parserState.parsedFile, nil
}

// processFileLine handles the processing of a single line from the request file
func processFileLine(parserState *requestParserState, line string) error {
	// Remove trailing newline and carriage return if present
	line = strings.TrimRight(line, "\r\n")
	trimmedLine := strings.TrimSpace(line)

	// Process the line based on content
	if trimmedLine == "" {
		return parserState.handleEmptyLine(line)
	}

	// Process non-empty line
	lineType := determineLineType(trimmedLine)
	return parserState.processLine(lineType, trimmedLine, line)
}

func determineLineType(trimmedLine string) lineType {
	if strings.HasPrefix(trimmedLine, requestSeparator) {
		return lineTypeSeparator
	}

	variableParts := strings.Split(trimmedLine, "=")
	if len(variableParts) > 1 && strings.HasPrefix(trimmedLine, "@") {
		return lineTypeVariableDefinition
	}

	// Check for @import directive (can be at beginning of line or in a comment)
	if strings.Contains(trimmedLine, "@import") {
		return lineTypeImportDirective
	}

	if strings.HasPrefix(trimmedLine, commentPrefix) || strings.HasPrefix(trimmedLine, slashCommentPrefix) {
		return lineTypeComment
	}

	return lineTypeContent
}

// ...

// ensureCurrentRequest creates a new request if one doesn't exist yet
// isRequestLine determines if a line is an HTTP request line (e.g., GET https://example.com)
func (p *requestParserState) isRequestLine(trimmedLine string) bool {
	// HTTP request lines start with the HTTP method followed by the URL and optionally the HTTP version
	parts := strings.Fields(trimmedLine)
	if len(parts) < 2 { // Need at least method + URL
		return false
	}

	// Check if first part is a valid HTTP method
	method := parts[0]
	validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "TRACE", "CONNECT"}
	for _, validMethod := range validMethods {
		if method == validMethod {
			// Check if the second part looks like a URL
			urlPart := parts[1]
			return strings.HasPrefix(urlPart, "http://") ||
				strings.HasPrefix(urlPart, "https://") ||
				strings.HasPrefix(urlPart, "//")
		}
	}

	return false
}

// processLine handles processing of a single line based on its determined type
func (p *requestParserState) processLine(lineType lineType, trimmedLine, originalLine string) error {
	switch lineType {
	case lineTypeSeparator:
		// Handle request separator (### or ---)
		return p.handleRequestLine(trimmedLine, originalLine) // Use handleRequestLine for requestSeparator
	case lineTypeVariableDefinition:
		return p.handleVariableDefinition(trimmedLine)
	case lineTypeImportDirective:
		return p.handleImportDirective(trimmedLine)
	case lineTypeComment:
		return p.handleComment(trimmedLine, originalLine)
	case lineTypeContent:
		// Handle general content - could be a request line, header, or body
		return p.handleContent(trimmedLine, originalLine)
	}
	return nil
}

// handleContent processes general content lines that could be request lines, headers, or body content
func (p *requestParserState) handleContent(trimmedLine, originalLine string) error {
	// First check if it's a request line (e.g., GET https://example.com)
	if p.isRequestLine(trimmedLine) {
		return p.handleRequestLine(trimmedLine, originalLine)
	}

	// Next, check if it's a header line (contains colon)
	if strings.Contains(trimmedLine, ":") && !p.parsingBody {
		return p.handleHeader(trimmedLine)
	}

	// If not a header and we have a current request, treat as body content
	p.handleBodyContent(originalLine)
	return nil
}

func (p *requestParserState) ensureCurrentRequest() {
	if p.currentRequest == nil {
		slog.Debug("ensureCurrentRequest: p.currentRequest is nil, creating new Request object", "line", p.lineNumber, "filePath", p.filePath)
		p.currentRequest = &Request{
			Headers:    make(http.Header),
			FilePath:   p.filePath,
			LineNumber: p.lineNumber, // Line number of the first significant line of this request
			// ActiveVariables will be populated by finalizeCurrentRequest or when a new request context truly begins.
		}
	} else {
		slog.Debug("ensureCurrentRequest: p.currentRequest already exists", "line", p.lineNumber, "filePath", p.filePath, "requestPtr", fmt.Sprintf("%p", p.currentRequest))
	}
}

// handleComment processes a comment line and extracts special directives (e.g., @name).
// Supports both # and // style comments (FR1.4)
func (p *requestParserState) handleComment(trimmedLine, originalLine string) error {
	var commentContent string

	// FR1.4: Support for both # and // style comments
	if strings.HasPrefix(trimmedLine, commentPrefix) {
		commentContent = strings.TrimPrefix(trimmedLine, commentPrefix)
	} else if strings.HasPrefix(trimmedLine, slashCommentPrefix) {
		commentContent = strings.TrimPrefix(trimmedLine, slashCommentPrefix)
	}

	commentContent = strings.TrimSpace(commentContent)

	// Check for request separator in the comment content
	if strings.HasPrefix(commentContent, requestSeparator) {
		// This is a commented-out separator, don't process it.
		return nil
	}

	p.ensureCurrentRequest() // Comments might have directives that require a request context

	// Process directives
	// Handle @name directive
	if strings.HasPrefix(commentContent, "@name ") {
		nameValue := strings.TrimSpace(commentContent[len("@name "):])
		p.currentRequest.Name = nameValue
		return nil
	}

	// Handle @no-redirect directive
	if strings.HasPrefix(commentContent, "@no-redirect") {
		p.currentRequest.NoRedirect = true
		return nil
	}

	// Handle @no-cookie-jar directive
	if strings.HasPrefix(commentContent, "@no-cookie-jar") {
		p.currentRequest.NoCookieJar = true
		return nil
	}

	// Handle @timeout directive with milliseconds value
	if strings.HasPrefix(commentContent, "@timeout ") {
		p.processTimeoutDirective(commentContent)
		return nil
	}

	// Other comment content - no special handling needed
	return nil
}

// handleEmptyLine processes an empty line, which can be used to separate headers from body
func (p *requestParserState) handleEmptyLine(trimmedLine string) error {
	// If a method has been defined (i.e., we are past the request line),
	// this empty line acts as the separator before the body.
	if p.currentRequest != nil && p.currentRequest.Method != "" {
		p.justSawEmptyLineSeparator = true
	}
	// If no method yet, it's an ignored empty line (e.g., between directives or before the first request).
	return nil
}

// Removed unused function handleRequestLineParsing

// handleRequestLine processes a potential HTTP request line (METHOD URL HTTP/Version).
func (p *requestParserState) handleRequestLine(trimmedLine, originalLine string) error {
	slog.Debug("handleRequestLine: Entered", "trimmedLine", trimmedLine, "requestPtr", fmt.Sprintf("%p", p.currentRequest), "line", p.lineNumber)

	if strings.HasPrefix(trimmedLine, requestSeparator) {
		slog.Debug("handleRequestLine: Detected request separator '###'", "line", trimmedLine, "requestPtr", fmt.Sprintf("%p", p.currentRequest))
		p.finalizeCurrentRequest() // Finalize the previous request

		// Reset parser state fields that are per-request, but p.currentRequest itself is now nil.
		// p.ensureCurrentRequest() will be called by the next line processor (e.g. handleComment, or this function again if it's not ###)
		// or when a new request line is actually encountered.

		// FR1.3: Support for request naming via ### Request Name
		requestName := strings.TrimSpace(strings.TrimPrefix(trimmedLine, requestSeparator))
		if requestName != "" {
			p.ensureCurrentRequest() // Ensure a request object exists to hold the name
			slog.Debug("handleRequestLine: Setting request name from separator line", "name", requestName, "requestPtr", fmt.Sprintf("%p", p.currentRequest))
			p.currentRequest.Name = requestName
		}
		return nil
	}

	// Not a separator, so it's a potential request line (METHOD URL)
	p.ensureCurrentRequest() // Ensure a request object is available
	slog.Debug("handleRequestLine: About to call parseRequestLineDetails", "trimmedLine", trimmedLine, "requestPtr", fmt.Sprintf("%p", p.currentRequest), "currentMethod", p.currentRequest.Method, "currentRawURL", p.currentRequest.RawURLString)

	_ = p.parseRequestLineDetails(trimmedLine) // Pass trimmedLine as requestLine; it returns a boolean indicating if it finalized due to same-line separator

	slog.Debug("handleRequestLine: Returned from parseRequestLineDetails", "trimmedLine", trimmedLine, "requestPtr", fmt.Sprintf("%p", p.currentRequest), "method", p.currentRequest.Method, "rawURL", p.currentRequest.RawURLString)
	return nil
}

// handleHeader processes header lines with the format: Header-Name: value
func (p *requestParserState) handleHeader(trimmedLine string) error {
	p.ensureCurrentRequest()

	// Split the header into name and value
	parts := strings.SplitN(trimmedLine, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("malformed header line: %s", trimmedLine)
	}

	headerName := strings.TrimSpace(parts[0])
	headerValue := strings.TrimSpace(parts[1])

	// Add or append the header
	p.currentRequest.Headers.Add(headerName, headerValue)
	return nil
}

// handleBodyContent processes a line that belongs to the request body.
func (p *requestParserState) handleBodyContent(line string) {
	p.ensureCurrentRequest()

	// Ensure we're in body parsing mode
	p.parsingBody = true

	// Add the line to the body
	p.bodyLines = append(p.bodyLines, line)
}

// handleVariableDefinition processes file-level variables (e.g., @variable = value)
func (p *requestParserState) handleVariableDefinition(trimmedLine string) error {
	parts := strings.SplitN(trimmedLine, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("malformed variable definition: %s", trimmedLine)
	}

	varName := strings.TrimSpace(parts[0])
	varValue := strings.TrimSpace(parts[1])

	// Store in the file variables
	p.currentFileVariables[varName] = varValue
	return nil
}

// handleImportDirective silently ignores @import directives as they are not supported in http_syntax.md.
func (p *requestParserState) handleImportDirective(trimmedLine string) error {
	// Silently ignore @import directives - they are not documented in http_syntax.md
	return nil
}

// finalizeCurrentRequest adds the current request to the parsed file's requests list
// and prepares for a new request
func (p *requestParserState) finalizeCurrentRequest() {
	slog.Debug("finalizeCurrentRequest: Attempting to finalize request", "requestPtr", fmt.Sprintf("%p", p.currentRequest), "line", p.lineNumber)
	if p.currentRequest == nil {
		slog.Debug("finalizeCurrentRequest: p.currentRequest is nil, nothing to finalize.", "line", p.lineNumber)
		return
	}

	// A request is only considered valid and added if it has both a method and a URL.
	// Body, headers, etc., are optional.
	if p.currentRequest.Method == "" || p.currentRequest.RawURLString == "" {
		slog.Debug("finalizeCurrentRequest: Request not added. Missing Method or RawURLString.",
			"requestPtr", fmt.Sprintf("%p", p.currentRequest),
			"method", p.currentRequest.Method,
			"rawURL", p.currentRequest.RawURLString,
			"line", p.lineNumber,
			"filePath", p.filePath)
	} else {
		// Set the request body from collected lines
		rawBody := strings.Join(p.bodyLines, "\n") // Use \n as per HTTP spec for line endings in body
		p.currentRequest.RawBody = rawBody
		// Note: p.currentRequest.Body (io.Reader) will be set by the consumer (e.g., Send) after variable substitution

		// Populate ActiveVariables for this request from currentFileVariables
		// This ensures the request captures variables defined before it in the file.
		p.currentRequest.ActiveVariables = make(map[string]string)
		for k, v := range p.currentFileVariables {
			p.currentRequest.ActiveVariables[k] = v
		}
		// Also include any variables defined within the request block itself (e.g. via @variable in comments)
		// This part might need refinement if @variable directives inside a request block are meant to be request-scoped
		// For now, assuming currentFileVariables is the primary source at finalization.

		slog.Debug("finalizeCurrentRequest: Adding request to parsedFile.Requests",
			"requestPtr", fmt.Sprintf("%p", p.currentRequest),
			"method", p.currentRequest.Method,
			"rawURL", p.currentRequest.RawURLString,
			"name", p.currentRequest.Name,
			"line", p.lineNumber,
			"filePath", p.filePath)
		p.parsedFile.Requests = append(p.parsedFile.Requests, p.currentRequest)
	}

	// Reset parser state for a new potential request, but don't create the new p.currentRequest here.
	// ensureCurrentRequest will handle creation when the next relevant line is processed.
	p.currentRequest = nil // Mark current request as finalized and processed.
	p.bodyLines = []string{}
	p.parsingBody = false
	p.justSawEmptyLineSeparator = false // Reset separator state
	slog.Debug("finalizeCurrentRequest: p.currentRequest set to nil. Body/parsing flags reset.", "line", p.lineNumber)
}

// processTimeoutDirective handles the @timeout directive with milliseconds value
func (p *requestParserState) processTimeoutDirective(commentContent string) {
	p.ensureCurrentRequest()
	timeoutStr := strings.TrimSpace(commentContent[len("@timeout "):])
	if timeoutStr == "" {
		return
	}

	timeoutMs, err := strconv.Atoi(timeoutStr)
	if err != nil || timeoutMs <= 0 {
		slog.Warn("Invalid timeout value in @timeout directive",
			"value", timeoutStr,
			"lineNumber", p.lineNumber,
			"filePath", p.filePath)
		return
	}

	p.currentRequest.Timeout = time.Duration(timeoutMs) * time.Millisecond
}

// _setRawURLFromLine sets the RawURLString and attempts to parse it into the URL field of the current request.
// It logs the outcome with the provided context hint.
func (p *requestParserState) _setRawURLFromLine(requestLine, contextHint string) {
	trimmedURL := strings.TrimSpace(requestLine)
	p.currentRequest.RawURLString = trimmedURL
	parsedURL, err := url.Parse(trimmedURL)
	if err != nil {
		slog.Warn("Failed to parse RawURLString",
			"context", contextHint, "rawURL", trimmedURL, "error", err,
			"line", p.lineNumber, "requestPtr", fmt.Sprintf("%p", p.currentRequest))
	} else {
		p.currentRequest.URL = parsedURL
	}
	slog.Debug("Set RawURLString",
		"context", contextHint, "RawURLString", trimmedURL,
		"requestPtr", fmt.Sprintf("%p", p.currentRequest))
}

// handleNonMethodRequestLine processes a request line where the first token is not a recognized HTTP method.
// It updates the current request's URL based on whether a method was already set.
func (p *requestParserState) handleNonMethodRequestLine(requestLine string, firstToken string) {
	if p.currentRequest.Method == "" {
		slog.Debug("First token not a method, and no method on currentRequest. Treating entire line as URL.",
			"token", firstToken, "requestLine", requestLine, "line", p.lineNumber, "requestPtr", fmt.Sprintf("%p", p.currentRequest))
		p._setRawURLFromLine(requestLine, "entire line as URL, no method previously set")
		return
	}

	// A method is already set on currentRequest.
	if p.currentRequest.RawURLString == "" {
		slog.Debug("Method already set, current line not a method, RawURLString is empty. Treating as URL.",
			"token", firstToken, "requestLine", requestLine, "currentMethod", p.currentRequest.Method, "line", p.lineNumber, "requestPtr", fmt.Sprintf("%p", p.currentRequest))
		p._setRawURLFromLine(requestLine, "URL part, method previously set")
		return
	}

	// Method and RawURLString already set, but current line starts with non-method.
	slog.Warn("Method and RawURLString already set, but current line starts with non-method. Ignoring.",
		"token", firstToken, "requestLine", requestLine, "currentMethod", p.currentRequest.Method, "currentRawURL", p.currentRequest.RawURLString, "line", p.lineNumber, "requestPtr", fmt.Sprintf("%p", p.currentRequest))
}

// parseRequestLineDetails attempts to parse the given trimmedLine as a request line (METHOD URL HTTP/Version).
// It updates p.currentRequest with the parsed details.
// If the request line includes a same-line request separator (###), it finalizes the current request
// and prepares for a new one.
// It returns true if the request was finalized due to a same-line separator, otherwise false.
// originalLine is used for logging/error context.
func (p *requestParserState) parseRequestLineDetails(originalRequestLine string) (finalizedBySeparator bool) {
	slog.Debug("parseRequestLineDetails: Parsing request line", "originalRequestLine", originalRequestLine, "requestPtr", fmt.Sprintf("%p", p.currentRequest), "line", p.lineNumber)

	requestLine := originalRequestLine
	finalizedBySeparator = false // Initialize

	// Check for same-line request separator
	if sepIndex := strings.Index(requestLine, requestSeparator); sepIndex != -1 {
		actualRequestPart := strings.TrimSpace(requestLine[:sepIndex])
		nextNamePart := ""
		if len(requestLine) > sepIndex+len(requestSeparator) {
			nextNamePart = strings.TrimSpace(requestLine[sepIndex+len(requestSeparator):])
		}

		if nextNamePart != "" {
			p.nextRequestName = nextNamePart
			slog.Debug("parseRequestLineDetails: Found same-line separator, captured next request name", "nextRequestName", p.nextRequestName, "line", p.lineNumber)
		} else {
			slog.Debug("parseRequestLineDetails: Found same-line separator, no specific name for next request", "line", p.lineNumber)
		}

		requestLine = actualRequestPart // Process only the part before "###"
		finalizedBySeparator = true     // Mark that this request needs finalization
	}

	parts := strings.Fields(requestLine)
	if len(parts) == 0 {
		if finalizedBySeparator {
			if p.currentRequest != nil && (p.currentRequest.Method != "" || p.currentRequest.RawURLString != "") {
				slog.Debug("parseRequestLineDetails: Finalizing potentially non-empty current request before empty line with separator", "requestPtr", fmt.Sprintf("%p", p.currentRequest))
				p.finalizeCurrentRequest()
			}
			p.ensureCurrentRequest()
			return true
		}
		slog.Warn("parseRequestLineDetails: Empty request line after processing potential separator", "originalRequestLine", originalRequestLine, "line", p.lineNumber, "requestPtr", fmt.Sprintf("%p", p.currentRequest))
		return false
	}

	methodToken := strings.ToUpper(parts[0])
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true, "HEAD": true, "OPTIONS": true, "TRACE": true, "CONNECT": true,
	}

	if !validMethods[methodToken] {
		p.handleNonMethodRequestLine(requestLine, methodToken)
		if finalizedBySeparator {
			if p.currentRequest.RawURLString != "" {
				slog.Debug("parseRequestLineDetails: Finalizing current request (non-method line) due to same-line separator", "requestPtr", fmt.Sprintf("%p", p.currentRequest))
				p.finalizeCurrentRequest()
				p.ensureCurrentRequest()
			}
			return true
		}
		return false
	}

	p.currentRequest.Method = methodToken
	slog.Debug("parseRequestLineDetails: Set Method", "Method", p.currentRequest.Method, "requestPtr", fmt.Sprintf("%p", p.currentRequest))

	if len(parts) < 2 {
		slog.Warn("parseRequestLineDetails: Method found, but no URL part.", "method", methodToken, "line", p.lineNumber, "requestPtr", fmt.Sprintf("%p", p.currentRequest))
		if finalizedBySeparator {
			slog.Debug("parseRequestLineDetails: Finalizing current request (method, no URL) due to same-line separator", "requestPtr", fmt.Sprintf("%p", p.currentRequest))
			p.finalizeCurrentRequest()
			p.ensureCurrentRequest()
			return true
		}
		return false
	}

	urlAndVersionStr := strings.TrimSpace(strings.Join(parts[1:], " "))
	urlStr, httpVersion := p.extractURLAndVersion(urlAndVersionStr)

	p.currentRequest.RawURLString = urlStr
	p.currentRequest.HTTPVersion = httpVersion
	slog.Debug("parseRequestLineDetails: Set RawURLString and HTTPVersion", "RawURLString", p.currentRequest.RawURLString, "HTTPVersion", p.currentRequest.HTTPVersion, "requestPtr", fmt.Sprintf("%p", p.currentRequest))

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		slog.Warn("parseRequestLineDetails: Failed to parse RawURLString", "rawURL", urlStr, "error", err, "line", p.lineNumber, "requestPtr", fmt.Sprintf("%p", p.currentRequest))
	} else {
		p.currentRequest.URL = parsedURL
	}

	if finalizedBySeparator {
		slog.Debug("parseRequestLineDetails: Finalizing current request (method and URL) due to same-line separator", "requestPtr", fmt.Sprintf("%p", p.currentRequest))
		p.finalizeCurrentRequest()
		p.ensureCurrentRequest()
		return true
	}

	return false
}

// extractURLAndVersion splits a string like "/path HTTP/1.1" or "/path" into URL and HTTP version.
func (p *requestParserState) extractURLAndVersion(urlAndVersionStr string) (urlStr, httpVersion string) {
	lastSpaceIdx := strings.LastIndex(urlAndVersionStr, " ")
	if lastSpaceIdx != -1 {
		potentialVersion := strings.TrimSpace(urlAndVersionStr[lastSpaceIdx+1:])
		if strings.HasPrefix(strings.ToUpper(potentialVersion), "HTTP/") {
			return strings.TrimSpace(urlAndVersionStr[:lastSpaceIdx]), potentialVersion
		}
	}
	return urlAndVersionStr, "" // No valid HTTP version found, assume entire string is URL
}

// Removed unused function parseHeaderOrStartBody

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

// parseExpectedStatusLine parses a line as an HTTP status line (HTTP_VERSION STATUS_CODE [STATUS_TEXT]).
// It updates the provided ExpectedResponse with the parsed status code and status string.
func parseExpectedStatusLine(line string, lineNumber int, resp *ExpectedResponse) error {
	parts := strings.Fields(line)
	if len(parts) < 2 { // Must have at least HTTP_VERSION STATUS_CODE [STATUS_TEXT]
		return fmt.Errorf("line %d: invalid status line: '%s'. Expected HTTP_VERSION STATUS_CODE [STATUS_TEXT]", lineNumber, line)
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
			// Not parsingBody, line is not empty, not separator, not comment.
			// Must be a status line or a header.
			if err := processExpectedStatusOrHeaderLine(processedLine, lineNumber, currentExpectedResponse); err != nil {
				return nil, err
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
