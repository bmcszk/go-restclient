package restclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecuteFile_InPlaceVars_HeaderSubstitution extracted from TestExecuteFile_InPlaceVariables
func TestExecuteFile_InPlaceVars_HeaderSubstitution(t *testing.T) {
	// Given
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	httpFileContent := fmt.Sprintf(`
@auth_token = Bearer_secret_token_123

GET %s/checkheaders
Authorization: {{auth_token}}
User-Agent: test-client
`, server.URL)

	requestFilePath := createTempHTTPFileFromString(t, httpFileContent)

	client, err := NewClient()
	require.NoError(t, err)

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err, "ExecuteFile should not return an error")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	assert.NoError(t, resp.Error, "Response error should be nil")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code mismatch")

	// Verify the server received the request with the substituted header
	assert.Equal(t, "Bearer_secret_token_123", capturedHeaders.Get("Authorization"))
	assert.Equal(t, "test-client", capturedHeaders.Get("User-Agent")) // Ensure other headers are preserved

	// Verify ParsedFile.FileVariables
	file, err := os.Open(requestFilePath)
	require.NoError(t, err)
	defer file.Close()
	parsedFile, pErr := ParseRequests(file, requestFilePath, client, make(map[string]string), os.LookupEnv, make(map[string]string), nil)
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	assert.Equal(t, "Bearer_secret_token_123", parsedFile.FileVariables["auth_token"])
}
