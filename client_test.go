package restclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"text/template"

	"github.com/google/uuid"
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
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			bodyBytes, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.JSONEq(t, `{"key": "value"}`, string(bodyBytes))
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

	validationErr := ValidateResponses(expectedFilePath, resp1, responses[1])
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
	_, err := client.ExecuteFile(context.Background(), "testdata/http_request_files/comment_only_file.http")
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

	require.Error(t, err, "Expected an error from ExecuteFile when multiple requests fail")
	assert.Contains(t, err.Error(), "request 1 (GET http://localhost:12347/badreq1) failed", "Error message should contain info about first failed request")
	assert.Contains(t, err.Error(), ":12347: connect: connection refused", "Error message should contain specific connection error for first request")
	assert.Contains(t, err.Error(), "request 2 (POST http://localhost:12348/badreq2) failed", "Error message should contain info about second failed request")
	assert.Contains(t, err.Error(), ":12348: connect: connection refused", "Error message should contain specific connection error for second request")

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

func TestExecuteFile_CapturesResponseHeaders(t *testing.T) {
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.Header().Add("X-Custom-Header", "value1")
		w.Header().Add("X-Custom-Header", "value2")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "{\"data\": \"headers test\"}")
	})
	defer server.Close()

	client, _ := NewClient()
	content := "GET " + server.URL + "/testheaders"
	tempFile, err := os.CreateTemp("", "test_headers_*.rest")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tempFile.Name()) }()
	_, err = tempFile.WriteString(content)
	require.NoError(t, err)
	_ = tempFile.Close()

	responses, err := client.ExecuteFile(context.Background(), tempFile.Name())
	require.NoError(t, err)
	require.Len(t, responses, 1)

	resp := responses[0]
	require.NotNil(t, resp)
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	assert.Equal(t, "application/vnd.api+json", resp.Headers.Get("Content-Type"))
	assert.Equal(t, []string{"value1", "value2"}, resp.Headers["X-Custom-Header"]) // Check multi-value header
	assert.Empty(t, resp.Headers.Get("Non-Existent-Header"))
}

func TestExecuteFile_SimpleGetHTTP(t *testing.T) {
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

	responses, err := client.ExecuteFile(context.Background(), "testdata/http_request_files/invalid_method.http")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 error occurred:")
	assert.Contains(t, err.Error(), "unsupported protocol scheme")
	assert.Contains(t, err.Error(), "request 1 (INVALIDMETHOD /test) failed")

	require.Len(t, responses, 1)

	resp1 := responses[0]
	assert.Error(t, resp1.Error, "Expected an error for invalid method/scheme")
	assert.Contains(t, resp1.Error.Error(), "unsupported protocol scheme", "Error message should indicate unsupported protocol scheme")
	assert.Contains(t, resp1.Error.Error(), "Invalidmethod", "Error message should contain the problematic method string as used")
}

