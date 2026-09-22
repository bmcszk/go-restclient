package restclient

// HTTP transport, request preparation, and response population for the client.

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/go-multierror"
)

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
	c.populateResponseDetails(clientResponse, httpResp, bodyBytes, readErr)

	return clientResponse, nil
}

// prepareRequestURL handles URL preparation and variable substitution
func (c *Client) prepareRequestURL(rcRequest *Request) error {
	if rcRequest.URL == nil && rcRequest.RawURLString != "" {
		rctx := c.directiveResolveContext(nil, rcRequest, os.LookupEnv,
			c.generateRequestScopedSystemVariables(), nil, -1)
		rctx.dotEnvVars = nil // no .env context for direct single-request execution
		substitutedAndParsedURL, subsErr := substituteRequestVariables(
			rcRequest,
			nil, // parsedFile - no file context for direct executeRequest
			c.BaseURL,
			rctx,
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
		c.populateResponseDetails(clientResponse, httpResp, bodyBytes, doErr)
		if httpResp.Body != nil {
			_ = httpResp.Body.Close()
		}
	}
	// Log critical HTTP errors only
	return clientResponse
}

// populateResponseDetails copies relevant information from an *http.Response and body to our *Response.
func (*Client) populateResponseDetails(resp *Response, httpResp *http.Response, bodyBytes []byte, bodyReadErr error) {
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
