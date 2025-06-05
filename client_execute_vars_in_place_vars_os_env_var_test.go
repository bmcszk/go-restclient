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

// TestExecuteFile_InPlaceVars_OsEnvVar extracted from TestExecuteFile_InPlaceVariables
func TestExecuteFile_InPlaceVars_OsEnvVar(t *testing.T) {
	// Given: an OS environment variable and an .http file with an in-place variable defined by it
	const testEnvVarName = "TEST_USER_HOME_INPLACE_OS" // Changed name slightly to avoid potential collision
	const testEnvVarValue = "/testhome/userosdir"      // Changed value slightly
	t.Setenv(testEnvVarName, testEnvVarValue)

	// Debug: Check if t.Setenv is working as expected in the test goroutine
	val, ok := os.LookupEnv(testEnvVarName)
	require.True(t, ok, "os.LookupEnv should find the var set by t.Setenv")
	require.Equal(t, testEnvVarValue, val, "os.LookupEnv should return the correct value set by t.Setenv")

	var capturedURLPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURLPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// server.URL does not have a trailing slash. {{my_home_dir}} will resolve to testEnvVarValue, which starts with a slash.
	// So, %s{{my_home_dir}} ensures no double slash.
	fileContent := fmt.Sprintf(`
@my_home_dir = {{$processEnv %s}}

### Test Request With OS Env Var In In-Place Var
GET %s{{my_home_dir}}/files
`, testEnvVarName, server.URL)

	requestFilePath := createTempHTTPFileFromString(t, fileContent)

	client, err := NewClient()
	require.NoError(t, err)

	// When: the .http file is executed
	results, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: no error should occur and the path should contain the resolved OS env variable
	require.NoError(t, execErr, "ExecuteFile should not return an error for in-place OS env var")
	require.Len(t, results, 1, "Should have one result for in-place OS env var")
	require.Nil(t, results[0].Error, "Request execution error should be nil for in-place OS env var")
	// capturedURLPath should be "/testhome/userosdir/files"
	assert.Equal(t, testEnvVarValue+"/files", capturedURLPath, "The URL path should be correctly substituted with the OS environment variable via in-place var")
}
