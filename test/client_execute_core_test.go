package test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	rc "github.com/bmcszk/go-restclient"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PRD-COMMENT: FR10.1 - Client Core Execution: Single Request
// Corresponds to: Basic client capability to parse and execute a single valid HTTP request 
// from a .http file (http_syntax.md).
// This test verifies that the client can successfully execute one request defined in 
// 'testdata/http_request_files/single_request.http' and retrieve the response.
func TestExecuteFile_SingleRequest(t *testing.T) {
	// Given
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/users", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "user data")
	})
	defer server.Close()

	client, _ := rc.NewClient()
	requestFilePath := createTestFileFromTemplate(t, 
		"testdata/http_request_files/single_request.http", 
		struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 1)
	resp := responses[0]
	require.NotNil(t, resp)
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "user data", resp.BodyString)
}

// PRD-COMMENT: FR10.2 - Client Core Execution: Multiple Requests
// Corresponds to: Client capability to parse and execute multiple sequential HTTP requests 
// from a single .http file (http_syntax.md "Request Separation").
// This test verifies that the client can execute all requests in 
// 'testdata/http_request_files/multiple_requests.http', collect all responses, and 
// optionally validate them against 
// 'testdata/http_response_files/client_multiple_requests_expected.hresp'.
func TestExecuteFile_MultipleRequests(t *testing.T) {
	// Given
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

	client, _ := rc.NewClient()
	processedFilePath := createTestFileFromTemplate(t, 
		"testdata/http_request_files/multiple_requests.http", 
		struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), processedFilePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 2)
	assert.Equal(t, 2, requestCounter, "Server should have received two requests")

	resp1 := responses[0]
	assert.NoError(t, resp1.Error)
	assert.Equal(t, http.StatusOK, resp1.StatusCode)
	assert.Equal(t, "response1", resp1.BodyString)

	expectedFilePath := "testdata/http_response_files/client_multiple_requests_expected.hresp"
	validationErr := client.ValidateResponses(expectedFilePath, resp1, responses[1])
	assert.NoError(t, validationErr)

	resp2 := responses[1]
	assert.NoError(t, resp2.Error)
	assert.Equal(t, http.StatusCreated, resp2.StatusCode)
	assert.Equal(t, "response2", resp2.BodyString)
}

// PRD-COMMENT: FR10.3 - Client Core Execution: Request Execution Error Handling
// Corresponds to: Client error handling for individual request failures during execution 
// of a multi-request file (http_syntax.md).
// This test verifies that if a request in 
// 'testdata/http_request_files/request_with_error.http' fails (e.g., network error), the 
// client reports the error for that specific request but continues to process subsequent 
// requests. The overall operation should also report an aggregated error.
func TestExecuteFile_RequestWithError(t *testing.T) {
	// Given
	server2 := startMockServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "good response")
	})
	defer server2.Close()

	client, _ := rc.NewClient()
	processedFilePath := createTestFileFromTemplate(t, 
		"testdata/http_request_files/request_with_error.http", 
		struct{ ServerURL string }{ServerURL: server2.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), processedFilePath)

	// Then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 error occurred:")
	assert.Contains(t, err.Error(), 
		"request 1 (GET http://localhost:12346/bad) processing resulted in error")

	require.Len(t, responses, 2)

	resp1 := responses[0]
	assert.Error(t, resp1.Error)
	assert.Contains(t, resp1.Error.Error(), "failed to execute HTTP request:")

	resp2 := responses[1]
	assert.NoError(t, resp2.Error)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	assert.Equal(t, "good response", resp2.BodyString)
}

// PRD-COMMENT: FR10.4 - Client Core Execution: File Parse Error Handling
// Corresponds to: Client error handling when a .http file has parsing issues, such as 
// no valid requests found (http_syntax.md).
// This test verifies that the client reports a suitable error if the provided file 
// 'testdata/http_request_files/parse_error.http' (which is expected to be empty or 
// syntactically invalid to the point of yielding no requests) cannot be successfully 
// parsed into executable requests.
func TestExecuteFile_ParseError(t *testing.T) {
	// Given
	client, _ := rc.NewClient()
	filePath := "testdata/http_request_files/parse_error.http"

	// When
	_, err := client.ExecuteFile(context.Background(), filePath)

	// Then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no requests found in file "+filePath)
}

