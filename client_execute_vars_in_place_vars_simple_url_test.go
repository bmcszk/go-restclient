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

// TestExecuteFile_InPlaceVars_SimpleURL extracted from TestExecuteFile_InPlaceVariables
func TestExecuteFile_InPlaceVars_SimpleURL(t *testing.T) {
	// Given
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	httpFileContent := fmt.Sprintf(`
@hostname = %s
@path_segment = /api/v1/items

GET {{hostname}}{{path_segment}}/123
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

	// Verify the server received the request with the substituted URL
	expectedPath := "/api/v1/items/123"
	assert.Equal(t, expectedPath, capturedPath, "Captured path by server mismatch")

	// Verify ParsedFile.FileVariables
	file, err := os.Open(requestFilePath)
	require.NoError(t, err)
	defer file.Close()
	// Note: The environment and programmaticVars maps are empty here as per original sub-test.
	parsedFile, pErr := ParseRequests(file, requestFilePath, client, make(map[string]string), os.LookupEnv, make(map[string]string), nil)
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	assert.Equal(t, server.URL, parsedFile.FileVariables["hostname"])
	assert.Equal(t, "/api/v1/items", parsedFile.FileVariables["path_segment"])
}
