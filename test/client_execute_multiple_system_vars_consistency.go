package test

import (
	rc "github.com/bmcszk/go-restclient"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecuteFile_MultipleSystemVarsConsistency tests that multiple system variables 
// defined in file-scoped variables maintain consistency across all requests in the file
func TestExecuteFile_MultipleSystemVarsConsistency(t *testing.T) {
	// Given
	var interceptedRequests []struct {
		URL     string
		Headers http.Header
		Body    string
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequests = append(interceptedRequests, struct {
			URL     string
			Headers http.Header
			Body    string
		}{
			URL:     r.URL.String(),
			Headers: r.Header.Clone(),
			Body:    string(bodyBytes),
		})
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status": "ok"}`)
	}))
	defer server.Close()

	client, _ := rc.NewClient()
	requestFilePath := createTestFileFromTemplate(t, "test/data/system_variables/multiple_system_vars_consistency.http",
		struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err, "ExecuteFile should not return an error")
	require.Len(t, responses, 3, "Expected 3 responses for 3 requests")
	require.Len(t, interceptedRequests, 3, "Should have intercepted 3 requests")

	// All responses should be successful
	for i, resp := range responses {
		assert.NoError(t, resp.Error, "Response %d should not have error", i+1)
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Response %d should have status 200", i+1)
	}

	// Extract and validate UUID consistency
	var extractedUUIDs []string
	var extractedTimestamps []string
	var extractedSessionTokens []string

	// From GET request
	getRequest := interceptedRequests[0]
	urlParts := strings.Split(getRequest.URL, "/")
	uuidFromURL := urlParts[len(urlParts)-1]
	_, err = uuid.Parse(uuidFromURL)
	assert.NoError(t, err, "UUID from GET URL should be valid: %s", uuidFromURL)
	extractedUUIDs = append(extractedUUIDs, uuidFromURL)

	timestampFromHeader := getRequest.Headers.Get("X-Request-Time")
	timestamp, err := strconv.ParseInt(timestampFromHeader, 10, 64)
	assert.NoError(t, err, "Timestamp from GET header should be valid integer: %s", timestampFromHeader)
	assert.Greater(t, timestamp, int64(0), "Timestamp should be positive")
	extractedTimestamps = append(extractedTimestamps, timestampFromHeader)

	sessionTokenFromHeader := getRequest.Headers.Get("X-Session-Token")
	sessionToken, err := strconv.Atoi(sessionTokenFromHeader)
	assert.NoError(t, err, "Session token from GET header should be valid integer: %s", sessionTokenFromHeader)
	assert.GreaterOrEqual(t, sessionToken, 1000000, "Session token should be >= 1000000")
	assert.LessOrEqual(t, sessionToken, 9999999, "Session token should be <= 9999999")
	extractedSessionTokens = append(extractedSessionTokens, sessionTokenFromHeader)

	// From POST request body
	postRequest := interceptedRequests[1]
	var postBody map[string]any
	err = json.Unmarshal([]byte(postRequest.Body), &postBody)
	require.NoError(t, err, "POST body should be valid JSON")

	transactionID, ok := postBody["transaction_id"].(string)
	require.True(t, ok, "POST body should contain transaction_id")
	_, err = uuid.Parse(transactionID)
	assert.NoError(t, err, "UUID from POST transaction_id should be valid: %s", transactionID)
	extractedUUIDs = append(extractedUUIDs, transactionID)

	postTimestamp, ok := postBody["timestamp"].(float64)
	require.True(t, ok, "POST body should contain timestamp as number")
	extractedTimestamps = append(extractedTimestamps, strconv.FormatInt(int64(postTimestamp), 10))

	session, ok := postBody["session"].(map[string]any)
	require.True(t, ok, "POST body should contain session object")
	
	sessionTokenFromBody, ok := session["token"].(string)
	require.True(t, ok, "session should contain token")
	extractedSessionTokens = append(extractedSessionTokens, sessionTokenFromBody)

	sessionId, ok := session["id"].(string)
	require.True(t, ok, "session should contain id")
	_, err = uuid.Parse(sessionId)
	assert.NoError(t, err, "UUID from POST session.id should be valid: %s", sessionId)
	extractedUUIDs = append(extractedUUIDs, sessionId)

	postHeaderTimestamp := postRequest.Headers.Get("X-Request-Time")
	extractedTimestamps = append(extractedTimestamps, postHeaderTimestamp)

	// From PUT request
	putRequest := interceptedRequests[2]
	urlParts = strings.Split(putRequest.URL, "/")
	sessionTokenFromPutURL := urlParts[len(urlParts)-1]
	extractedSessionTokens = append(extractedSessionTokens, sessionTokenFromPutURL)

	uuidFromPutHeader := putRequest.Headers.Get("X-Transaction-ID")
	_, err = uuid.Parse(uuidFromPutHeader)
	assert.NoError(t, err, "UUID from PUT header should be valid: %s", uuidFromPutHeader)
	extractedUUIDs = append(extractedUUIDs, uuidFromPutHeader)

	var putBody map[string]any
	err = json.Unmarshal([]byte(putRequest.Body), &putBody)
	require.NoError(t, err, "PUT body should be valid JSON")

	sessionTokenFromPutBody, ok := putBody["session_token"].(string)
	require.True(t, ok, "PUT body should contain session_token")
	extractedSessionTokens = append(extractedSessionTokens, sessionTokenFromPutBody)

	lastActivity, ok := putBody["last_activity"].(float64)
	require.True(t, ok, "PUT body should contain last_activity as number")
	extractedTimestamps = append(extractedTimestamps, strconv.FormatInt(int64(lastActivity), 10))

	transactionRef, ok := putBody["transaction_ref"].(string)
	require.True(t, ok, "PUT body should contain transaction_ref")
	_, err = uuid.Parse(transactionRef)
	assert.NoError(t, err, "UUID from PUT transaction_ref should be valid: %s", transactionRef)
	extractedUUIDs = append(extractedUUIDs, transactionRef)

	// Verify all UUIDs are the same
	firstUUID := extractedUUIDs[0]
	for i, extractedUUID := range extractedUUIDs {
		assert.Equal(t, firstUUID, extractedUUID, 
			"UUID %d should match the first UUID. All UUIDs should be consistent across the file", i+1)
	}

	// Verify all timestamps are the same
	firstTimestamp := extractedTimestamps[0]
	for i, extractedTimestamp := range extractedTimestamps {
		assert.Equal(t, firstTimestamp, extractedTimestamp, 
			"Timestamp %d should match the first timestamp. All timestamps should be consistent across the file", i+1)
	}

	// Verify all session tokens are the same
	firstSessionToken := extractedSessionTokens[0]
	for i, extractedSessionToken := range extractedSessionTokens {
		assert.Equal(t, firstSessionToken, extractedSessionToken, 
			"Session token %d should match the first session token. All tokens should be consistent", i+1)
	}

	// Log the consistent values for verification
	t.Logf("All requests used the same values:")
	t.Logf("  - UUID: %s", firstUUID)
	t.Logf("  - Timestamp: %s", firstTimestamp)
	t.Logf("  - Session Token: %s", firstSessionToken)
}