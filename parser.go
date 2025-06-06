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
	"strings"

	"github.com/joho/godotenv"
)

const (
	requestSeparator = "###"
	commentPrefix    = "#"
)

// requestParserState holds the state during the parsing of a request file.
type requestParserState struct {
	scanner                 *bufio.Scanner
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

// finalizeRequest populates the RawBody, Body, GetBody, and ActiveVariables fields of a request.
func finalizeRequest(req *Request, bodyLines []string, fileVars map[string]string) {
	if req == nil {
		return
	}
	req.RawBody = strings.Join(bodyLines, "\n")
	req.RawBody = strings.TrimRight(req.RawBody, " \t\r\n") // Trim trailing whitespace/newlines
	req.Body = strings.NewReader(req.RawBody)               // For single read if needed directly by parser consumers
	// GetBody allows the body to be read multiple times, as required by http.Request
	rawBodyCopy := req.RawBody // Capture rawBody for the closure
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(rawBodyCopy)), nil
	}

	// Ensure ActiveVariables is initialized and populated with current file-level variables
	if req.ActiveVariables == nil {
		req.ActiveVariables = make(map[string]string)
	}
	for k, v := range fileVars {
		req.ActiveVariables[k] = v
	}
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

	for _, importedPath := range importStack {
		if importedPath == absFilePath {
			return nil, fmt.Errorf("circular import detected: '%s' already in import stack %v", absFilePath, importStack)
		}
	}
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
	parsedFile, err := ParseRequests(file, absFilePath, client, requestScopedSystemVarsForFileParse, osEnvGetter, dotEnvVarsForParser, newImportStack)
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
		if strings.Contains(val, "{{" ) && strings.Contains(val, "}}") {
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

const (
	lineTypeSeparator = iota
	lineTypeVariableDefinition
	lineTypeImportDirective
	lineTypeComment
	lineTypeContent // For request lines, headers, or body
)

func determineLineType(trimmedLine string) int {
	if strings.HasPrefix(trimmedLine, requestSeparator) {
		return lineTypeSeparator
	}
	if strings.HasPrefix(trimmedLine, "@") {
		return lineTypeVariableDefinition
	}
	if strings.HasPrefix(trimmedLine, "// @import ") || strings.HasPrefix(trimmedLine, "# @import ") {
		return lineTypeImportDirective
	}
	// Check for general comments after specific comment-like directives (e.g. @import)
	if strings.HasPrefix(trimmedLine, commentPrefix) || strings.HasPrefix(trimmedLine, "//") {
		return lineTypeComment
	}
	return lineTypeContent
}

// ParseRequests performs the core parsing logic for an HTTP request file.
func ParseRequests(reader io.Reader, filePath string, client *Client,
	requestScopedSystemVars map[string]string,
	osEnvGetter func(string) (string, bool),
	dotEnvVars map[string]string,
	importStack []string) (*ParsedFile, error) {

	pState := &requestParserState{
		scanner:                 bufio.NewScanner(reader),
		filePath:                filePath,
		client:                  client,
		requestScopedSystemVars: requestScopedSystemVars,
		osEnvGetter:             osEnvGetter,
		dotEnvVars:              dotEnvVars,
		importStack:             importStack,
		parsedFile: &ParsedFile{
			FilePath:      filePath,
			Requests:      []*Request{},
			FileVariables: make(map[string]string),
		},
		currentFileVariables: make(map[string]string),
		bodyLines:            []string{},
	}

	for pState.scanner.Scan() {
		pState.lineNumber++
		originalLine := pState.scanner.Text()
		trimmedLine := strings.TrimSpace(originalLine)

		lineType := determineLineType(trimmedLine)

		switch lineType {
		case lineTypeSeparator:
			pState.handleRequestSeparator(trimmedLine)
		case lineTypeVariableDefinition:
			if err := pState.handleVariableDefinition(trimmedLine, originalLine); err != nil {
				return nil, err
			}
		case lineTypeImportDirective:
			if err := pState.handleImportDirective(trimmedLine, originalLine); err != nil {
				return nil, err
			}
		case lineTypeComment:
			if err := pState.handleComment(trimmedLine, originalLine); err != nil {
				return nil, err
			}
		case lineTypeContent:
			pState.ensureCurrentRequest()
			if pState.parsingBody {
				pState.handleLineWhenParsingBody(originalLine)
			} else {
				if err := pState.processContentLineWhenNotParsingBody(originalLine, trimmedLine); err != nil {
					return nil, err
				}
			}
		default:
			// Should not happen with current line types
			return nil, fmt.Errorf("line %d: unknown line type for line: %s", pState.lineNumber, originalLine)
		}
	}

	// After the loop, add the last pending request
	if pState.currentRequest != nil && (pState.currentRequest.Method != "" || pState.currentRequest.RawURLString != "" || len(pState.bodyLines) > 0) {
		finalizeRequest(pState.currentRequest, pState.bodyLines, pState.currentFileVariables)
		pState.parsedFile.Requests = append(pState.parsedFile.Requests, pState.currentRequest)
	}

	if err := pState.scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading request file %s: %w", pState.filePath, err)
	}

	// Assign accumulated file-level variables to the ParsedFile struct
	for k, v := range pState.currentFileVariables {
		pState.parsedFile.FileVariables[k] = v
	}

	return pState.parsedFile, nil
}

