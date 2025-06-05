package restclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecuteFile_InPlaceVars_VarDefinedByVar extracted from TestExecuteFile_InPlaceVariables
func TestExecuteFile_InPlaceVars_VarDefinedByVar(t *testing.T) {
	// Given
	var capturedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Use server.URL to make the test robust against dynamic port allocation
	parsedServerURL, psuErr := url.Parse(server.URL)
	require.NoError(t, psuErr)
	hostFromServer := parsedServerURL.Host // e.g., 127.0.0.1:PORT

	httpFileContent := fmt.Sprintf(`
@my_host = %s
@base_api_path = /api/v2
@full_api_url = http://{{my_host}}{{base_api_path}}
@items_endpoint = {{full_api_url}}/items

GET {{items_endpoint}}?host_check={{my_host}}
`, hostFromServer)

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

	// Verify the server received the request at the correctly resolved URL
	expectedPathAndQuery := fmt.Sprintf("/api/v2/items?host_check=%s", hostFromServer)
	assert.Equal(t, expectedPathAndQuery, capturedURL)

	// Verify ParsedFile.FileVariables (should store raw definitions)
	file, err := os.Open(requestFilePath)
	require.NoError(t, err)
	defer file.Close()
	parsedFile, pErr := ParseRequests(file, requestFilePath, client, make(map[string]string), os.LookupEnv, make(map[string]string), nil)
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	assert.Equal(t, hostFromServer, parsedFile.FileVariables["my_host"])
	assert.Equal(t, "/api/v2", parsedFile.FileVariables["base_api_path"])
	assert.Equal(t, "http://{{my_host}}{{base_api_path}}", parsedFile.FileVariables["full_api_url"])
	assert.Equal(t, "{{full_api_url}}/items", parsedFile.FileVariables["items_endpoint"])
}
