package restclient

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/joho/godotenv"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
)


// Client is the main struct for interacting with the REST client library.
// It holds configuration like the HTTP client, base URL, default headers,
// and programmatic variables for substitution.
type Client struct {
	httpClient              *http.Client
	BaseURL                 string
	DefaultHeaders          http.Header
	currentDotEnvVars       map[string]string
	programmaticVars        map[string]any
	selectedEnvironmentName string // Added for T4
}

// NewClient creates a new instance of the REST client.
// Options for customization (e.g., timeout, custom transport) can be added later.
func NewClient(options ...ClientOption) (*Client, error) {
	c := &Client{
		httpClient:     &http.Client{},
		DefaultHeaders: make(http.Header),
	}

	for _, option := range options {
		err := option(c)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}


// ExecuteFile parses a request file (.http, .rest), executes all requests found, and returns their responses.
// It returns an error if the file cannot be parsed or no requests are found.
// Individual request execution errors are stored within each Response object.
//
// Variable Substitution Workflow:
// 1. File Parsing (`parseRequestFile`):
//   - Loads .env file from the request file's directory.
//   - Generates request-scoped system variables (e.g., `{{$uuid}}`) once for the entire file parsing pass.
//   - Resolves `@variable = value` definitions. The `value` itself can contain placeholders,
//     which are resolved using: Client programmatic vars > request-scoped system vars > OS env vars > .env vars.
//
// 2. Request Execution (within `ExecuteFile` loop for each request):
//   - Re-generates request-scoped system variables (e.g., `{{$uuid}}`) *once per individual request* to ensure
//     uniqueness if needed across multiple requests in the same file, but consistency within a single request.
//   - For each part of the request (URL, headers, body):
//     a. `resolveVariablesInText` is called. For {{variableName}} placeholders
//        (where 'variableName' does not start with '$'),
//     the precedence is: Client programmatic vars > file-scoped `@vars` (rcRequest.ActiveVariables) >
//     Environment vars (parsedFile.EnvironmentVariables) > Global vars (parsedFile.GlobalVariables) >
//     OS env vars > .env vars > fallback.
//     System variables like {{$uuid}} are resolved from the request-scoped map
//     if the placeholder is {{$systemVarName}}.
//     It resolves simple system variables like `{{$uuid}}` from the request-scoped map.
//     It leaves dynamic system variables (e.g., `{{$dotenv NAME}}`) untouched for the next step.
//     b. `substituteDynamicSystemVariables` is called: This handles system variables requiring arguments
//     (e.g., `{{$dotenv NAME}}`, `{{$processEnv NAME}}`, `{{$randomInt MIN MAX}}`).
//
// Programmatic variables for substitution can be set on the Client using `WithVars()`.
func (c *Client) ExecuteFile(ctx context.Context, requestFilePath string) ([]*Response, error) {
	parsedFile, err := c.parseAndValidateFile(requestFilePath)
	if err != nil {
		return nil, err
	}

	c.loadDotEnvVars(requestFilePath)

	var responses []*Response
	var multiErr *multierror.Error
	osEnvGetter := func(key string) (string, bool) { return os.LookupEnv(key) }

	for i, restClientReq := range parsedFile.Requests {
		response, err := c.executeRequestWithVariables(ctx, restClientReq, parsedFile, osEnvGetter, i)
		response, shouldSkip := c.handleRequestExecutionError(response, err, restClientReq, i, &multiErr)
		if shouldSkip {
			continue
		}
		if response != nil {
			responses = append(responses, response)
		}
	}

	return responses, multiErr.ErrorOrNil()
}

// handleRequestExecutionError processes errors from request execution and manages error wrapping
// Returns the processed response and a boolean indicating if the request should be skipped
func (c *Client) handleRequestExecutionError(
	response *Response,
	err error,
	restClientReq *Request,
	index int,
	multiErr **multierror.Error,
) (*Response, bool) {
	if err != nil {
		*multiErr = multierror.Append(*multiErr, err)
		if shouldSkipRequest(response, err) {
			return nil, true
		}
		response = ensureResponseExists(response, restClientReq)
	}
	
	c.wrapResponseError(response, restClientReq, index, multiErr)
	return response, false
}

// shouldSkipRequest determines if a request should be skipped based on error type
func shouldSkipRequest(response *Response, err error) bool {
	return response != nil && response.Error != nil &&
		(strings.Contains(err.Error(), "error processing body for request") ||
			strings.Contains(err.Error(), "failed to read external file"))
}

// ensureResponseExists creates a response if none exists
func ensureResponseExists(response *Response, restClientReq *Request) *Response {
	if response == nil {
		return &Response{Request: restClientReq, Error: errors.New("request processing failed")}
	}
	return response
}

// wrapResponseError wraps response errors for logging
func (*Client) wrapResponseError(
	response *Response,
	restClientReq *Request,
	index int,
	multiErr **multierror.Error,
) {
	if response != nil && response.Error != nil {
		urlForError := restClientReq.RawURLString
		if restClientReq.URL != nil {
			urlForError = restClientReq.URL.String()
		}
		wrappedErr := fmt.Errorf(
			"request %d (%s %s) processing resulted in error: %w",
			index+1, restClientReq.Method, urlForError, response.Error)
		*multiErr = multierror.Append(*multiErr, wrappedErr)
	}
}

// End of function resolveVariablesInText

// substituteDynamicSystemVariables handles system variables that require argument
// parsing or dynamic evaluation at substitution time.

// generateRequestScopedSystemVariables creates a map of system variables that are generated once per request.
// This ensures that if, for example, {{$uuid}} is used multiple times within the same request
// (e.g., in the URL and a header), it resolves to the same value for that specific request.
func (*Client) generateRequestScopedSystemVariables() map[string]string {
	vars := make(map[string]string)
	vars["$uuid"] = uuid.NewString()
	vars["$guid"] = vars["$uuid"]        // Alias $guid to $uuid
	vars["$random.uuid"] = vars["$uuid"] // Add $random.uuid as alias
	vars["$timestamp"] = strconv.FormatInt(time.Now().UTC().Unix(), 10)
	vars["$isoTimestamp"] = time.Now().UTC().Format(time.RFC3339) // Add $isoTimestamp
	vars["$randomInt"] = strconv.Itoa(rand.Intn(1001))            // 0-1000 inclusive as per PRD
	// Add other simple, no-argument system variables here if any

	return vars
}

// _resolveRequestURL resolves the final request URL based on the client's BaseURL and the request's URL.
// It returns the resolved URL or an error if the BaseURL is invalid or requestURL is nil.
// _resolveRequestURL resolves the final request URL based on the client's BaseURL,
// the request's initial URL (if parsed),
// and the request's RawURLString (if initial URL parsing was deferred).
// It returns the resolved URL or an error.
func (*Client) _resolveRequestURL(
	baseURLStr string,
	initialRequestURL *url.URL,
	rawRequestURLStr string,
) (*url.URL, error) {
	currentRequestURL, err := determineCurrentRequestURL(initialRequestURL, rawRequestURLStr)
	if err != nil {
		return nil, err
	}

	freshRequestURL, err := sanitizeRequestURL(currentRequestURL)
	if err != nil {
		return nil, err
	}

	return resolveWithBaseURL(freshRequestURL, baseURLStr)
}

// determineCurrentRequestURL determines which URL to use for processing
func determineCurrentRequestURL(initialRequestURL *url.URL, rawRequestURLStr string) (*url.URL, error) {
	if initialRequestURL != nil {
		return initialRequestURL, nil
	}
	if rawRequestURLStr != "" {
		parsedRawURL, err := url.Parse(rawRequestURLStr)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to parse rawRequestURLString '%s' after variable expansion: %w",
				rawRequestURLStr, err)
		}
		return parsedRawURL, nil
	}
	return nil, errors.New("request URL is unexpectedly nil and rawRequestURLString is empty")
}

