package restclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
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