func (p *requestParserState) ensureCurrentRequest() {
	if p.currentRequest == nil {
		p.currentRequest = &Request{
			Headers:         make(http.Header),
			FilePath:        p.filePath,
			LineNumber:      p.lineNumber, // Line number of the first significant line of this request
			ActiveVariables: make(map[string]string),
		}
	}
}

func (p *requestParserState) handleRequestSeparator(trimmedLine string) {
	if p.currentRequest != nil && (p.currentRequest.Method != "" || p.currentRequest.RawURLString != "" || len(p.bodyLines) > 0) {
		finalizeRequest(p.currentRequest, p.bodyLines, p.currentFileVariables)
		p.parsedFile.Requests = append(p.parsedFile.Requests, p.currentRequest)
	}
	p.bodyLines = []string{}
	p.parsingBody = false
	p.justSawEmptyLineSeparator = false
	requestName := strings.TrimSpace(trimmedLine[len(requestSeparator):])
	p.currentRequest = &Request{
		Name:            requestName,
		Headers:         make(http.Header),
		FilePath:        p.filePath,
		LineNumber:      p.lineNumber,
		ActiveVariables: make(map[string]string),
	}
}

func (p *requestParserState) handleVariableDefinition(trimmedLine, originalLine string) error {
	parts := strings.SplitN(trimmedLine[1:], "=", 2) // remove leading '@'
	if len(parts) != 2 {
		return fmt.Errorf("line %d: malformed in-place variable definition, missing '=' or name: %s", p.lineNumber, originalLine)
	}

	varName := strings.TrimSpace(parts[0])
	varValue := strings.TrimSpace(parts[1])

	if varName == "" {
		return fmt.Errorf("line %d: variable name cannot be empty in definition: %s", p.lineNumber, originalLine)
	}

	// Call the new helper to resolve and set the variable
	return p.resolveAndSetFileVariable(varName, varValue)
}