// sanitizeRequestURL re-parses a URL to ensure it's valid
func sanitizeRequestURL(currentRequestURL *url.URL) (*url.URL, error) {
	currentRequestURLStr := currentRequestURL.String()
	freshRequestURL, err := url.Parse(currentRequestURLStr)
	if err != nil {
		return nil, fmt.Errorf("failed to re-parse current requestURL string '%s': %w", currentRequestURLStr, err)
	}
	return freshRequestURL, nil
}

// resolveWithBaseURL resolves a request URL against a base URL
func resolveWithBaseURL(freshRequestURL *url.URL, baseURLStr string) (*url.URL, error) {
	if freshRequestURL.IsAbs() {
		return freshRequestURL, nil
	}
	if baseURLStr == "" {
		return freshRequestURL, nil
	}

	freshBase, err := parseAndSanitizeBaseURL(baseURLStr)
	if err != nil {
		return nil, err
	}

	return handleSpecialPathJoining(freshRequestURL, freshBase)
}

// parseAndSanitizeBaseURL parses and sanitizes a base URL
func parseAndSanitizeBaseURL(baseURLStr string) (*url.URL, error) {
	base, err := url.Parse(baseURLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid BaseURL %s: %w", baseURLStr, err)
	}

	baseStr := base.String()
	freshBase, err := url.Parse(baseStr)
	if err != nil {
		return nil, fmt.Errorf("failed to re-parse base URL string '%s': %w", baseStr, err)
	}
	return freshBase, nil
}

