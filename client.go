package restclient

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
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
	envName                 string
	oauth2Tokens            map[string]oauth2Token
	oauth2Mu                sync.Mutex
}

// NewClient creates a new instance of the REST client.
// Options for customization (e.g., timeout, custom transport) can be added later.
func NewClient(options ...ClientOption) (*Client, error) {
	c := &Client{
		httpClient:     &http.Client{},
		DefaultHeaders: make(http.Header),
		oauth2Tokens:   make(map[string]oauth2Token),
	}

	for _, option := range options {
		err := option(c)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

// SetProgrammaticVars sets variables with highest precedence for variable substitution.
// These override .env, OS env, file-scoped vars, and globals.
func (c *Client) SetProgrammaticVars(vars map[string]any) {
	if c.programmaticVars == nil {
		c.programmaticVars = make(map[string]any)
	}
	for k, v := range vars {
		c.programmaticVars[k] = v
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
//     a. `resolveVariablesInText` is called. For {{variableName}} placeholders
//     (where 'variableName' does not start with '$'),
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
	c.resolveFileScopedSystemVariables(parsedFile)

	parsedFile.ResponseMap = make(map[string]*Response)
	var responses []*Response
	var multiErr *multierror.Error
	osEnvGetter := func(key string) (string, bool) { return os.LookupEnv(key) }
	refState := newRefExecutionState()

	for i, restClientReq := range parsedFile.Requests {
		// @import'd requests run only when invoked via @ref/@forceRef from a local request.
		if restClientReq.Imported {
			continue
		}
		reqResponses, err := c.runOneRequest(ctx, restClientReq, i, parsedFile, refState, osEnvGetter)
		responses = append(responses, reqResponses...)
		multiErr = multierror.Append(multiErr, err)
	}

	return responses, multiErr.ErrorOrNil()
}

// runOneRequest executes one request plus its @ref/@forceRef deps; returns appended responses and errors.
func (c *Client) runOneRequest(
	ctx context.Context,
	restClientReq *Request,
	index int,
	parsedFile *ParsedFile,
	refState *refExecutionState,
	osEnvGetter func(string) (string, bool),
) ([]*Response, *multierror.Error) {
	var respOut []*Response
	var multiErr *multierror.Error
	if refState.alreadyExecuted(restClientReq.Name) {
		return respOut, multiErr
	}
	skip, err := c.skipRequest(restClientReq, parsedFile, osEnvGetter)
	if err != nil {
		return respOut, multierror.Append(multiErr, fmt.Errorf("evaluating @disabled: %w", err))
	}
	if skip {
		skipped := &Response{Request: restClientReq, Skipped: true}
		respOut = append(respOut, skipped)
		storeResponse(parsedFile, restClientReq, skipped)
		return respOut, multiErr
	}
	if handled, loopResponses, loopErr := c.runRequestWithLoops(
		ctx, restClientReq, index, parsedFile, refState, osEnvGetter,
	); handled {
		respOut = append(respOut, loopResponses...)
		return respOut, multierror.Append(multiErr, loopErr)
	}
	if restClientReq.SleepDuration > 0 {
		time.Sleep(restClientReq.SleepDuration)
	}
	response, err := c.executeRequestWithVariables(ctx, restClientReq, parsedFile, osEnvGetter, index)
	response, shouldSkip, combinedErr := c.handleRequestExecutionError(response, err, restClientReq, index)
	multiErr = multierror.Append(multiErr, combinedErr)
	if shouldSkip || response == nil {
		return respOut, multiErr
	}
	respOut = append(respOut, response)
	storeResponse(parsedFile, restClientReq, response)
	refState.recordExecuted(restClientReq.Name, response)
	return respOut, multiErr
}

// isLoopedRequest returns true when the request declares any @loop directive.
func (*Client) isLoopedRequest(req *Request) bool {
	return req.LoopDeclared
}

// skipRequest decides @disabled / @disabled !<expr> skipping; returns (skip, err).
func (c *Client) skipRequest(
	restClientReq *Request,
	parsedFile *ParsedFile,
	osEnvGetter func(string) (string, bool),
) (bool, error) {
	if restClientReq.Disabled {
		return true, nil
	}
	if restClientReq.DisabledExpr == "" {
		return false, nil
	}
	return c.evaluateDisabledExpr(restClientReq.DisabledExpr, parsedFile, restClientReq, osEnvGetter)
}

// refExecutionState tracks per-ExecuteFile execution of @ref/@forceRef dependencies.
type refExecutionState struct {
	executed map[string]*Response // name -> most recent response
	onStack  map[string]bool      // names currently being resolved (cycle detection)
}

func newRefExecutionState() *refExecutionState {
	return &refExecutionState{
		executed: make(map[string]*Response),
		onStack:  make(map[string]bool),
	}
}

func (s *refExecutionState) alreadyExecuted(name string) bool {
	return name != "" && s.executed[name] != nil
}

func (s *refExecutionState) recordExecuted(name string, resp *Response) {
	if name != "" && resp != nil {
		s.executed[name] = resp
	}
}

func (s *refExecutionState) pushStack(name string) func() {
	s.onStack[name] = true
	return func() { delete(s.onStack, name) }
}

// resolveRequestRefs resolves refs depth-first; @ref caches, @forceRef re-runs.
func (c *Client) resolveRequestRefs(
	ctx context.Context,
	req *Request,
	parsedFile *ParsedFile,
	state *refExecutionState,
	osEnvGetter func(string) (string, bool),
) error {
	for _, ref := range req.Refs {
		if refCacheReuseable(ref, state) {
			continue
		}
		if err := c.executeReferencedRequest(ctx, ref, parsedFile, state, osEnvGetter); err != nil {
			return err
		}
	}
	return nil
}

func refCacheReuseable(ref RequestRef, state *refExecutionState) bool {
	return !ref.Force && state.executed[ref.Name] != nil
}

func (c *Client) executeReferencedRequest(
	ctx context.Context,
	ref RequestRef,
	parsedFile *ParsedFile,
	state *refExecutionState,
	osEnvGetter func(string) (string, bool),
) error {
	if state.onStack[ref.Name] {
		return fmt.Errorf("cycle detected in @ref graph involving request %q", ref.Name)
	}
	target := findRequestByName(parsedFile, ref.Name)
	if target == nil {
		return fmt.Errorf("unknown referenced request %q", ref.Name)
	}

	popStack := state.pushStack(ref.Name)
	defer popStack()

	if err := c.resolveRequestRefs(ctx, target, parsedFile, state, osEnvGetter); err != nil {
		return err
	}
	if !ref.Force && state.executed[ref.Name] != nil {
		return nil
	}
	c.runAndCacheReferenced(ctx, target, ref.Name, parsedFile, state, osEnvGetter)
	return nil
}

func findRequestByName(parsedFile *ParsedFile, name string) *Request {
	for _, r := range parsedFile.Requests {
		if r.Name == name {
			return r
		}
	}
	return nil
}

func (c *Client) runAndCacheReferenced(
	ctx context.Context,
	target *Request,
	name string,
	parsedFile *ParsedFile,
	state *refExecutionState,
	osEnvGetter func(string) (string, bool),
) {
	response, _ := c.executeRequestWithVariables(ctx, target, parsedFile, osEnvGetter, requestIndex(parsedFile, target))
	state.recordExecuted(name, response)
	if response != nil {
		storeResponse(parsedFile, target, response)
	}
}

func requestIndex(parsedFile *ParsedFile, target *Request) int {
	for i, r := range parsedFile.Requests {
		if r == target {
			return i
		}
	}
	return -1
}

// storeResponse saves a response in the response map keyed by request name.
func storeResponse(parsedFile *ParsedFile, req *Request, resp *Response) {
	if req.Name != "" {
		parsedFile.ResponseMap[req.Name] = resp
	}
}

// ParseFile parses the request file (.http, .rest) without executing requests.
// Returns the parsed file with all requests, variables, and environment resolved.
func (c *Client) ParseFile(requestFilePath string) (*ParsedFile, error) {
	parsedFile, err := c.parseAndValidateFile(requestFilePath)
	if err != nil {
		return nil, err
	}

	c.loadDotEnvVars(requestFilePath)
	c.resolveFileScopedSystemVariables(parsedFile)

	parsedFile.ResponseMap = make(map[string]*Response)
	return parsedFile, nil
}

// ExecuteRequest executes a single request from a parsed file by index.
// The caller must first call ParseFile to obtain the parsed file.
// @ref/@forceRef dependencies of the request are resolved depth-first before execution.
func (c *Client) ExecuteRequest(ctx context.Context, parsedFile *ParsedFile, index int) (*Response, error) {
	if index < 0 || index >= len(parsedFile.Requests) {
		return nil, fmt.Errorf("request index %d out of range (file has %d requests)", index, len(parsedFile.Requests))
	}

	osEnvGetter := func(key string) (string, bool) { return os.LookupEnv(key) }
	restClientReq := parsedFile.Requests[index]
	skip, err := c.skipRequest(restClientReq, parsedFile, osEnvGetter)
	if err != nil {
		return nil, fmt.Errorf("evaluating @disabled: %w", err)
	}
	if skip {
		return &Response{Request: restClientReq, Skipped: true}, nil
	}
	if err := c.resolveRequestRefs(ctx, restClientReq, parsedFile, newRefExecutionState(), osEnvGetter); err != nil {
		return nil, err
	}
	if restClientReq.SleepDuration > 0 {
		time.Sleep(restClientReq.SleepDuration)
	}
	return c.executeAndStoreRequest(ctx, restClientReq, parsedFile, osEnvGetter, index)
}

// executeAndStoreRequest executes the request and stores the response in the file's response map.
func (c *Client) executeAndStoreRequest(
	ctx context.Context,
	restClientReq *Request,
	parsedFile *ParsedFile,
	osEnvGetter func(string) (string, bool),
	index int,
) (*Response, error) {
	if c.isLoopedRequest(restClientReq) {
		return c.executeLoopAndStore(ctx, restClientReq, parsedFile, osEnvGetter, index)
	}
	response, err := c.executeRequestWithVariables(ctx, restClientReq, parsedFile, osEnvGetter, index)
	if err != nil {
		if response == nil {
			response = &Response{Request: restClientReq, Error: err}
		}
		return response, err
	}
	storeResponse(parsedFile, restClientReq, response)
	return response, nil
}

// handleRequestExecutionError processes errors from request execution; returns (response, shouldSkip, combined error).
func (c *Client) handleRequestExecutionError(
	response *Response,
	err error,
	restClientReq *Request,
	index int,
) (*Response, bool, error) {
	if err != nil {
		if shouldSkipRequest(response, err) {
			return nil, true, err
		}
		response = ensureResponseExists(response, restClientReq)
	}

	wrappedErr := c.wrapResponseError(response, restClientReq, index)
	return response, false, multierror.Append(err, wrappedErr).ErrorOrNil()
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

// wrapResponseError returns a wrapped error describing the request's failed processing, or nil when nothing to wrap.
func (*Client) wrapResponseError(
	response *Response,
	restClientReq *Request,
	index int,
) error {
	if response == nil || response.Error == nil {
		return nil
	}
	urlForError := restClientReq.RawURLString
	if restClientReq.URL != nil {
		urlForError = restClientReq.URL.String()
	}
	return fmt.Errorf(
		"request %d (%s %s) processing resulted in error: %w",
		index+1, restClientReq.Method, urlForError, response.Error)
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

// resolveFileScopedSystemVariables resolves file-scoped variables that contain system variable placeholders.
// This ensures that system variables in file-scoped variable definitions (like @scenarioId = {{$uuid}})
// are resolved once per file, not once per request, maintaining consistency across all requests in the file.
func (c *Client) resolveFileScopedSystemVariables(parsedFile *ParsedFile) {
	if parsedFile == nil || parsedFile.FileVariables == nil {
		return
	}

	// Generate file-scoped system variables once for the entire file
	fileScopedSystemVars := c.generateRequestScopedSystemVariables()

	// Resolve file-scoped variables and track resolved ones
	resolvedVariables := c.resolveFileVariables(parsedFile, fileScopedSystemVars)

	// Update all requests' ActiveVariables to reflect the resolved values
	c.updateRequestActiveVariables(parsedFile.Requests, resolvedVariables)
}

// resolveFileVariables processes each file-scoped variable that contains system variable placeholders
func (c *Client) resolveFileVariables(
	parsedFile *ParsedFile,
	fileScopedSystemVars map[string]string,
) map[string]string {
	resolvedVariables := make(map[string]string)

	for varName, varValue := range parsedFile.FileVariables {
		if isSystemVariablePlaceholder(varValue) {
			resolvedValue := resolveSystemVariablePlaceholder(
				varValue, fileScopedSystemVars, c.currentDotEnvVars, c.programmaticVars)
			parsedFile.FileVariables[varName] = resolvedValue
			resolvedVariables[varName] = resolvedValue
		}
	}

	return resolvedVariables
}

// updateRequestActiveVariables updates all requests' ActiveVariables with resolved values
func (c *Client) updateRequestActiveVariables(requests []*Request, resolvedVariables map[string]string) {
	for _, request := range requests {
		c.updateSingleRequestActiveVariables(request, resolvedVariables)
	}
}

// updateSingleRequestActiveVariables updates a single request's ActiveVariables with resolved values
func (*Client) updateSingleRequestActiveVariables(request *Request, resolvedVariables map[string]string) {
	if request.ActiveVariables == nil {
		return
	}

	for varName, resolvedValue := range resolvedVariables {
		if _, exists := request.ActiveVariables[varName]; exists {
			request.ActiveVariables[varName] = resolvedValue
		}
	}
}

// isSystemVariablePlaceholder checks if a string contains a system variable placeholder
func isSystemVariablePlaceholder(value string) bool {
	if !strings.HasPrefix(value, "{{") || !strings.HasSuffix(value, "}}") {
		return false
	}

	innerDirective := strings.TrimSpace(value[2 : len(value)-2])
	return strings.HasPrefix(innerDirective, "$")
}

// resolveSystemVariablePlaceholder resolves a system variable placeholder to its value
func resolveSystemVariablePlaceholder(
	placeholder string,
	systemVars map[string]string,
	dotEnvVars map[string]string,
	programmaticVars map[string]any,
) string {
	innerDirective := strings.TrimSpace(placeholder[2 : len(placeholder)-2])

	// Check if it's a simple system variable that we have pre-generated
	if val, ok := systemVars[innerDirective]; ok {
		return val
	}

	// For dynamic system variables, use the existing substitution logic
	return substituteDynamicSystemVariables(placeholder, dotEnvVars, programmaticVars)
}

// executeRequest sends a given Request and returns the Response.
// Errors during execution (e.g. network, body read) are captured in Response.Error.
// A non-nil error is returned by this function only for critical pre-execution
// failures (e.g. nil request, bad BaseURL).
func (c *Client) doHTTPRequest(ctx context.Context, rcRequest *Request) (*Response, error) {
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
	rcRequest.URL, err = c.resolveRequestURL(c.BaseURL, rcRequest.URL, rcRequest.RawURLString)
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
		loopAliases, loopIndex := currentLoopBindings(restClientReq)
		resolvedContent := resolveVariablesInText(content, resolveContext{
			programmaticVars: c.programmaticVars, fileScopedVars: restClientReq.ActiveVariables,
			environmentVars: parsedFile.EnvironmentVariables, globalVars: parsedFile.GlobalVariables,
			systemVars: requestScopedSystemVars, osEnvGetter: osEnvGetter,
			dotEnvVars: c.currentDotEnvVars, responseMap: parsedFile.ResponseMap,
			loopItemAliases: loopAliases, loopIndex: loopIndex,
		})
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

// loadDotEnvVars loads .env and .env.<name> (WithEnvName) from the request
// file directory into the client's dotenv vars.
func (c *Client) loadDotEnvVars(requestFilePath string) {
	c.currentDotEnvVars = loadDotEnvDir(filepath.Dir(requestFilePath), c.envName)
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

	// Substituted request headers carry the oauth2 directive; it is replaced
	// here before the request is sent, leaving the resolved Bearer token as
	// the actual Authorization header.
	if err := c.applyOAuth2(restClientReq, parsedFile, osEnvGetter); err != nil {
		return &Response{Request: restClientReq, Error: err}, fmt.Errorf(
			"oauth2 token fetch failed for request %s (index %d): %w",
			restClientReq.Name, index, err)
	}

	// Execute the HTTP request
	resp, execErr := c.doHTTPRequest(ctx, restClientReq)
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
	loopAliases, loopIndex := currentLoopBindings(restClientReq)
	resolvedBody := resolveVariablesInText(restClientReq.RawBody, resolveContext{
		programmaticVars: c.programmaticVars, fileScopedVars: restClientReq.ActiveVariables,
		environmentVars: parsedFile.EnvironmentVariables, globalVars: parsedFile.GlobalVariables,
		systemVars: requestScopedSystemVars, osEnvGetter: osEnvGetter,
		dotEnvVars: c.currentDotEnvVars, responseMap: parsedFile.ResponseMap,
		loopItemAliases: loopAliases, loopIndex: loopIndex,
	})
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