// resolveAndSetFileVariable resolves the given variable value against various sources
// (programmatic, system, OS, .env) and then substitutes dynamic system variables.
// Finally, it stores the resolved variable in the current file's variables.
func (p *requestParserState) resolveAndSetFileVariable(varName, varValue string) error {
	resolvedValue := varValue // Start with the literal value

	if p.client != nil {
		// Step 1: Resolve simple system variables like {{$uuid}}, {{$timestamp}}
		// These are substituted from the pre-generated requestScopedSystemVars map.
		// We pass nil for other variable sources to prevent premature resolution of {{client_var}}, etc.
		if p.requestScopedSystemVars != nil { // Ensure map exists
			resolvedValue = p.client.resolveVariablesInText(
				resolvedValue,
				nil,                       // programmaticVars
				nil,                       // requestSpecificVars
				nil,                       // responseVars
				nil,                       // envFileVars (these are distinct from dotEnvVars used by $dotenv)
				p.requestScopedSystemVars, // systemVars (for $uuid, $timestamp)
				nil,                       // osEnvGetter (used by $env, handled by substituteDynamicSystemVariables)
				nil,                       // dotEnvVars (used by $dotenv, handled by substituteDynamicSystemVariables)
				&ResolveOptions{FallbackToOriginal: true},
			)
		}

		// Step 2: Resolve complex dynamic system variables like {{$processEnv VAR}}, {{$datetime "format"}}
		// This function also handles {{$randomInt}}, {{$env.VAR_NAME}}, {{$dotenv VAR_NAME}}
		// It expects its input `resolvedValue` to have simple system vars already processed.
		// It uses activeDotEnvVars for {{$dotenv VAR_NAME}}
		resolvedValue = p.client.substituteDynamicSystemVariables(resolvedValue, p.dotEnvVars)
	}

	p.currentFileVariables[varName] = resolvedValue
	return nil
}

func (p *requestParserState) handleImportDirective(trimmedLine, originalLine string) error {
	prefixLen := 0
	if strings.HasPrefix(trimmedLine, "// @import ") {
		prefixLen = len("// @import ")
	} else {
		prefixLen = len("# @import ")
	}
	pathPart := strings.TrimSpace(trimmedLine[prefixLen:])

	importedFilePath, wasQuoted := p.unquoteImportPath(pathPart)
	if !wasQuoted {
		slog.Warn("Malformed @import directive: path not correctly quoted", "lineContent", originalLine, "lineNumber", p.lineNumber, "filePath", p.filePath)
		return nil // Or return an error if strict quoting is required
	}

	slog.Debug("Found @import directive", "importingFile", p.filePath, "importedFileRelativePath", importedFilePath, "lineNumber", p.lineNumber)
	absImportedFilePath := filepath.Clean(filepath.Join(filepath.Dir(p.filePath), importedFilePath))
	slog.Debug("Attempting to import file", "absolutePath", absImportedFilePath, "importingFile", p.filePath)

	importedData, err := parseRequestFile(absImportedFilePath, p.client, p.importStack)
	if err != nil {
		return fmt.Errorf("line %d: error importing file '%s' (from '%s'): %w", p.lineNumber, importedFilePath, p.filePath, err)
	}
	if importedData != nil {
		p.parsedFile.Requests = append(p.parsedFile.Requests, importedData.Requests...)
		slog.Debug("Merged requests from imported file", "importedFile", absImportedFilePath, "requestCount", len(importedData.Requests))
		for key, val := range importedData.FileVariables {
			if _, exists := p.currentFileVariables[key]; !exists {
				p.currentFileVariables[key] = val
			}
		}
		slog.Debug("Merged file variables from imported file", "importedFile", absImportedFilePath, "variableCount", len(importedData.FileVariables))
	}
	return nil
}

// unquoteImportPath checks if the given pathPart is enclosed in double quotes.
// If so, it removes the quotes and returns the inner path and true.
// Otherwise, it returns the original pathPart and false.
func (p *requestParserState) unquoteImportPath(pathPart string) (string, bool) {
	if len(pathPart) > 1 && strings.HasPrefix(pathPart, "\"") && strings.HasSuffix(pathPart, "\"") {
		return pathPart[1 : len(pathPart)-1], true
	}
	return pathPart, false
}

