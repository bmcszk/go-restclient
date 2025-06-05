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

// TestExecuteFile_InPlaceVars_SubstInBody extracted from TestExecuteFile_InPlaceVariables
func TestExecuteFile_InPlaceVars_SubstInBody(t *testing.T) {
	// Given: an .http file with an in-place variable used in a JSON request body
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var errReadBody error
		capturedBody, errReadBody = io.ReadAll(r.Body)
		require.NoError(t, errReadBody) // Use errReadBody
		defer r.Body.Close()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fileContent := fmt.Sprintf(`
@user_id = user123

### Test Request With Body Var
POST %s/users
Content-Type: application/json

{
  "id": "{{user_id}}",
  "status": "active"
}
`, server.URL)

	requestFilePath := createTempHTTPFileFromString(t, fileContent)

	client, err := NewClient()
	require.NoError(t, err)

	// When: the .http file is executed
	_, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: no error should occur and the body should be substituted correctly
	require.NoError(t, execErr, "ExecuteFile should not return an error")
	assert.JSONEq(t, `{"id": "user123", "status": "active"}`, string(capturedBody), "The request body should be correctly substituted")
}
