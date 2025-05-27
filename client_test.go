package restclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"text/template"

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

// createTestFileFromTemplate processes a template file and returns the path to the processed file.
func createTestFileFromTemplate(t *testing.T, templatePath string, data interface{}) string {
	t.Helper()
	tmplContent, err := os.ReadFile(templatePath)
	require.NoError(t, err)

	tmpl, err := template.New("testfile").Parse(string(tmplContent))
	require.NoError(t, err)

	tempFile, err := os.CreateTemp(t.TempDir(), "processed_*.http")
	require.NoError(t, err)

	err = tmpl.Execute(tempFile, data)
	require.NoError(t, err)

	err = tempFile.Close()
	require.NoError(t, err)

	return tempFile.Name()
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
	processedFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/multiple_requests.http", struct{ ServerURL string }{ServerURL: server.URL})

	responses, err := client.ExecuteFile(context.Background(), processedFilePath)
	require.NoError(t, err)
	require.Len(t, responses, 2)
	assert.Equal(t, 2, requestCounter, "Server should have received two requests")

	resp1 := responses[0]
	assert.NoError(t, resp1.Error)
	assert.Equal(t, http.StatusOK, resp1.StatusCode)
	assert.Equal(t, "response1", resp1.BodyString)

	// Define expected response for request 1 & 2 in a single file
	expectedFilePath := "testdata/http_response_files/client_multiple_requests_expected.hresp"

	validationErr := ValidateResponses(context.Background(), expectedFilePath, resp1, responses[1])
	assert.NoError(t, validationErr, "Validation errors for responses should be nil")

	resp2 := responses[1]
	assert.NoError(t, resp2.Error)
	assert.Equal(t, http.StatusCreated, resp2.StatusCode)
	assert.Equal(t, "response2", resp2.BodyString)
}

func TestExecuteFile_RequestWithError(t *testing.T) {
	// serverURL := "http://localhost:12346" // Non-existent server for first request - This is now in the .http file
	server2 := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "good response")
	})
	defer server2.Close()

	client, _ := NewClient()
	processedFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/request_with_error.http", struct{ ServerURL string }{ServerURL: server2.URL})

	responses, err := client.ExecuteFile(context.Background(), processedFilePath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 error occurred:")
	assert.Contains(t, err.Error(), "http request failed")
	assert.Contains(t, err.Error(), "request 1 (GET http://localhost:12346/bad) failed")

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
	_, err := client.ExecuteFile(context.Background(), "testdata/http_request_files/parse_error.http")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse request file")
}

func TestExecuteFile_NoRequestsInFile(t *testing.T) {
	client, _ := NewClient()
	_, err := client.ExecuteFile(context.Background(), "testdata/http_request_files/no_requests.http")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid requests found in file")
}

func TestExecuteFile_ValidThenInvalidSyntax(t *testing.T) {
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/first" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "response from /first")
		} else if r.Method == "INVALID_METHOD" && r.URL.Path == "/second" {
			// The Go http server by default will respond with 501 Not Implemented
			// if it receives a method it doesn't understand, or 405 if the handler is more specific.
			// httptest.Server uses DefaultServeMux which would result in 404 if no path matches,
			// but if a path *could* match but method doesn't, it's 405.
			// Let's assume the default http server behavior for an unknown method is 501.
			w.WriteHeader(http.StatusNotImplemented)
			fmt.Fprint(w, "method not implemented")
		} else {
			t.Logf("Mock server received UNEXPECTED request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusTeapot)
		}
	})
	defer server.Close()

	client, _ := NewClient()
	tempFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/valid_then_invalid_syntax.http", struct{ ServerURL string }{ServerURL: server.URL})

	responses, err := client.ExecuteFile(context.Background(), tempFilePath)

	// ExecuteFile itself should not return an error, as parsing succeeds and requests are attempted.
	// Errors from server (like 501) are captured in the Response object, not as a Go error from ExecuteFile directly unless it's a client-side execution failure (e.g. network unreachable)
	require.NoError(t, err, "ExecuteFile should not return an error if requests are merely rejected by server")

	require.Len(t, responses, 2, "Should have two response objects")

	// First response should be successful
	resp1 := responses[0]
	require.NotNil(t, resp1, "First response object should not be nil")
	assert.NoError(t, resp1.Error, "Error in first response object should be nil")
	assert.Equal(t, http.StatusOK, resp1.StatusCode, "Status code for first response should be OK")
	assert.Equal(t, "response from /first", resp1.BodyString)

	// Second response should indicate server error (e.g., 501 Not Implemented)
	resp2 := responses[1]
	require.NotNil(t, resp2, "Second response object should not be nil")
	assert.NoError(t, resp2.Error, "Error in second object should be nil as it's a server response, not client-side exec error")
	assert.Equal(t, http.StatusNotImplemented, resp2.StatusCode, "Status code for second response should be Not Implemented")
	assert.Contains(t, resp2.BodyString, "method not implemented", "Body for second response should indicate method error")
}

