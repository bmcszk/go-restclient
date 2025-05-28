package restclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	// "regexp" // Unused

	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/joho/godotenv"
)

// Client is the main struct for interacting with the REST client library.
// It will hold configuration and methods to execute requests.
type Client struct {
	httpClient        *http.Client
	BaseURL           string
	DefaultHeaders    http.Header
	currentDotEnvVars map[string]string
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
			// Default to a new client if nil is provided
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

// ExecuteFile parses a request file, executes all requests found, and returns their responses.
// It returns an error if the file cannot be parsed or no requests are found.
// Individual request execution errors are stored within each Response object.
func (c *Client) ExecuteFile(ctx context.Context, requestFilePath string) ([]*Response, error) {
	parsedFile, err := parseRequestFile(requestFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse request file %s: %w", requestFilePath, err)
	}
	if len(parsedFile.Requests) == 0 {
		return nil, fmt.Errorf("no requests found in file %s", requestFilePath)
	}

	// Load .env file from the same directory as the request file
	c.currentDotEnvVars = make(map[string]string)
	envFilePath := filepath.Join(filepath.Dir(requestFilePath), ".env")
	if _, err := os.Stat(envFilePath); err == nil {
		loadedVars, loadErr := godotenv.Read(envFilePath)
		if loadErr != nil {
			// Log or return an error? For now, let's treat a load error as non-fatal,
			// meaning $dotenv vars will just be empty if the file is malformed.
			// Or, we could append to multiErr.
			// For now, just print a warning perhaps, or let it be silent.
			// Let's be silent for now, consistent with undefined vars being empty.
		} else {
			c.currentDotEnvVars = loadedVars
		}
	}

	responses := make([]*Response, len(parsedFile.Requests))
	var multiErr *multierror.Error

	for i, restClientReq := range parsedFile.Requests {
		// Substitute custom variables in RawURLString
		substitutedRawURL := restClientReq.RawURLString
		for k, v := range restClientReq.ActiveVariables {
			placeholder := "{{" + k + "}}"
			substitutedRawURL = strings.ReplaceAll(substitutedRawURL, placeholder, v)
		}

		// Substitute system variables in RawURLString
		substitutedRawURL = c.substituteSystemVariables(substitutedRawURL)

		finalParsedURL, parseErr := url.Parse(substitutedRawURL)
		if parseErr != nil {
			// If URL parsing fails after substitution, this is a critical error for this request
			resp := &Response{Request: restClientReq}
			resp.Error = fmt.Errorf("failed to parse URL after variable substitution: %s (original: %s): %w", substitutedRawURL, restClientReq.RawURLString, parseErr)
			wrappedErr := fmt.Errorf("request %d (%s %s) failed URL parsing: %w", i+1, restClientReq.Method, restClientReq.RawURLString, resp.Error)
			multiErr = multierror.Append(multiErr, wrappedErr)
			responses[i] = resp
			continue // Skip execution for this request
		}
		restClientReq.URL = finalParsedURL // Update the URL with the fully parsed and substituted one

		// Apply variables to Headers and Body
		c.applyVariables(restClientReq, restClientReq.ActiveVariables)

		resp, execErr := c.executeRequest(ctx, restClientReq)
		if execErr != nil {
			if resp == nil {
				resp = &Response{Request: restClientReq}
			}
			if resp.Error == nil {
				resp.Error = execErr
			} else {
				// If resp.Error was already set (e.g., by URL parse failure), wrap execErr if it's new
				resp.Error = fmt.Errorf("execution error: %w (prior error: %s)", execErr, resp.Error)
			}
			wrappedExecErr := fmt.Errorf("request %d (%s %s) failed with critical error: %w", i+1, restClientReq.Method, restClientReq.URL.String(), execErr)
			multiErr = multierror.Append(multiErr, wrappedExecErr)
		}

		if resp != nil && resp.Error != nil {
			// Ensure URL string in error message is from the substituted URL if available, otherwise RawURLString
			urlForError := restClientReq.RawURLString
			if restClientReq.URL != nil {
				urlForError = restClientReq.URL.String()
			}
			wrappedRespErr := fmt.Errorf("request %d (%s %s) failed: %w", i+1, restClientReq.Method, urlForError, resp.Error)
			multiErr = multierror.Append(multiErr, wrappedRespErr)
		}
		responses[i] = resp
	}

	return responses, multiErr.ErrorOrNil()
}

// applyVariables substitutes custom placeholders in the request's Headers, and Body.
// It then substitutes system variables.
func (c *Client) applyVariables(req *Request, vars map[string]string) {
	if req == nil {
		return
	}

	// Substitute custom variables in Headers
	if len(vars) > 0 {
		for headerKey, headerValues := range req.Headers {
			newValues := make([]string, len(headerValues))
			for i, val := range headerValues {
				tempVal := val
				for key, value := range vars {
					placeholder := "{{" + key + "}}"
					tempVal = strings.ReplaceAll(tempVal, placeholder, value)
				}
				newValues[i] = tempVal
			}
			req.Headers[headerKey] = newValues
		}
	}

	// Substitute system variables in Headers
	for headerKey, headerValues := range req.Headers {
		newValues := make([]string, len(headerValues))
		for i, val := range headerValues {
			newValues[i] = c.substituteSystemVariables(val)
		}
		req.Headers[headerKey] = newValues
	}

	// Substitute in RawBody
	if req.RawBody != "" {
		bodyChanged := false
		currentBody := req.RawBody

		// Substitute custom variables
		if len(vars) > 0 {
			tempCustomSubstBody := currentBody
			for key, value := range vars {
				placeholder := "{{" + key + "}}"
				tempCustomSubstBody = strings.ReplaceAll(tempCustomSubstBody, placeholder, value)
			}
			if tempCustomSubstBody != currentBody {
				currentBody = tempCustomSubstBody
				bodyChanged = true
			}
		}

		// Substitute system variables
		tempSystemSubstBody := c.substituteSystemVariables(currentBody)
		if tempSystemSubstBody != currentBody {
			currentBody = tempSystemSubstBody
			bodyChanged = true
		}

		if bodyChanged {
			req.RawBody = currentBody
			// Important: Update the Body io.Reader as well
			req.Body = strings.NewReader(req.RawBody)
		}
	}
}

// substituteSystemVariables replaces system variable placeholders in a string.
func (c *Client) substituteSystemVariables(text string) string {
	// Handle {{$randomInt min max}} - MUST be before {{$randomInt}} (no-args)
	reRandomIntWithArgs := regexp.MustCompile(`\{\{\$randomInt\s+(-?\d+)\s+(-?\d+)\}\}`)
	text = reRandomIntWithArgs.ReplaceAllStringFunc(text, func(match string) string {
		parts := reRandomIntWithArgs.FindStringSubmatch(match)
		if len(parts) == 3 { // parts[0] is full match, parts[1] is min, parts[2] is max
			min, errMin := strconv.Atoi(parts[1])
			max, errMax := strconv.Atoi(parts[2])
			if errMin == nil && errMax == nil {
				if min > max { // Swap if min > max
					min, max = max, min
				}
				return strconv.Itoa(rand.Intn(max-min+1) + min)
			}
		}
		return match // Malformed or error, leave as is
	})

	// Handle {{$randomInt}} (no arguments, defaults to 0-100)
	reRandomIntNoArgs := regexp.MustCompile(`\{\{\$randomInt\}\}`)
	text = reRandomIntNoArgs.ReplaceAllStringFunc(text, func(match string) string {
		return strconv.Itoa(rand.Intn(101)) // 0-100 inclusive
	})

	// Handle {{$guid}}
	for strings.Contains(text, "{{$guid}}") {
		text = strings.Replace(text, "{{$guid}}", uuid.NewString(), 1)
	}

	// Handle {{$processEnv variableName}}
	// Regex to find {{$processEnv ENV_VAR_NAME}}
	// The variable name must start with a letter or underscore, followed by letters, numbers, or underscores.
	reProcessEnv := regexp.MustCompile(`{{\$processEnv\s+([a-zA-Z_][a-zA-Z0-9_]*)\}\}`)
	text = reProcessEnv.ReplaceAllStringFunc(text, func(match string) string {
		parts := reProcessEnv.FindStringSubmatch(match)
		if len(parts) == 2 { // parts[0] is full match, parts[1] is envVarName
			envVarName := parts[1]
			return os.Getenv(envVarName) // Returns empty string if not found, which is desired behavior
		}
		return match // Should not happen if regex matches, but as a fallback
	})

	// Handle {{$timestamp}}
	// Note: Using ReplaceAll ensures all occurrences are replaced. If multiple timestamps
	// in the same string should be identical, this is correct. If they should be unique
	// (like $guid), a loop with strings.Replace (count 1) would be needed, but for timestamp,
	// all occurrences in one substitution pass having the same value is generally expected.
	if strings.Contains(text, "{{$timestamp}}") {
		text = strings.ReplaceAll(text, "{{$timestamp}}", fmt.Sprintf("%d", time.Now().UTC().Unix()))
	}

	// Handle {{$dotenv variableName}}
	reDotEnv := regexp.MustCompile(`\{\{\$dotenv\s+([a-zA-Z_][a-zA-Z0-9_]*)\}\}`)
	text = reDotEnv.ReplaceAllStringFunc(text, func(match string) string {
		parts := reDotEnv.FindStringSubmatch(match)
		if len(parts) == 2 {
			varName := parts[1]
			if val, ok := c.currentDotEnvVars[varName]; ok {
				return val
			}
			return "" // Variable not found in .env, return empty string
		}
		return match // Should not happen
	})

	// TODO: Add other system variables here like {{$datetime}}, {{$timestamp}}, {{$randomInt}}, etc. when they are unblocked.

	return text
}

// executeRequest sends a given Request and returns the Response.
// Errors during execution (e.g. network, body read) are captured in Response.Error.
// A non-nil error is returned by this function only for critical pre-execution failures (e.g. nil request, bad BaseURL).
func (c *Client) executeRequest(ctx context.Context, rcRequest *Request) (*Response, error) {
	if rcRequest == nil {
		// For a nil request, we can't even populate a Response struct meaningfully.
		return nil, fmt.Errorf("cannot execute a nil request")
	}

	// Initialize a response object upfront to hold results or errors
	clientResponse := &Response{
		Request: rcRequest, // Link the request early
	}

	urlToUse := rcRequest.URL
	if !urlToUse.IsAbs() && c.BaseURL != "" {
		base, err := url.Parse(c.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("invalid BaseURL %s: %w", c.BaseURL, err)
		}
		if rcRequest.URL.Scheme == "" && rcRequest.URL.Host == "" {
			if base.Path != "" && !strings.HasSuffix(base.Path, "/") && !strings.HasPrefix(rcRequest.URL.Path, "/") {
				base.Path += "/"
			}
			urlToUse = base.ResolveReference(rcRequest.URL)
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, rcRequest.Method, urlToUse.String(), rcRequest.Body)
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
		httpReq.Header.Del(key)
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}
	if httpReq.Header.Get("Host") == "" && httpReq.URL.Host != "" {
		httpReq.Host = httpReq.URL.Host
	}

	startTime := time.Now()
	httpResp, err := c.httpClient.Do(httpReq)
	duration := time.Since(startTime)
	clientResponse.Duration = duration // Set duration regardless of http error

	if err != nil {
		clientResponse.Error = fmt.Errorf("http request failed: %w", err)
		// Attempt to get some info from httpResp even if err != nil (e.g. if it's a redirect error httpClient is configured not to follow)
		if httpResp != nil {
			clientResponse.Status = httpResp.Status
			clientResponse.StatusCode = httpResp.StatusCode
			clientResponse.Proto = httpResp.Proto
			clientResponse.Headers = httpResp.Header
			// Don't try to read body if there was an error from Do(), as httpResp.Body might be nil or invalid.
			// But ensure it's closed if non-nil to prevent resource leaks.
			defer func() { _ = httpResp.Body.Close() }()
		}
		return clientResponse, nil // Return response with error populated
	}
	defer func() { _ = httpResp.Body.Close() }()

	// 4. Capture response details into clientResponse
	clientResponse.Status = httpResp.Status
	clientResponse.StatusCode = httpResp.StatusCode
	clientResponse.Proto = httpResp.Proto
	clientResponse.Headers = httpResp.Header
	clientResponse.Size = httpResp.ContentLength

	bodyBytes, readErr := io.ReadAll(httpResp.Body)
	if readErr != nil {
		clientResponse.Error = fmt.Errorf("failed to read response body: %w", readErr)
		// BodyBytes will be nil or partial, BodyString will be empty or partial
		// Still return clientResponse with the error
	} else {
		clientResponse.Body = bodyBytes
		clientResponse.BodyString = string(bodyBytes)
	}

	// TODO: Populate TLS details if applicable
	// (Requires inspecting httpResp.TLS which is *tls.ConnectionState)
	if httpResp.TLS != nil {
		clientResponse.IsTLS = true
		// Basic TLS info, more can be added from httpResp.TLS
		switch httpResp.TLS.Version {
		case tls.VersionTLS10:
			clientResponse.TLSVersion = "TLS 1.0"
		case tls.VersionTLS11:
			clientResponse.TLSVersion = "TLS 1.1"
		case tls.VersionTLS12:
			clientResponse.TLSVersion = "TLS 1.2"
		case tls.VersionTLS13:
			clientResponse.TLSVersion = "TLS 1.3"
		default:
			clientResponse.TLSVersion = "unknown"
		}
		clientResponse.TLSCipherSuite = tls.CipherSuiteName(httpResp.TLS.CipherSuite)
	}

	return clientResponse, nil
}

// TODO: Add other public methods as needed, e.g.:
// - Execute(request *Request) (*Response, error) for programmatic requests
// - Methods for setting request-specific options (timeout, retries etc.)
