package restclient

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateResponses_StatusString(t *testing.T) {
	// Given: Test cases defined in 'tests' slice
	tests := []struct {
		name             string
		actualResponse   *Response
		expectedFilePath string
		expectedErrCount int
		expectedErrText  string // Kept for single error cases not using multierror helper
		expectedErrTexts []string
	}{
		{
			name:             "matching status string",
			actualResponse:   &Response{StatusCode: 200, Status: "200 OK"},
			expectedFilePath: "testdata/http_response_files/validator_body_exact_no_body_exp.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "mismatching status string",
			actualResponse:   &Response{StatusCode: 200, Status: "200 Something Else"},
			expectedFilePath: "testdata/http_response_files/validator_body_exact_no_body_exp.hresp",
			expectedErrCount: 1,
			expectedErrText:  "status string mismatch: expected '200 OK', got '200 Something Else'",
		},
		{
			name:             "actual status string is correct, expected file has only status code",
			actualResponse:   &Response{StatusCode: 200, Status: "200 OK"},
			expectedFilePath: "testdata/http_response_files/validator_partial_status_code_mismatch.hresp",
			expectedErrCount: 1,
			expectedErrText:  "status string mismatch: expected '200', got '200 OK'",
		},
		{
			name:             "mismatching status code, status strings also mismatch",
			actualResponse:   &Response{StatusCode: 404, Status: "404 Not Found"},
			expectedFilePath: "testdata/http_response_files/validator_body_exact_no_body_exp.hresp",
			expectedErrCount: 2,
			expectedErrTexts: []string{"status code mismatch: expected 200, got 404", "status string mismatch: expected '200 OK', got '404 Not Found'"},
		},
		{
			name:             "matching status code, expected file only code, actual also only code in status",
			actualResponse:   &Response{StatusCode: 200, Status: "200"},
			expectedFilePath: "testdata/http_response_files/validator_partial_status_code_mismatch.hresp",
			expectedErrCount: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: actualResponse and expectedFilePath from the test case tt

			// When
			err := ValidateResponses(tt.expectedFilePath, tt.actualResponse)

			// Then
			if tt.expectedErrCount == 0 {
				assert.NoError(t, err)
			} else {
				// Use direct assertion for single, specific error messages if tt.expectedErrText is set
				// Otherwise, use the multierror helper.
				if tt.expectedErrText != "" && tt.expectedErrCount == 1 && len(tt.expectedErrTexts) == 0 {
					require.Error(t, err)
					merr, ok := err.(*multierror.Error)
					require.True(t, ok, "Expected a multierror.Error for single error case with expectedErrText")
					require.Len(t, merr.Errors, 1)
					assert.Contains(t, merr.Errors[0].Error(), tt.expectedErrText)
				} else {
					assertMultierrorContains(t, err, tt.expectedErrCount, tt.expectedErrTexts)
				}
			}
		})
	}
}

func TestValidateResponses_StatusCode(t *testing.T) {
	// Given: Test cases defined in 'tests' slice
	tests := []struct {
		name                string
		actualResponse      *Response
		actualResponseCode  *int   // Pointer to allow nil
		expectedFileContent string // To become FilePath
		expectedFilePath    string // New
		expectedErrCount    int
		expectedErrText     string
	}{
		{
			name:               "matching status code only",
			actualResponseCode: intPtr(200),
			expectedFilePath:   "testdata/http_response_files/validator_partial_status_code_mismatch.hresp",
			expectedErrCount:   0,
		},
		{
			name:               "mismatching status code only",
			actualResponseCode: intPtr(404),
			expectedFilePath:   "testdata/http_response_files/validator_partial_status_code_mismatch.hresp", // Expect 200
			expectedErrCount:   2,                                                                           // Expects 2 errors: status code and status string mismatch
			expectedErrText:    "",                                                                          // Clear this as we use currentExpectedErrTexts now
			// currentExpectedErrTexts will be: ["status code mismatch: expected 200, got 404", "status string mismatch: expected '200', got '404'"]
		},
		{
			name:               "nil actual status code (should not happen with real http.Response)",
			actualResponseCode: nil, // Actual code is 0, so if file expects 200, it's a mismatch
			expectedFilePath:   "testdata/http_response_files/validator_partial_status_code_mismatch.hresp",
			expectedErrCount:   2,  // Expects 2 errors: status code and status string mismatch
			expectedErrText:    "", // Clear this
			// currentExpectedErrTexts will be: ["status code mismatch: expected 200, got 0", "status string mismatch: expected '200', got '0'"]
		},
		{
			name:               "nil expected status code (file has no status line)",
			actualResponseCode: intPtr(200),
			expectedFilePath:   "testdata/http_response_files/validator_status_code_nil_expected.hresp",
			expectedErrCount:   2, // This will fail parsing + count mismatch
			// currentExpectedErrTexts will be: ["failed to parse expected response file", "mismatch in number of responses"]
		},
		{
			// This test case specifically tests when the expected file ONLY has a status code (no text reason phrase)
			// and the actual response also only provides a status code (no text reason phrase in its .Status field).
			name:               "matching status code, actual and expected only have code",
			actualResponseCode: intPtr(200),
			expectedFilePath:   "testdata/http_response_files/validator_status_code_only.hresp",
			expectedErrCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			actual := &Response{}
			if tt.actualResponseCode != nil {
				actual.StatusCode = *tt.actualResponseCode
				actual.Status = fmt.Sprintf("%d", actual.StatusCode) // Populate actual.Status
			} else {
				actual.Status = "0"   // Default if StatusCode is 0 (from nil actualResponseCode)
				actual.StatusCode = 0 // ensure StatusCode is also 0 if actualResponseCode is nil
			}

			currentExpectedErrCount := tt.expectedErrCount
			var currentExpectedErrTexts []string

			if tt.name == "mismatching status code" {
				currentExpectedErrTexts = []string{"status code mismatch: expected 200, got 404", "status string mismatch: expected '200', got '404'"}
			} else if tt.name == "nil actual status code (should not happen with real http.Response)" {
				currentExpectedErrTexts = []string{"status code mismatch: expected 200, got 0", "status string mismatch: expected '200', got '0'"}
			} else if tt.name == "nil expected status code (file has no status line)" {
				currentExpectedErrTexts = []string{
					"failed to parse expected response file", // This will contain the "invalid status code 'application/json'" detail
					"mismatch in number of responses: got 1 actual, but expected 0",
				}
			} else if tt.expectedErrText != "" {
				currentExpectedErrTexts = []string{tt.expectedErrText}
			}

			// When
			err := ValidateResponses(tt.expectedFilePath, actual)

			// Then
			if currentExpectedErrCount == 0 {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				merr, ok := err.(*multierror.Error)
				require.True(t, ok, "Expected a multierror.Error")
				assert.Len(t, merr.Errors, currentExpectedErrCount)
				// Check specific error texts if provided for the adjusted expectation
				if len(currentExpectedErrTexts) > 0 {
					for _, expectedText := range currentExpectedErrTexts {
						found := false
						for _, e := range merr.Errors {
							if strings.Contains(e.Error(), expectedText) {
								found = true
								break
							}
						}
						assert.True(t, found, "Expected error text '%s' not found in %v", expectedText, merr.Errors)
					}
				} else if tt.expectedErrText != "" && currentExpectedErrCount == 1 { // Original single error check
					assert.ErrorContains(t, merr.Errors[0], tt.expectedErrText)
				}
			}
		})
	}
}