func TestExecuteFile_MultipleRequests_GreaterThanTwo(t *testing.T) {
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

	validationErr := ValidateResponses(expectedResponseFilePath, actualResponses...)
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

func TestExecuteFile_WithCustomVariables(t *testing.T) {
	var requestCount int32
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		currentCount := atomic.AddInt32(&requestCount, 1)
		t.Logf("Mock server received request #%d: %s %s", currentCount, r.Method, r.URL.Path)
		switch r.URL.Path {
		case "/users/testuser123": // SCENARIO-LIB-013-001, SCENARIO-LIB-013-002, SCENARIO-LIB-013-003
			assert.Equal(t, http.MethodPost, r.Method)
			bodyBytes, _ := io.ReadAll(r.Body)
			assert.JSONEq(t, `{"id": "testuser123"}`, string(bodyBytes))
			assert.Equal(t, "Bearer secret-token-value", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "response for user testuser123")
		case "/products/testuser123": // SCENARIO-LIB-013-004 (variable override for pathSegment)
			assert.Equal(t, http.MethodGet, r.Method)
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "response from products/testuser123")
		case "/items/{{undefined_path_var}}": // SCENARIO-LIB-013-005 (undefined variable left as-is in path)
			assert.Equal(t, http.MethodGet, r.Method)
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "response for items (undefined_path_var)")
		default:
			t.Errorf("Unexpected request path to mock server: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer server.Close()

	client, _ := NewClient()

	requestFileContent := fmt.Sprintf(`
@fullServerUrl = %s
@pathSegment = users
@userId = testuser123
@token = secret-token-value

# Request 1: Uses fullServerUrl, pathSegment, userId, token
POST {{fullServerUrl}}/{{pathSegment}}/{{userId}}
Authorization: Bearer {{token}}
Content-Type: application/json

{
  "id": "{{userId}}"
}

###
# Request 2: Override pathSegment, still uses fullServerUrl
@pathSegment = products
GET {{fullServerUrl}}/{{pathSegment}}/{{userId}}

###
# Request 3: Undefined variable in path, still uses fullServerUrl
GET {{fullServerUrl}}/items/{{undefined_path_var}}

`, server.URL) // Use full server.URL

	tempFile, err := os.CreateTemp(t.TempDir(), "test_vars_*.http")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tempFile.Name()) }()
	_, err = tempFile.WriteString(requestFileContent)
	require.NoError(t, err)
	_ = tempFile.Close()

	responses, err := client.ExecuteFile(context.Background(), tempFile.Name())
	require.NoError(t, err, "ExecuteFile should not return an error for variable processing")
	require.Len(t, responses, 3, "Expected 3 responses")
	assert.EqualValues(t, 3, atomic.LoadInt32(&requestCount), "Mock server should have been hit 3 times")

	// Check response 1 (SCENARIO-LIB-013-001, SCENARIO-LIB-013-002, SCENARIO-LIB-013-003)
	resp1 := responses[0]
	assert.NoError(t, resp1.Error)
	assert.Equal(t, http.StatusOK, resp1.StatusCode)
	assert.Equal(t, "response for user testuser123", resp1.BodyString)

	// Check response 2 (SCENARIO-LIB-013-004)
	resp2 := responses[1]
	assert.NoError(t, resp2.Error)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	assert.Equal(t, "response from products/testuser123", resp2.BodyString)

	// Check response 3 (SCENARIO-LIB-013-005)
	resp3 := responses[2]
	assert.NoError(t, resp3.Error)
	assert.Equal(t, http.StatusOK, resp3.StatusCode)
	assert.Equal(t, "response for items (undefined_path_var)", resp3.BodyString)
}

func TestExecuteFile_WithGuidSystemVariable(t *testing.T) {
	var interceptedRequest struct {
		URL    string
		Header string
		Body   string
	}

	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		interceptedRequest.URL = r.URL.String()
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequest.Body = string(bodyBytes)
		interceptedRequest.Header = r.Header.Get("X-Request-ID")
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client, _ := NewClient()

	requestFileContent := fmt.Sprintf(`
GET %s/users/{{$guid}}
User-Agent: test-client
X-Request-ID: {{$guid}}

{
  "transactionId": "{{$guid}}",
  "correlationId": "{{$guid}}"
}
`, server.URL)

	tempFile, err := os.CreateTemp(t.TempDir(), "test_guid_*.http")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tempFile.Name()) }()
	_, err = tempFile.WriteString(requestFileContent)
	require.NoError(t, err)
	_ = tempFile.Close()

	responses, err := client.ExecuteFile(context.Background(), tempFile.Name())
	require.NoError(t, err, "ExecuteFile should not return an error for GUID processing")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// SCENARIO-LIB-014-001: {{$guid}} in URL
	// Example URL: /users/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	urlParts := strings.Split(interceptedRequest.URL, "/")
	require.True(t, len(urlParts) >= 2, "URL should have at least two parts after splitting by /")
	guidFromURL := urlParts[len(urlParts)-1]
	_, err = uuid.Parse(guidFromURL)
	assert.NoError(t, err, "GUID from URL should be a valid UUID: %s", guidFromURL)

	// SCENARIO-LIB-014-002: {{$guid}} in header
	guidFromHeader := interceptedRequest.Header
	_, err = uuid.Parse(guidFromHeader)
	assert.NoError(t, err, "GUID from X-Request-ID header should be a valid UUID: %s", guidFromHeader)

	// SCENARIO-LIB-014-003: {{$guid}} in body
	// SCENARIO-LIB-014-004: Multiple {{$guid}} in one request yield different GUIDs
	var bodyJSON map[string]string
	err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal response body JSON")

	guidFromBody1, ok1 := bodyJSON["transactionId"]
	require.True(t, ok1, "transactionId not found in body")
	_, err = uuid.Parse(guidFromBody1)
	assert.NoError(t, err, "GUID from body (transactionId) should be a valid UUID: %s", guidFromBody1)

	guidFromBody2, ok2 := bodyJSON["correlationId"]
	require.True(t, ok2, "correlationId not found in body")
	_, err = uuid.Parse(guidFromBody2)
	assert.NoError(t, err, "GUID from body (correlationId) should be a valid UUID: %s", guidFromBody2)

	// Check all GUIDs are different
	assert.NotEqual(t, guidFromURL, guidFromHeader, "GUID from URL and header should be different")
	assert.NotEqual(t, guidFromURL, guidFromBody1, "GUID from URL and body1 should be different")
	assert.NotEqual(t, guidFromURL, guidFromBody2, "GUID from URL and body2 should be different")
	assert.NotEqual(t, guidFromHeader, guidFromBody1, "GUID from header and body1 should be different")
	assert.NotEqual(t, guidFromHeader, guidFromBody2, "GUID from header and body2 should be different")
	assert.NotEqual(t, guidFromBody1, guidFromBody2, "GUIDs from body (transactionId and correlationId) should be different")
}