// handleSpecialPathJoining handles special cases for URL path joining
func handleSpecialPathJoining(freshRequestURL, freshBase *url.URL) (*url.URL, error) {
	if strings.HasPrefix(freshRequestURL.Path, "/") && freshBase.Path != "" && freshBase.Path != "/" {
		finalResolvedURL := joinURLPaths(freshBase, freshRequestURL)
		if finalResolvedURL == nil {
			return nil, fmt.Errorf("failed to join URL paths: %s and %s", freshBase.Path, freshRequestURL.Path)
		}
		return finalResolvedURL, nil
	}
	return freshBase.ResolveReference(freshRequestURL), nil
}

// executeRequest sends a given Request and returns the Response.
// Errors during execution (e.g. network, body read) are captured in Response.Error.
// A non-nil error is returned by this function only for critical pre-execution
// failures (e.g. nil request, bad BaseURL).
func (c *Client) executeRequest(ctx context.Context, rcRequest *Request) (*Response, error) {
	if rcRequest == nil {
		return nil, errors.New("cannot execute a nil request")
	}

	clientResponse := &Response{Request: rcRequest}

	if err := c.prepareRequestURL(rcRequest); err != nil {
		return nil, err
	}

	httpReq, err := c.createHTTPRequest(ctx, rcRequest)
	if err != nil {
		clientResponse.Error = err
		return clientResponse, nil
	}

	httpResp, duration, doErr := c.executeHTTPRequest(httpReq, rcRequest)
	clientResponse.Duration = duration

	if doErr != nil {
		return c.handleHTTPError(clientResponse, httpResp, doErr, httpReq), nil
	}

	defer func() { _ = httpResp.Body.Close() }()
	bodyBytes, readErr := io.ReadAll(httpResp.Body)
	c._populateResponseDetails(clientResponse, httpResp, bodyBytes, readErr)

	return clientResponse, nil
}

// prepareRequestURL handles URL preparation and variable substitution
func (c *Client) prepareRequestURL(rcRequest *Request) error {
	if rcRequest.URL == nil && rcRequest.RawURLString != "" {
		substitutedAndParsedURL, subsErr := substituteRequestVariables(
			rcRequest,
			nil, // parsedFile - no file context for direct executeRequest
			c.generateRequestScopedSystemVariables(),
			os.LookupEnv,
			c.programmaticVars,
			nil,       // currentDotEnvVars - no specific .env file for direct call
			c.BaseURL, // Pass client's BaseURL for consistency
		)
		if subsErr != nil {
			return fmt.Errorf("variable substitution failed for request '%s': %w", rcRequest.Name, subsErr)
		}
		rcRequest.URL = substitutedAndParsedURL
	}

	var err error
	rcRequest.URL, err = c._resolveRequestURL(c.BaseURL, rcRequest.URL, rcRequest.RawURLString)
	return err
}

