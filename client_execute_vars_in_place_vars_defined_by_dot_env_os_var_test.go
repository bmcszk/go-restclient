package restclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecuteFile_InPlaceVars_DefinedByDotEnvOsVar tests in-place variable substitution
// where the variable is defined by an OS environment variable using {{$env.VAR_NAME}} syntax.
func TestExecuteFile_InPlaceVars_DefinedByDotEnvOsVar(t *testing.T) {
	// Given: an .http file with an in-place variable defined by an OS environment variable using {{$env.VAR_NAME}}
	const testEnvVarName = "MY_CONFIG_PATH_DOT_ENV_EXTRACTED"       // Modified for isolation
	const testEnvVarValue = "/usr/local/appconfig_dotenv_extracted" // Modified for isolation
	var capturedURLPath string

	t.Setenv(testEnvVarName, testEnvVarValue)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURLPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fileContent := fmt.Sprintf(`
@my_path_from_env_ext = {{$env.%s}}

### Test Request With OS Env Var ({{$env.VAR}}) In In-Place Var
GET %s{{my_path_from_env_ext}}/data_ext
`, testEnvVarName, server.URL)

	requestFilePath := createTempHTTPFileFromString(t, fileContent)

	client, err := NewClient()
	require.NoError(t, err)

	// When: the .http file is executed
	results, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: no error should occur and the URL path should be correctly substituted
	require.NoError(t, execErr, "ExecuteFile should not return an error")
	require.Len(t, results, 1, "Should have one result")
	require.Nil(t, results[0].Error, "Request execution error should be nil")

	expectedPath := testEnvVarValue + "/data_ext"
	assert.Equal(t, expectedPath, capturedURLPath, "The URL path should be correctly substituted with the OS environment variable via {{$env.VAR_NAME}} in-place var")
}