func TestExecuteFile_WithProcessEnvSystemVariable(t *testing.T) {
	// Set up environment variables for the test
	const testEnvVarName = "GO_RESTCLIENT_TEST_VAR"
	const testEnvVarValue = "test_env_value_123"
	const undefinedEnvVarName = "GO_RESTCLIENT_UNDEFINED_VAR"

	err := os.Setenv(testEnvVarName, testEnvVarValue)
	require.NoError(t, err, "Failed to set environment variable for test")
	defer func() { _ = os.Unsetenv(testEnvVarName) }() // Clean up

	// Ensure the undefined variable is indeed not set
	_ = os.Unsetenv(undefinedEnvVarName)

	var interceptedRequest struct {
		URL    string
		Header string
		Body   string
	}

	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		interceptedRequest.URL = r.URL.String()
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequest.Body = string(bodyBytes)
		interceptedRequest.Header = r.Header.Get("X-Env-Value")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	client, _ := NewClient()

	requestFileContent := fmt.Sprintf(`
GET %s/path-{{$processEnv %s}}/data
Content-Type: application/json
Cache-Control: {{$processEnv UNDEFINED_CACHE_VAR_SHOULD_BE_EMPTY}}
User-Agent: test-client
X-Env-Value: {{$processEnv %s}}

{
  "env_payload": "{{$processEnv %s}}",
  "undefined_payload": "{{$processEnv %s}}"
}
`, server.URL, testEnvVarName, testEnvVarName, testEnvVarName, undefinedEnvVarName)

	tempFile, err := os.CreateTemp(t.TempDir(), "test_process_env_*.http")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tempFile.Name()) }()
	_, err = tempFile.WriteString(requestFileContent)
	require.NoError(t, err)
	_ = tempFile.Close()

	responses, err := client.ExecuteFile(context.Background(), tempFile.Name())
	require.NoError(t, err, "ExecuteFile should not return an error for $processEnv processing")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// SCENARIO-LIB-019-001: Correctly substitutes an existing environment variable
	// Check URL
	expectedURL := fmt.Sprintf("/path-%s/data", testEnvVarValue)
	assert.Equal(t, expectedURL, interceptedRequest.URL, "URL should contain substituted env variable")

	// Check Header
	assert.Equal(t, testEnvVarValue, interceptedRequest.Header, "X-Env-Value header should contain substituted env variable")

	// Check Body
	var bodyJSON map[string]string
	err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal request body JSON")

	envPayload, ok := bodyJSON["env_payload"]
	require.True(t, ok, "env_payload not found in body")
	assert.Equal(t, testEnvVarValue, envPayload, "Body env_payload should contain substituted env variable")

	// SCENARIO-LIB-019-002: Substitutes with an empty string if the environment variable is not defined
	// Check undefined in URL (implicitly via full URL check) - the original path segment was `data`
	// Check Header for undefined variable
	// The Cache-Control header in the .http file was Cache-Control: {{$processEnv UNDEFINED_CACHE_VAR_SHOULD_BE_EMPTY}}
	// The actual header sent to the mock server, after substitution, should be 'Cache-Control: '. The value part is empty.
	// However, how Go's http.Request.Header handles this needs care. If the value is empty, it might omit the header or format it as 'Key:'.
	// Let's check the Request object *before* it's sent, specifically the restClientReq.Headers
	// This test is better performed by checking the effective header on the server side if possible, or by inspecting the RawRequest string.
	// For now, let's check the 'undefined_payload' in the body.

	undefinedPayload, ok := bodyJSON["undefined_payload"]
	require.True(t, ok, "undefined_payload not found in body")
	assert.Empty(t, undefinedPayload, "Body undefined_payload should be empty for an undefined env variable")

	// Also explicitly check the Cache-Control header as received by the mock server
	// A header with an empty value after substitution might be sent as "HeaderName: " or omitted.
	// net/http server behavior: if a header `Key: ` is sent, `r.Header.Get("Key")` returns `""`.
	// Let's assume the .http file explicitly defines Cache-Control, it should be present, even if value is empty after substitution.
	actualCacheControl := resp.Request.Headers.Get("Cache-Control")
	assert.Equal(t, "", actualCacheControl, "Cache-Control header value after undefined variable substitution should be empty")

}

