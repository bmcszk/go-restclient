package test

import (
	"fmt"
	"testing"

	rc "github.com/bmcszk/go-restclient"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func RunValidateResponses_StatusString(t *testing.T) {
	t.Helper()
	// Given: Test cases defined in 'tests' slice
	tests := []statusStringTestCase{
		{
			name:             "matching status string",
			actualResponse:   &rc.Response{StatusCode: 200, Status: "200 OK"},
			expectedFilePath: "test/data/http_response_files/validator_body_exact_no_body_exp.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "mismatching status string",
			actualResponse:   &rc.Response{StatusCode: 200, Status: "200 Something Else"},
			expectedFilePath: "test/data/http_response_files/validator_body_exact_no_body_exp.hresp",
			expectedErrCount: 1,
			expectedErrText:  "status string mismatch: expected '200 OK', got '200 Something Else'",
		},
		{
			name:             "actual status string is correct, expected file has only status code",
			actualResponse:   &rc.Response{StatusCode: 200, Status: "200 OK"},
			expectedFilePath: "test/data/http_response_files/validator_partial_status_code_mismatch.hresp",
			expectedErrCount: 1,
			expectedErrText:  "status string mismatch: expected '200', got '200 OK'",
		},
		{
			name:             "mismatching status code, status strings also mismatch",
			actualResponse:   &rc.Response{StatusCode: 404, Status: "404 Not Found"},
			expectedFilePath: "test/data/http_response_files/validator_body_exact_no_body_exp.hresp",
			expectedErrCount: 2,
			expectedErrTexts: []string{
				"status code mismatch: expected 200, got 404",
				"status string mismatch: expected '200 OK', got '404 Not Found'",
			},
		},
		{
			name:             "matching status code, expected file only code, actual also only code in status",
			actualResponse:   &rc.Response{StatusCode: 200, Status: "200"},
			expectedFilePath: "test/data/http_response_files/validator_partial_status_code_mismatch.hresp",
			expectedErrCount: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runStatusStringSubtest(t, tt)
		})
	}
}

func runStatusStringSubtest(t *testing.T, tt statusStringTestCase) {
	t.Helper()
	// Given: actualResponse and expectedFilePath from the test case tt
	client, _ := rc.NewClient()

	// When
	err := client.ValidateResponses(tt.expectedFilePath, tt.actualResponse)

	// Then
	validateStatusStringResult(t, tt, err)
}

type statusStringTestCase struct {
	name             string
	actualResponse   *rc.Response
	expectedFilePath string
	expectedErrCount int
	expectedErrText  string
	expectedErrTexts []string
}

func validateStatusStringResult(t *testing.T, tt statusStringTestCase, err error) {
	t.Helper()
	if tt.expectedErrCount == 0 {
		assert.NoError(t, err)
		return
	}

	if shouldUseSingleErrorAssertion(tt) {
		validateSingleError(t, err, tt.expectedErrText)
	} else {
		assertMultierrorContains(t, err, tt.expectedErrCount, tt.expectedErrTexts)
	}
}

func shouldUseSingleErrorAssertion(tt statusStringTestCase) bool {
	return tt.expectedErrText != "" && tt.expectedErrCount == 1 && len(tt.expectedErrTexts) == 0
}

func validateSingleError(t *testing.T, err error, expectedErrText string) {
	t.Helper()
	require.Error(t, err)
	merr, ok := err.(*multierror.Error)
	require.True(t, ok, "Expected a multierror.Error for single error case with expectedErrText")
	require.Len(t, merr.Errors, 1)
	assert.Contains(t, merr.Errors[0].Error(), expectedErrText)
}

type statusCodeTestCase struct {
	name               string
	actualResponseCode *int
	expectedFilePath   string
	expectedErrCount   int
	expectedErrText    string
}

