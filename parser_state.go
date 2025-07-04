package restclient

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// RequestLineResult represents the result of parsing a request line
type RequestLineResult int

const (
	// RequestLineContinues indicates the request line was processed normally
	RequestLineContinues RequestLineResult = iota
	// RequestLineFinalizedBySeparator indicates the request was finalized due to a same-line separator
	RequestLineFinalizedBySeparator
)

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
	
	// Multi-line query parameter support
	queryParams               []string          // Accumulated query parameters from multi-line syntax
	parsingQueryParams        bool              // Flag to indicate we're collecting query parameters
}

// processFileLines reads and processes all lines from the reader
func processFileLines(reader *bufio.Reader, parserState *requestParserState) error {
	for {
		line, err := reader.ReadString('\n')
		if readErr := handleReadError(err); readErr != nil {
			return readErr
		}
		
		if processErr := processLineIfNeeded(line, parserState); processErr != nil {
			return processErr
		}
		
		if err == io.EOF {
			break
		}
	}
	return nil
}

// processLineIfNeeded processes a line if it should be processed
func processLineIfNeeded(line string, parserState *requestParserState) error {
	if shouldProcessLine(line, parserState) {
		return processFileLine(parserState, line)
	}
	return nil
}

// handleReadError checks for read errors excluding EOF
func handleReadError(err error) error {
	if err != nil && err != io.EOF {
		return fmt.Errorf("error reading request file: %w", err)
	}
	return nil
}

// shouldProcessLine determines if a line should be processed
func shouldProcessLine(line string, parserState *requestParserState) bool {
	if line != "" {
		parserState.lineNumber++
		return true
	}
	return false
}

