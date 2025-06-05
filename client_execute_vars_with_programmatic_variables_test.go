package restclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteFile_WithProgrammaticVariables(t *testing.T) {
	// Given
	var interceptedRequest struct {
		URL    string
		Header string
		Body   string
	}
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequest.URL = r.URL.Path // Only path for easier assertion
		interceptedRequest.Header = r.Header.Get("X-Test-Header")
		interceptedRequest.Body = string(bodyBytes)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	clientProgrammaticVars := map[string]interface{}{
		"prog_baseUrl":         server.URL,
		"prog_path":            "items",
		"prog_id":              "prog123",
		"prog_headerVal":       "ProgrammaticHeaderValue",
		"prog_bodyField":       "dataFromProgrammatic",
		"file_var_to_override": "overridden_by_programmatic",
		"PROG_ENV_VAR":         "programmatic_wins_over_env",
	}

	// Set an OS env var that will be overridden by programmatic var
	_ = os.Setenv("PROG_ENV_VAR", "env_value_should_be_overridden")
	defer os.Unsetenv("PROG_ENV_VAR")

	client, err := NewClient(WithVars(clientProgrammaticVars))
	require.NoError(t, err)

	requestFilePath := "testdata/http_request_files/programmatic_variables.http"

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err)
	require.Len(t, responses, 1)
	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Assertions for the request sent to the server
	assert.Equal(t, "/items/prog123", interceptedRequest.URL, "URL path mismatch")
	assert.Equal(t, "ProgrammaticHeaderValue", interceptedRequest.Header, "X-Test-Header mismatch")

	var bodyJSON map[string]string
	err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal request body JSON")

	assert.Equal(t, "dataFromProgrammatic", bodyJSON["field"], "Body field 'field' mismatch")
	assert.Equal(t, "overridden_by_programmatic", bodyJSON["overridden_file_var"], "Body field 'overridden_file_var' mismatch")
	assert.Equal(t, "programmatic_wins_over_env", bodyJSON["env_var_check"], "Body field 'env_var_check' mismatch")
	assert.Equal(t, "file_only", bodyJSON["file_only_check"], "Body field 'file_only_check' mismatch")

	// Also check headers received by the server for variable substitution confirmation
	// These were set up in the new programmatic_variables.http file to check different sources
	assert.Equal(t, "overridden_by_programmatic", resp.Request.Headers.Get("X-File-Var"))
	assert.Equal(t, "programmatic_wins_over_env", resp.Request.Headers.Get("X-Env-Var"))
	assert.Equal(t, "file_only", resp.Request.Headers.Get("X-Unused-File-Var"))
}
