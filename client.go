package restclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is the main struct for interacting with the REST client library.
// It will hold configuration and methods to execute requests.
type Client struct {
	// httpClient is the underlying HTTP client used to make requests.
	httpClient *http.Client
	// BaseURL can be used to set a common base URL for all requests.
	BaseURL string
	// DefaultHeaders can be used to set headers that apply to all requests.
	DefaultHeaders http.Header
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
			// Or return an error, TBD
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
	parsedFile, err := ParseRequestFile(requestFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse request file %s: %w", requestFilePath, err)
	}
	if len(parsedFile.Requests) == 0 {
		return nil, fmt.Errorf("no requests found in file %s", requestFilePath)
	}

	responses := make([]*Response, 0, len(parsedFile.Requests))
	for _, restClientReq := range parsedFile.Requests {
		// Each executeRequest call will now handle its own errors internally by populating resp.Error
		resp, _ := c.executeRequest(ctx, restClientReq) // Top-level error from executeRequest is mostly for critical failures
		responses = append(responses, resp)
	}

	return responses, nil
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

	// 1. Construct http.Request from restclient.Request
	urlToUse := rcRequest.URL
	if !urlToUse.IsAbs() && c.BaseURL != "" {
		base, err := url.Parse(c.BaseURL)
		if err != nil {
			// This is a client configuration error, critical.
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
		return clientResponse, nil // Return partially filled response with error
	}

	// 2. Apply DefaultHeaders and then request-specific headers
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

	// 3. Execute request using c.httpClient
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
