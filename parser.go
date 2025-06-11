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
	"unicode"

	"time"

	"github.com/joho/godotenv"
)

const (
	requestSeparator   = "###"
	commentPrefix      = "#"
	slashCommentPrefix = "//"
)

// isPotentialRequestLine checks if a line starts with a known HTTP method.
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

// requestParserState holds the state during the parsing of a request file.
type requestParserState struct {
	// Stores the name for the *next* request, captured from '### name' or 'METHOD URL ### name'
	nextRequestName         string
	filePath                string
	client                  *Client
	requestScopedSystemVars map[string]string
	osEnvGetter             func(string) (string, bool)
	dotEnvVars              map[string]string
	importStack             []string

	parsedFile                *ParsedFile
	currentRequest            *Request
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
			return nil, fmt.Errorf(
				"circular import detected: '%s' already in import stack %v",
				absFilePath, importStack)
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
	parsedFile, err := parseRequests(
		reader, absFilePath, client, requestScopedSystemVarsForFileParse,
		osEnvGetter, dotEnvVarsForParser, newImportStack)
	if err != nil {
		return nil, err // Error already wrapped by parseRequests or is a direct parsing error
	}

	// Load http-client.env.json for environment-specific variables (Task T4)
	loadEnvironmentSpecificVariables(filePath, client, parsedFile) // Pass original filePath

	return parsedFile, nil
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
		slog.Warn(
			"Error loading public environment file",
			"file", publicEnvFilePath, "environment", client.selectedEnvironmentName,
			"error", err)
	}
	for k, v := range publicEnvVars {
		mergedEnvVars[k] = v
	}

	// Load private environment file: http-client.private.env.json
	privateEnvFilePath := filepath.Join(fileDir, "http-client.private.env.json")
	privateEnvVars, err := loadEnvironmentFile(privateEnvFilePath, client.selectedEnvironmentName)
	if err != nil {
		slog.Warn(
			"Error loading private environment file",
			"file", privateEnvFilePath, "environment", client.selectedEnvironmentName,
			"error", err)
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
		slog.Debug(
			"No environment variables loaded for selected environment",
			"environment", client.selectedEnvironmentName,
			"public_file", publicEnvFilePath, "private_file", privateEnvFilePath)
	}
}

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
		parsedFile: &ParsedFile{
			Requests: make([]*Request, 0),
			FileVariables: make(map[string]string),
			FilePath: filePath,
		},
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
		return parserState.handleEmptyLine()
	}
	// Process non-empty line
	lineType := determineLineType(trimmedLine)
	return parserState.processLine(lineType, trimmedLine, line)
}

// ensureCurrentRequest creates a new request if one doesn't exist yet
// isRequestLine determines if a line is an HTTP request line
// (e.g., GET https://example.com or just https://example.com for a GET).
func (p *requestParserState) isRequestLine(trimmedLine string) bool {
	parts := strings.Fields(trimmedLine)
	if len(parts) == 0 {
		return false
	}

	if len(parts) == 1 {
		// Potential short-form GET if it looks like a URL.
		// This check helps distinguish it from other single-word lines that are not URLs.
		// More robust parsing of the URL itself happens later.
		lineIsURL := strings.HasPrefix(parts[0], "http://") || strings.HasPrefix(parts[0], "https://")
		if lineIsURL {
			slog.Debug(
				"isRequestLine: Single token line identified as potential short-form GET URL",
				"token", parts[0], "line", p.lineNumber)
		}
		return lineIsURL
	}

	// len(parts) >= 2, check if the first part is a valid HTTP method.
	method := parts[0]
	// The actual parsing and validation of the URL happens in handleRequestLine/parseRequestLineDetails.
	return isValidHTTPToken(method)
}

// processLine handles processing of a single line based on its determined type
func (p *requestParserState) processLine(lineType lineType, trimmedLine, originalLine string) error {
	switch lineType {
	case lineTypeSeparator:
		// Handle request separator (### or ---)
		return p.handleRequestLine(trimmedLine) // Use handleRequestLine for requestSeparator
	case lineTypeVariableDefinition:
		return p.handleVariableDefinition(trimmedLine)
	case lineTypeComment:
		return p.handleComment(trimmedLine)
	case lineTypeContent:
		// Handle general content - could be a request line, header, or body
		return p.handleContent(trimmedLine, originalLine)
	}
	return nil
}

