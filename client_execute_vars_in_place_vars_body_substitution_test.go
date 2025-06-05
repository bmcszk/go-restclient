package restclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecuteFile_InPlaceVars_BodySubstitution extracted from TestExecuteFile_InPlaceVariables
func TestExecuteFile_InPlaceVars_BodySubstitution(t *testing.T) {
	// Given
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var errReadBody error
		capturedBody, errReadBody = io.ReadAll(r.Body)
		require.NoError(t, errReadBody)
		defer r.Body.Close()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"created"}`))
	}))
	defer server.Close()

	httpFileContent := fmt.Sprintf(`
@product_name = SuperWidget
@product_id = SW1000
@product_price = 49.99

POST %s/products
Content-Type: application/json

{
  "id": "{{product_id}}",
  "name": "{{product_name}}",
  "price": {{product_price}}
}
`, server.URL)

	requestFilePath := createTempHTTPFileFromString(t, httpFileContent)

	client, err := NewClient()
	require.NoError(t, err)

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err, "ExecuteFile should not return an error")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	assert.NoError(t, resp.Error, "Response error should be nil")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code mismatch")

	// Verify the server received the request with the substituted body
	expectedBodyJSON := `{
  "id": "SW1000",
  "name": "SuperWidget",
  "price": 49.99
}`
	assert.JSONEq(t, expectedBodyJSON, string(capturedBody), "Captured body by server mismatch")

	// Verify ParsedFile.FileVariables
	file, err := os.Open(requestFilePath)
	require.NoError(t, err)
	defer file.Close()
	parsedFile, pErr := ParseRequests(file, requestFilePath, client, make(map[string]string), os.LookupEnv, make(map[string]string), nil)
	require.NoError(t, pErr)
	require.NotNil(t, parsedFile.FileVariables)
	assert.Equal(t, "SuperWidget", parsedFile.FileVariables["product_name"])
	assert.Equal(t, "SW1000", parsedFile.FileVariables["product_id"])
	assert.Equal(t, "49.99", parsedFile.FileVariables["product_price"])
}
