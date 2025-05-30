package restclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
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
	assert.Contains(t, err.Error(), "request 1 (INVALIDMETHOD /test) failed")

	require.Len(t, responses, 1)

	resp1 := responses[0]
	assert.Error(t, resp1.Error, "Expected an error for invalid method/scheme")
	assert.Contains(t, resp1.Error.Error(), "unsupported protocol scheme", "Error message should indicate unsupported protocol scheme")
	assert.Contains(t, resp1.Error.Error(), "Invalidmethod", "Error message should contain the problematic method string as used")
}

func TestExecuteFile_IgnoreEmptyBlocks_Client(t *testing.T) {
	// Given common setup for all subtests
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
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
			bodyBytes, _ := io.ReadAll(r.Body)
			assert.JSONEq(t, `{"key": "value"}`, string(bodyBytes))
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprint(w, "response from /req2")
		default:
			t.Errorf("Unexpected request to mock server: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer server.Close()
	client, _ := NewClient()

	tests := []struct {
		name               string
		requestFileContent string
		expectedResponses  int
		expectedError      bool
		responseValidators []func(t *testing.T, resp *Response)
	}{
		{
			name: "SCENARIO-LIB-028-004: Valid request, then separator, then only comments",
			requestFileContent: fmt.Sprintf(`
GET %s/first

###
# This block is empty
`, server.URL),
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
			name: "SCENARIO-LIB-028-005: Only comments, then separator, then valid request",
			requestFileContent: fmt.Sprintf(`
# This block is empty
###
GET %s/second
`, server.URL),
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
			name: "SCENARIO-LIB-028-006: Valid request, separator with comments, then another valid request",
			requestFileContent: fmt.Sprintf(`
GET %s/req1

### Comment for empty block
# More comments

###
POST %s/req2
Content-Type: application/json

{
  "key": "value"
}
`, server.URL, server.URL),
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
			name: "File with only variable definitions - ExecuteFile",
			requestFileContent: `
@host=localhost
@port=8080
`,
			expectedResponses: 0,
			expectedError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given specific setup for this subtest
			tempFile, err := os.CreateTemp(t.TempDir(), "test_*.http")
			require.NoError(t, err)
			defer os.Remove(tempFile.Name())

			_, err = tempFile.WriteString(tt.requestFileContent)
			require.NoError(t, err)
			require.NoError(t, tempFile.Close())

			// When
			responses, execErr := client.ExecuteFile(context.Background(), tempFile.Name())

			// Then
			if tt.expectedError {
				assert.Error(t, execErr)
				if strings.Contains(tt.name, "variable definitions") {
					assert.Contains(t, execErr.Error(), "no requests found in file")
				}
			} else {
				assert.NoError(t, execErr)
				require.Len(t, responses, tt.expectedResponses, "Number of responses mismatch")
				for i, validator := range tt.responseValidators {
					if i < len(responses) {
						validator(t, responses[i])
					}
				}
			}
		})
	}
}