// handleContent processes general content lines that could be request lines, headers, or body content
func (p *requestParserState) handleContent(trimmedLine, originalLine string) error {
	// If we are already in the body parsing state, all non-empty, non-comment lines are body content.
	if p.parsingBody {
		p.handleBodyContent(originalLine)
		return nil
	}

	// Not parsing body. This line could be a request line or a header.
	if p.isRequestLine(trimmedLine) {
		return p.handleRequestLine(trimmedLine)
	}

	// Handle header-like lines
	if strings.Contains(trimmedLine, ":") {
		return p.handlePotentialHeaderLine(trimmedLine)
	}

	// Handle orphaned content
	return p.handleOrphanedContent(originalLine)
}

// handlePotentialHeaderLine processes lines that look like headers
func (p *requestParserState) handlePotentialHeaderLine(trimmedLine string) error {
	// Ensure we have a current request to attach this header to.
	if p.currentRequest == nil || p.currentRequest.Method == "" {
		slog.Warn(
			"Parser: Encountered header-like line without an active request "+
				"context or before a request line",
			"line", trimmedLine, "lineNumber", p.lineNumber)
		return nil
	}
	return p.handleHeader(trimmedLine)
}

// handleOrphanedContent processes lines that don't fit other categories
func (p *requestParserState) handleOrphanedContent(originalLine string) error {
	// If there's no current request context, it's likely an error or ignorable.
	if p.currentRequest == nil || p.currentRequest.Method == "" {
		slog.Warn(
			"Parser: Encountered orphaned line without an active request context",
			"line", originalLine, "lineNumber", p.lineNumber)
		return nil
	}

	// If there is a request context, treat as body content
	p.parsingBody = true
	p.handleBodyContent(originalLine)
	return nil
}

func (p *requestParserState) ensureCurrentRequest() {
	if p.currentRequest == nil {
		p.currentRequest = &Request{
			Headers:    make(http.Header),
			FilePath:   p.filePath,
			LineNumber: p.lineNumber, // Line number of the first significant line of this request
			// ActiveVariables will be populated by finalizeCurrentRequest or when a new request context truly begins.
		}
	}
}

