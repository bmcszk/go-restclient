package restclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
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

// ResolveOptions controls the behavior of variable substitution.
// If both FallbackToOriginal and FallbackToEmpty are false, and a variable is not found,
// an error or specific handling might occur (though current implementation defaults to empty string if not original).
type ResolveOptions struct {
	FallbackToOriginal bool // If true, an unresolved placeholder {{var}} becomes "{{var}}"
	FallbackToEmpty    bool // If true, an unresolved placeholder {{var}} becomes "" (empty string)
}

// Client is the main struct for interacting with the REST client library.
// It holds configuration like the HTTP client, base URL, default headers,
// and programmatic variables for substitution.
type Client struct {
	httpClient              *http.Client
	BaseURL                 string
	DefaultHeaders          http.Header
	currentDotEnvVars       map[string]string
	programmaticVars        map[string]interface{}
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

// ClientOption is a functional option for configuring the Client.
type ClientOption func(*Client) error

// WithHTTPClient allows providing a custom http.Client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) error {
		if hc == nil {
			c.httpClient = &http.Client{}
		} else {
			c.httpClient = hc
		}
		return nil
	}
}

// WithBaseURL sets a base URL for the client.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) error {
		c.BaseURL = baseURL
		return nil
	}
}

// WithDefaultHeader adds a default header to be sent with every request.
func WithDefaultHeader(key, value string) ClientOption {
	return func(c *Client) error {
		c.DefaultHeaders.Add(key, value)
		return nil
	}
}

// WithDefaultHeaders adds multiple default headers.
func WithDefaultHeaders(headers http.Header) ClientOption {
	return func(c *Client) error {
		for key, values := range headers {
			for _, value := range values {
				c.DefaultHeaders.Add(key, value)
			}
		}
		return nil
	}
}

// WithVars sets programmatic variables for the client instance.
// These variables can be used in .http and .hresp files.
// Programmatic variables have the highest precedence during substitution,
// overriding file-defined variables, environment variables, and .env variables.
// If called multiple times, the provided vars are merged with existing ones,
// with new values for existing keys overwriting old ones.
func WithVars(vars map[string]interface{}) ClientOption {
	return func(c *Client) error {
		if c.programmaticVars == nil {
			c.programmaticVars = make(map[string]interface{})
		}
		for k, v := range vars {
			c.programmaticVars[k] = v
		}
		return nil
	}
}

// WithEnvironment sets the name of the environment to be used from http-client.env.json.
func WithEnvironment(name string) ClientOption {
	return func(c *Client) error {
		c.selectedEnvironmentName = name
		return nil
	}
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
//     a. `resolveVariablesInText` is called. For {{variableName}} placeholders (where 'variableName' does not start with '$'),
//     the precedence is: Client programmatic vars > file-scoped `@vars` (rcRequest.ActiveVariables) >
//     Environment vars (parsedFile.EnvironmentVariables) > Global vars (parsedFile.GlobalVariables) >
//     OS env vars > .env vars > fallback.
//     System variables like {{$uuid}} are resolved from the request-scoped map if the placeholder is {{$systemVarName}}.
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
		if err != nil {
			multiErr = multierror.Append(multiErr, err)
			// For body processing errors (like external file not found), skip this request entirely
			if response != nil && response.Error != nil &&
				(strings.Contains(err.Error(), "error processing body for request") ||
					strings.Contains(err.Error(), "failed to read external file")) {
				continue // Skip adding this response to the results
			}
			// Ensure we have a response for other types of errors
			if response == nil {
				response = &Response{Request: restClientReq, Error: fmt.Errorf("request processing failed")}
			}
		}
		// Handle error wrapping for logging
		if response != nil && response.Error != nil {
			urlForError := restClientReq.RawURLString
			if restClientReq.URL != nil {
				urlForError = restClientReq.URL.String()
			}
			wrappedErr := fmt.Errorf("request %d (%s %s) processing resulted in error: %w", i+1, restClientReq.Method, urlForError, response.Error)
			multiErr = multierror.Append(multiErr, wrappedErr)
		}
		if response != nil {
			responses = append(responses, response)
		}
	}

	return responses, multiErr.ErrorOrNil()
}

// End of function resolveVariablesInText

// substituteDynamicSystemVariables handles system variables that require argument parsing or dynamic evaluation at substitution time.

