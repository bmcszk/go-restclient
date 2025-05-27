package restclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create a mock server
func startMockServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func TestNewClient(t *testing.T) {
	c, err := NewClient()
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.NotNil(t, c.httpClient)
	assert.Empty(t, c.BaseURL)
	assert.NotNil(t, c.DefaultHeaders)
	assert.Empty(t, c.DefaultHeaders)
}

func TestNewClient_WithOptions(t *testing.T) {
	customHTTPClient := &http.Client{Timeout: 15 * time.Second} // Note: time not imported yet
	baseURL := "https://api.example.com"
	defaultHeaderKey := "X-Default"
	defaultHeaderValue := "DefaultValue"

	c, err := NewClient(
		WithHTTPClient(customHTTPClient),
		WithBaseURL(baseURL),
		WithDefaultHeader(defaultHeaderKey, defaultHeaderValue),
	)
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, customHTTPClient, c.httpClient)
	assert.Equal(t, baseURL, c.BaseURL)
	assert.Equal(t, defaultHeaderValue, c.DefaultHeaders.Get(defaultHeaderKey))

	// Test nil http client option
	c2, err2 := NewClient(WithHTTPClient(nil))
	require.NoError(t, err2)
	require.NotNil(t, c2.httpClient, "httpClient should default if nil provided")
}

func TestExecuteFile_SingleRequest(t *testing.T) {
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/users", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "user data")
	})
	defer server.Close()

	client, _ := NewClient()
	content := "GET " + server.URL + "/users"
	tempFile, err := os.CreateTemp("", "test_single_*.rest")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tempFile.Name()) }()
	_, err = tempFile.WriteString(content)
	require.NoError(t, err)
	_ = tempFile.Close() // Close the file before ParseRequestFile reads it

	responses, err := client.ExecuteFile(context.Background(), tempFile.Name())
	require.NoError(t, err)
	require.Len(t, responses, 1)
	resp := responses[0]
	require.NotNil(t, resp)
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "user data", resp.BodyString)
}

func TestExecuteFile_MultipleRequests(t *testing.T) {
	var requestCounter int
	server := startMockServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCounter++
		switch r.URL.Path {
		case "/req1":
			assert.Equal(t, http.MethodGet, r.Method)
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "response1")
		case "/req2":
			assert.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprint(w, "response2")
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, _ := NewClient()
	content := fmt.Sprintf("GET %s/req1\n###\nPOST %s/req2", server.URL, server.URL)
	tempFile, err := os.CreateTemp("", "test_multi_*.rest")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tempFile.Name()) }()
	_, err = tempFile.WriteString(content)
	require.NoError(t, err)
	_ = tempFile.Close()

	responses, err := client.ExecuteFile(context.Background(), tempFile.Name())
	require.NoError(t, err)
	require.Len(t, responses, 2)
	assert.Equal(t, 2, requestCounter, "Server should have received two requests")

	resp1 := responses[0]
	assert.NoError(t, resp1.Error)
	assert.Equal(t, http.StatusOK, resp1.StatusCode)
	assert.Equal(t, "response1", resp1.BodyString)

	resp2 := responses[1]
	assert.NoError(t, resp2.Error)
	assert.Equal(t, http.StatusCreated, resp2.StatusCode)
	assert.Equal(t, "response2", resp2.BodyString)
}

func TestExecuteFile_RequestWithError(t *testing.T) {
	serverURL := "http://localhost:12346" // Non-existent server for first request
	server2 := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "good response")
	})
	defer server2.Close()

	client, _ := NewClient()
	content := fmt.Sprintf("GET %s/bad\n###\nGET %s/good", serverURL, server2.URL)
	tempFile, err := os.CreateTemp("", "test_err_*.rest")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tempFile.Name()) }()
	_, err = tempFile.WriteString(content)
	require.NoError(t, err)
	_ = tempFile.Close()

	responses, err := client.ExecuteFile(context.Background(), tempFile.Name())
	require.NoError(t, err) // ExecuteFile itself shouldn't error for per-request errors
	require.Len(t, responses, 2)

	resp1 := responses[0]
	assert.Error(t, resp1.Error)
	assert.Contains(t, resp1.Error.Error(), "http request failed")

	resp2 := responses[1]
	assert.NoError(t, resp2.Error)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	assert.Equal(t, "good response", resp2.BodyString)
}

func TestExecuteFile_ParseError(t *testing.T) {
	client, _ := NewClient()
	content := "# just a comment" // This content will cause ParseRequestFile to return an error
	tempFile, err := os.CreateTemp("", "test_parse_err_*.rest")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tempFile.Name()) }()
	_, err = tempFile.WriteString(content)
	require.NoError(t, err)
	_ = tempFile.Close()

	_, err = client.ExecuteFile(context.Background(), tempFile.Name())
	assert.Error(t, err)
	// The error should be "no valid requests found in file" from the parser
	assert.Contains(t, err.Error(), "no valid requests found in file")
}

func TestExecuteFile_NoRequestsInFile(t *testing.T) {
	client, _ := NewClient()
	content := "# Only comments"
	tempFile, err := os.CreateTemp("", "test_no_reqs_*.rest")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tempFile.Name()) }()
	_, err = tempFile.WriteString(content)
	require.NoError(t, err)
	_ = tempFile.Close()

	_, err = client.ExecuteFile(context.Background(), tempFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid requests found in file")
}

