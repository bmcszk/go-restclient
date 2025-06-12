package test

import (
	"testing"

	rc "github.com/bmcszk/go-restclient"

	"github.com/stretchr/testify/assert"
)

func RunValidateResponses_Body_ExactMatch(t *testing.T) {
	t.Helper()
	// Given: Test cases defined in 'tests' slice
	body1 := "Hello World"
	body2 := "Hello Go"
	tests := []struct {
		name             string
		actualResponse   *rc.Response
		expectedFilePath string // Changed from expectedFileContent
		expectedErrCount int
		expectedErrTexts []string // Changed from expectedErrText
	}{
		{
			name:             "matching body",
			actualResponse:   &rc.Response{StatusCode: 200, Status: "200 OK", BodyString: "{\"key\":\"value\"}"},
			expectedFilePath: "testdata/http_response_files/validator_body_exact_match_ok.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "mismatching body",
			actualResponse:   &rc.Response{StatusCode: 200, Status: "200 OK", BodyString: body2},
			// file has `body1` equivalent
			expectedFilePath: "testdata/http_response_files/validator_body_exact_match_ok.hresp",
			expectedErrCount: 1,
			expectedErrTexts: []string{"body mismatch"}, // Diff will be part of the message
		},
		{
			name:             "empty body in file, actual has content",
			actualResponse:   &rc.Response{StatusCode: 200, Status: "200 OK", BodyString: body1},
			expectedFilePath: "testdata/http_response_files/validator_body_exact_no_body_exp.hresp", // Empty body
			expectedErrCount: 1,
			expectedErrTexts: []string{"body mismatch"},
		},
		{
			name:             "empty body in file, actual also empty",
			actualResponse:   &rc.Response{StatusCode: 200, Status: "200 OK", BodyString: ""},
			expectedFilePath: "testdata/http_response_files/validator_body_exact_no_body_exp.hresp", // Empty body
			expectedErrCount: 0,
		},
		// Examples for new test cases based on your original structure:
		// {
		// 	name:                "file has body, actual has no body (nil body string ptr)",
		// 	actualResponse:      &rc.Response{StatusCode: 200, Status: "200 OK" /* BodyString is nil */},
		// 	expectedFilePath:    "testdata/http_response_files/validator_body_exact_match_ok.hresp",
		// 	expectedErrCount:    1,
		// 	expectedErrTexts:    []string{"body mismatch"},
		// },
		// {
		// 	name:                "file has no body, actual has body",
		// 	actualResponse:      &rc.Response{StatusCode: 200, Status: "200 OK", BodyString: body1},
		// No body in file
		// 	expectedFilePath:    "testdata/http_response_files/validator_body_exact_no_body_exp.hresp",
		// 	expectedErrCount:    1,
		// 	expectedErrTexts:    []string{"body mismatch"},
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: actualResponse and expectedFilePath from the test case tt
			client, _ := rc.NewClient()

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
// bodyValidationTestCase defines the structure for test cases used in body validation tests.
type bodyValidationTestCase struct {
	name             string
	actualResponse   *rc.Response
	expectedFilePath string
	expectedErrCount int
	expectedErrTexts []string
}

// runBodyValidationTest is a helper function to execute a single body validation test case.
func runBodyValidationTest(t *testing.T, client *rc.Client, tt bodyValidationTestCase) {
	t.Helper()
	// When
	err := client.ValidateResponses(tt.expectedFilePath, tt.actualResponse)

	// Then
	if tt.expectedErrCount == 0 {
		assert.NoError(t, err)
	} else {
		assertMultierrorContains(t, err, tt.expectedErrCount, tt.expectedErrTexts)
	}
}

// Since ParseExpectedResponseFile does not populate ExpectedResponse.BodyContains,
// this test verifies that the BodyContains logic in ValidateResponses is benign
// (doesn't cause errors) when the expected response comes from a file.
func RunValidateResponses_BodyContains(t *testing.T) {
	t.Helper()
	// Given: Test cases defined in 'tests' slice
	tests := []bodyValidationTestCase{
		{
			name:           "BodyContains logic is not triggered by file (positive case)",
			actualResponse: &rc.Response{StatusCode: 200, Status: "200 OK", BodyString: "Hello World Wide Web"},
			// Exact match for body
			expectedFilePath: "testdata/http_response_files/validator_bodycontains_positive.hresp",
			expectedErrCount: 0,
		},
		{
			name:           "BodyContains logic not triggered, body mismatch handled by exact check",
			actualResponse: &rc.Response{StatusCode: 200, Status: "200 OK", BodyString: "Hello World"},
			// File expects "Universe" - this will cause an exact body mismatch.
			// The BodyContains part of ValidateResponses will not run due to empty expected.BodyContains.
			expectedFilePath: "testdata/http_response_files/validator_bodycontains_mismatch.hresp",
			expectedErrCount: 1,
			expectedErrTexts: []string{"body mismatch"},
		},
	}

	client, _ := rc.NewClient() // Initialize client once

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: actualResponse and expectedFilePath from the test case tt
			runBodyValidationTest(t, client, tt)
		})
	}
}

// TestValidateResponses_BodyNotContains is similar to BodyContains.
// It verifies that BodyNotContains logic in ValidateResponses is benign
// when the expected response comes from a file, as the file cannot specify BodyNotContains.
func RunValidateResponses_BodyNotContains(t *testing.T) {
	t.Helper()
	// Given: Test cases defined in 'tests' slice
	tests := []bodyValidationTestCase{
		{
			name:             "BodyNotContains logic is not triggered by file (positive case)",
			actualResponse:   &rc.Response{StatusCode: 200, Status: "200 OK", BodyString: "Hello World"},
			expectedFilePath: "testdata/http_response_files/validator_bodynotcontains_exact_mismatch.hresp",
			expectedErrCount: 0,
		},
		{
			name: "BodyNotContains logic not triggered, actual contains something, " +
				"file expects different exact body",
			actualResponse:   &rc.Response{StatusCode: 200, Status: "200 OK", BodyString: "Hello Universe"},
			// Exact body mismatch
			expectedFilePath: "testdata/http_response_files/validator_bodynotcontains_exact_mismatch.hresp",
			expectedErrCount: 1,
			expectedErrTexts: []string{"body mismatch"},
		},
	}

	client, _ := rc.NewClient() // Initialize client once

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: actualResponse and expectedFilePath from the test case tt
			runBodyValidationTest(t, client, tt)
		})
	}
}
