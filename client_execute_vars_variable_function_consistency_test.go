package restclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteFile_VariableFunctionConsistency(t *testing.T) {
	// This server will capture the path, headers, and body to check for consistency.
	var capturedPathUUID, capturedHeaderUUID, capturedBodyUUID, capturedBodyAnotherUUID string
	var capturedHeaderTimestamp, capturedBodyTimestamp string
	var capturedHeaderRandomInt, capturedBodyRandomInt string

	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) == 3 && pathParts[1] == "test-uuid" {
			capturedPathUUID = pathParts[2]
		} else {
			t.Logf("Unexpected path format: %s", r.URL.Path)
		}

		capturedHeaderUUID = r.Header.Get("X-Request-UUID")
		capturedHeaderTimestamp = r.Header.Get("X-Request-Timestamp")
		capturedHeaderRandomInt = r.Header.Get("X-Request-RandomInt")

		bodyBytes, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var bodyJSON map[string]interface{}
		err = json.Unmarshal(bodyBytes, &bodyJSON)
		require.NoError(t, err)

		if id, ok := bodyJSON["id"].(string); ok {
			capturedBodyUUID = id
		}
		if anotherID, ok := bodyJSON["another_id"].(string); ok {
			capturedBodyAnotherUUID = anotherID
		}
		if ts, ok := bodyJSON["timestamp"].(string); ok {
			capturedBodyTimestamp = ts
		}
		if ri, ok := bodyJSON["randomInt"].(string); ok {
			capturedBodyRandomInt = ri
		}

		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL)) // Set BaseURL to mock server
	require.NoError(t, err)

	requestFilePath := "testdata/http_request_files/variable_function_consistency.rest"

	responses, err := client.ExecuteFile(context.Background(), requestFilePath)
	require.NoError(t, err, "ExecuteFile should not return an error")
	require.Len(t, responses, 1, "Should have one response")

	resp := responses[0]
	require.NotNil(t, resp, "Response object should not be nil")
	assert.NoError(t, resp.Error, "Error in response object should be nil")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be OK")

	// Assert that captured values are not the placeholders themselves
	assert.NotEmpty(t, capturedPathUUID, "Path UUID should not be empty")
	assert.NotEqual(t, "{{$uuid}}", capturedPathUUID, "Path UUID should be resolved from {{$uuid}}")
	_, parseUUIDErr := uuid.Parse(capturedPathUUID)
	assert.NoError(t, parseUUIDErr, "Captured Path UUID should be a valid UUID")

	assert.NotEmpty(t, capturedHeaderTimestamp, "Header Timestamp should not be empty")
	assert.NotEqual(t, "{{$timestamp}}", capturedHeaderTimestamp, "Header Timestamp should be resolved")
	_, parseIntErr := strconv.ParseInt(capturedHeaderTimestamp, 10, 64)
	assert.NoError(t, parseIntErr, "Captured Header Timestamp should be a valid integer")

	assert.NotEmpty(t, capturedHeaderRandomInt, "Header RandomInt should not be empty")
	assert.NotEqual(t, "{{$randomInt}}", capturedHeaderRandomInt, "Header RandomInt should be resolved")
	_, parseIntErr = strconv.ParseInt(capturedHeaderRandomInt, 10, 64) // Re-check, should be parsable as int
	assert.NoError(t, parseIntErr, "Captured Header RandomInt should be a valid integer")

	// Assert UUID consistency
	assert.Equal(t, capturedPathUUID, capturedHeaderUUID, "Path UUID and Header UUID should be the same")
	assert.Equal(t, capturedPathUUID, capturedBodyUUID, "Path UUID and Body UUID should be the same")
	assert.Equal(t, capturedPathUUID, capturedBodyAnotherUUID, "Path UUID and Body Another UUID should be the same")

	// Assert Timestamp consistency
	assert.Equal(t, capturedHeaderTimestamp, capturedBodyTimestamp, "Header Timestamp and Body Timestamp should be the same")

	// Assert RandomInt consistency
	assert.Equal(t, capturedHeaderRandomInt, capturedBodyRandomInt, "Header RandomInt and Body RandomInt should be the same")

	// Additionally, verify that the actual substituted values in the request object (client-side) are consistent.
	parsedReq := resp.Request
	require.NotNil(t, parsedReq)

	// Check substituted URL Path
	// capturedPathUUID is what the server received and should be the actual resolved UUID.
	assert.Equal(t, "/test-uuid/"+capturedPathUUID, parsedReq.URL.Path, "Parsed request URL path mismatch")

	// Check substituted Header
	assert.Equal(t, capturedPathUUID, parsedReq.Headers.Get("X-Request-UUID"))
	assert.Equal(t, capturedHeaderTimestamp, parsedReq.Headers.Get("X-Request-Timestamp"))
	assert.Equal(t, capturedHeaderRandomInt, parsedReq.Headers.Get("X-Request-RandomInt"))

	// Check substituted Body
	var finalBodyJSON map[string]interface{}
	err = json.Unmarshal([]byte(parsedReq.RawBody), &finalBodyJSON)
	require.NoError(t, err, "Failed to parse RawBody as JSON")

	assert.Equal(t, capturedPathUUID, finalBodyJSON["id"].(string))
	assert.Equal(t, capturedPathUUID, finalBodyJSON["another_id"].(string))
	assert.Equal(t, capturedHeaderTimestamp, finalBodyJSON["timestamp"].(string))
	assert.Equal(t, capturedHeaderRandomInt, finalBodyJSON["randomInt"].(string))
}
