package restclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteFile_SingleRequest(t *testing.T) {
	// Given
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/users", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "user data")
	})
	defer server.Close()

	client, _ := NewClient()
	requestFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/single_request.http", struct{ ServerURL string }{ServerURL: server.URL})

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

	client, _ := NewClient()
	processedFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/multiple_requests.http", struct{ ServerURL string }{ServerURL: server.URL})

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
	validationErr := client.ValidateResponses(expectedFilePath, []*Response{resp1, responses[1]})
	assert.NoError(t, validationErr)

	resp2 := responses[1]
	assert.NoError(t, resp2.Error)
	assert.Equal(t, http.StatusCreated, resp2.StatusCode)
	assert.Equal(t, "response2", resp2.BodyString)
}

func TestExecuteFile_RequestWithError(t *testing.T) {
	// Given
	server2 := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "good response")
	})
	defer server2.Close()

	client, _ := NewClient()
	processedFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/request_with_error.http", struct{ ServerURL string }{ServerURL: server2.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), processedFilePath)

	// Then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 error occurred:")
	assert.Contains(t, err.Error(), "request 1 (GET http://localhost:12346/bad) processing resulted in error")

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
	// Given
	client, _ := NewClient()
	filePath := "testdata/http_request_files/parse_error.http"

	// When
	_, err := client.ExecuteFile(context.Background(), filePath)

	// Then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no requests found in file "+filePath)
}

func TestExecuteFile_NoRequestsInFile(t *testing.T) {
	// Given
	client, _ := NewClient()
	filePath := "testdata/http_request_files/comment_only_file.http"

	// When
	_, err := client.ExecuteFile(context.Background(), filePath)

	// Then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no requests found in file "+filePath)
}

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

	client, _ := NewClient()
	tempFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/valid_then_invalid_syntax.http", struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), tempFilePath)

	// Then
	require.NoError(t, err, "ExecuteFile should not return an error if requests are merely rejected by server")
	require.Len(t, responses, 2, "Should have two response objects")

	resp1 := responses[0]
	require.NotNil(t, resp1, "First response object should not be nil")
	assert.NoError(t, resp1.Error, "Error in first response object should be nil")
	assert.Equal(t, http.StatusOK, resp1.StatusCode)
	assert.Equal(t, "response from /first", resp1.BodyString)

	resp2 := responses[1]
	require.NotNil(t, resp2, "Second response object should not be nil")
	assert.NoError(t, resp2.Error, "Error in second object should be nil as it's a server response, not client-side exec error")
	assert.Equal(t, http.StatusNotImplemented, resp2.StatusCode, "Status code for second response should be Not Implemented")
	assert.Contains(t, resp2.BodyString, "method not implemented", "Body for second response should indicate method error")
}

func TestExecuteFile_MultipleErrors(t *testing.T) {
	// Given
	client, _ := NewClient()
	filePath := "testdata/http_request_files/multiple_errors.http"

	// When
	responses, err := client.ExecuteFile(context.Background(), filePath)

	// Then
	require.Error(t, err, "Expected an error from ExecuteFile when multiple requests fail")
	assert.Contains(t, err.Error(), "request 1 (GET http://localhost:12347/badreq1) processing resulted in error", "Error message should contain info about first failed request")
	assert.Contains(t, err.Error(), ":12347: connect: connection refused", "Error message should contain specific connection error for first request")
	assert.Contains(t, err.Error(), "request 2 (POST http://localhost:12348/badreq2) processing resulted in error", "Error message should contain info about second failed request")
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
	// Given
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.Header().Add("X-Custom-Header", "value1")
		w.Header().Add("X-Custom-Header", "value2")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "{\"data\": \"headers test\"}")
	})
	defer server.Close()

	client, _ := NewClient()
	requestFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/captures_response_headers.http", struct{ ServerURL string }{ServerURL: server.URL})

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

	clientWithMockTransport, err := NewClient(WithHTTPClient(&http.Client{Transport: mockTransport}))
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
	assert.Equal(t, "https://jsonplaceholder.typicode.com/todos/1", interceptedReq.URL.String(), "Expected full URL from file")
	assert.Empty(t, interceptedReq.Header, "Expected no headers from simple_get.http")
}

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

	client, _ := NewClient()
	requestFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/multiple_requests_gt2.http", struct{ ServerURL string }{ServerURL: server.URL})

	// When
	actualResponses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err)
	require.Len(t, actualResponses, 3, "Should have received 3 responses")

	expectedResponseFilePath := "testdata/http_response_files/multiple_responses_gt2_expected.http"

	validationErr := client.ValidateResponses(expectedResponseFilePath, actualResponses)
	assert.NoError(t, validationErr, "Validation of responses against file failed")
}