// PRD-COMMENT: FR10.5 - Client Core Execution: Empty Request File
// Corresponds to: Client behavior when processing a .http file that syntactically 
// parses but contains no actual HTTP requests (http_syntax.md).
// This test uses 'testdata/http_request_files/no_requests.http' to verify that the 
// client correctly identifies that no requests are present and returns an appropriate 
// error or empty response set.
func TestExecuteFile_NoRequestsInFile(t *testing.T) {
	// Given
	client, _ := rc.NewClient()
	filePath := "testdata/http_request_files/comment_only_file.http"

	// When
	_, err := client.ExecuteFile(context.Background(), filePath)

	// Then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no requests found in file "+filePath)
}

// PRD-COMMENT: FR10.6 - Client Core Execution: Mixed Validity Parse Error
// Corresponds to: Client behavior when a .http file contains a mix of valid requests 
// followed by content that causes a parsing error (http_syntax.md).
// This test uses 'testdata/http_request_files/valid_then_invalid_syntax.http' to ensure 
// the client executes valid requests up to the point of the parse error and then reports 
// the parsing error, potentially halting further execution from that file.
func TestExecuteFile_ValidThenInvalidSyntax(t *testing.T) {
	// Given
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/first" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "response from /first")
		} else if r.Method == "INVALID_METHOD" && r.URL.Path == "/second" {
			w.WriteHeader(http.StatusNotImplemented)
			fmt.Fprint(w, "method not implemented")
		} else {
			t.Logf("Mock server received UNEXPECTED request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusTeapot)
		}
	})
	defer server.Close()

	client, _ := rc.NewClient()
	tempFilePath := createTestFileFromTemplate(t, 
		"testdata/http_request_files/valid_then_invalid_syntax.http", 
		struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), tempFilePath)

	// Then
	require.NoError(t, err, 
		"ExecuteFile should not return an error if requests are merely rejected by server")
	require.Len(t, responses, 2, "Should have two response objects")

	resp1 := responses[0]
	require.NotNil(t, resp1, "First response object should not be nil")
	assert.NoError(t, resp1.Error, "Error in first response object should be nil")
	assert.Equal(t, http.StatusOK, resp1.StatusCode)
	assert.Equal(t, "response from /first", resp1.BodyString)

	resp2 := responses[1]
	require.NotNil(t, resp2, "Second response object should not be nil")
	assert.NoError(t, resp2.Error, 
		"Error in second object should be nil as it's a server response, not client-side exec error")
	assert.Equal(t, http.StatusNotImplemented, resp2.StatusCode, 
		"Status code for second response should be Not Implemented")
	assert.Contains(t, resp2.BodyString, "method not implemented", 
		"Body for second response should indicate method error")
}

// PRD-COMMENT: FR10.7 - Client Core Execution: Multiple Execution Errors
// Corresponds to: Client's ability to handle and aggregate multiple errors if several 
// requests within a single file fail during execution (http_syntax.md).
// This test uses 'testdata/http_request_files/multiple_errors.http' (containing requests 
// designed to fail) to verify that each failing request's error is captured in its 
// respective response object and that an aggregated error is returned by ExecuteFile.
func TestExecuteFile_MultipleErrors(t *testing.T) {
	// Given
	client, _ := rc.NewClient()
	filePath := "testdata/http_request_files/multiple_errors.http"

	// When
	responses, err := client.ExecuteFile(context.Background(), filePath)

	// Then
	require.Error(t, err, 
		"Expected an error from ExecuteFile when multiple requests fail")
	assert.Contains(t, err.Error(), 
		"request 1 (GET http://localhost:12347/badreq1) processing resulted in error", 
		"Error message should contain info about first failed request")
	assert.Contains(t, err.Error(), ":12347: connect: connection refused", 
		"Error message should contain specific connection error for first request")
	assert.Contains(t, err.Error(), 
		"request 2 (POST http://localhost:12348/badreq2) processing resulted in error", 
		"Error message should contain info about second failed request")
	assert.Contains(t, err.Error(), ":12348: connect: connection refused", 
		"Error message should contain specific connection error for second request")

	require.Len(t, responses, 2, 
		"Should receive two response objects, even if they contain errors")

	resp1 := responses[0]
	require.NotNil(t, resp1, "First response object should not be nil")
	assert.Error(t, resp1.Error, "Error in first response object should be set")
	assert.Contains(t, resp1.Error.Error(), ":12347: connect: connection refused")

	resp2 := responses[1]
	require.NotNil(t, resp2, "Second response object should not be nil")
	assert.Error(t, resp2.Error, "Error in second response object should be set")
	assert.Contains(t, resp2.Error.Error(), ":12348: connect: connection refused")
}

