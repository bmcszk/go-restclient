package restclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateResponses_Body_ExactMatch(t *testing.T) {
	// Given: Test cases defined in 'tests' slice
	body1 := "Hello World"
	body2 := "Hello Go"
	tests := []struct {
		name             string
		actualResponse   *Response
		expectedFilePath string // Changed from expectedFileContent
		expectedErrCount int
		expectedErrTexts []string // Changed from expectedErrText
	}{
		{
			name:             "matching body",
			actualResponse:   &Response{StatusCode: 200, Status: "200 OK", BodyString: "{\"key\":\"value\"}"},
			expectedFilePath: "testdata/http_response_files/validator_body_exact_match_ok.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "mismatching body",
			actualResponse:   &Response{StatusCode: 200, Status: "200 OK", BodyString: body2},
			expectedFilePath: "testdata/http_response_files/validator_body_exact_match_ok.hresp", // file has `body1` equivalent
			expectedErrCount: 1,
			expectedErrTexts: []string{"body mismatch"}, // Diff will be part of the message
		},
		{
			name:             "empty body in file, actual has content",
			actualResponse:   &Response{StatusCode: 200, Status: "200 OK", BodyString: body1},
			expectedFilePath: "testdata/http_response_files/validator_body_exact_no_body_exp.hresp", // Empty body
			expectedErrCount: 1,
			expectedErrTexts: []string{"body mismatch"},
		},
		{
			name:             "empty body in file, actual also empty",
			actualResponse:   &Response{StatusCode: 200, Status: "200 OK", BodyString: ""},
			expectedFilePath: "testdata/http_response_files/validator_body_exact_no_body_exp.hresp", // Empty body
			expectedErrCount: 0,
		},
		// Examples for new test cases based on your original structure:
		// {
		// 	name:                "file has body, actual has no body (nil body string ptr)",
		// 	actualResponse:      &Response{StatusCode: 200, Status: "200 OK" /* BodyString is nil */},
		// 	expectedFilePath:    "testdata/http_response_files/validator_body_exact_match_ok.hresp",
		// 	expectedErrCount:    1,
		// 	expectedErrTexts:    []string{"body mismatch"},
		// },
		// {
		// 	name:                "file has no body, actual has body",
		// 	actualResponse:      &Response{StatusCode: 200, Status: "200 OK", BodyString: body1},
		// 	expectedFilePath:    "testdata/http_response_files/validator_body_exact_no_body_exp.hresp", // No body in file
		// 	expectedErrCount:    1,
		// 	expectedErrTexts:    []string{"body mismatch"},
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: actualResponse and expectedFilePath from the test case tt
			client, _ := NewClient()

			// When
			err := client.ValidateResponses(tt.expectedFilePath, tt.actualResponse)

			// Then
			if tt.expectedErrCount == 0 {
				assert.NoError(t, err)
			} else {
				assertMultierrorContains(t, err, tt.expectedErrCount, tt.expectedErrTexts)
			}
		})
	}
}

// TestValidateResponses_BodyContains tests the BodyContains logic.
// Since ParseExpectedResponseFile does not populate ExpectedResponse.BodyContains,
// this test verifies that the BodyContains logic in ValidateResponses is benign
// (doesn't cause errors) when the expected response comes from a file.
func TestValidateResponses_BodyContains(t *testing.T) {
	// Given: Test cases defined in 'tests' slice
	tests := []struct {
		name           string
		actualResponse *Response
		// expectedFileContent will be simple, as it cannot express BodyContains - Now expectedFilePath
		expectedFilePath string // Was expectedFileContent
		expectedErrCount int    // Should be 0 if actual matches file's explicit parts
		expectedErrTexts []string
	}{
		{
			name:           "BodyContains logic is not triggered by file (positive case)",
			actualResponse: &Response{StatusCode: 200, Status: "200 OK", BodyString: "Hello World Wide Web"},
			// Exact match for body
			expectedFilePath: "testdata/http_response_files/validator_bodycontains_positive.hresp",
			expectedErrCount: 0,
		},
		{
			name:           "BodyContains logic not triggered, body mismatch handled by exact check",
			actualResponse: &Response{StatusCode: 200, Status: "200 OK", BodyString: "Hello World"},
			// File expects "Universe" - this will cause an exact body mismatch.
			// The BodyContains part of ValidateResponses will not run due to empty expected.BodyContains.
			expectedFilePath: "testdata/http_response_files/validator_bodycontains_mismatch.hresp",
			expectedErrCount: 1,
			expectedErrTexts: []string{"body mismatch"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: actualResponse and expectedFilePath from the test case tt
			client, _ := NewClient()

			// When
			err := client.ValidateResponses(tt.expectedFilePath, tt.actualResponse)

			// Then
			if tt.expectedErrCount == 0 {
				assert.NoError(t, err)
			} else {
				assertMultierrorContains(t, err, tt.expectedErrCount, tt.expectedErrTexts)
			}
		})
	}
}

// TestValidateResponses_BodyNotContains is similar to BodyContains.
// It verifies that BodyNotContains logic in ValidateResponses is benign
// when the expected response comes from a file, as the file cannot specify BodyNotContains.
func TestValidateResponses_BodyNotContains(t *testing.T) {
	// Given: Test cases defined in 'tests' slice
	tests := []struct {
		name             string
		actualResponse   *Response
		expectedFilePath string // Was expectedFileContent
		expectedErrCount int
		expectedErrTexts []string
	}{
		{
			name:             "BodyNotContains logic is not triggered by file (positive case)",
			actualResponse:   &Response{StatusCode: 200, Status: "200 OK", BodyString: "Hello World"},
			expectedFilePath: "testdata/http_response_files/validator_bodynotcontains_exact_mismatch.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "BodyNotContains logic not triggered, actual contains something, file expects different exact body",
			actualResponse:   &Response{StatusCode: 200, Status: "200 OK", BodyString: "Hello Universe"},
			expectedFilePath: "testdata/http_response_files/validator_bodynotcontains_exact_mismatch.hresp", // Exact body mismatch
			expectedErrCount: 1,
			expectedErrTexts: []string{"body mismatch"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: actualResponse and expectedFilePath from the test case tt
			client, _ := NewClient()

			// When
			err := client.ValidateResponses(tt.expectedFilePath, tt.actualResponse)

			// Then
			if tt.expectedErrCount == 0 {
				assert.NoError(t, err)
			} else {
				assertMultierrorContains(t, err, tt.expectedErrCount, tt.expectedErrTexts)
			}
		})
	}
}
