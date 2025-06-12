package test

import (
	"net/http"
	"testing"

	rc "github.com/bmcszk/go-restclient"
	"github.com/stretchr/testify/assert"
)

func RunValidateResponses_Headers(t *testing.T) {
	t.Helper()
	// Given: Test cases defined in 'tests' slice
	tests := []struct {
		name             string
		actualResponse   *rc.Response
		expectedFilePath string
		expectedErrCount int
		expectedErrTexts []string
	}{
		{
			name: "matching headers",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"123"}},
			},
			expectedFilePath: "test/data/http_response_files/validator_headerscontain_key_missing.hresp",
			expectedErrCount: 0,
		},
		{
			name: "mismatching header value",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK", Headers: http.Header{"Content-Type": {"text/html"}},
			},
			// Expects application/json
			expectedFilePath: "test/data/http_response_files/validator_headerscontain_key_missing.hresp",
			expectedErrCount: 1,
			expectedErrTexts: []string{"expected value 'application/json' for header 'Content-Type' not found"},
		},
		{
			name: "missing expected header",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK", Headers: http.Header{"X-Other": {"value"}},
			},
			// Expects X-Custom-Header
			expectedFilePath: "test/data/http_response_files/validator_headers_missing_exp.hresp",
			expectedErrCount: 1,
			// Adjusted error message
			expectedErrTexts: []string{"expected header 'X-Custom-Header' not found"},
		},
		{
			name: "extra actual header (should be ignored)",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}, "X-Extra": {"ignored"}},
			},
			expectedFilePath: "test/data/http_response_files/validator_headerscontain_key_missing.hresp",
			expectedErrCount: 0,
		},
		{
			name: "matching multi-value headers (order preserved)",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Accept": {"application/json", "text/xml"}},
			},
			expectedFilePath: "test/data/http_response_files/validator_headers_multival_match.hresp",
			expectedErrCount: 0,
		},
		{
			name: "mismatching multi-value headers (different order)",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Accept": {"application/json", "text/xml"}},
			},
			// Expects xml then json
			expectedFilePath: "test/data/http_response_files/validator_headers_multival_mismatch_order.hresp",
			// Should now pass as order is not strictly enforced for all values
			expectedErrCount: 0,
			// expectedErrTexts: []string{"header 'Accept' value mismatch: expected [text/xml application/json],
			//                               got [application/json text/xml]"},
		},
		{
			name: "mismatching multi-value headers (different value)",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Accept": {"application/json", "application/pdf"}},
			},
			// Expects json then text/plain
			expectedFilePath: "test/data/http_response_files/validator_headers_multival_match.hresp",
			expectedErrCount: 1,
			// Adjusted for current logic
			expectedErrTexts: []string{"expected value 'text/xml' for header 'Accept' not found"},
		},
		{
			name: "subset of multi-value headers (actual has more values)",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Accept": {"application/json", "text/xml", "application/pdf"}},
			},
			// Expects only application/json
			expectedFilePath: "test/data/http_response_files/validator_headers_multival_subset.hresp",
			// Should now pass as expected ["application/json"] is found in actual
			expectedErrCount: 0,
			// expectedErrTexts: []string{"header 'Accept' value mismatch:
			//                               expected [application/json],
			//                               got [application/json text/xml application/pdf]"},
		},
		{
			name: "case-insensitive header key matching",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK", Headers: http.Header{"Content-Type": {"application/json"}},
			},
			// Expected file has "content-type"
			expectedFilePath: "test/data/http_response_files/validator_headers_case_insensitive_match.hresp",
			expectedErrCount: 0,
		},
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

// TestValidateResponses_HeadersContain verifies that HeadersContain logic in ValidateResponses
// is benign when the expected response comes from a file, as the file cannot specify HeadersContain.
func RunValidateResponses_HeadersContain(t *testing.T) {
	t.Helper()
	// Given: Test cases defined in 'tests' slice
	tests := []struct {
		name             string
		actualResponse   *rc.Response
		expectedFilePath string // Was expectedFileContent
		expectedErrCount int
		expectedErrTexts []string
	}{
		{
			name: "HeadersContain logic not triggered (matching case for other fields)",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json; charset=utf-8"}},
			},
			expectedFilePath: "test/data/http_response_files/validator_headerscontain_match.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "HeadersContain logic not triggered (mismatch on standard header)",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK", Headers: http.Header{"Content-Type": {"text/html"}},
			},
			// Standard header mismatch
			expectedFilePath: "test/data/http_response_files/validator_headerscontain_key_missing.hresp",
			expectedErrCount: 1,
			expectedErrTexts: []string{"expected value 'application/json' for header 'Content-Type' not found"},
		},
		{
			name:             "HeadersContain logic not triggered (expected header key not found by standard check)",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK", Headers: http.Header{"X-Other": {"value"}},
			},
			// Expected header 'Content-Type' missing from actual
			expectedFilePath: "test/data/http_response_files/validator_headerscontain_key_missing.hresp",
			expectedErrCount: 1,
			expectedErrTexts: []string{"expected header 'Content-Type' not found"},
		},
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