// createHTTPRequest creates an HTTP request with headers
func (c *Client) createHTTPRequest(ctx context.Context, rcRequest *Request) (*http.Request, error) {
	httpReq, err := http.NewRequestWithContext(ctx, rcRequest.Method, rcRequest.URL.String(), rcRequest.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	c.setRequestHeaders(httpReq, rcRequest)
	return httpReq, nil
}

// setRequestHeaders sets default and request-specific headers
func (c *Client) setRequestHeaders(httpReq *http.Request, rcRequest *Request) {
	c.addDefaultHeaders(httpReq)
	c.addRequestHeaders(httpReq, rcRequest)
	c.setHostHeader(httpReq)
}

// addDefaultHeaders adds client default headers to the request
func (c *Client) addDefaultHeaders(httpReq *http.Request) {
	for key, values := range c.DefaultHeaders {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}
}

// addRequestHeaders adds request-specific headers
func (*Client) addRequestHeaders(httpReq *http.Request, rcRequest *Request) {
	for key, values := range rcRequest.Headers {
		httpReq.Header.Del(key)
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}
}

// setHostHeader sets the Host header if not already set
func (*Client) setHostHeader(httpReq *http.Request) {
	if httpReq.Header.Get("Host") == "" && httpReq.URL.Host != "" {
		httpReq.Host = httpReq.URL.Host
	}
}

// executeHTTPRequest executes the HTTP request and returns response, duration, and error
func (c *Client) executeHTTPRequest(httpReq *http.Request, rcRequest *Request) (*http.Response, time.Duration, error) {
	startTime := time.Now()
	var httpResp *http.Response
	var doErr error

	if rcRequest.NoCookieJar {
		tempClient := *c.httpClient
		tempClient.Jar = nil
		httpResp, doErr = tempClient.Do(httpReq)
	} else {
		httpResp, doErr = c.httpClient.Do(httpReq)
	}

	duration := time.Since(startTime)
	return httpResp, duration, doErr
}

// handleHTTPError handles HTTP execution errors
func (c *Client) handleHTTPError(
	clientResponse *Response,
	httpResp *http.Response,
	doErr error,
	_ *http.Request,
) *Response {
	clientResponse.Error = fmt.Errorf("failed to execute HTTP request: %w", doErr)
	if httpResp != nil {
		var bodyBytes []byte
		c._populateResponseDetails(clientResponse, httpResp, bodyBytes, doErr)
		if httpResp.Body != nil {
			_ = httpResp.Body.Close()
		}
	}
	// Log critical HTTP errors only
	return clientResponse
}

// _populateResponseDetails copies relevant information from an *http.Response and body to our *Response.
func (*Client) _populateResponseDetails(resp *Response, httpResp *http.Response, bodyBytes []byte, bodyReadErr error) {
	if httpResp == nil {
		return
	}

	populateBasicResponseData(resp, httpResp)
	populateBodyData(resp, bodyBytes, bodyReadErr)
	populateTLSData(resp, httpResp)
}

// populateBasicResponseData sets basic response fields
func populateBasicResponseData(resp *Response, httpResp *http.Response) {
	resp.Status = httpResp.Status
	resp.StatusCode = httpResp.StatusCode
	resp.Proto = httpResp.Proto
	resp.Headers = httpResp.Header
	resp.Size = httpResp.ContentLength
}

// populateBodyData handles body data and errors
func populateBodyData(resp *Response, bodyBytes []byte, bodyReadErr error) {
	if bodyReadErr != nil {
		readErrWrapped := fmt.Errorf("failed to read response body: %w", bodyReadErr)
		resp.Error = multierror.Append(resp.Error, readErrWrapped).ErrorOrNil()
	} else {
		resp.Body = bodyBytes
		resp.BodyString = string(bodyBytes)
		if resp.Size == -1 || (resp.Size == 0 && len(bodyBytes) > 0) {
			resp.Size = int64(len(bodyBytes))
		}
	}
}

// populateTLSData handles TLS-related response data
func populateTLSData(resp *Response, httpResp *http.Response) {
	if httpResp.TLS != nil {
		resp.IsTLS = true
		resp.TLSVersion = getTLSVersionString(httpResp.TLS.Version)
		resp.TLSCipherSuite = tls.CipherSuiteName(httpResp.TLS.CipherSuite)
	}
}

// getTLSVersionString converts TLS version to string
func getTLSVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("TLS unknown (0x%04x)", version)
	}
}