func (p *requestParserState) handleComment(trimmedLine, originalLine string) error {
	// This function is now called only after determineLineType has identified the line as a comment.
	// We still need to correctly extract the content part of the comment.
	var commentContent string
	if strings.HasPrefix(trimmedLine, commentPrefix) { // '#' comment
		commentContent = strings.TrimSpace(trimmedLine[len(commentPrefix):])
	} else if strings.HasPrefix(trimmedLine, "//") { // '//' comment
		// Ensure it's not an import directive like "// @import" which is handled by lineTypeImportDirective
		// This check might be redundant if determineLineType is perfect, but good for safety.
		if !strings.HasPrefix(trimmedLine, "// @import") {
			commentContent = strings.TrimSpace(trimmedLine[len("//"):])
		}
	} else {
		// Should not happen if determineLineType is correct
		return fmt.Errorf("line %d: handleComment called with non-comment line: %s", p.lineNumber, originalLine)
	}

	p.processCommentContent(commentContent)

	p.justSawEmptyLineSeparator = false // Comments reset the empty line separator state
	return nil
}

// processCommentContent checks if the comment content contains a @name directive
// and updates the current request's name if found.
func (p *requestParserState) processCommentContent(commentContent string) {
	if strings.HasPrefix(commentContent, "@name ") {
		requestNameFromComment := strings.TrimSpace(commentContent[len("@name "):])
		if requestNameFromComment != "" {
			p.ensureCurrentRequest()
			p.currentRequest.Name = requestNameFromComment
		}
	}
}

func (p *requestParserState) handleEmptyLineWhenNotParsingBody() {
	// If a method has been defined (i.e., we are past the request line),
	// this empty line acts as the separator before the body.
	if p.currentRequest.Method != "" {
		p.justSawEmptyLineSeparator = true
	}
	// If no method yet, it's an ignored empty line (e.g., between directives or before the first request).
}

// handleLineWhenParsingBody appends the given line to the current request's body.
// It's called when p.parsingBody is true.
func (p *requestParserState) handleLineWhenParsingBody(originalLine string) {
	p.bodyLines = append(p.bodyLines, originalLine)
}

// processContentLineWhenNotParsingBody handles a content line when not already parsing the body.
func (p *requestParserState) processContentLineWhenNotParsingBody(originalLine, trimmedLine string) error {
	// This function is called when p.parsingBody is false.
	// The p.ensureCurrentRequest() has already been called by the caller (processContentLine).

	if trimmedLine == "" {
		// Empty line, and we are NOT parsing body.
		p.handleEmptyLineWhenNotParsingBody() // This helper sets justSawEmptyLineSeparator
		return nil
	}

	// Not parsingBody, and line is NOT empty.
	// Check if this non-empty line starts the body because the previous line was a separator.
	if p.justSawEmptyLineSeparator {
		p.parsingBody = true                // Start parsing body
		p.justSawEmptyLineSeparator = false // Consume the separator state
		// Since we are now parsing the body, use the dedicated helper for adding to body.
		p.handleLineWhenParsingBody(originalLine)
		return nil
	}

	// Not parsingBody, line is not empty, and previous was not a separator.
	// This means it's a request line (METHOD URL HTTP/Version) or a header.
	return p.processNewLineWhenNotParsingBody(trimmedLine, originalLine)
}

// handleRequestLineParsing attempts to parse the current line as a request line (METHOD URL HTTP/Version).
// It returns true if the request was finalized (e.g., by a same-line separator), and an error if parsing fails.
func (p *requestParserState) handleRequestLineParsing(trimmedLine, originalLine string) (bool, error) {
	finalized, err := p.parseRequestLineDetails(trimmedLine, originalLine)
	if err != nil {
		// This path is not currently hit as parseRequestLineDetails doesn't return errors,
		// but kept for robustness if error handling is added there.
		return false, err // Propagate error
	}
	return finalized, nil
}

