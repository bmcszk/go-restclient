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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteFile_InvalidMethodInFile(t *testing.T) {
	// Given
	client, _ := NewClient()
	requestFilePath := "testdata/http_request_files/invalid_method.http"

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 error occurred:")
	assert.Contains(t, err.Error(), "unsupported protocol scheme")
	assert.Contains(t, err.Error(), "request 1 (INVALIDMETHOD /test) processing resulted in error")

	require.Len(t, responses, 1)

	resp1 := responses[0]
	assert.Error(t, resp1.Error, "Expected an error for invalid method/scheme")
	assert.Contains(t, resp1.Error.Error(), "unsupported protocol scheme", "Error message should indicate unsupported protocol scheme")
	assert.Contains(t, resp1.Error.Error(), "Invalidmethod", "Error message should contain the problematic method string as used")
}

// newEdgeCaseTestServer provides a mock HTTP server for edge case testing scenarios.
func newEdgeCaseTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/first":
			assert.Equal(t, http.MethodGet, r.Method)
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "response from /first")
		case "/second":
			assert.Equal(t, http.MethodGet, r.Method)
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "response from /second")
		case "/req1":
			assert.Equal(t, http.MethodGet, r.Method)
			w.WriteHeader(http.StatusAccepted)
			_, _ = fmt.Fprint(w, "response from /req1")
		case "/req2":
			assert.Equal(t, http.MethodPost, r.Method)
			bodyBytes, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.JSONEq(t, `{"key": "value"}`, string(bodyBytes))
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprint(w, "response from /req2")
		default:
			t.Errorf("Unexpected request to mock server: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// runEdgeCaseTestSubTest executes a single sub-test for TestExecuteFile_IgnoreEmptyBlocks_Client.
func runEdgeCaseTestSubTest(t *testing.T, client *Client, serverURL string, tt struct {
	name               string
	requestFilePath    string
	expectedResponses  int
	expectedError      bool
	responseValidators []func(t *testing.T, resp *Response)
}) {
	t.Helper()
	// Given specific setup for this subtest
	var requestFilePathToUse string

	if tt.requestFilePath == "" {
		t.Fatalf("requestFilePath cannot be empty for test: %s", tt.name)
	}

	contentBytes, err := os.ReadFile(tt.requestFilePath)
	require.NoError(t, err, "Failed to read: %s", tt.requestFilePath)

	// Create a temporary file for the processed request content
	tempFile, err := os.CreateTemp(t.TempDir(), "test_*.http")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	var finalContent string
	switch tt.requestFilePath {
	case "testdata/client_execute_edgecases/only_vars.http":
		finalContent = string(contentBytes)
	case "testdata/client_execute_edgecases/valid_comment_separator_valid.http":
		finalContent = fmt.Sprintf(string(contentBytes), serverURL, serverURL) // two URLs for this specific file
	default: // Handles other files like valid_then_empty_comments.http, empty_comments_then_valid.http
		finalContent = fmt.Sprintf(string(contentBytes), serverURL) // one URL for other files
	}

	_, err = tempFile.WriteString(finalContent)
	require.NoError(t, err)
	err = tempFile.Close()
	require.NoError(t, err)
	requestFilePathToUse = tempFile.Name()

	// When
	responses, execErr := client.ExecuteFile(context.Background(), requestFilePathToUse)

	// Then
	if tt.expectedError {
		assert.Error(t, execErr)
		if strings.Contains(tt.name, "variable definitions") {
			assert.Contains(t, execErr.Error(), "no requests found in file")
		}
	} else {
		assert.NoError(t, execErr)
		require.Len(t, responses, tt.expectedResponses, "Number of responses mismatch for test: %s", tt.name)
		for i, validator := range tt.responseValidators {
			if i < len(responses) {
				validator(t, responses[i])
			}
		}
	}
}

func TestExecuteFile_IgnoreEmptyBlocks_Client(t *testing.T) {
	// Given common setup
	server := newEdgeCaseTestServer(t) // Use the new helper
	defer server.Close()
	client, _ := NewClient()

	tests := []struct {
		name               string
		requestFilePath    string // Changed from requestFileContent
		expectedResponses  int
		expectedError      bool
		responseValidators []func(t *testing.T, resp *Response)
	}{
		{
			name:              "SCENARIO-LIB-028-004: Valid request, then separator, then only comments",
			requestFilePath:   "testdata/client_execute_edgecases/valid_then_empty_comments.http",
			expectedResponses: 1,
			expectedError:     false,
			responseValidators: []func(t *testing.T, resp *Response){
				func(t *testing.T, resp *Response) {
					assert.NoError(t, resp.Error)
					assert.Equal(t, http.StatusOK, resp.StatusCode)
					assert.Equal(t, "response from /first", resp.BodyString)
				},
			},
		},
		{
			name:              "SCENARIO-LIB-028-005: Only comments, then separator, then valid request",
			requestFilePath:   "testdata/client_execute_edgecases/empty_comments_then_valid.http",
			expectedResponses: 1,
			expectedError:     false,
			responseValidators: []func(t *testing.T, resp *Response){
				func(t *testing.T, resp *Response) {
					assert.NoError(t, resp.Error)
					assert.Equal(t, http.StatusOK, resp.StatusCode)
					assert.Equal(t, "response from /second", resp.BodyString)
				},
			},
		},
		{
			name:              "SCENARIO-LIB-028-006: Valid request, separator with comments, then another valid request",
			requestFilePath:   "testdata/client_execute_edgecases/valid_comment_separator_valid.http",
			expectedResponses: 2,
			expectedError:     false,
			responseValidators: []func(t *testing.T, resp *Response){
				func(t *testing.T, resp *Response) { // For GET /req1
					assert.NoError(t, resp.Error)
					assert.Equal(t, http.StatusAccepted, resp.StatusCode)
					assert.Equal(t, "response from /req1", resp.BodyString)
				},
				func(t *testing.T, resp *Response) { // For POST /req2
					assert.NoError(t, resp.Error)
					assert.Equal(t, http.StatusCreated, resp.StatusCode)
					assert.Equal(t, "response from /req2", resp.BodyString)
				},
			},
		},
		{
			name:              "File with only variable definitions - ExecuteFile",
			requestFilePath:   "testdata/client_execute_edgecases/only_vars.http",
			expectedResponses: 0,
			expectedError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runEdgeCaseTestSubTest(t, client, server.URL, tt)
		})
	}
}
