package restclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecuteFile_InPlaceVars_PrecedenceOverEnv extracted from TestExecuteFile_InPlaceVariables
func TestExecuteFile_InPlaceVars_PrecedenceOverEnv(t *testing.T) {
	// Given: an .http file with an in-place variable and an environment variable with the same name
	var capturedURLPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURLPath = r.URL.Path // We only care about the path part
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// The in-place @host should point to the mock server's URL
	fileContent := fmt.Sprintf(`
@host = %s

### Test Request
GET {{host}}/expected_path
`, server.URL)

	requestFilePath := createTempHTTPFileFromString(t, fileContent)
	tempDir := filepath.Dir(requestFilePath)

	// Create a temporary environment file for this test
	envName := "testPrecedenceEnv"
	envFileName := fmt.Sprintf("http-client.env.%s.json", envName)
	envFilePath := filepath.Join(tempDir, envFileName)
	envData := map[string]string{
		"host": "http://env.example.com/should_not_be_used", // This should be overridden by the in-place variable
	}
	envJSON, err := json.Marshal(envData)
	require.NoError(t, err, "Failed to marshal env data to JSON")
	err = os.WriteFile(envFilePath, envJSON, 0644)
	require.NoError(t, err, "Failed to write temp env file")
	defer os.Remove(envFilePath) // Clean up the temp env file

	client, err := NewClient(WithEnvironment(envName))
	require.NoError(t, err)

	// When: the .http file is executed, it should load the environment from the temp file
	_, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: no error should occur and the in-place variable should take precedence
	require.NoError(t, execErr, "ExecuteFile should not return an error")
	assert.Equal(t, "/expected_path", capturedURLPath, "The request path should match, indicating in-place var was used")
}