func TestExecuteFile_WithDotEnvSystemVariable(t *testing.T) {
	var interceptedRequest struct {
		URL    string
		Header string
		Body   string
	}

	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		interceptedRequest.URL = r.URL.String()
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequest.Body = string(bodyBytes)
		interceptedRequest.Header = r.Header.Get("X-Dotenv-Value")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	client, _ := NewClient()

	// Create a temporary directory for the .http file and .env file
	tempDir := t.TempDir()

	// SCENARIO-LIB-020-001: Variable exists in .env file
	// Create .env file
	dotEnvContent1 := "DOTENV_VAR1=dotenv_value_one\nDOTENV_VAR2=another val from dotenv"
	dotEnvFile1Path := filepath.Join(tempDir, ".env")
	err := os.WriteFile(dotEnvFile1Path, []byte(dotEnvContent1), 0644)
	require.NoError(t, err)

	requestFileContent1 := fmt.Sprintf(`
GET %s/path-{{$dotenv DOTENV_VAR1}}/data
Content-Type: application/json
X-Dotenv-Value: {{$dotenv DOTENV_VAR2}}

{
  "payload": "{{$dotenv DOTENV_VAR1}}",
  "missing_payload": "{{$dotenv MISSING_DOTENV_VAR}}"
}
`, server.URL)

	httpFile1Path := filepath.Join(tempDir, "request1.http")
	err = os.WriteFile(httpFile1Path, []byte(requestFileContent1), 0644)
	require.NoError(t, err)

	responses1, err1 := client.ExecuteFile(context.Background(), httpFile1Path)
	require.NoError(t, err1, "ExecuteFile (scenario 1) should not return an error for $dotenv processing")
	require.Len(t, responses1, 1, "Expected 1 response for scenario 1")

	resp1 := responses1[0]
	assert.NoError(t, resp1.Error)
	assert.Equal(t, http.StatusOK, resp1.StatusCode)

	// Check URL (SCENARIO-LIB-020-001)
	expectedURL1 := "/path-dotenv_value_one/data"
	assert.Equal(t, expectedURL1, interceptedRequest.URL, "URL (scenario 1) should contain substituted dotenv variable")

	// Check Header (SCENARIO-LIB-020-001)
	assert.Equal(t, "another val from dotenv", interceptedRequest.Header, "X-Dotenv-Value header (scenario 1) should contain substituted dotenv variable")

	// Check Body (SCENARIO-LIB-020-001 and SCENARIO-LIB-020-002)
	var bodyJSON1 map[string]string
	err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON1)
	require.NoError(t, err, "Failed to unmarshal request body JSON (scenario 1)")

	dotenvPayload1, ok1 := bodyJSON1["payload"]
	require.True(t, ok1, "payload not found in body (scenario 1)")
	assert.Equal(t, "dotenv_value_one", dotenvPayload1, "Body payload (scenario 1) should contain substituted dotenv variable")

	missingPayload1, ok2 := bodyJSON1["missing_payload"]
	require.True(t, ok2, "missing_payload not found in body (scenario 1)")
	assert.Empty(t, missingPayload1, "Body missing_payload (scenario 1) should be empty for a missing dotenv variable")

	// Clean up .env for the next scenario - remove it so it's not found
	err = os.Remove(dotEnvFile1Path)
	require.NoError(t, err, "Failed to remove .env file for scenario 2 prep")

	// SCENARIO-LIB-020-003: .env file not found
	requestFileContent2 := fmt.Sprintf(`
GET %s/path-{{$dotenv DOTENV_VAR_SHOULD_BE_EMPTY}}/data
User-Agent: test-client

{
  "payload": "{{$dotenv DOTENV_VAR_ALSO_EMPTY}}"
}
`, server.URL)

	httpFile2Path := filepath.Join(tempDir, "request2.http")
	err = os.WriteFile(httpFile2Path, []byte(requestFileContent2), 0644)
	require.NoError(t, err)

	responses2, err2 := client.ExecuteFile(context.Background(), httpFile2Path)
	require.NoError(t, err2, "ExecuteFile (scenario 2) should not return an error if .env not found")
	require.Len(t, responses2, 1, "Expected 1 response for scenario 2")

	resp2 := responses2[0]
	assert.NoError(t, resp2.Error)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	// Check URL (SCENARIO-LIB-020-003)
	expectedURL2 := "/path-/data" // DOTENV_VAR_SHOULD_BE_EMPTY becomes empty
	assert.Equal(t, expectedURL2, interceptedRequest.URL, "URL (scenario 2) should have empty substitution for dotenv variable")

	// Check Body (SCENARIO-LIB-020-003)
	var bodyJSON2 map[string]string
	err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON2)
	require.NoError(t, err, "Failed to unmarshal request body JSON (scenario 2)")

	dotenvPayload2, ok3 := bodyJSON2["payload"]
	require.True(t, ok3, "payload not found in body (scenario 2)")
	assert.Empty(t, dotenvPayload2, "Body payload (scenario 2) should be empty if .env not found")
}