func RunValidateResponses_StatusCode(t *testing.T) {
	t.Helper()
	// Given: Test cases defined in 'tests' slice
	tests := []statusCodeTestCase{
		{
			name:               "matching status code only",
			actualResponseCode: intPtr(200),
			expectedFilePath:   "test/data/http_response_files/validator_partial_status_code_mismatch.hresp",
			expectedErrCount:   0,
		},
		{
			name:               "mismatching status code only",
			actualResponseCode: intPtr(404),
			// Expect 200
			expectedFilePath: "test/data/http_response_files/validator_partial_status_code_mismatch.hresp",
			// Expects 2 errors: status code and status string mismatch
			expectedErrCount: 2,
			// Clear this as we use currentExpectedErrTexts now
			expectedErrText: "",
			// currentExpectedErrTexts will be:
			// ["status code mismatch: expected 200, got 404", "status string mismatch: expected '200', got '404'"]
		},
		{
			name:               "nil actual status code (should not happen with real http.Response)",
			// Actual code is 0, so if file expects 200, it's a mismatch
			actualResponseCode: nil,
			expectedFilePath: "test/data/http_response_files/validator_partial_status_code_mismatch.hresp",
			// Expects 2 errors: status code and status string mismatch
			expectedErrCount: 2,
			// Clear this
			expectedErrText: "",
			// currentExpectedErrTexts will be:
			// ["status code mismatch: expected 200, got 0", "status string mismatch: expected '200', got '0'"]
		},
		{
			name:               "nil expected status code (file has no status line)",
			actualResponseCode: intPtr(200),
			expectedFilePath:   "test/data/http_response_files/validator_status_code_nil_expected.hresp",
			expectedErrCount:   2, // This will fail parsing + count mismatch
			// currentExpectedErrTexts will be:
			// ["failed to parse expected response file", "mismatch in number of responses"]
		},
		{
			// This test case specifically tests when the expected file ONLY has a status code
			// (no text reason phrase) and the actual response also only provides a status code
			// (no text reason phrase in its .Status field).
			name:               "matching status code, actual and expected only have code",
			actualResponseCode: intPtr(200),
			expectedFilePath:   "test/data/http_response_files/validator_status_code_only.hresp",
			expectedErrCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runStatusCodeSubtest(t, tt)
		})
	}
}

func runStatusCodeSubtest(t *testing.T, tt statusCodeTestCase) {
	t.Helper()
	actual := buildResponseFromTestCase(tt)
	client, _ := rc.NewClient()
	currentExpectedErrCount, currentExpectedErrTexts := getExpectedErrorsForTestCase(tt)

	// When
	err := client.ValidateResponses(tt.expectedFilePath, actual)

	// Then
	validateStatusCodeResult(t, currentExpectedErrCount, currentExpectedErrTexts, err)
}

func buildResponseFromTestCase(tt statusCodeTestCase) *rc.Response {
	actual := &rc.Response{}
	if tt.actualResponseCode != nil {
		actual.StatusCode = *tt.actualResponseCode
		actual.Status = fmt.Sprintf("%d", actual.StatusCode)
	} else {
		actual.Status = "0"
		actual.StatusCode = 0
	}
	return actual
}

func getExpectedErrorsForTestCase(tt statusCodeTestCase) (int, []string) {
	currentExpectedErrCount := tt.expectedErrCount
	var currentExpectedErrTexts []string

	switch tt.name {
	case "mismatching status code":
		currentExpectedErrTexts = []string{
			"status code mismatch: expected 200, got 404",
			"status string mismatch: expected '200', got '404'",
		}
	case "nil actual status code (should not happen with real http.Response)":
		currentExpectedErrTexts = []string{
			"status code mismatch: expected 200, got 0",
			"status string mismatch: expected '200', got '0'",
		}
	case "nil expected status code (file has no status line)":
		currentExpectedErrTexts = []string{
			"failed to parse expected response file",
			"mismatch in number of responses: got 1 actual, but expected 0",
		}
	default:
		if tt.expectedErrText != "" {
			currentExpectedErrTexts = []string{tt.expectedErrText}
		}
	}
	return currentExpectedErrCount, currentExpectedErrTexts
}

func validateStatusCodeResult(t *testing.T, expectedErrCount int, expectedErrTexts []string, err error) {
	t.Helper()
	if expectedErrCount == 0 {
		assert.NoError(t, err)
	} else {
		assertMultierrorContains(t, err, expectedErrCount, expectedErrTexts)
	}
}