// PRD-COMMENT: FR10.8 - Client Core Execution: Response Header Capturing
// Corresponds to: Client's capability to accurately capture all HTTP response headers 
// (http_syntax.md "Response Object").
// This test uses 'testdata/http_request_files/captures_response_headers.http' to verify that 
// the client correctly stores received headers, including multi-value headers, in the Response 
// object.
func TestExecuteFile_CapturesResponseHeaders(t *testing.T) {
	// Given
	server := startMockServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.Header().Add("X-Custom-Header", "value1")
		w.Header().Add("X-Custom-Header", "value2")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "{\"data\": \"headers test\"}")
	})
	defer server.Close()

	client, _ := rc.NewClient()
	requestFilePath := createTestFileFromTemplate(t, 
		"testdata/http_request_files/captures_response_headers.http", 
		struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 1)

	resp := responses[0]
	require.NotNil(t, resp)
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	assert.Equal(t, "application/vnd.api+json", resp.Headers.Get("Content-Type"))
	assert.Equal(t, []string{"value1", "value2"}, resp.Headers["X-Custom-Header"])
	assert.Empty(t, resp.Headers.Get("Non-Existent-Header"))
}

// PRD-COMMENT: FR10.9 - Client Core Execution: Basic HTTP GET
// Corresponds to: Client's ability to execute a simple GET request over plain HTTP using a 
// mock transport (http_syntax.md).
// This test uses 'testdata/http_request_files/simple_get.http' and a mock HTTP transport to 
// verify the fundamental request execution flow, ensuring the correct method, URL, and headers 
// are prepared and sent.
func TestExecuteFile_SimpleGetHTTP(t *testing.T) {
	// Given
	var interceptedReq *http.Request
	mockTransport := &mockRoundTripper{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			interceptedReq = req.Clone(req.Context())
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("mocked response")),
				Header:     make(http.Header),
			}, nil
		},
	}

	clientWithMockTransport, err := rc.NewClient(rc.WithHTTPClient(&http.Client{Transport: mockTransport}))
	require.NoError(t, err)
	requestFilePath := "testdata/http_request_files/simple_get.http"

	// When
	responses, err := clientWithMockTransport.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err, "ExecuteFile should not fail")
	require.Len(t, responses, 1, "Expected one response")
	resp := responses[0]
	require.NotNil(t, resp, "Response should not be nil")
	assert.NoError(t, resp.Error, "Response error should be nil")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status OK from mock")

	require.NotNil(t, interceptedReq, "Request should have been intercepted")
	assert.Equal(t, http.MethodGet, interceptedReq.Method, "Expected GET method")
	assert.Equal(t, "https://jsonplaceholder.typicode.com/todos/1", 
		interceptedReq.URL.String(), "Expected full URL from file")
	assert.Empty(t, interceptedReq.Header, "Expected no headers from simple_get.http")
}

// PRD-COMMENT: FR10.10 - Client Core Execution: Multiple Requests (Extended)
// Corresponds to: Robustness of client in handling .http files with more than two requests, 
// ensuring all are processed sequentially (http_syntax.md "Request Separation").
// This test uses 'testdata/http_request_files/multiple_requests_gt2.http' and validates 
// responses against 'testdata/http_response_files/multiple_responses_gt2_expected.http' to 
// ensure the client can handle a larger number of requests in a file.
func TestExecuteFile_MultipleRequests_GreaterThanTwo(t *testing.T) {
	// Given
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

	client, _ := rc.NewClient()
	requestFilePath := createTestFileFromTemplate(t, 
		"testdata/http_request_files/multiple_requests_gt2.http", 
		struct{ ServerURL string }{ServerURL: server.URL})

	// When
	actualResponses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err)
	require.Len(t, actualResponses, 3, "Should have received 3 responses")

	expectedResponseFilePath := "testdata/http_response_files/multiple_responses_gt2_expected.http"

	validationErr := client.ValidateResponses(expectedResponseFilePath, actualResponses...)
	assert.NoError(t, validationErr, "Validation of responses against file failed")
}