func TestExecuteFile_WithTimestampSystemVariable(t *testing.T) {
	var interceptedRequest struct {
		URL    string
		Header string
		Body   string
	}

	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		interceptedRequest.URL = r.URL.String()
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequest.Body = string(bodyBytes)
		interceptedRequest.Header = r.Header.Get("X-Request-Time")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	client, _ := NewClient()

	// Capture current time to compare against, allowing for slight delay
	beforeTime := time.Now().UTC().Unix()

	requestFileContent := fmt.Sprintf(`
GET %s/events/{{$timestamp}}
Content-Type: application/json
X-Request-Time: {{$timestamp}}
User-Agent: test-client

{
  "event_time": "{{$timestamp}}",
  "processed_at": "{{$timestamp}}"
}
`, server.URL)

	tempFile, err := os.CreateTemp(t.TempDir(), "test_timestamp_*.http")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tempFile.Name()) }()
	_, err = tempFile.WriteString(requestFileContent)
	require.NoError(t, err)
	_ = tempFile.Close()

	responses, err := client.ExecuteFile(context.Background(), tempFile.Name())
	require.NoError(t, err, "ExecuteFile should not return an error for $timestamp processing")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	afterTime := time.Now().UTC().Unix()

	// SCENARIO-LIB-016-001: {{$timestamp}} in URL, header, body
	// Check URL
	urlParts := strings.Split(interceptedRequest.URL, "/")
	require.True(t, len(urlParts) >= 2, "URL path should have at least two parts")
	timestampFromURLStr := urlParts[len(urlParts)-1]
	timestampFromURL, parseErrURL := strconv.ParseInt(timestampFromURLStr, 10, 64)
	assert.NoError(t, parseErrURL, "Timestamp from URL should be a valid integer")
	assert.GreaterOrEqual(t, timestampFromURL, beforeTime, "Timestamp from URL should be >= time before request")
	assert.LessOrEqual(t, timestampFromURL, afterTime, "Timestamp from URL should be <= time after request")

	// Check Header
	timestampFromHeader, parseErrHeader := strconv.ParseInt(interceptedRequest.Header, 10, 64)
	assert.NoError(t, parseErrHeader, "Timestamp from Header should be a valid integer")
	assert.GreaterOrEqual(t, timestampFromHeader, beforeTime, "Timestamp from Header should be >= time before request")
	assert.LessOrEqual(t, timestampFromHeader, afterTime, "Timestamp from Header should be <= time after request")

	// Check Body
	var bodyJSON map[string]string
	err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal request body JSON")

	timestampFromBody1Str, ok1 := bodyJSON["event_time"]
	require.True(t, ok1, "event_time not found in body")
	timestampFromBody1, parseErrBody1 := strconv.ParseInt(timestampFromBody1Str, 10, 64)
	assert.NoError(t, parseErrBody1, "Timestamp from body (event_time) should be valid int")
	assert.GreaterOrEqual(t, timestampFromBody1, beforeTime)
	assert.LessOrEqual(t, timestampFromBody1, afterTime)

	timestampFromBody2Str, ok2 := bodyJSON["processed_at"]
	require.True(t, ok2, "processed_at not found in body")
	timestampFromBody2, parseErrBody2 := strconv.ParseInt(timestampFromBody2Str, 10, 64)
	assert.NoError(t, parseErrBody2, "Timestamp from body (processed_at) should be valid int")
	assert.GreaterOrEqual(t, timestampFromBody2, beforeTime)
	assert.LessOrEqual(t, timestampFromBody2, afterTime)

	// SCENARIO-LIB-016-002: Multiple {{$timestamp}} instances yield the same value for that pass
	assert.Equal(t, timestampFromURL, timestampFromHeader, "Timestamp in URL and Header should be the same for one request pass")
	assert.Equal(t, timestampFromHeader, timestampFromBody1, "Timestamp in Header and Body (event_time) should be the same")
	assert.Equal(t, timestampFromBody1, timestampFromBody2, "Timestamp in Body (event_time and processed_at) should be the same")
}