func TestExecuteFile_MultipleErrors(t *testing.T) {
	client, _ := NewClient()
	filePath := "testdata/http_request_files/multiple_errors.http"

	responses, err := client.ExecuteFile(context.Background(), filePath)

	// Check the combined error from ExecuteFile
	require.Error(t, err, "Expected an error from ExecuteFile when multiple requests fail")
	// Using multierror, the message often starts with "X errors occurred:"
	// We are checking for the presence of specific error messages for each request.
	// The exact count "2 errors occurred" might be brittle if multierror changes formatting slightly,
	// so we check for the presence of individual failure messages.
	assert.Contains(t, err.Error(), "request 1 (GET http://localhost:12347/badreq1) failed", "Error message should contain info about first failed request")
	// Check for port and connection refused, being flexible about IPv4/IPv6 loopback
	assert.Contains(t, err.Error(), ":12347: connect: connection refused", "Error message should contain specific connection error for first request")
	assert.Contains(t, err.Error(), "request 2 (POST http://localhost:12348/badreq2) failed", "Error message should contain info about second failed request")
	assert.Contains(t, err.Error(), ":12348: connect: connection refused", "Error message should contain specific connection error for second request")

	// Check individual response objects
	require.Len(t, responses, 2, "Should receive two response objects, even if they contain errors")

	resp1 := responses[0]
	require.NotNil(t, resp1, "First response object should not be nil")
	assert.Error(t, resp1.Error, "Error in first response object should be set")
	assert.Contains(t, resp1.Error.Error(), ":12347: connect: connection refused")

	resp2 := responses[1]
	require.NotNil(t, resp2, "Second response object should not be nil")
	assert.Error(t, resp2.Error, "Error in second response object should be set")
	assert.Contains(t, resp2.Error.Error(), ":12348: connect: connection refused")
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
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 error occurred:")
	assert.Contains(t, err.Error(), "unsupported protocol scheme")
	assert.Contains(t, err.Error(), "request 1 (INVALIDMETHOD /test) failed")

	require.Len(t, responses, 1)

	resp1 := responses[0]
	assert.Error(t, resp1.Error, "Expected an error for invalid method/scheme")
	// The error comes from httpClient.Do() due to the malformed request (non-standard method with path-only URL)
	assert.Contains(t, resp1.Error.Error(), "unsupported protocol scheme", "Error message should indicate unsupported protocol scheme")
	// Check for the method string as it appears in the error message
	assert.Contains(t, resp1.Error.Error(), "Invalidmethod", "Error message should contain the problematic method string as used")
}

func TestExecuteFile_MultipleRequests_GreaterThanTwo(t *testing.T) {
	// Test setup similar to TestExecuteFile_MultipleRequests but with 3+ requests
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cIdx := atomic.AddInt32(&requestCount, 1)
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		t.Logf("Mock server received request #%d: %s %s, Body: %s", cIdx, r.Method, r.URL.Path, string(body))

		switch r.URL.Path {
		case "/req1":
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "response1")
		case "/req2":
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, "response2")
		case "/req3":
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, "response3")
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, _ := NewClient()
	requestFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/multiple_requests_gt2.http", struct{ ServerURL string }{ServerURL: server.URL})

	actualResponses, err := client.ExecuteFile(context.Background(), requestFilePath)
	require.NoError(t, err)
	require.Len(t, actualResponses, 3, "Should have received 3 responses")

	// Validate using the existing expected response file
	expectedResponseFilePath := "testdata/http_response_files/multiple_responses_gt2_expected.http"
	// previousExpectedContent, err := os.ReadFile(expectedResponseFilePath) // Removed
	// require.NoError(t, err) // Removed
	// tempExpectedFile := writeExpectedResponseFile(t, string(previousExpectedContent)) // Removed

	validationErr := ValidateResponses(context.Background(), expectedResponseFilePath, actualResponses...)
	assert.NoError(t, validationErr, "Validation against multiple_responses_gt2_expected.http failed")
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
