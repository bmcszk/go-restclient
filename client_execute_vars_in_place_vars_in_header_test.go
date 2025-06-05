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

// TestExecuteFile_InPlaceVars_InHeader extracted from TestExecuteFile_InPlaceVariables
func TestExecuteFile_InPlaceVars_InHeader(t *testing.T) {
	// Given: an .http file with an in-place variable used in a header
	const headerKey = "X-Auth-Token-InPlace"         // Slightly changed key for independence
	const headerValue = "secret-token-inplace-54321" // Slightly changed value

	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fileContent := fmt.Sprintf(`
@my_token = %s

### Test Request With In-Place Var in Header
GET %s/some/path
%s: {{my_token}}
`, headerValue, server.URL, headerKey)

	requestFilePath := createTempHTTPFileFromString(t, fileContent)

	client, err := NewClient()
	require.NoError(t, err)

	// When: the .http file is executed
	results, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: no error should occur and the header should be correctly substituted
	require.NoError(t, execErr, "ExecuteFile should not return an error")
	require.Len(t, results, 1, "Should have one result")
	require.Nil(t, results[0].Error, "Request execution error should be nil")
	assert.Equal(t, headerValue, capturedHeaders.Get(headerKey), "The header should be correctly substituted with the in-place variable")
}