// processExternalFile reads and processes external file references with optional variable substitution and encoding
func (c *Client) processExternalFile(
	restClientReq *Request,
	parsedFile *ParsedFile,
	requestScopedSystemVars map[string]string,
	osEnvGetter func(string) (string, bool),
) (string, error) {
	// Resolve the file path relative to the request's file directory
	requestDir := filepath.Dir(restClientReq.FilePath)
	fullPath := restClientReq.ExternalFilePath
	if !filepath.IsAbs(fullPath) {
		fullPath = filepath.Join(requestDir, restClientReq.ExternalFilePath)
	}

	// Read the file with appropriate encoding
	content, err := c.readFileWithEncoding(fullPath, restClientReq.ExternalFileEncoding)
	if err != nil {
		return "", fmt.Errorf("failed to read external file %s: %w", restClientReq.ExternalFilePath, err)
	}

	// Apply variable substitution if requested
	if restClientReq.ExternalFileWithVariables {
		resolvedContent := resolveVariablesInText(
			content,
			c.programmaticVars,
			restClientReq.ActiveVariables,
			parsedFile.EnvironmentVariables,
			parsedFile.GlobalVariables,
			requestScopedSystemVars,
			osEnvGetter,
			c.currentDotEnvVars,
		)
		content = substituteDynamicSystemVariables(
			resolvedContent,
			c.currentDotEnvVars,
			c.programmaticVars,
		)
	}

	return content, nil
}

// readFileWithEncoding reads a file with the specified encoding, defaulting to UTF-8
func (c *Client) readFileWithEncoding(filePath, encodingName string) (string, error) {
	// Read the file as bytes
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// If no encoding specified or UTF-8, return as-is
	if encodingName == "" || strings.ToLower(encodingName) == "utf-8" || strings.ToLower(encodingName) == "utf8" {
		return string(data), nil
	}

	// Get the decoder for the specified encoding
	decoder, err := c.getEncodingDecoder(encodingName)
	if err != nil {
		return "", fmt.Errorf("unsupported encoding %s: %w", encodingName, err)
	}

	// Decode the content
	decodedContent, err := decoder.Bytes(data)
	if err != nil {
		return "", fmt.Errorf("failed to decode content with encoding %s: %w", encodingName, err)
	}

	return string(decodedContent), nil
}

// getEncodingDecoder returns the appropriate decoder for the given encoding name
func (*Client) getEncodingDecoder(encodingName string) (*encoding.Decoder, error) {
	encodingName = strings.ToLower(encodingName)

	switch encodingName {
	case "latin1", "iso-8859-1":
		return charmap.ISO8859_1.NewDecoder(), nil
	case "cp1252", "windows-1252":
		return charmap.Windows1252.NewDecoder(), nil
	case "ascii":
		// ASCII is a subset of UTF-8, so we can use UTF-8 decoder
		return unicode.UTF8.NewDecoder(), nil
	default:
		return nil, fmt.Errorf("unsupported encoding: %s", encodingName)
	}
}

// parseAndValidateFile parses the request file and validates it has requests
func (c *Client) parseAndValidateFile(requestFilePath string) (*ParsedFile, error) {
	parsedFile, err := parseRequestFile(requestFilePath, c, make([]string, 0))
	if err != nil {
		return nil, fmt.Errorf("failed to parse request file %s: %w", requestFilePath, err)
	}
	if len(parsedFile.Requests) == 0 {
		return nil, fmt.Errorf("no requests found in file %s", requestFilePath)
	}
	return parsedFile, nil
}

// loadDotEnvVars loads .env variables from the same directory as the request file
func (c *Client) loadDotEnvVars(requestFilePath string) {
	c.currentDotEnvVars = make(map[string]string)
	envFilePath := filepath.Join(filepath.Dir(requestFilePath), ".env")
	if _, err := os.Stat(envFilePath); err == nil {
		loadedVars, loadErr := godotenv.Read(envFilePath)
		if loadErr == nil {
			c.currentDotEnvVars = loadedVars
		}
	}
}

