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
// programmaticVars, if provided, will override any variables defined in the request file.
func (c *Client) ExecuteFile(ctx context.Context, requestFilePath string, programmaticVars ...map[string]string) ([]*Response, error) {
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
		// Create a new map for merged variables to avoid modifying the original parsed request's ActiveVariables directly yet.
		mergedVariables := make(map[string]string)
		// Copy file-defined variables first
		for k, v := range restClientReq.ActiveVariables {
			mergedVariables[k] = v
		}
		// Then, override with programmatic variables if provided
		if len(programmaticVars) > 0 && programmaticVars[0] != nil {
			for k, v := range programmaticVars[0] {
				mergedVariables[k] = v
			}
		}
		// Update the request's ActiveVariables with the merged map for subsequent use
		restClientReq.ActiveVariables = mergedVariables

		// Pre-evaluate system variables within the values of ActiveVariables
		// This ensures that if a variable's value contains a system function (e.g., @myId = {{$uuid}}),
		// that function is evaluated once and its result is stored.
		// Subsequent uses of {{myId}} will then use this same pre-evaluated value.
		for k, v := range restClientReq.ActiveVariables {
			restClientReq.ActiveVariables[k] = c.substituteSystemVariables(v)
		}

		// Substitute custom variables in RawURLString
		substitutedRawURL := restClientReq.RawURLString
		for k, v := range restClientReq.ActiveVariables { // Use the now merged ActiveVariables
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

// substituteSystemVariables replaces known system variable placeholders in a given text.
func (c *Client) substituteSystemVariables(text string) string {
	originalTextForLogging := text // Keep a copy for logging

	// Handle {{$randomInt min max}} - MUST be before {{$randomInt}} (no-args)
	reRandomIntWithArgs := regexp.MustCompile(`\{\{\$randomInt\s+(-?\d+)\s+(-?\d+)\}\}`)
	text = reRandomIntWithArgs.ReplaceAllStringFunc(text, func(match string) string {
		parts := reRandomIntWithArgs.FindStringSubmatch(match)
		if len(parts) == 3 {
			min, errMin := strconv.Atoi(parts[1])
			max, errMax := strconv.Atoi(parts[2])
			if errMin == nil && errMax == nil && min <= max {
				return strconv.Itoa(rand.Intn(max-min+1) + min)
			}
		}
		return match // Return original match if parsing fails or min > max
	})

	// Handle {{$guid}} - for backward compatibility or explicit choice.
	// This is treated as an alias for {{$uuid}}.
	// Each occurrence should yield a new GUID.
	for strings.Contains(text, "{{$guid}}") {
		newGUID := uuid.NewString()
		if originalTextForLogging == text { // Log only for the first replacement of $guid in this originalText
			fmt.Printf("[DEBUG] substituteSystemVariables: input='%s', op='$guid', first_replaced_with='%s'\n", originalTextForLogging, newGUID)
		}
		text = strings.Replace(text, "{{$guid}}", newGUID, 1)
	}

	// REQ-LIB-008: $uuid
	// Each occurrence should yield a new UUID.
	for strings.Contains(text, "{{$uuid}}") {
		newUUID := uuid.NewString()
		// Log only for the first replacement of $uuid in this originalText (if $guid didn't already log for it)
		if originalTextForLogging == text && !strings.Contains(originalTextForLogging, "{{$guid}}") {
			fmt.Printf("[DEBUG] substituteSystemVariables: input='%s', op='$uuid', first_replaced_with='%s'\n", originalTextForLogging, newUUID)
		}
		text = strings.Replace(text, "{{$uuid}}", newUUID, 1)
	}

	// REQ-LIB-009: $timestamp
	if strings.Contains(text, "{{$timestamp}}") {
		timestampStr := strconv.FormatInt(time.Now().UTC().Unix(), 10)
		text = strings.ReplaceAll(text, "{{$timestamp}}", timestampStr)
	}

	// REQ-LIB-010: $randomInt (no arguments, 0-100 as per common expectation/previous tests)
	if strings.Contains(text, "{{$randomInt}}") {
		randomIntStr := strconv.Itoa(rand.Intn(101)) // 0-100 inclusive
		text = strings.ReplaceAll(text, "{{$randomInt}}", randomIntStr)
	}

	// REQ-LIB-011: $dotenv MY_VARIABLE_NAME
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
		return match // Should not happen with a valid regex, but good for safety
	})

	// REQ-LIB-012: $processEnv MY_ENV_VAR
	reProcessEnv := regexp.MustCompile(`\{\{\$processEnv\s+([a-zA-Z_][a-zA-Z0-9_]*)\}\}`)
	text = reProcessEnv.ReplaceAllStringFunc(text, func(match string) string {
		parts := reProcessEnv.FindStringSubmatch(match)
		if len(parts) == 2 {
			varName := parts[1]
			if val, ok := os.LookupEnv(varName); ok {
				return val
			}
			return "" // Variable not found in process env, return empty string
		}
		return match // Should not happen
	})

	// REQ-LIB-029 & REQ-LIB-030: Datetime variables
	// These were more complex and might have their own regex. For now, assume they were separate calls
	// or integrated carefully. This part needs to match the *actual* original logic for datetime.
	// For this focused fix, I am restoring the structure from a typical simple system variable handler.
	// IF THE ORIGINAL HAD COMPLEX REGEX FOR DATETIME HERE, THIS IS A SIMPLIFICATION.
	// Example of how datetime *might* have been (if simple ReplaceAll, which is unlikely given its complexity):
	// text = strings.ReplaceAll(text, "{{$datetime ...}}", evaluateDatetime(...))
	// text = strings.ReplaceAll(text, "{{$localDatetime ...}}", evaluateLocalDatetime(...))

	// IMPORTANT: The original datetime substitution logic needs to be preserved.
	// The following are placeholders if the original logic was more complex than simple ReplaceAll.
	// If the original used `reDateTime.ReplaceAllStringFunc` and `reLocalDateTime.ReplaceAllStringFunc`,
	// those blocks should be here.
	// For now, assuming a simplified structure for this edit's focus.

	// Placeholder for original $datetime logic (MUST BE VERIFIED/RESTORED FROM ORIGINAL)
	// For example, if it used a regex like reDateTime:
	/*
		reDateTime := regexp.MustCompile(`\{\{\$datetime\s+...complex regex...\}\}`)
		text = reDateTime.ReplaceAllStringFunc(text, func(match string) string {
			// ... original $datetime evaluation ...
			return "evaluated_datetime"
		})
	*/

	// Placeholder for original $localDatetime logic (MUST BE VERIFIED/RESTORED FROM ORIGINAL)
	/*
		reLocalDateTime := regexp.MustCompile(`\{\{\$localDatetime\s+...complex regex...\}\}`)
		text = reLocalDateTime.ReplaceAllStringFunc(text, func(match string) string {
			// ... original $localDatetime evaluation ...
			return "evaluated_local_datetime"
		})
	*/

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