// processNewLineWhenNotParsingBody handles a non-empty line when not currently parsing the request body.
// It determines if the line is a request line or a header.
func (p *requestParserState) processNewLineWhenNotParsingBody(trimmedLine, originalLine string) error {
	if p.currentRequest.Method == "" { // Expecting a request line
		finalized, err := p.handleRequestLineParsing(trimmedLine, originalLine)
		if err != nil {
			return err
		}
		if finalized {
			// Request was finalized (e.g. due to same-line ###), processing for this line is done.
			return nil
		}
	} else { // Parsing headers
		p.parseHeaderOrStartBody(trimmedLine, originalLine)
	}
	return nil
}

// parseRequestLineDetails attempts to parse the given trimmedLine as a request line (METHOD URL HTTP/Version).
// It updates p.currentRequest with the parsed details.
// If the request line includes a same-line request separator (###), it finalizes the current request
// and prepares for a new one.
// It returns true if the request was finalized due to a same-line separator, otherwise false.
// originalLine is used for logging/error context.
func (p *requestParserState) parseRequestLineDetails(trimmedLine, originalLine string) (finalizedDueToSeparator bool, err error) {
	parts := strings.SplitN(trimmedLine, " ", 2)
	if len(parts) < 2 {
		// Malformed request line (not enough parts), treat as start of body.
		slog.Warn("Malformed request line (expected METHOD URL), not treating as request or body", "line", trimmedLine, "filePath", p.filePath, "lineNumber", p.lineNumber)
		// Do not treat as body start; let it be skipped. The currentRequest will remain empty
		// and will not be added by finalize handlers due to existing guards.
		return false, nil // Not a valid request line, not finalized
	}

	method := strings.ToUpper(strings.TrimSpace(parts[0]))
	urlAndVersionStr := strings.TrimSpace(parts[1])
	finalizeAfterThisLine := false

	if sepIndex := strings.Index(urlAndVersionStr, requestSeparator); sepIndex != -1 {
		urlAndVersionStr = strings.TrimSpace(urlAndVersionStr[:sepIndex])
		finalizeAfterThisLine = true
	}

	urlStr, httpVersion := p.extractURLAndVersion(urlAndVersionStr)

	p.currentRequest.Method = method
	p.currentRequest.RawURLString = urlStr
	p.currentRequest.HTTPVersion = httpVersion
	parsedURL, _ := url.Parse(urlStr) // Error ignored as per original logic, URL might have vars
	p.currentRequest.URL = parsedURL

	if finalizeAfterThisLine {
		finalizeRequest(p.currentRequest, p.bodyLines, p.currentFileVariables) // p.bodyLines should be empty here
		p.parsedFile.Requests = append(p.parsedFile.Requests, p.currentRequest)
		p.bodyLines = []string{}                                                      // Reset for the new implicit request
		p.parsingBody = false                                                         // Reset for the new implicit request
		p.currentRequest = &Request{Headers: make(http.Header), FilePath: p.filePath} // Prepare for next request
		return true, nil
	}
	return false, nil // Request line parsed, but not finalized by separator
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

// parseHeaderOrStartBody attempts to parse the given trimmedLine as an HTTP header.
// If it's a valid header, it's added to p.currentRequest.Headers.
// If not, it's treated as the start of the request body.
// originalLine is used for appending to body to preserve whitespace.
func (p *requestParserState) parseHeaderOrStartBody(trimmedLine, originalLine string) {
	parts := strings.SplitN(trimmedLine, ":", 2)
	if len(parts) == 2 {
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key != "" { // Basic validation: header key should not be empty
			p.currentRequest.Headers.Add(key, value)
		} else {
			// Empty header key, treat as body line
			slog.Debug("Empty header key, treating as body line", "line", trimmedLine, "filePath", p.filePath, "lineNumber", p.lineNumber)
			p.parsingBody = true
			p.bodyLines = append(p.bodyLines, originalLine)
		}
	} else {
		// Not a valid header (no colon, or malformed), assume this is the start of the body.
		p.parsingBody = true
		p.bodyLines = append(p.bodyLines, originalLine)
	}
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