// generateRequestScopedSystemVariables creates a map of system variables that are generated once per request.
// This ensures that if, for example, {{$uuid}} is used multiple times within the same request
// (e.g., in the URL and a header), it resolves to the same value for that specific request.
func (c *Client) generateRequestScopedSystemVariables() map[string]string {
	vars := make(map[string]string)
	vars["$uuid"] = uuid.NewString()
	vars["$guid"] = vars["$uuid"]        // Alias $guid to $uuid
	vars["$random.uuid"] = vars["$uuid"] // Add $random.uuid as alias
	vars["$timestamp"] = strconv.FormatInt(time.Now().UTC().Unix(), 10)
	vars["$isoTimestamp"] = time.Now().UTC().Format(time.RFC3339) // Add $isoTimestamp
	vars["$randomInt"] = strconv.Itoa(rand.Intn(1001))            // 0-1000 inclusive as per PRD
	// Add other simple, no-argument system variables here if any

	// For logging/debugging purposes, to see what was generated once per request
	// fmt.Printf("[DEBUG] Generated request-scoped system variables: %v\n", vars)
	slog.Debug("generateRequestScopedSystemVariables: Generated map", "vars", vars) // Cascade: Added for debugging
	return vars
}

// _resolveRequestURL resolves the final request URL based on the client's BaseURL and the request's URL.
// It returns the resolved URL or an error if the BaseURL is invalid or requestURL is nil.
// _resolveRequestURL resolves the final request URL based on the client's BaseURL, the request's initial URL (if parsed),
// and the request's RawURLString (if initial URL parsing was deferred).
// It returns the resolved URL or an error.
func (c *Client) _resolveRequestURL(baseURLStr string, initialRequestURL *url.URL, rawRequestURLStr string) (*url.URL, error) {
	slog.Debug("_resolveRequestURL: Entered function", "baseURL", baseURLStr, "initialRequestURL_is_nil", initialRequestURL == nil, "rawRequestURLStr", rawRequestURLStr)

	var currentRequestURL *url.URL

	if initialRequestURL != nil {
		currentRequestURL = initialRequestURL
	} else if rawRequestURLStr != "" {
		// If initialRequestURL was nil (e.g., deferred parsing due to variables),
		// try to parse the rawRequestURLStr. This string should have had variables expanded by now.
		parsedRawURL, err := url.Parse(rawRequestURLStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse rawRequestURLString '%s' after variable expansion: %w", rawRequestURLStr, err)
		}
		currentRequestURL = parsedRawURL
		slog.Debug("_resolveRequestURL: Parsed rawRequestURLStr successfully", "parsedURL", currentRequestURL.String())
	} else {
		// Both initialRequestURL is nil and rawRequestURLStr is empty. This is an error.
		return nil, fmt.Errorf("request URL is unexpectedly nil and rawRequestURLString is empty")
	}

	// Sanitize the currentRequestURL (which could be from initialRequestURL or parsed rawRequestURLStr)
	// This re-parsing ensures we work with a 'fresh' copy and validates its structure.
	currentRequestURLStr := currentRequestURL.String()
	freshRequestURL, err := url.Parse(currentRequestURLStr)
	if err != nil {
		return nil, fmt.Errorf("failed to re-parse current requestURL string '%s': %w", currentRequestURLStr, err)
	}

	// If freshRequestURL is absolute, use it directly
	if freshRequestURL.IsAbs() {
		return freshRequestURL, nil
	}

	// If no BaseURL, return freshRequestURL as is
	if baseURLStr == "" {
		return freshRequestURL, nil
	}

	// Parse and sanitize the base URL
	base, err := url.Parse(baseURLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid BaseURL %s: %w", baseURLStr, err)
	}

	baseStr := base.String()
	freshBase, err := url.Parse(baseStr)
	if err != nil {
		return nil, fmt.Errorf("failed to re-parse base URL string '%s': %w", baseStr, err)
	}

	// Handle special path joining for absolute paths
	if strings.HasPrefix(freshRequestURL.Path, "/") && freshBase.Path != "" && freshBase.Path != "/" {
		finalResolvedURL := joinURLPaths(freshBase, freshRequestURL)
		if finalResolvedURL == nil {
			return nil, fmt.Errorf("failed to join URL paths: %s and %s", freshBase.Path, freshRequestURL.Path)
		}
		return finalResolvedURL, nil
	}

	// Default behavior for other cases
	return freshBase.ResolveReference(freshRequestURL), nil
}