// executeRequestWithVariables handles variable substitution and execution for a single request
func (c *Client) executeRequestWithVariables(
	ctx context.Context,
	restClientReq *Request,
	parsedFile *ParsedFile,
	osEnvGetter func(string) (string, bool),
	index int,
) (*Response, error) {
	requestScopedSystemVars := c.generateRequestScopedSystemVariables()

	// Substitute variables for URL and Headers
	err := c.substituteRequestURLAndHeaders(restClientReq, parsedFile, requestScopedSystemVars, osEnvGetter)
	if err != nil {
		return &Response{Request: restClientReq, Error: err}, fmt.Errorf(
			"variable substitution failed for request %s (index %d): %w",
			restClientReq.Name, index, err)
	}

	// Substitute variables for Body
	err = c.substituteRequestBody(restClientReq, parsedFile, requestScopedSystemVars, osEnvGetter)
	if err != nil {
		return &Response{Request: restClientReq, Error: err}, fmt.Errorf(
			"error processing body for request %s (index %d): %w",
			restClientReq.Name, index, err)
	}

	// Execute the HTTP request
	resp, execErr := c.executeRequest(ctx, restClientReq)
	if execErr != nil {
		return &Response{Request: restClientReq, Error: execErr}, nil
	}
	return resp, nil
}

// substituteRequestURLAndHeaders handles URL and header variable substitution
func (c *Client) substituteRequestURLAndHeaders(
	restClientReq *Request,
	parsedFile *ParsedFile,
	requestScopedSystemVars map[string]string,
	osEnvGetter func(string) (string, bool),
) error {
	finalParsedURL, subsErr := substituteRequestVariables(
		restClientReq,
		parsedFile,
		requestScopedSystemVars,
		osEnvGetter,
		c.programmaticVars,
		c.currentDotEnvVars,
		c.BaseURL,
	)
	if subsErr != nil {
		return subsErr
	}
	restClientReq.URL = finalParsedURL
	return nil
}

// substituteRequestBody handles body variable substitution including external files
func (c *Client) substituteRequestBody(
	restClientReq *Request,
	parsedFile *ParsedFile,
	requestScopedSystemVars map[string]string,
	osEnvGetter func(string) (string, bool),
) error {
	finalSubstitutedBody, err := c.resolveRequestBody(restClientReq, parsedFile, requestScopedSystemVars, osEnvGetter)
	if err != nil {
		return err
	}

	c.setRequestBody(restClientReq, finalSubstitutedBody)
	return nil
}

// resolveRequestBody handles the core body resolution logic
func (c *Client) resolveRequestBody(
	restClientReq *Request,
	parsedFile *ParsedFile,
	requestScopedSystemVars map[string]string,
	osEnvGetter func(string) (string, bool),
) (string, error) {
	if restClientReq.ExternalFilePath != "" {
		return c.processExternalFile(restClientReq, parsedFile, requestScopedSystemVars, osEnvGetter)
	}

	if restClientReq.RawBody == "" {
		return "", nil
	}

	if c.isMultipartFormWithFileReferences(restClientReq) {
		return c.processMultipartFormWithFiles(restClientReq, parsedFile, requestScopedSystemVars, osEnvGetter)
	}

	return c.processRegularBody(restClientReq, parsedFile, requestScopedSystemVars, osEnvGetter), nil
}

// processRegularBody handles regular body processing (non-multipart, non-external)
func (c *Client) processRegularBody(
	restClientReq *Request,
	parsedFile *ParsedFile,
	requestScopedSystemVars map[string]string,
	osEnvGetter func(string) (string, bool),
) string {
	resolvedBody := resolveVariablesInText(
		restClientReq.RawBody,
		c.programmaticVars,
		restClientReq.ActiveVariables,
		parsedFile.EnvironmentVariables,
		parsedFile.GlobalVariables,
		requestScopedSystemVars,
		osEnvGetter,
		c.currentDotEnvVars,
	)
	return substituteDynamicSystemVariables(resolvedBody, c.currentDotEnvVars, c.programmaticVars)
}

// setRequestBody sets the final body content on the request
func (*Client) setRequestBody(restClientReq *Request, finalSubstitutedBody string) {
	if finalSubstitutedBody != "" {
		restClientReq.RawBody = finalSubstitutedBody
		restClientReq.Body = strings.NewReader(finalSubstitutedBody)
		restClientReq.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(finalSubstitutedBody)), nil
		}
	} else {
		restClientReq.Body = nil
		restClientReq.GetBody = nil
	}
}

// TODO: Add other public methods as needed, e.g.:
// - Execute(ctx context.Context, request *Request, options ...RequestOption) (*Response, error)
// - A method to validate a single response if users construct ExpectedResponse manually.
//
