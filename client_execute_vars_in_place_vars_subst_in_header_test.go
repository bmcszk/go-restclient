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

// TestExecuteFile_InPlaceVars_SubstInHeader extracted from TestExecuteFile_InPlaceVariables
func TestExecuteFile_InPlaceVars_SubstInHeader(t *testing.T) {
	// Given: an .http file with an in-place variable used in a header
	var capturedHeaderValue string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaderValue = r.Header.Get("X-Custom-Header")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fileContent := fmt.Sprintf(`
@my_header_value = secret-token

### Test Request With Header Var
GET %s/somepath
X-Custom-Header: {{my_header_value}}
`, server.URL)

	requestFilePath := createTempHTTPFileFromString(t, fileContent)

	client, err := NewClient()
	require.NoError(t, err)

	// When: the .http file is executed
	_, execErr := client.ExecuteFile(context.Background(), requestFilePath)

	// Then: no error should occur and the header should be substituted correctly
	require.NoError(t, execErr, "ExecuteFile should not return an error")
	assert.Equal(t, "secret-token", capturedHeaderValue, "The X-Custom-Header should be correctly substituted")
}