func TestExecuteFile_WithRandomIntSystemVariable(t *testing.T) {
	var interceptedRequest struct {
		URL    string
		Header string
		Body   string
	}

	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		interceptedRequest.URL = r.URL.String()
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequest.Body = string(bodyBytes)
		interceptedRequest.Header = r.Header.Get("X-Random-ID")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	client, _ := NewClient()

	// Test cases
	tests := []struct {
		name               string
		httpFileContent    string
		validate           func(t *testing.T, url, header, body string)
		expectErrorInExec  bool
		expectErrorInParse bool // For malformed variable syntax causing URL parse issues
	}{ // SCENARIO-LIB-015-001: With valid min max args
		{
			name: "valid min max args",
			httpFileContent: fmt.Sprintf(`
GET %s/item/{{$randomInt 10 20}}/details
X-Random-ID: {{$randomInt 1 5}}

{
  "value": {{$randomInt 100 105}}
}
`, server.URL),
			validate: func(t *testing.T, url, header, body string) {
				urlParts := strings.Split(url, "/")
				valURL, err := strconv.Atoi(urlParts[len(urlParts)-2])
				require.NoError(t, err, "Random int from URL should be valid int")
				assert.True(t, valURL >= 10 && valURL <= 20, "URL random int %d out of range [10,20]", valURL)

				valHeader, err := strconv.Atoi(header)
				require.NoError(t, err, "Random int from Header should be valid int")
				assert.True(t, valHeader >= 1 && valHeader <= 5, "Header random int %d out of range [1,5]", valHeader)

				var bodyJSON map[string]int
				err = json.Unmarshal([]byte(body), &bodyJSON)
				require.NoError(t, err, "Failed to unmarshal body")
				assert.True(t, bodyJSON["value"] >= 100 && bodyJSON["value"] <= 105, "Body random int %d out of range [100,105]", bodyJSON["value"])
			},
		},
		// SCENARIO-LIB-015-002: No args (default 0-100)
		{
			name: "no args",
			httpFileContent: fmt.Sprintf(`
GET %s/item/{{$randomInt}}/default
X-Random-ID: {{$randomInt}}

{
  "value": {{$randomInt}}
}
`, server.URL),
			validate: func(t *testing.T, url, header, body string) {
				urlParts := strings.Split(url, "/")
				valURL, err := strconv.Atoi(urlParts[len(urlParts)-2])
				require.NoError(t, err, "Random int from URL (no args) should be valid int")
				assert.True(t, valURL >= 0 && valURL <= 100, "URL random int (no args) %d out of range [0,100]", valURL)

				valHeader, err := strconv.Atoi(header)
				require.NoError(t, err, "Random int from Header (no args) should be valid int")
				assert.True(t, valHeader >= 0 && valHeader <= 100, "Header random int (no args) %d out of range [0,100]", valHeader)

				var bodyJSON map[string]int
				err = json.Unmarshal([]byte(body), &bodyJSON)
				require.NoError(t, err, "Failed to unmarshal body (no args)")
				assert.True(t, bodyJSON["value"] >= 0 && bodyJSON["value"] <= 100, "Body random int (no args) %d out of range [0,100]", bodyJSON["value"])
			},
		},
		// SCENARIO-LIB-015-003: Swapped min max args
		{
			name: "swapped min max args",
			httpFileContent: fmt.Sprintf(`
GET %s/item/{{$randomInt 30 25}}/swapped

{
  "value": {{$randomInt 90 80}}
}
`, server.URL),
			validate: func(t *testing.T, url, header, body string) {
				urlParts := strings.Split(url, "/")
				valURL, err := strconv.Atoi(urlParts[len(urlParts)-2])
				require.NoError(t, err, "Random int from URL (swapped) should be valid int")
				assert.True(t, valURL >= 25 && valURL <= 30, "URL random int (swapped) %d out of range [25,30]", valURL)

				var bodyJSON map[string]int
				err = json.Unmarshal([]byte(body), &bodyJSON)
				require.NoError(t, err, "Failed to unmarshal body (swapped)")
				assert.True(t, bodyJSON["value"] >= 80 && bodyJSON["value"] <= 90, "Body random int (swapped) %d out of range [80,90]", bodyJSON["value"])
			},
		},
		// SCENARIO-LIB-015-004: Malformed args (non-integer)
		{
			name: "malformed args",
			httpFileContent: fmt.Sprintf(`
GET %s/item/{{$randomInt abc def}}/malformed
X-Random-ID: {{$randomInt 1 xyz}}

{
  "value": "{{$randomInt foo bar}}"
}
`, server.URL),
			validate: func(t *testing.T, urlStr, header, body string) {
				// Placeholder: {{$randomInt abc def}}
				// Expected URL encoding: %7B%7B$randomInt%20abc%20def%7D%7D
				expectedEncodedPlaceholder := "%7B%7B$randomInt%20abc%20def%7D%7D"
				assert.Contains(t, urlStr, expectedEncodedPlaceholder, "URL should contain URL-encoded malformed $randomInt")
				assert.Equal(t, "{{$randomInt 1 xyz}}", header, "Header should retain malformed $randomInt")
				var bodyJSON map[string]string // Expecting string due to non-substitution
				err := json.Unmarshal([]byte(body), &bodyJSON)
				require.NoError(t, err, "Failed to unmarshal body (malformed)")
				assert.Equal(t, "{{$randomInt foo bar}}", bodyJSON["value"], "Body should retain malformed $randomInt")
			},
			expectErrorInParse: false, // Changed: url.Parse might not fail on this. The literal string is sent.
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempFile, err := os.CreateTemp(t.TempDir(), "test_randomint_*.http")
			require.NoError(t, err)
			defer func() { _ = os.Remove(tempFile.Name()) }()
			_, err = tempFile.WriteString(tc.httpFileContent)
			require.NoError(t, err)
			_ = tempFile.Close()

			responses, err := client.ExecuteFile(context.Background(), tempFile.Name())

			if tc.expectErrorInExec {
				require.Error(t, err, "Expected error during ExecuteFile")
				return
			}
			// Some malformed variables might not cause ExecuteFile to error directly,
			// but might cause the individual request.URL to fail parsing later.
			// If tc.expectErrorInParse is true, we check resp.Error
			if !tc.expectErrorInParse {
				require.NoError(t, err, "ExecuteFile returned an unexpected error: %v", err)
			}

			require.Len(t, responses, 1, "Expected 1 response")
			resp := responses[0]

			if tc.expectErrorInParse {
				require.Error(t, resp.Error, "Expected error in response due to parsing/substitution issue")
				assert.Contains(t, resp.Error.Error(), "failed to parse URL after variable substitution")
			} else {
				assert.NoError(t, resp.Error, "Response error should be nil for case: %s", tc.name)
			}

			// If we expected a parse error for the request, validation of substituted values might not be meaningful
			// or possible if the request didn't even reach the server.
			if resp.Error == nil { // Only validate if the request was successful or server responded.
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				tc.validate(t, interceptedRequest.URL, interceptedRequest.Header, interceptedRequest.Body)
			}
		})
	}
}

