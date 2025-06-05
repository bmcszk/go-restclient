package restclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecuteFile_InPlaceVars_SystemVar extracted from TestExecuteFile_InPlaceVariables
func TestExecuteFile_InPlaceVars_SystemVar(t *testing.T) {
	// Given: an .http file with an in-place variable defined by a system variable {{$uuid}}
	var capturedURLPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURLPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fileContent := fmt.Sprintf(`
@my_request_id = {{$uuid}}

### Test Request With System Var In In-Place Var
GET %s/{{my_request_id}}/resource
`, server.URL)

	requestFilePath := createTempHTTPFileFromString(t, fileContent)

	client, err := NewClient()
	require.NoError(t, err)

	// When: the .http file is executed
	_, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: no error should occur and the path should contain a resolved UUID
	require.NoError(t, execErr, "ExecuteFile should not return an error")
	require.NotEmpty(t, capturedURLPath, "Captured URL path should not be empty")

	pathSegments := strings.Split(strings.Trim(capturedURLPath, "/"), "/")
	require.Len(t, pathSegments, 2, "URL path should have two segments")
	assert.Len(t, pathSegments[0], 36, "The first path segment (resolved UUID) should be 36 characters long")
	assert.Equal(t, "resource", pathSegments[1], "The second path segment should be 'resource'")
	assert.NotEqual(t, "{{$uuid}}", pathSegments[0], "The UUID part should not be the literal system variable")
	assert.NotEqual(t, "{{my_request_id}}", pathSegments[0], "The UUID part should not be the literal in-place variable")
}