// handleComment processes a comment line and extracts special directives (e.g., @name).
// Supports both # and // style comments (FR1.4)
func (p *requestParserState) handleComment(trimmedLine string) error {
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
	parsedName, isNameDirective := parseNameFromAtNameDirective(commentContent)
	if isNameDirective {
		if parsedName != "" {
			p.currentRequest.Name = parsedName // Apply directly
		}
		return nil // @name directive was recognized and handled (even if name was empty)
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
func (p *requestParserState) handleEmptyLine() error {
	// If a method has been defined (i.e., we are past the request line),
	// and we are not already parsing the body,
	// this empty line acts as the separator before the body.
	if p.currentRequest != nil && p.currentRequest.Method != "" && !p.parsingBody {
		p.parsingBody = true               // Crucial: now we are officially parsing the body
		p.justSawEmptyLineSeparator = true // This flag can still be useful for other logic
	}
	// If no method yet, or already parsing body, it's an ignored empty line or an empty line within the body.
	return nil
}

// Removed unused function handleRequestLineParsing

// handleRequestLine processes a potential HTTP request line (METHOD URL HTTP/Version).
func (p *requestParserState) handleRequestLine(trimmedLine string) error {

	if p.justSawEmptyLineSeparator && p.currentRequest != nil &&
		p.currentRequest.Method != "" && isPotentialRequestLine(trimmedLine) {
		p.finalizeCurrentRequest()
		// ensureCurrentRequest() will be called by subsequent logic if needed, preparing for the new request.
	}
	p.justSawEmptyLineSeparator = false // Reset flag as we are processing a non-empty line.

	if strings.HasPrefix(trimmedLine, requestSeparator) {
		p.finalizeCurrentRequest() // Finalize the previous request

		// Reset parser state fields that are per-request, but p.currentRequest itself is now nil.
		// p.ensureCurrentRequest() will be called by the next line processor
		// (e.g. handleComment, or this function again if it's not ###)
		// or when a new request line is actually encountered.

		// FR1.3: Support for request naming via ### Request Name
		requestNameFromSeparator := strings.TrimSpace(strings.TrimPrefix(trimmedLine, requestSeparator))
		if requestNameFromSeparator != "" {
			p.nextRequestName = requestNameFromSeparator
		}
		// After a separator, currentRequest should be nil (finalized by finalizeCurrentRequest).
		// The next actual request line will create a new currentRequest via ensureCurrentRequest.
		return nil
	}

	// Not a separator, so it's a potential request line (METHOD URL)
	p.ensureCurrentRequest() // Ensure a request object is available

	finalizedBySeparatorInLine := p.parseRequestLineDetails(trimmedLine) // Pass trimmedLine as requestLine

	// Apply stored nextRequestName if available and current request has no name yet.
	// @name directive would have already set p.currentRequest.Name directly.
	if !finalizedBySeparatorInLine && p.currentRequest != nil &&
		p.currentRequest.Method != "" && p.nextRequestName != "" {
		if p.currentRequest.Name == "" { // Only apply if no name is set yet (e.g. by @name)
			p.currentRequest.Name = p.nextRequestName
		}
		p.nextRequestName = "" // Clear nextRequestName as it has been considered/applied or overridden
	}

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

	// Check for external file reference syntax - only if this is the first body line
	// and no other body content exists (to avoid interfering with multipart form data)
	trimmedLine := strings.TrimSpace(line)
	if len(p.bodyLines) == 0 && strings.HasPrefix(trimmedLine, "<") &&
		(strings.HasPrefix(trimmedLine, "< ") || strings.HasPrefix(trimmedLine, "<@")) {
		p.handleExternalFileReference(trimmedLine)
		return
	}

	// Add the line to the body
	p.bodyLines = append(p.bodyLines, line)
}

// handleExternalFileReference processes external file references in request body
// Supports formats:
// - < ./path/to/file (static file content)
// - <@ ./path/to/file (file content with variable substitution)
// - <@encoding ./path/to/file (file content with variable substitution and specific encoding)
func (p *requestParserState) handleExternalFileReference(line string) {
	p.ensureCurrentRequest()

	// Remove leading whitespace and '<' character
	content := strings.TrimSpace(line[1:]) // Remove the '<'

	// Check for variable substitution syntax (<@)
	if strings.HasPrefix(content, "@") {
		p.parseExternalFileWithVariables(content[1:]) // Remove the '@'
	} else {
		// Static file reference (< ./path/to/file)
		p.currentRequest.ExternalFilePath = strings.TrimSpace(content)
		p.currentRequest.ExternalFileWithVariables = false
	}

	// Set RawBody to indicate external file usage (for backward compatibility)
	p.currentRequest.RawBody = line
}

// parseExternalFileWithVariables handles parsing of external file references with variable substitution
func (p *requestParserState) parseExternalFileWithVariables(contentAfterAt string) {
	p.currentRequest.ExternalFileWithVariables = true
	parts := strings.Fields(contentAfterAt)
	if len(parts) >= 2 && isValidEncoding(parts[0]) {
		p.currentRequest.ExternalFileEncoding = parts[0]
		p.currentRequest.ExternalFilePath = strings.Join(parts[1:], " ")
	} else {
		p.currentRequest.ExternalFilePath = strings.TrimSpace(contentAfterAt)
	}
}

// isValidEncoding checks if the given string is a valid encoding name
func isValidEncoding(encoding string) bool {
	validEncodings := map[string]bool{
		"utf-8": true, "utf8": true, "latin1": true, "iso-8859-1": true,
		"ascii": true, "cp1252": true, "windows-1252": true,
	}
	return validEncodings[strings.ToLower(encoding)]
}

// handleVariableDefinition processes file-level variables (e.g., @variable = value)
func (p *requestParserState) handleVariableDefinition(trimmedLine string) error {
	parts := strings.SplitN(trimmedLine, "=", 2)

	if len(parts) != 2 { // Case: "@name_only_var" or just "@"
		// If trimmedLine is just "@", it's missing name and equals.
		// If trimmedLine is "@foo", it's missing equals.
		return fmt.Errorf("malformed in-place variable definition, missing '=' or name part invalid: %s", trimmedLine)
	}

	varNameWithAt := strings.TrimSpace(parts[0]) // e.g., "@foo", or "@"
	varValue := strings.TrimSpace(parts[1])

	if !strings.HasPrefix(varNameWithAt, "@") {
		// This case should ideally be caught by determineLineType if it's not starting with @
		// but good to be defensive.
		return fmt.Errorf("malformed variable definition, must start with '@': %s", trimmedLine)
	}

	// Check if the name part after "@" is empty or just whitespace
	actualVarName := strings.TrimSpace(varNameWithAt[1:])
	if actualVarName == "" { // Case: "@ = value" or "@   = value"
		return fmt.Errorf("malformed in-place variable definition, variable name cannot be empty: %s", trimmedLine)
	}

	// Store in the file variables using the full @name (e.g. "@foo")
	p.currentFileVariables[varNameWithAt] = varValue
	return nil
}

// finalizeCurrentRequest adds the current request to the parsed file's requests list
// and prepares for a new request
func (p *requestParserState) finalizeCurrentRequest() {
	if p.currentRequest == nil {
		return
	}

	// A request is only considered valid and added if it has both a method and a URL.
	// Body, headers, etc., are optional.
	if p.currentRequest.Method == "" || p.currentRequest.RawURLString == "" {
	} else {
		// Set the request body from collected lines (only if external file is not used)
		if p.currentRequest.ExternalFilePath == "" {
			rawBody := strings.Join(p.bodyLines, "\n") // Use \n as per HTTP spec for line endings in body
			p.currentRequest.RawBody = rawBody
		}
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

		p.parsedFile.Requests = append(p.parsedFile.Requests, p.currentRequest)
	}

	// Reset parser state for a new potential request, but don't create the new p.currentRequest here.
	// ensureCurrentRequest will handle creation when the next relevant line is processed.
	p.currentRequest = nil // Mark current request as finalized and processed.
	p.bodyLines = []string{}
	p.parsingBody = false
	p.justSawEmptyLineSeparator = false // Reset separator state
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
// It updates the current request's URL and potentially method (for short-form GETs).
func (p *requestParserState) handleNonMethodRequestLine(requestLine string, firstToken string) {
	if p.currentRequest.Method == "" {
		// No method set on currentRequest yet.
		// Check if the firstToken (which is the whole requestLine if it's a single token line)
		// looks like a URL, implying a short-form GET.
		if strings.HasPrefix(firstToken, "http://") || strings.HasPrefix(firstToken, "https://") {
			slog.Debug("Interpreting as short-form GET request.",
				"urlToken", firstToken, "line", p.lineNumber, "requestPtr", fmt.Sprintf("%p", p.currentRequest))
			p.currentRequest.Method = "GET"
			p.currentRequest.HTTPVersion = "HTTP/1.1" // Default for short-form
			p._setRawURLFromLine(firstToken, "short-form GET URL")
		} else {
			// First token is not a method, and not a URL. It's an orphaned line or unexpected content.
			slog.Warn("First token not a method or URL, and no method on currentRequest. Treating as orphaned line.",
				"token", firstToken, "requestLine", requestLine,
				"line", p.lineNumber, "requestPtr", fmt.Sprintf("%p", p.currentRequest))
			// Potentially set as body or log as error, for now, it's an orphaned line that might be ignored
			// or become part of a body if subsequent lines suggest that.
			// If it was truly intended as a URL but didn't start with http(s), it won't be parsed as such here.
		}
		return
	}

	// A method is already set on currentRequest.
	if p.currentRequest.RawURLString == "" {
		// Method is set, but URL is not. This line could be the URL part.
		slog.Debug("Method already set, current line not a method, RawURLString is empty. Treating as URL part.",
			"token", firstToken, "requestLine", requestLine, "currentMethod", p.currentRequest.Method, "line", p.lineNumber, "requestPtr", fmt.Sprintf("%p", p.currentRequest))
		p._setRawURLFromLine(requestLine, "URL part, method previously set")
		return
	}

	// Method and RawURLString already set, but current line starts with non-method. This is unexpected.
	slog.Warn("Method and RawURLString already set, but current line starts with non-method. Ignoring line.",
		"token", firstToken, "requestLine", requestLine, "currentMethod", p.currentRequest.Method, "currentRawURL", p.currentRequest.RawURLString, "line", p.lineNumber, "requestPtr", fmt.Sprintf("%p", p.currentRequest))
}

// processSameLineSeparator handles same-line request separators (### on the same line as request)
func (p *requestParserState) processSameLineSeparator(requestLine string) (processedLine string, hasSeparator bool) {
	sepIndex := strings.Index(requestLine, requestSeparator)
	if sepIndex == -1 {
		return requestLine, false
	}

	actualRequestPart := strings.TrimSpace(requestLine[:sepIndex])
	nextNamePart := ""
	if len(requestLine) > sepIndex+len(requestSeparator) {
		nextNamePart = strings.TrimSpace(requestLine[sepIndex+len(requestSeparator):])
	}

	if nextNamePart != "" {
		p.nextRequestName = nextNamePart
	}

	return actualRequestPart, true
}

// parseRequestLineDetails attempts to parse the given trimmedLine as a request line (METHOD URL HTTP/Version).
// It updates p.currentRequest with the parsed details.
// If the request line includes a same-line request separator (###), it finalizes the current request
// and prepares for a new one.
// It returns true if the request was finalized due to a same-line separator, otherwise false.
// originalLine is used for logging/error context.
func (p *requestParserState) parseRequestLineDetails(originalRequestLine string) (finalizedBySeparator bool) {

	requestLine, finalizedBySeparator := p.processSameLineSeparator(originalRequestLine)

	parts := strings.Fields(requestLine)
	if len(parts) == 0 {
		if finalizedBySeparator {
			// If there was a separator, and the part before it was empty,
			// finalize any existing request and prepare for a new one.
			if p.currentRequest != nil && (p.currentRequest.Method != "" || p.currentRequest.RawURLString != "") {
				p.finalizeCurrentRequest()
			}
			p.ensureCurrentRequest() // Prepare for the request that might be named by nextRequestName
			return true
		}
		slog.Warn("parseRequestLineDetails: Empty request line after processing potential separator", "originalRequestLine", originalRequestLine, "line", p.lineNumber, "requestPtr", fmt.Sprintf("%p", p.currentRequest))
		return false
	}

	methodCandidate := parts[0] // Method candidate in its original case

	if !isValidHTTPToken(methodCandidate) {
		// First token is not a valid HTTP method token.
		// Try to handle as a non-method line (e.g., a bare URL which implies GET).
		p.handleNonMethodRequestLine(requestLine, methodCandidate)
		// handleNonMethodRequestLine might set RawURLString and imply a method (e.g. GET)
		// or leave Method empty if it can't determine one.

		if finalizedBySeparator {
			// If there was a separator, finalize this "non-method" request if it's valid enough.
			if p.currentRequest != nil && (p.currentRequest.Method != "" || p.currentRequest.RawURLString != "") {
				p.finalizeCurrentRequest()
			}
			p.ensureCurrentRequest() // Prepare for the next request
			return true
		}
		// If not finalized by separator, the validity of this non-method line stands on its own.
		// finalizeCurrentRequest will later determine if it's a complete request.
		return false
	}

	// methodCandidate IS a valid HTTP token, so treat it as the method.
	// Store in original case as per RFC 7230 (methods are case-sensitive).
	p.currentRequest.Method = methodCandidate

	if len(parts) < 2 {
		slog.Warn("parseRequestLineDetails: Method found, but no URL part.", "method", methodCandidate, "line", p.lineNumber, "requestPtr", fmt.Sprintf("%p", p.currentRequest))
		// Even if there's no URL, if a separator follows, we finalize this incomplete request.
		if finalizedBySeparator {
			p.finalizeCurrentRequest()
			p.ensureCurrentRequest() // Prepare for the next request
			return true
		}
		return false // Incomplete request line
	}

	urlAndVersionStr := strings.TrimSpace(strings.Join(parts[1:], " "))
	urlStr, httpVersion := p.extractURLAndVersion(urlAndVersionStr)

	p.currentRequest.RawURLString = urlStr
	p.currentRequest.HTTPVersion = httpVersion // Can be empty if not specified

	// Check if URL contains variables (using {{ and }} as variable markers)
	containsVariables := strings.Contains(urlStr, "{{") || strings.Contains(urlStr, "}}")

	if containsVariables {
		// p.currentRequest.URL remains nil as parsing is deferred
	} else {
		// No variables, try to parse URL now
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			slog.Warn("parseRequestLineDetails: Failed to parse RawURLString (no variables)", "rawURL", urlStr, "error", err, "line", p.lineNumber, "requestPtr", fmt.Sprintf("%p", p.currentRequest))
			// p.currentRequest.URL remains nil or as set by url.Parse on error (which is typically nil for parse errors)
		} else {
			p.currentRequest.URL = parsedURL
		}
	}

	if finalizedBySeparator {
		// If there was a separator on the same line, finalize this now-parsed request.
		p.finalizeCurrentRequest()
		p.ensureCurrentRequest() // Prepare for the next request
		return true
	}

	return false // Not finalized by a same-line separator
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