// finalizeParseResults completes the parsing process
func finalizeParseResults(parserState *requestParserState) {
	if parserState.currentRequest != nil {
		parserState.finalizeCurrentRequest()
	}
	
	for k, v := range parserState.currentFileVariables {
		parserState.parsedFile.FileVariables[k] = v
	}
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

	// Check for multi-line query parameters (? or & prefixed lines)
	if p.isQueryParameterLine(trimmedLine) {
		return p.handleQueryParameterLine(trimmedLine)
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
	// Finalize any accumulated query parameters before starting body parsing
	p.finalizeQueryParameters()
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
	commentContent := p.extractCommentContent(trimmedLine)
	
	if p.isCommentedSeparator(commentContent) {
		return nil
	}

	p.ensureCurrentRequest() // Comments might have directives that require a request context
	return p.processCommentDirectives(commentContent)
}

// extractCommentContent extracts the content from comment lines
func (*requestParserState) extractCommentContent(trimmedLine string) string {
	var commentContent string
	if strings.HasPrefix(trimmedLine, commentPrefix) {
		commentContent = strings.TrimPrefix(trimmedLine, commentPrefix)
	} else if strings.HasPrefix(trimmedLine, slashCommentPrefix) {
		commentContent = strings.TrimPrefix(trimmedLine, slashCommentPrefix)
	}
	return strings.TrimSpace(commentContent)
}

// isCommentedSeparator checks if the comment contains a request separator
func (*requestParserState) isCommentedSeparator(commentContent string) bool {
	return strings.HasPrefix(commentContent, requestSeparator)
}

// processCommentDirectives processes various comment directives
func (p *requestParserState) processCommentDirectives(commentContent string) error {
	if p.handleNameDirective(commentContent) {
		return nil
	}
	if p.handleNoRedirectDirective(commentContent) {
		return nil
	}
	if p.handleNoCookieJarDirective(commentContent) {
		return nil
	}
	if p.handleTimeoutDirective(commentContent) {
		return nil
	}
	return nil // Other comment content - no special handling needed
}

// handleNameDirective processes @name directives
func (p *requestParserState) handleNameDirective(commentContent string) bool {
	parsedName, isNameDirective := parseNameFromAtNameDirective(commentContent)
	if isNameDirective && parsedName != "" {
		p.currentRequest.Name = parsedName
	}
	return isNameDirective
}

// handleNoRedirectDirective processes @no-redirect directives
func (p *requestParserState) handleNoRedirectDirective(commentContent string) bool {
	if strings.HasPrefix(commentContent, "@no-redirect") {
		p.currentRequest.NoRedirect = true
		return true
	}
	return false
}

// handleNoCookieJarDirective processes @no-cookie-jar directives
func (p *requestParserState) handleNoCookieJarDirective(commentContent string) bool {
	if strings.HasPrefix(commentContent, "@no-cookie-jar") {
		p.currentRequest.NoCookieJar = true
		return true
	}
	return false
}

// handleTimeoutDirective processes @timeout directives
func (p *requestParserState) handleTimeoutDirective(commentContent string) bool {
	if strings.HasPrefix(commentContent, "@timeout ") {
		p.processTimeoutDirective(commentContent)
		return true
	}
	return false
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

// handleRequestLine processes a potential HTTP request line (METHOD URL HTTP/Version).
func (p *requestParserState) handleRequestLine(trimmedLine string) error {
	p.handleEmptyLineSeparatorState(trimmedLine)
	p.justSawEmptyLineSeparator = false

	if p.isRequestSeparator(trimmedLine) {
		return p.handleRequestSeparator(trimmedLine)
	}

	return p.processActualRequestLine(trimmedLine)
}

// handleEmptyLineSeparatorState handles state when empty line separator was seen
func (p *requestParserState) handleEmptyLineSeparatorState(trimmedLine string) {
	if p.justSawEmptyLineSeparator && p.currentRequest != nil &&
		p.currentRequest.Method != "" && isPotentialRequestLine(trimmedLine) {
		p.finalizeCurrentRequest()
	}
}

// isRequestSeparator checks if the line is a request separator
func (*requestParserState) isRequestSeparator(trimmedLine string) bool {
	return strings.HasPrefix(trimmedLine, requestSeparator)
}

// handleRequestSeparator processes request separator lines
func (p *requestParserState) handleRequestSeparator(trimmedLine string) error {
	p.finalizeCurrentRequest()

	requestNameFromSeparator := strings.TrimSpace(strings.TrimPrefix(trimmedLine, requestSeparator))
	if requestNameFromSeparator != "" {
		p.nextRequestName = requestNameFromSeparator
	}
	return nil
}

// processActualRequestLine processes actual request lines (not separators)
func (p *requestParserState) processActualRequestLine(trimmedLine string) error {
	p.ensureCurrentRequest()

	result := p.parseRequestLineDetails(trimmedLine)
	p.applyStoredRequestName(result)

	return nil
}

// applyStoredRequestName applies stored request name if conditions are met
func (p *requestParserState) applyStoredRequestName(result RequestLineResult) {
	if result == RequestLineContinues && p.currentRequest != nil &&
		p.currentRequest.Method != "" && p.nextRequestName != "" {
		if p.currentRequest.Name == "" {
			p.currentRequest.Name = p.nextRequestName
		}
		p.nextRequestName = ""
	}
}

// handleHeader processes header lines with the format: Header-Name: value
func (p *requestParserState) handleHeader(trimmedLine string) error {
	p.ensureCurrentRequest()

	// Finalize any accumulated query parameters before processing headers
	p.finalizeQueryParameters()

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

	// Finalize any accumulated query parameters before processing body content
	if !p.parsingBody {
		p.finalizeQueryParameters()
	}

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

	// Handle multi-line form data (VS Code REST Client syntax)
	if p.isFormUrlEncodedContent() && p.shouldHandleFormDataContinuation(line) {
		p.handleFormDataLine(line)
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

	// Finalize any accumulated query parameters before finalizing the request
	p.finalizeQueryParameters()

	// A request is only considered valid and added if it has both a method and a URL.
	// Body, headers, etc., are optional.
	if p.currentRequest.Method != "" && p.currentRequest.RawURLString != "" {
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
	p.parsingQueryParams = false         // Reset query parameter state
	p.queryParams = []string{}
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
			"token", firstToken, "requestLine", requestLine,
			"currentMethod", p.currentRequest.Method, "line", p.lineNumber,
			"requestPtr", fmt.Sprintf("%p", p.currentRequest))
		p._setRawURLFromLine(requestLine, "URL part, method previously set")
		return
	}

	// Method and RawURLString already set, but current line starts with non-method. This is unexpected.
	slog.Warn("Method and RawURLString already set, but current line starts with non-method. Ignoring line.",
		"token", firstToken, "requestLine", requestLine,
		"currentMethod", p.currentRequest.Method,
		"currentRawURL", p.currentRequest.RawURLString, "line", p.lineNumber,
		"requestPtr", fmt.Sprintf("%p", p.currentRequest))
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
func (p *requestParserState) parseRequestLineDetails(originalRequestLine string) RequestLineResult {
	requestLine, finalizedBySeparator := p.processSameLineSeparator(originalRequestLine)

	parts := strings.Fields(requestLine)
	var result RequestLineResult
	if finalizedBySeparator {
		result = RequestLineFinalizedBySeparator
	} else {
		result = RequestLineContinues
	}
	
	if len(parts) == 0 {
		return p.handleEmptyRequestLine(result, originalRequestLine)
	}

	methodCandidate := parts[0]
	if !isValidHTTPToken(methodCandidate) {
		return p.handleNonMethodLine(requestLine, methodCandidate, result)
	}

	return p.handleValidMethodLine(parts, methodCandidate, result)
}

// handleEmptyRequestLine handles empty request lines
func (p *requestParserState) handleEmptyRequestLine(
	result RequestLineResult, originalRequestLine string) RequestLineResult {
	if result == RequestLineFinalizedBySeparator {
		if p.currentRequest != nil && (p.currentRequest.Method != "" || p.currentRequest.RawURLString != "") {
			p.finalizeCurrentRequest()
		}
		p.ensureCurrentRequest()
		return RequestLineFinalizedBySeparator
	}
	slog.Warn(
		"parseRequestLineDetails: Empty request line after processing potential separator",
		"originalRequestLine", originalRequestLine, "line", p.lineNumber,
		"requestPtr", fmt.Sprintf("%p", p.currentRequest))
	return RequestLineContinues
}

// handleNonMethodLine handles lines that don't start with a valid HTTP method
func (p *requestParserState) handleNonMethodLine(
	requestLine, methodCandidate string, result RequestLineResult) RequestLineResult {
	p.handleNonMethodRequestLine(requestLine, methodCandidate)

	if result == RequestLineFinalizedBySeparator {
		if p.currentRequest != nil && (p.currentRequest.Method != "" || p.currentRequest.RawURLString != "") {
			p.finalizeCurrentRequest()
		}
		p.ensureCurrentRequest()
		return RequestLineFinalizedBySeparator
	}
	return RequestLineContinues
}

// handleValidMethodLine handles lines that start with a valid HTTP method
func (p *requestParserState) handleValidMethodLine(parts []string, methodCandidate string, 
	result RequestLineResult) RequestLineResult {
	p.currentRequest.Method = methodCandidate

	if len(parts) < 2 {
		return p.handleIncompleteMethodLine(methodCandidate, result)
	}

	p.parseURLAndVersion(parts)

	if result == RequestLineFinalizedBySeparator {
		p.finalizeCurrentRequest()
		p.ensureCurrentRequest()
		return RequestLineFinalizedBySeparator
	}

	return RequestLineContinues
}

// handleIncompleteMethodLine handles method lines without URL parts
func (p *requestParserState) handleIncompleteMethodLine(
	methodCandidate string, result RequestLineResult) RequestLineResult {
	slog.Warn(
		"parseRequestLineDetails: Method found, but no URL part.",
		"method", methodCandidate, "line", p.lineNumber,
		"requestPtr", fmt.Sprintf("%p", p.currentRequest))

	if result == RequestLineFinalizedBySeparator {
		p.finalizeCurrentRequest()
		p.ensureCurrentRequest()
		return RequestLineFinalizedBySeparator
	}
	return RequestLineContinues
}

// parseURLAndVersion parses URL and HTTP version from request line parts
func (p *requestParserState) parseURLAndVersion(parts []string) {
	urlAndVersionStr := strings.TrimSpace(strings.Join(parts[1:], " "))
	urlStr, httpVersion := p.extractURLAndVersion(urlAndVersionStr)

	p.currentRequest.RawURLString = urlStr
	p.currentRequest.HTTPVersion = httpVersion

	p.parseURLIfNoVariables(urlStr)
}

// parseURLIfNoVariables parses URL immediately if it contains no variables
func (p *requestParserState) parseURLIfNoVariables(urlStr string) {
	containsVariables := strings.Contains(urlStr, "{{") || strings.Contains(urlStr, "}}")

	if !containsVariables {
		if parsedURL, err := url.Parse(urlStr); err != nil {
			slog.Warn(
				"parseRequestLineDetails: Failed to parse RawURLString (no variables)",
				"rawURL", urlStr, "error", err, "line", p.lineNumber,
				"requestPtr", fmt.Sprintf("%p", p.currentRequest))
		} else {
			p.currentRequest.URL = parsedURL
		}
	}
}


// isQueryParameterLine checks if a line is a multi-line query parameter
// VS Code REST Client syntax: lines starting with ? or & (after whitespace)
func (p *requestParserState) isQueryParameterLine(trimmedLine string) bool {
	// Must have a current request to append query parameters to
	if p.currentRequest == nil || p.currentRequest.Method == "" {
		return false
	}

	// Check for ? (first query param) or & (continuation query param)
	return strings.HasPrefix(trimmedLine, "?") || strings.HasPrefix(trimmedLine, "&")
}

// handleQueryParameterLine processes multi-line query parameter syntax
func (p *requestParserState) handleQueryParameterLine(trimmedLine string) error {
	if p.currentRequest == nil {
		return errors.New("query parameter line without active request context")
	}

	// Start collecting query parameters if this is the first one
	if !p.parsingQueryParams {
		p.parsingQueryParams = true
		p.queryParams = []string{}
	}

	// Remove the ? or & prefix and trim whitespace
	paramLine := strings.TrimSpace(trimmedLine[1:])
	if paramLine != "" {
		p.queryParams = append(p.queryParams, paramLine)
	}

	return nil
}

// finalizeQueryParameters appends accumulated query parameters to the current request URL
func (p *requestParserState) finalizeQueryParameters() {
	if !p.parsingQueryParams || len(p.queryParams) == 0 || p.currentRequest == nil {
		return
	}

	// Join all query parameters with &
	queryString := strings.Join(p.queryParams, "&")
	
	// Append to the URL
	p.appendQueryStringToURL(queryString)

	// Reset query parameter state
	p.parsingQueryParams = false
	p.queryParams = []string{}
}

// appendQueryStringToURL appends the query string to the current request URL
func (p *requestParserState) appendQueryStringToURL(queryString string) {
	if p.currentRequest.RawURLString == "" {
		return
	}
	
	separator := "?"
	if strings.Contains(p.currentRequest.RawURLString, "?") {
		separator = "&"
	}
	p.currentRequest.RawURLString += separator + queryString
	
	// Re-parse the URL if it doesn't contain variables
	p.reparseURLIfNoVariables()
}

// reparseURLIfNoVariables re-parses the URL if it doesn't contain template variables
func (p *requestParserState) reparseURLIfNoVariables() {
	rawURL := p.currentRequest.RawURLString
	containsVariables := strings.Contains(rawURL, "{{") || strings.Contains(rawURL, "}}")
	if !containsVariables {
		if parsedURL, err := url.Parse(p.currentRequest.RawURLString); err == nil {
			p.currentRequest.URL = parsedURL
		}
	}
}

// isFormUrlEncodedContent checks if the current request has form-urlencoded content type
func (p *requestParserState) isFormUrlEncodedContent() bool {
	if p.currentRequest == nil {
		return false
	}
	
	contentType := p.currentRequest.Headers.Get("Content-Type")
	return strings.Contains(strings.ToLower(contentType), "application/x-www-form-urlencoded")
}

// shouldHandleFormDataContinuation checks if this line should be handled as form data continuation
func (*requestParserState) shouldHandleFormDataContinuation(line string) bool {
	trimmedLine := strings.TrimSpace(line)
	
	// Handle & line continuations (VS Code REST Client syntax)
	if strings.HasPrefix(trimmedLine, "&") {
		return true
	}
	
	// Handle regular form data lines (key=value format)
	hasEquals := strings.Contains(trimmedLine, "=")
	notJSON := !strings.HasPrefix(trimmedLine, "{") && !strings.HasPrefix(trimmedLine, "[")
	return hasEquals && notJSON
}

// handleFormDataLine processes a form data line, handling & continuations properly
func (p *requestParserState) handleFormDataLine(line string) {
	trimmedLine := strings.TrimSpace(line)
	
	if strings.HasPrefix(trimmedLine, "&") {
		// Remove the & and add to existing form data
		formParam := strings.TrimSpace(trimmedLine[1:])
		if len(p.bodyLines) > 0 {
			// Append to the last line with & separator
			lastIndex := len(p.bodyLines) - 1
			p.bodyLines[lastIndex] = strings.TrimSpace(p.bodyLines[lastIndex]) + "&" + formParam
		} else {
			// First line starts with &, treat as form parameter
			p.bodyLines = append(p.bodyLines, formParam)
		}
	} else {
		// Regular form data line - trim newlines but preserve the content
		p.bodyLines = append(p.bodyLines, strings.TrimSpace(line))
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