// executeRequest sends a given Request and returns the Response.
// Errors during execution (e.g. network, body read) are captured in Response.Error.
// A non-nil error is returned by this function only for critical pre-execution failures (e.g. nil request, bad BaseURL).
func (c *Client) executeRequest(ctx context.Context, rcRequest *Request) (*Response, error) {
	if rcRequest == nil {
		slog.Error("executeRequest: called with nil rcRequest", "rcRequest_addr", fmt.Sprintf("%p", rcRequest))
		// For a nil request, we can't even populate a Response struct meaningfully.
		return nil, fmt.Errorf("cannot execute a nil request")
	}

	slog.Debug("executeRequest: Entry point", "rcRequest_addr", fmt.Sprintf("%p", rcRequest), "baseURL", c.BaseURL, "rcRequest.URL_is_nil", rcRequest.URL == nil, "rcRequest.RawURLString", rcRequest.RawURLString, "rcRequest.Name", rcRequest.Name)

	// Initialize a response object upfront to hold results or errors
	clientResponse := &Response{
		Request: rcRequest, // Link the request early
	}

	slog.Debug("executeRequest: Before _resolveRequestURL block", "baseURL", c.BaseURL, "rcRequest.URL_is_nil", rcRequest.URL == nil, "rcRequest.RawURLString", rcRequest.RawURLString, "rcRequest.Name", rcRequest.Name)

	// If rcRequest.URL is nil but RawURLString is present, it means the URL hasn't been parsed yet,
	// likely because it contained variables. Perform substitution now.
	if rcRequest.URL == nil && rcRequest.RawURLString != "" {
		slog.Debug("executeRequest: rcRequest.URL is nil and RawURLString is present, performing variable substitution", "requestName", rcRequest.Name, "rawURL", rcRequest.RawURLString)

		// For currentDotEnvVars, in a direct executeRequest call, there's no specific .env file context.
		// $dotenv substitutions will fall back to OS env vars if not found in ProgrammaticVars.
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
			err := fmt.Errorf("variable substitution failed in executeRequest for '%s': %w", rcRequest.Name, subsErr)
			slog.Error("executeRequest: Variable substitution error", "error", err, "requestName", rcRequest.Name)
			return nil, err // Critical pre-execution failure, similar to _resolveRequestURL failure
		}
		rcRequest.URL = substitutedAndParsedURL
		var logURLString string
		if rcRequest.URL != nil {
			logURLString = rcRequest.URL.String()
		} else {
			logURLString = "<nil>"
		}
		slog.Debug("executeRequest: Variable substitution complete", "requestName", rcRequest.Name, "new_rcRequest.URL_is_nil", rcRequest.URL == nil, "new_rcRequest.URL_string", logURLString)
	}

	slog.Debug("executeRequest: Before c._resolveRequestURL call", "baseURL", c.BaseURL, "rcRequest.URL_is_nil", rcRequest.URL == nil, "rcRequest.RawURLString", rcRequest.RawURLString, "rcRequest.Name", rcRequest.Name)
	// Resolve the request URL. If rcRequest.URL is already parsed (e.g., from the substitution step above or programmatic setup),
	// _resolveRequestURL will use it. Otherwise, it might re-parse RawURLString if URL is still nil (though less likely now)
	// or primarily handle BaseURL resolution against the now-populated rcRequest.URL.
	var err error
	rcRequest.URL, err = c._resolveRequestURL(c.BaseURL, rcRequest.URL, rcRequest.RawURLString)
	if err != nil {
		// An error from _resolveRequestURL implies a bad BaseURL or nil rcRequest.URL.
		// Per original logic for bad BaseURL, return nil for *Response.
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, rcRequest.Method, rcRequest.URL.String(), rcRequest.Body)
	if err != nil {
		clientResponse.Error = fmt.Errorf("failed to create http request: %w", err)
		return clientResponse, nil
	}

	for key, values := range c.DefaultHeaders {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}
	for key, values := range rcRequest.Headers {
		// "[DEBUG_HEADER_LOOP_ITERATION]", "original_key", key, "canonicalKey", textproto.CanonicalMIMEHeaderKey(key), "values_from_rcRequest", values)
		// "[DEBUG_HEADER_STATE_BEFORE_DEL]", "canonicalKey", canonicalKey, "slice_in_httpReq", httpReq.Header[canonicalKey])

		httpReq.Header.Del(key) // Uses canonicalKey internally
		// "[DEBUG_HEADER_STATE_AFTER_DEL]", "canonicalKey", textproto.CanonicalMIMEHeaderKey(key), "slice_in_httpReq", httpReq.Header[textproto.CanonicalMIMEHeaderKey(key)])

		for _, value := range values {
			httpReq.Header.Add(key, value) // Uses canonicalKey internally
			// "[DEBUG_HEADER_STATE_AFTER_ADD]", "canonicalKey", canonicalKey, "slice_in_httpReq", httpReq.Header[canonicalKey], "value_idx", i)
		}
		// "[DEBUG_HEADER_STATE_AFTER_ALL_ADDS_FOR_KEY]", "canonicalKey", canonicalKey, "slice_in_httpReq", httpReq.Header[canonicalKey])
	}

	// "[DEBUG_SPECIFIC_HEADER_AFTER_LOOP]",
	// 	"key_original", "X-Datetime-RFC1123",
	// 	"value_get", httpReq.Header.Get("X-Datetime-RFC1123"),
	// 	"key_canonical", "X-Datetime-Rfc1123",
	// 	"value_slice_direct", httpReq.Header["X-Datetime-Rfc1123"],
	// )

	if httpReq.Header.Get("Host") == "" && httpReq.URL.Host != "" {
		httpReq.Host = httpReq.URL.Host
	}

	// "[DEBUG_HTTPREQ_FINAL_HEADERS]", "headers", httpReq.Header) // Added for debugging
	startTime := time.Now()
	httpResp, err := c.httpClient.Do(httpReq)
	duration := time.Since(startTime)
	clientResponse.Duration = duration // Set duration regardless of http error

	if err != nil {
		clientResponse.Error = fmt.Errorf("http request failed: %w", err)
		if httpResp == nil {
			return clientResponse, nil // Early return if httpResp is nil, error is already set
		}

		// At this point, httpResp is non-nil. Attempt to process its body and populate details.
		var bodyBytes []byte
		var readErr error
		if httpResp.Body != nil {
			defer func() { _ = httpResp.Body.Close() }() // Defer close if body exists
			bodyBytes, readErr = io.ReadAll(httpResp.Body)
		}
		// If httpResp.Body was nil, bodyBytes and readErr remain nil.
		// _populateResponseDetails is expected to handle this gracefully.
		c._populateResponseDetails(clientResponse, httpResp, bodyBytes, readErr)
		return clientResponse, nil // Return response with error and any populated details
	}

	// Success path: ensure body is closed after reading.
	// httpResp and httpResp.Body are guaranteed non-nil here if err is nil.
	defer func() { _ = httpResp.Body.Close() }()

	bodyBytes, readErr := io.ReadAll(httpResp.Body)
	c._populateResponseDetails(clientResponse, httpResp, bodyBytes, readErr)

	return clientResponse, nil
}

