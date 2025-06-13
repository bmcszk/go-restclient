package test

import (
	rc "github.com/bmcszk/go-restclient"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecuteFile_UuidVariableConsistency tests that a variable defined as @scenarioId = {{$uuid}}
// maintains the same value throughout all uses in the file.
// This test verifies that once a variable is assigned a system variable value like {{$uuid}},
// that value should be consistent across all requests within the same file execution.
func TestExecuteFile_UuidVariableConsistency(t *testing.T) {
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
		
		// Mock different responses based on method
		switch r.Method {
		case "GET":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, `{"uuid": "%s"}`, strings.TrimPrefix(r.URL.Path, "/uuid/"))
		case "POST", "PUT":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, `{"json": %s, "headers": {"X-Scenario-Id": "%s"}}`, 
				string(bodyBytes), r.Header.Get("X-Scenario-ID"))
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client, _ := rc.NewClient()
	requestFilePath := createTestFileFromTemplate(t, "test/data/system_variables/uuid_consistency.http",
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

	// Extract UUID from each request and verify they're all the same
	var extractedUUIDs []string

	// From GET request URL
	getRequest := interceptedRequests[0]
	urlParts := strings.Split(getRequest.URL, "/")
	uuidFromURL := urlParts[len(urlParts)-1]
	_, err = uuid.Parse(uuidFromURL)
	assert.NoError(t, err, "UUID from GET URL should be valid: %s", uuidFromURL)
	extractedUUIDs = append(extractedUUIDs, uuidFromURL)

	// From POST request body
	postRequest := interceptedRequests[1]
	var postBody map[string]any
	err = json.Unmarshal([]byte(postRequest.Body), &postBody)
	require.NoError(t, err, "POST body should be valid JSON")
	
	scenarioID, ok := postBody["scenario_id"].(string)
	require.True(t, ok, "POST body should contain scenario_id")
	_, err = uuid.Parse(scenarioID)
	assert.NoError(t, err, "UUID from POST scenario_id should be valid: %s", scenarioID)
	extractedUUIDs = append(extractedUUIDs, scenarioID)

	testData, ok := postBody["test_data"].(map[string]any)
	require.True(t, ok, "POST body should contain test_data object")
	
	testDataUuid, ok := testData["uuid"].(string)
	require.True(t, ok, "test_data should contain uuid")
	_, err = uuid.Parse(testDataUuid)
	assert.NoError(t, err, "UUID from POST test_data.uuid should be valid: %s", testDataUuid)
	extractedUUIDs = append(extractedUUIDs, testDataUuid)

	metadata, ok := testData["metadata"].(map[string]any)
	require.True(t, ok, "test_data should contain metadata object")
	
	metadataScenario, ok := metadata["scenario"].(string)
	require.True(t, ok, "metadata should contain scenario")
	_, err = uuid.Parse(metadataScenario)
	assert.NoError(t, err, "UUID from POST metadata.scenario should be valid: %s", metadataScenario)
	extractedUUIDs = append(extractedUUIDs, metadataScenario)

	// From PUT request header
	putRequest := interceptedRequests[2]
	uuidFromHeader := putRequest.Headers.Get("X-Scenario-ID")
	_, err = uuid.Parse(uuidFromHeader)
	assert.NoError(t, err, "UUID from PUT header should be valid: %s", uuidFromHeader)
	extractedUUIDs = append(extractedUUIDs, uuidFromHeader)

	// From PUT request body
	var putBody map[string]any
	err = json.Unmarshal([]byte(putRequest.Body), &putBody)
	require.NoError(t, err, "PUT body should be valid JSON")
	
	updateScenario, ok := putBody["update_scenario"].(string)
	require.True(t, ok, "PUT body should contain update_scenario")
	_, err = uuid.Parse(updateScenario)
	assert.NoError(t, err, "UUID from PUT update_scenario should be valid: %s", updateScenario)
	extractedUUIDs = append(extractedUUIDs, updateScenario)

	// Verify all UUIDs are the same
	firstUUID := extractedUUIDs[0]
	for i, extractedUUID := range extractedUUIDs {
		assert.Equal(t, firstUUID, extractedUUID, 
			"UUID %d should match the first UUID. All UUIDs should be consistent across the file", i+1)
	}

	// Log the consistent UUID for verification
	t.Logf("All requests used the same UUID: %s", firstUUID)
	t.Logf("UUID was found in:")
	t.Logf("  - GET URL: %s", uuidFromURL)
	t.Logf("  - POST body scenario_id: %s", scenarioID)
	t.Logf("  - POST body test_data.uuid: %s", testDataUuid)
	t.Logf("  - POST body metadata.scenario: %s", metadataScenario)
	t.Logf("  - PUT header X-Scenario-ID: %s", uuidFromHeader)
	t.Logf("  - PUT body update_scenario: %s", updateScenario)
}