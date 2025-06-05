package restclient

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecuteFile_InPlaceVars_MalformedDefinitions tests the behavior of in-place variable
// substitution when the variable definitions are malformed.
func TestExecuteFile_InPlaceVars_MalformedDefinitions(t *testing.T) {
	tests := []struct {
		name            string
		httpFileContent string
		expectedError   string // Substring of the expected error message from ExecuteFile
	}{
		{
			name: "name_only_no_equals_no_value_extracted", // Modified for isolation
			httpFileContent: `
@name_only_var_ext

### Test Request
GET http://localhost/test_ext
`,
			expectedError: "malformed in-place variable definition, missing '=' or name",
		},
		{
			name: "no_name_equals_value_extracted", // Modified for isolation
			httpFileContent: `
@=value_only_val_ext

### Test Request
GET http://localhost/test_ext
`,
			expectedError: "variable name cannot be empty in definition",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Given: an .http file with a malformed in-place variable definition
			requestFilePath := createTempHTTPFileFromString(t, tc.httpFileContent)
			client, err := NewClient()
			require.NoError(t, err)

			// When: the .http file is executed
			_, execErr := client.ExecuteFile(context.Background(), requestFilePath)

			// Then: an error should occur indicating a parsing failure due to the malformed variable
			require.Error(t, execErr, "ExecuteFile should return an error for malformed variable definition")
			assert.Contains(t, execErr.Error(), "failed to parse request file", "Error message should indicate parsing failure")
			assert.Contains(t, execErr.Error(), tc.expectedError, "Error message should contain specific malformed reason")
		})
	}
}
