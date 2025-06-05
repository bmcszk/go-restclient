package restclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecuteFile_InPlaceVars_DefinedByDotEnvSystemVar tests in-place variable substitution
// where the variable is defined by a {{$dotenv VAR_NAME}} system variable.
func TestExecuteFile_InPlaceVars_DefinedByDotEnvSystemVar(t *testing.T) {
	// Given: a .env file and an HTTP file using {{$dotenv VAR_NAME}} for an in-place variable
	tempDir := t.TempDir()
	const dotEnvVarName = "DOTENV_VAR_FOR_SYSTEM_TEST_EXTRACTED" // Modified for isolation
	const dotEnvVarValue = "actual_dotenv_value_extracted"       // Modified for isolation
	dotEnvContent := fmt.Sprintf("%s=%s", dotEnvVarName, dotEnvVarValue)
	dotEnvFilePath := filepath.Join(tempDir, ".env")
	err := os.WriteFile(dotEnvFilePath, []byte(dotEnvContent), 0600)
	require.NoError(t, err, "Failed to write .env file")

	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	httpFileContent := fmt.Sprintf(`
@my_api_key_dotenv_ext = {{$dotenv %s}}

### Test Request DotEnv Extracted
GET http://%s/{{my_api_key_dotenv_ext}}
`, dotEnvVarName, serverURL.Host)

	requestFilePath := filepath.Join(tempDir, "test_dotenv_ext.http") // Modified for isolation
	err = os.WriteFile(requestFilePath, []byte(httpFileContent), 0600)
	require.NoError(t, err, "Failed to write .http file")

	client, err := NewClient()
	require.NoError(t, err)

	// When: the HTTP file is executed
	responses, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: the request should be successful and the variable substituted correctly
	require.NoError(t, execErr, "ExecuteFile returned an unexpected error")
	require.Len(t, responses, 1, "Expected one response")
	require.Nil(t, responses[0].Error, "Response error should be nil")
	assert.Equal(t, "/"+dotEnvVarValue, capturedPath, "Expected path to be substituted with .env value via {{$dotenv}}")
}
