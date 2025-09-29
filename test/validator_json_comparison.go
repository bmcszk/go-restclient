package test

import (
	"testing"

	rc "github.com/bmcszk/go-restclient"

	"github.com/stretchr/testify/assert"
)

// RunValidateResponses_JSON_WhitespaceComparison tests JSON comparison with different whitespace
func RunValidateResponses_JSON_WhitespaceComparison(t *testing.T) {
	t.Helper()

	// Test case: JSON with different whitespace should fail currently (before implementation)
	// but should pass after JSON whitespace-agnostic comparison is implemented
	tests := []struct {
		name                          string
		actualResponse                *rc.Response
		expectedFilePath              string
		expectedErrCount              int // Expected error count BEFORE implementation (should be 1 for failing tests)
		expectedErrTexts              []string
		shouldPassAfterImplementation bool // Whether this test should pass after implementation
	}{
		{
			name:                          "JSON with different whitespace - should fail now, pass later",
			actualResponse:                &rc.Response{StatusCode: 200, Status: "200 OK", Headers: map[string][]string{"Content-Type": {"application/json"}}, BodyString: "{\"key\": \"value\"}"},
			expectedFilePath:              "test/data/http_response_files/validator_json_whitespace_different.hresp",
			expectedErrCount:              1, // Should fail before implementation
			expectedErrTexts:              []string{"body mismatch"},
			shouldPassAfterImplementation: true,
		},
		{
			name:                          "JSON with different indentation - should fail now, pass later",
			actualResponse:                &rc.Response{StatusCode: 200, Status: "200 OK", Headers: map[string][]string{"Content-Type": {"application/json"}}, BodyString: "{\n  \"key\": \"value\"\n}"},
			expectedFilePath:              "test/data/http_response_files/validator_json_indentation_different.hresp",
			expectedErrCount:              1, // Should fail before implementation
			expectedErrTexts:              []string{"body mismatch"},
			shouldPassAfterImplementation: true,
		},
		{
			name:                          "JSON with different line breaks - should fail now, pass later",
			actualResponse:                &rc.Response{StatusCode: 200, Status: "200 OK", Headers: map[string][]string{"Content-Type": {"application/json"}}, BodyString: "{\"key\":\"value\",\"nested\":{\"id\":1}}"},
			expectedFilePath:              "test/data/http_response_files/validator_json_linebreaks_different.hresp",
			expectedErrCount:              1, // Should fail before implementation
			expectedErrTexts:              []string{"body mismatch"},
			shouldPassAfterImplementation: true,
		},
		{
			name:                          "Non-JSON content should still work as before",
			actualResponse:                &rc.Response{StatusCode: 200, Status: "200 OK", BodyString: "{\"key\":\"value\"}"}, // Match the expected file content
			expectedFilePath:              "test/data/http_response_files/validator_body_exact_match_ok.hresp",                // Non-JSON
			expectedErrCount:              0,                                                                                  // Should pass before and after implementation
			shouldPassAfterImplementation: false,                                                                              // This is not a JSON-specific test
		},
		{
			name:                          "JSON with different indentation - should fail now, pass later",
			actualResponse:                &rc.Response{StatusCode: 200, Status: "200 OK", BodyString: "{\n  \"key\": \"value\"\n}"},
			expectedFilePath:              "test/data/http_response_files/validator_json_indentation_different.hresp",
			expectedErrCount:              1, // Should fail before implementation
			expectedErrTexts:              []string{"body mismatch"},
			shouldPassAfterImplementation: true,
		},
		{
			name:                          "JSON with different line breaks - should fail now, pass later",
			actualResponse:                &rc.Response{StatusCode: 200, Status: "200 OK", BodyString: "{\"key\":\"value\",\"nested\":{\"id\":1}}"},
			expectedFilePath:              "test/data/http_response_files/validator_json_linebreaks_different.hresp",
			expectedErrCount:              1, // Should fail before implementation
			expectedErrTexts:              []string{"body mismatch"},
			shouldPassAfterImplementation: true,
		},
		{
			name:                          "Non-JSON content should still work as before",
			actualResponse:                &rc.Response{StatusCode: 200, Status: "200 OK", BodyString: "Hello World"},
			expectedFilePath:              "test/data/http_response_files/validator_body_exact_match_ok.hresp", // Non-JSON
			expectedErrCount:              0,                                                                   // Should pass before and after implementation
			shouldPassAfterImplementation: false,                                                               // This is not a JSON-specific test
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			client, _ := rc.NewClient()
			responses := []*rc.Response{tt.actualResponse}

			// When
			err := client.ValidateResponses(tt.expectedFilePath, responses...)

			// Then
			if tt.expectedErrCount == 0 {
				assert.NoError(t, err, "Expected no validation error")
			} else {
				assertMultierrorContains(t, err, tt.expectedErrCount, tt.expectedErrTexts)
			}

			// TODO: After implementation, add logic to verify that shouldPassAfterImplementation tests actually pass
			// This will be added when we implement the feature
		})
	}
}

// RunValidateResponses_JSON_WithPlaceholders tests JSON comparison with placeholders
func RunValidateResponses_JSON_WithPlaceholders(t *testing.T) {
	t.Helper()

	tests := []struct {
		name                          string
		actualResponse                *rc.Response
		expectedFilePath              string
		expectedErrCount              int
		expectedErrTexts              []string
		shouldPassAfterImplementation bool
	}{
		{
			name:                          "JSON with placeholders and different formatting - should fail now, pass later",
			actualResponse:                &rc.Response{StatusCode: 200, Status: "200 OK", Headers: map[string][]string{"Content-Type": {"application/json"}}, BodyString: "{\"id\": \"550e8400-e29b-41d4-a716-446655440000\"}"},
			expectedFilePath:              "test/data/http_response_files/validator_json_placeholders_formatting.hresp",
			expectedErrCount:              1, // Should fail before implementation due to formatting differences
			expectedErrTexts:              []string{"body mismatch"},
			shouldPassAfterImplementation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			client, _ := rc.NewClient()
			responses := []*rc.Response{tt.actualResponse}

			// When
			err := client.ValidateResponses(tt.expectedFilePath, responses...)

			// Then
			if tt.expectedErrCount == 0 {
				assert.NoError(t, err, "Expected no validation error")
			} else {
				assertMultierrorContains(t, err, tt.expectedErrCount, tt.expectedErrTexts)
			}
		})
	}
}
