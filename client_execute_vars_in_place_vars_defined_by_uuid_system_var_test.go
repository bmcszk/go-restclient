package restclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecuteFile_InPlaceVars_DefinedByUUIDSystemVar tests the scenario where an in-place variable
// is defined by the {{$uuid}} system variable and used in a request header.
func TestExecuteFile_InPlaceVars_DefinedByUUIDSystemVar(t *testing.T) {
	// Given: an .http file with an in-place variable defined by the {{$uuid}} system variable
	var capturedHeaderValue string
	const headerKey = "X-Request-ID-UUID-Test" // Changed to avoid potential conflicts

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaderValue = r.Header.Get(headerKey)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fileContent := fmt.Sprintf(`
@my_request_uuid = {{$uuid}}

### Test Request With UUID In-Place Var in Header
GET %s/some/path/uuidtest
%s: {{my_request_uuid}}
`, server.URL, headerKey)

	requestFilePath := createTempHTTPFileFromString(t, fileContent)

	client, err := NewClient()
	require.NoError(t, err)

	// When: the .http file is executed
	results, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: no error should occur and the header should contain a valid UUID
	require.NoError(t, execErr, "ExecuteFile should not return an error")
	require.Len(t, results, 1, "Should have one result")
	require.Nil(t, results[0].Error, "Request execution error should be nil")

	// Validate that the captured header value is a valid UUID
	_, err = uuid.Parse(capturedHeaderValue)
	assert.NoError(t, err, "Header value should be a valid UUID. Got: %s", capturedHeaderValue)
	assert.NotEmpty(t, capturedHeaderValue, "Captured UUID header should not be empty")
}