func TestExecuteFile_WithDatetimeSystemVariable(t *testing.T) {
	var interceptedRequest struct {
		URL    string
		Header string
		Body   string
	}

	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		interceptedRequest.URL = r.URL.String()
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequest.Body = string(bodyBytes)
		interceptedRequest.Header = r.Header.Get("X-Request-DT")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	client, _ := NewClient()

	// Custom format for Go: YYYY-MM-DD HH:mm:ss -> 2006-01-02 15:04:05
	customGoFormat := "2006-01-02 15:04:05"

	tests := []struct {
		name                string
		httpVar             string // The variable part like `rfc1123` or `"YYYY-MM-DD HH:mm:ss"`
		expectedFormat      string // Go time format string for parsing the output, or empty if expecting placeholder
		expectAsPlaceholder bool   // If true, expect the variable to remain as a placeholder
	}{
		{
			name:           "rfc1123",
			httpVar:        "rfc1123",
			expectedFormat: time.RFC1123,
		},
		{
			name:           "iso8601",
			httpVar:        "iso8601",
			expectedFormat: time.RFC3339, // Go's equivalent
		},
		{
			name:           "custom format double quotes",
			httpVar:        fmt.Sprintf("\"%s\"", customGoFormat), // e.g. "2006-01-02 15:04:05"
			expectedFormat: customGoFormat,
		},
		{
			name:                "unknown keyword",
			httpVar:             "badkeyword",
			expectAsPlaceholder: true,
		},
		{
			name:                "malformed custom format string (unclosed quote)",
			httpVar:             "\"2006-01-02", // Unclosed double quote
			expectAsPlaceholder: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fullHttpVar := fmt.Sprintf("{{$datetime %s}}", tc.httpVar)

			requestFileContent := fmt.Sprintf(`
GET %s/events/%s
Content-Type: application/json
X-Request-DT: %s

{
  "event_dt": "%s"
}
`, server.URL, fullHttpVar, fullHttpVar, fullHttpVar)

			tempFile, err := os.CreateTemp(t.TempDir(), "test_datetime_*.http")
			require.NoError(t, err)
			// No defer remove, let each test run clean up or rely on t.TempDir()
			_, err = tempFile.WriteString(requestFileContent)
			require.NoError(t, err)
			_ = tempFile.Close()

			beforeRequest := time.Now().UTC()
			responses, err := client.ExecuteFile(context.Background(), tempFile.Name())
			afterRequest := time.Now().UTC()

			require.NoError(t, err, "ExecuteFile should not error for datetime processing")
			require.Len(t, responses, 1)
			resp := responses[0]
			require.NoError(t, resp.Error) // Expecting successful execution by client
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Validate URL, Header, Body
			valuesToTest := map[string]string{
				"URL":    strings.TrimPrefix(interceptedRequest.URL, "/events/"),
				"Header": interceptedRequest.Header,
			}

			// Special handling for body if it might be invalid JSON due to placeholder
			if tc.name == "malformed custom format string (unclosed quote)" {
				valuesToTest["BodyRaw"] = interceptedRequest.Body // Check raw body string
			} else {
				var bodyJSON map[string]string
				err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON)
				require.NoError(t, err, "Failed to unmarshal body for datetime test: %s. Body: %s", tc.name, interceptedRequest.Body)
				valuesToTest["BodyJSON"] = bodyJSON["event_dt"]
			}

			for K, V := range valuesToTest {
				t.Logf("Test: %s, Key: %s, Value: %s, Expected Placeholder: %t, Expected Format: %s", tc.name, K, V, tc.expectAsPlaceholder, tc.expectedFormat)
				// If the value came from a URL path, it might be URL-encoded. Unescape it first.
				actualValueToTest := V
				if K == "URL" {
					decodedVal, decodeErr := url.PathUnescape(V)
					if decodeErr == nil {
						actualValueToTest = decodedVal
					}
				}

				if tc.expectAsPlaceholder {
					if K == "BodyRaw" { // For raw body check, expect fullHttpVar within the JSON-like structure
						expectedBodyContent := fmt.Sprintf(`{
  "event_dt": "%s"
}`, fullHttpVar)
						assert.Equal(t, expectedBodyContent, actualValueToTest, "BodyRaw should contain the placeholder correctly formatted in JSON string")
					} else {
						assert.Equal(t, fullHttpVar, actualValueToTest, "%s value should be the placeholder %s", K, fullHttpVar)
					}
				} else {
					parsedTime, parseErr := time.Parse(tc.expectedFormat, actualValueToTest)
					assert.NoError(t, parseErr, "%s value '%s' (original '%s') should be parsable with format '%s'", K, actualValueToTest, V, tc.expectedFormat)
					if parseErr == nil {
						// Check if the parsed time is within a reasonable window of the request execution
						// Allow a small delta (e.g., 2 seconds) to account for execution time
						assert.WithinDuration(t, beforeRequest, parsedTime, 2*time.Second, "%s time should be close to request time (before)", K)
						assert.WithinDuration(t, parsedTime, afterRequest.Add(1*time.Second), 2*time.Second, "%s time should be close to request time (after)", K) // Add 1s to after to ensure window
					}
				}
			}
			_ = os.Remove(tempFile.Name()) // Clean up temp file
		})
	}
}
