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

// TestExecuteFile_InPlaceVars_DefinedByAnotherInPlaceVar tests the scenario where an in-place variable
// is defined by another in-place variable, and this composite variable is used in the request URL.
func TestExecuteFile_InPlaceVars_DefinedByAnotherInPlaceVar(t *testing.T) {
	// Given: an .http file with an in-place variable defined by another in-place variable
	const basePathValue = "/api/v1/nested"   // Changed to avoid potential conflicts if run in parallel
	const resourcePathValue = "items_nested" // Changed to avoid potential conflicts
	var capturedURLPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURLPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fileContent := fmt.Sprintf(`
@base_path = %s
@resource = %s
@full_url_segment = {{base_path}}/{{resource}}/123

### Test Request With Nested In-Place Var in URL
GET %s{{full_url_segment}}
`, basePathValue, resourcePathValue, server.URL)

	requestFilePath := createTempHTTPFileFromString(t, fileContent)

	client, err := NewClient()
	require.NoError(t, err)

	// When: the .http file is executed
	results, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: no error should occur and the URL path should be correctly substituted
	require.NoError(t, execErr, "ExecuteFile should not return an error")
	require.Len(t, results, 1, "Should have one result")
	require.Nil(t, results[0].Error, "Request execution error should be nil")
	assert.Equal(t, "/api/v1/nested/items_nested/123", capturedURLPath, "The URL path should be correctly substituted with nested in-place variables")
}