func TestExecuteFile_SimpleGetHTTP(t *testing.T) {
	// The purpose of THIS test is to ensure ExecuteFile
	// correctly parses the file and attempts to make a request.
	// We use a custom http.Client with a Transport that intercepts the request
	// to verify the outgoing http.Request object.

	var interceptedReq *http.Request
	mockTransport := &mockRoundTripper{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			interceptedReq = req.Clone(req.Context()) // Clone to inspect safely

			// Return a dummy response
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("mocked response")),
				Header:     make(http.Header),
			}, nil
		},
	}

	clientWithMockTransport, err := NewClient(WithHTTPClient(&http.Client{Transport: mockTransport}))
	require.NoError(t, err)

	responses, err := clientWithMockTransport.ExecuteFile(context.Background(), "testdata/http_request_files/simple_get.http")
	require.NoError(t, err, "ExecuteFile should not fail")
	require.Len(t, responses, 1, "Expected one response")
	resp := responses[0]
	require.NotNil(t, resp, "Response should not be nil")
	assert.NoError(t, resp.Error, "Response error should be nil")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status OK from mock")

	// Now assert the details of the *intercepted* request
	require.NotNil(t, interceptedReq, "Request should have been intercepted")
	assert.Equal(t, http.MethodGet, interceptedReq.Method, "Expected GET method")
	assert.Equal(t, "https://jsonplaceholder.typicode.com/todos/1", interceptedReq.URL.String(), "Expected full URL from file")
	assert.Empty(t, interceptedReq.Header, "Expected no headers from simple_get.http")
}

func TestExecuteFile_WithBaseURL(t *testing.T) {
	var interceptedReq *http.Request
	mockTransport := &mockRoundTripper{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			interceptedReq = req.Clone(req.Context())
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("mock"))}, nil
		},
	}

	mockServerURL := "http://localhost:12345" // Dummy URL, won't be hit
	client, err := NewClient(
		WithBaseURL(mockServerURL+"/api"),
		WithHTTPClient(&http.Client{Transport: mockTransport}),
	)
	require.NoError(t, err)

	responses, err := client.ExecuteFile(context.Background(), "testdata/http_request_files/relative_path_get.http")
	require.NoError(t, err)
	require.Len(t, responses, 1)
	assert.NoError(t, responses[0].Error)

	require.NotNil(t, interceptedReq)
	// BaseURL is mockServerURL + "/api" = "http://localhost:12345/api"
	// Request URL is "/todos/1"
	// Expected resolved URL is "http://localhost:12345/api/todos/1"
	assert.Equal(t, mockServerURL, interceptedReq.URL.Scheme+"://"+interceptedReq.URL.Host)
	assert.Equal(t, "/api/todos/1", interceptedReq.URL.Path)
}

func TestExecuteFile_WithDefaultHeaders(t *testing.T) {
	var interceptedReq *http.Request
	mockTransport := &mockRoundTripper{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			interceptedReq = req.Clone(req.Context())
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("mock"))}, nil
		},
	}

	client, err := NewClient(
		WithDefaultHeader("X-Default", "default-value"),
		WithDefaultHeader("X-Override", "default-should-be-overridden"),
		WithHTTPClient(&http.Client{Transport: mockTransport}),
		WithBaseURL("http://dummyserver.com"), // Base URL needed for relative path in .http file
	)
	require.NoError(t, err)

	responses, err := client.ExecuteFile(context.Background(), "testdata/http_request_files/get_with_override_header.http")
	require.NoError(t, err)
	require.Len(t, responses, 1)
	assert.NoError(t, responses[0].Error)

	require.NotNil(t, interceptedReq)
	assert.Equal(t, "default-value", interceptedReq.Header.Get("X-Default"))
	assert.Equal(t, "file-value", interceptedReq.Header.Get("X-Override"), "Header from file should override client default")
	assert.Equal(t, "present", interceptedReq.Header.Get("X-File-Only"))
}

func TestExecuteFile_InvalidMethodInFile(t *testing.T) {
	client, _ := NewClient()
	// No mock transport needed as the error occurs when the http.Client tries to execute it.

	responses, err := client.ExecuteFile(context.Background(), "testdata/http_request_files/invalid_method.http")
	require.NoError(t, err) // ExecuteFile itself shouldn't error here
	require.Len(t, responses, 1)

	resp1 := responses[0]
	assert.Error(t, resp1.Error, "Expected an error for invalid method/scheme")
	// The error comes from httpClient.Do() due to the malformed request (non-standard method with path-only URL)
	assert.Contains(t, resp1.Error.Error(), "unsupported protocol scheme", "Error message should indicate unsupported protocol scheme")
	// Check for the method string as it appears in the error message
	assert.Contains(t, resp1.Error.Error(), "Invalidmethod", "Error message should contain the problematic method string as used")
}

// mockRoundTripper is a helper for mocking http.RoundTripper
type mockRoundTripper struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.RoundTripFunc != nil {
		return m.RoundTripFunc(req)
	}
	return nil, fmt.Errorf("RoundTripFunc not set")
}

// TODO: Test TLS details in Response struct (requires HTTPS server and more setup)
