package restclient

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestExecuteRequest_SimpleGET(t *testing.T) {
	var capturedRequestHost string
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/testpath", r.URL.Path)
		assert.Equal(t, "TestValue", r.Header.Get("X-Test-Header"))
		capturedRequestHost = r.Host // Capture Host header from server side
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, `{"message": "success"}`)
	})
	defer server.Close()

	client, _ := NewClient()
	targetURL, _ := url.Parse(server.URL + "/testpath")
	restReq := &Request{
		Method: http.MethodGet,
		URL:    targetURL,
		Headers: http.Header{
			"X-Test-Header": []string{"TestValue"},
		},
		Body: strings.NewReader(""), // Empty body for GET
	}

	resp, err := client.ExecuteRequest(restReq)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NoError(t, resp.Error)
	assert.Equal(t, targetURL.Host, capturedRequestHost) // Assert captured host matches target
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Headers.Get("Content-Type"))
	assert.Equal(t, `{"message": "success"}`+"\n", resp.BodyString)
	assert.NotZero(t, resp.Duration)
}

func TestExecuteRequest_POSTWithBody(t *testing.T) {
	expectedBodyContent := `{"key":"value"}`
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		bodyBytes, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, expectedBodyContent, string(bodyBytes))
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, "created")
	})
	defer server.Close()

	client, _ := NewClient()
	parsedURL, _ := url.Parse(server.URL + "/resource")
	restReq := &Request{
		Method:  http.MethodPost,
		URL:     parsedURL,
		RawBody: expectedBodyContent,
		Body:    strings.NewReader(expectedBodyContent),
	}

	resp, err := client.ExecuteRequest(restReq)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "created", resp.BodyString)
}

func TestExecuteRequest_WithBaseURL(t *testing.T) {
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		// For the first request: BaseURL = ".../api", req.URL = "/endpoint"
		// ResolveReference makes it ".../endpoint"
		assert.Equal(t, "/endpoint", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client, _ := NewClient(WithBaseURL(server.URL + "/api"))
	absolutePathURL, _ := url.Parse("/endpoint") // Starts with "/"
	restReq := &Request{
		Method: http.MethodGet,
		URL:    absolutePathURL,
	}

	respBase1, errBase1 := client.ExecuteRequest(restReq)
	require.NoError(t, errBase1)
	require.NotNil(t, respBase1)
	assert.NoError(t, respBase1.Error)

	// Test with relative path that needs joining
	relativePathURL, _ := url.Parse("endpoint2") // No leading slash
	restReq2 := &Request{Method: http.MethodGet, URL: relativePathURL}
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For the second request: BaseURL = ".../api", req.URL = "endpoint2"
		// ResolveReference should make it ".../api/endpoint2"
		assert.Equal(t, "/api/endpoint2", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	})
	respBase2, errBase2 := client.ExecuteRequest(restReq2)
	require.NoError(t, errBase2)
	require.NotNil(t, respBase2)
	assert.NoError(t, respBase2.Error)
}

func TestExecuteRequest_WithDefaultHeaders(t *testing.T) {
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DefaultVal", r.Header.Get("X-Default-Global"))
		assert.Equal(t, "RequestSpecificVal", r.Header.Get("X-Request-Specific"))
		assert.Equal(t, "OverriddenValue", r.Header.Get("X-To-Override")) // Expect request header to win
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client, _ := NewClient(WithDefaultHeader("X-Default-Global", "DefaultVal"), WithDefaultHeader("X-To-Override", "DefaultShouldBeOverridden"))
	parsedURL, _ := url.Parse(server.URL)
	restReq := &Request{
		Method: http.MethodGet,
		URL:    parsedURL,
		Headers: http.Header{
			"X-Request-Specific": []string{"RequestSpecificVal"},
			"X-To-Override":      []string{"OverriddenValue"},
		},
	}

	resp, err := client.ExecuteRequest(restReq)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NoError(t, resp.Error)
}

func TestExecuteRequest_ErrorCases(t *testing.T) {
	client, _ := NewClient()

	// Nil request (still returns error from function)
	_, err := client.ExecuteRequest(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot execute a nil request")

	// Network error (now error is in Response.Error)
	parsedURL, _ := url.Parse("http://localhost:12345/nonexistent")
	restReqNetErr := &Request{Method: http.MethodGet, URL: parsedURL}
	respNetErr, funcErr := client.ExecuteRequest(restReqNetErr)
	assert.Nil(t, funcErr) // Function itself shouldn't error here
	require.NotNil(t, respNetErr)
	assert.Error(t, respNetErr.Error)
	assert.Contains(t, respNetErr.Error.Error(), "http request failed")

	// Test error during request creation (e.g. invalid method)
	badMethodReq := &Request{Method: "INVALID METHOD", URL: parsedURL}
	respBadMethod, funcErrBadMethod := client.ExecuteRequest(badMethodReq)
	assert.Nil(t, funcErrBadMethod)
	require.NotNil(t, respBadMethod)
	assert.Error(t, respBadMethod.Error)
	assert.Contains(t, respBadMethod.Error.Error(), "failed to create http request")
	assert.Contains(t, respBadMethod.Error.Error(), "invalid method")
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

	responses, err := client.ExecuteFile(tempFile.Name())
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

	responses, err := client.ExecuteFile(tempFile.Name())
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

	responses, err := client.ExecuteFile(tempFile.Name())
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

	_, err = client.ExecuteFile(tempFile.Name())
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

	_, err = client.ExecuteFile(tempFile.Name())
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

	responses, err := clientWithMockTransport.ExecuteFile("testdata/http_request_files/simple_get.http")
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
