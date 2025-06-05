package restclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecuteFile_InPlaceVars_InBody extracted from TestExecuteFile_InPlaceVariables
func TestExecuteFile_InPlaceVars_InBody(t *testing.T) {
	// Given: an .http file with an in-place variable used in the request body
	const userIdValue = "user-from-var-inplace-body-789"                                 // Slightly changed for independence
	const expectedBody = `{"id": "user-from-var-inplace-body-789", "status": "pending"}` // Adjusted expected body

	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		capturedBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close() // Added defer to close body
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fileContent := fmt.Sprintf(`
@my_user_id = %s

### Test Request With In-Place Var in Body
POST %s/submit
Content-Type: application/json

{
  "id": "{{my_user_id}}",
  "status": "pending"
}
`, userIdValue, server.URL)

	requestFilePath := createTempHTTPFileFromString(t, fileContent)

	client, err := NewClient()
	require.NoError(t, err)

	// When: the .http file is executed
	results, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: no error should occur and the body should be correctly substituted
	require.NoError(t, execErr, "ExecuteFile should not return an error")
	require.Len(t, results, 1, "Should have one result")
	require.Nil(t, results[0].Error, "Request execution error should be nil")
	assert.JSONEq(t, expectedBody, string(capturedBody), "The request body should be correctly substituted with the in-place variable")
}