// _populateResponseDetails copies relevant information from an *http.Response and body to our *Response.
func (c *Client) _populateResponseDetails(resp *Response, httpResp *http.Response, bodyBytes []byte, bodyReadErr error) {
	if httpResp == nil {
		return
	}

	resp.Status = httpResp.Status
	resp.StatusCode = httpResp.StatusCode
	resp.Proto = httpResp.Proto
	resp.Headers = httpResp.Header
	resp.Size = httpResp.ContentLength // This might be -1 if chunked, actual size is len(bodyBytes)

	if bodyReadErr != nil {
		readErrWrapped := fmt.Errorf("failed to read response body: %w", bodyReadErr)
		resp.Error = multierror.Append(resp.Error, readErrWrapped).ErrorOrNil()
	} else {
		resp.Body = bodyBytes
		resp.BodyString = string(bodyBytes)
		if resp.Size == -1 || (resp.Size == 0 && len(bodyBytes) > 0) { // Update size if not set or if chunked and body was read
			resp.Size = int64(len(bodyBytes))
		}
	}

	if httpResp.TLS != nil {
		resp.IsTLS = true
		switch httpResp.TLS.Version {
		case tls.VersionTLS10:
			resp.TLSVersion = "TLS 1.0"
		case tls.VersionTLS11:
			resp.TLSVersion = "TLS 1.1"
		case tls.VersionTLS12:
			resp.TLSVersion = "TLS 1.2"
		case tls.VersionTLS13:
			resp.TLSVersion = "TLS 1.3"
		default:
			resp.TLSVersion = fmt.Sprintf("TLS unknown (0x%04x)", httpResp.TLS.Version)
		}
		resp.TLSCipherSuite = tls.CipherSuiteName(httpResp.TLS.CipherSuite)
		// TODO: Add more TLS details like server name, peer certificates if needed
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
			nil, // default options
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
func (c *Client) getEncodingDecoder(encodingName string) (*encoding.Decoder, error) {
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
	slog.Debug("ExecuteFile: Entered function", "requestFilePath", requestFilePath)
	parsedFile, err := parseRequestFile(requestFilePath, c, make([]string, 0))
	if err != nil {
		return nil, fmt.Errorf("failed to parse request file %s: %w", requestFilePath, err)
	}
	if len(parsedFile.Requests) == 0 {
		return nil, fmt.Errorf("no requests found in file %s", requestFilePath)
	}
	slog.Debug("ExecuteFile: parseRequestFile completed successfully", "numRequests", len(parsedFile.Requests), "requestFilePath", requestFilePath)
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
func (c *Client) executeRequestWithVariables(ctx context.Context, restClientReq *Request, parsedFile *ParsedFile, osEnvGetter func(string) (string, bool), index int) (*Response, error) {
	slog.Debug("ExecuteFile: Loop start", "iteration", index, "requestName", restClientReq.Name)
	requestScopedSystemVars := c.generateRequestScopedSystemVariables()

	// Substitute variables for URL and Headers
	err := c.substituteRequestURLAndHeaders(restClientReq, parsedFile, requestScopedSystemVars, osEnvGetter)
	if err != nil {
		return &Response{Request: restClientReq, Error: err}, fmt.Errorf("variable substitution failed for request %s (index %d): %w", restClientReq.Name, index, err)
	}

	// Substitute variables for Body
	err = c.substituteRequestBody(restClientReq, parsedFile, requestScopedSystemVars, osEnvGetter)
	if err != nil {
		return &Response{Request: restClientReq, Error: err}, fmt.Errorf("error processing body for request %s (index %d): %w", restClientReq.Name, index, err)
	}

	// Execute the HTTP request
	resp, execErr := c.executeRequest(ctx, restClientReq)
	if execErr != nil {
		return &Response{Request: restClientReq, Error: execErr}, nil
	}
	return resp, nil
}

// substituteRequestURLAndHeaders handles URL and header variable substitution
func (c *Client) substituteRequestURLAndHeaders(restClientReq *Request, parsedFile *ParsedFile, requestScopedSystemVars map[string]string, osEnvGetter func(string) (string, bool)) error {
	if restClientReq.URL == nil {
		slog.Warn("ExecuteFile: restClientReq.URL is nil before variable substitution", "method", restClientReq.Method, "rawURL", restClientReq.RawURLString, "requestName", restClientReq.Name)
	}

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
func (c *Client) substituteRequestBody(restClientReq *Request, parsedFile *ParsedFile, requestScopedSystemVars map[string]string, osEnvGetter func(string) (string, bool)) error {
	var finalSubstitutedBody string
	var bodyErr error

	// Handle external file references
	if restClientReq.ExternalFilePath != "" {
		finalSubstitutedBody, bodyErr = c.processExternalFile(
			restClientReq,
			parsedFile,
			requestScopedSystemVars,
			osEnvGetter,
		)
		if bodyErr != nil {
			return bodyErr
		}
	} else if restClientReq.RawBody != "" {
		resolvedBody := resolveVariablesInText(
			restClientReq.RawBody,
			c.programmaticVars,
			restClientReq.ActiveVariables,
			parsedFile.EnvironmentVariables,
			parsedFile.GlobalVariables,
			requestScopedSystemVars,
			osEnvGetter,
			c.currentDotEnvVars,
			nil,
		)
		finalSubstitutedBody = substituteDynamicSystemVariables(
			resolvedBody,
			c.currentDotEnvVars,
			c.programmaticVars,
		)
	}

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
	return nil
}

// TODO: Add other public methods as needed, e.g.:
// - Execute(ctx context.Context, request *Request, options ...RequestOption) (*Response, error)
// - A method to validate a single response if users construct ExpectedResponse manually.
//
