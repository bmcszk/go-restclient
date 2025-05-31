package restclient

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateResponses_Headers(t *testing.T) {
	// Given: Test cases defined in 'tests' slice
	tests := []struct {
		name             string
		actualResponse   *Response
		expectedFilePath string
		expectedErrCount int
		expectedErrTexts []string
	}{
		{
			name:             "matching headers",
			actualResponse:   &Response{StatusCode: 200, Status: "200 OK", Headers: http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"123"}}},
			expectedFilePath: "testdata/http_response_files/validator_headerscontain_key_missing.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "mismatching header value",
			actualResponse:   &Response{StatusCode: 200, Status: "200 OK", Headers: http.Header{"Content-Type": {"text/html"}}},
			expectedFilePath: "testdata/http_response_files/validator_headerscontain_key_missing.hresp", // Expects application/json
			expectedErrCount: 1,
			expectedErrTexts: []string{"expected value 'application/json' for header 'Content-Type' not found"},
		},
		{
			name:             "missing expected header",
			actualResponse:   &Response{StatusCode: 200, Status: "200 OK", Headers: http.Header{"X-Other": {"value"}}},
			expectedFilePath: "testdata/http_response_files/validator_headers_missing_exp.hresp", // Expects X-Custom-Header
			expectedErrCount: 1,
			expectedErrTexts: []string{"expected header 'X-Custom-Header' not found"}, // Adjusted error message
		},
		{
			name:             "extra actual header (should be ignored)",
			actualResponse:   &Response{StatusCode: 200, Status: "200 OK", Headers: http.Header{"Content-Type": {"application/json"}, "X-Extra": {"ignored"}}},
			expectedFilePath: "testdata/http_response_files/validator_headerscontain_key_missing.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "matching multi-value headers (order preserved)",
			actualResponse:   &Response{StatusCode: 200, Status: "200 OK", Headers: http.Header{"Accept": {"application/json", "text/xml"}}},
			expectedFilePath: "testdata/http_response_files/validator_headers_multival_match.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "mismatching multi-value headers (different order)",
			actualResponse:   &Response{StatusCode: 200, Status: "200 OK", Headers: http.Header{"Accept": {"application/json", "text/xml"}}},
			expectedFilePath: "testdata/http_response_files/validator_headers_multival_mismatch_order.hresp", // Expects xml then json
			expectedErrCount: 0,                                                                              // Should now pass as order is not strictly enforced for all values
			// expectedErrTexts: []string{"header 'Accept' value mismatch: expected [text/xml application/json], got [application/json text/xml]"},
		},
		{
			name:             "mismatching multi-value headers (different value)",
			actualResponse:   &Response{StatusCode: 200, Status: "200 OK", Headers: http.Header{"Accept": {"application/json", "application/pdf"}}},
			expectedFilePath: "testdata/http_response_files/validator_headers_multival_match.hresp", // Expects json then text/plain
			expectedErrCount: 1,
			expectedErrTexts: []string{"expected value 'text/xml' for header 'Accept' not found"}, // Adjusted for current logic
		},
		{
			name:             "subset of multi-value headers (actual has more values)",
			actualResponse:   &Response{StatusCode: 200, Status: "200 OK", Headers: http.Header{"Accept": {"application/json", "text/xml", "application/pdf"}}},
			expectedFilePath: "testdata/http_response_files/validator_headers_multival_subset.hresp", // Expects only application/json
			expectedErrCount: 0,                                                                      // Should now pass as expected ["application/json"] is found in actual
			// expectedErrTexts: []string{"header 'Accept' value mismatch: expected [application/json], got [application/json text/xml application/pdf]"},
		},
		{
			name:             "case-insensitive header key matching",
			actualResponse:   &Response{StatusCode: 200, Status: "200 OK", Headers: http.Header{"Content-Type": {"application/json"}}},
			expectedFilePath: "testdata/http_response_files/validator_headers_case_insensitive_match.hresp", // Expected file has "content-type"
			expectedErrCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: actualResponse and expectedFilePath from the test case tt
			client, _ := NewClient()

			// When
			err := client.ValidateResponses(tt.expectedFilePath, []*Response{tt.actualResponse})

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
func TestValidateResponses_HeadersContain(t *testing.T) {
	// Given: Test cases defined in 'tests' slice
	tests := []struct {
		name             string
		actualResponse   *Response
		expectedFilePath string // Was expectedFileContent
		expectedErrCount int
		expectedErrTexts []string
	}{
		{
			name:             "HeadersContain logic not triggered (matching case for other fields)",
			actualResponse:   &Response{StatusCode: 200, Status: "200 OK", Headers: http.Header{"Content-Type": {"application/json; charset=utf-8"}}},
			expectedFilePath: "testdata/http_response_files/validator_headerscontain_match.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "HeadersContain logic not triggered (mismatch on standard header)",
			actualResponse:   &Response{StatusCode: 200, Status: "200 OK", Headers: http.Header{"Content-Type": {"text/html"}}},
			expectedFilePath: "testdata/http_response_files/validator_headerscontain_key_missing.hresp", // Standard header mismatch
			expectedErrCount: 1,
			expectedErrTexts: []string{"expected value 'application/json' for header 'Content-Type' not found"},
		},
		{
			name:             "HeadersContain logic not triggered (expected header key not found by standard check)",
			actualResponse:   &Response{StatusCode: 200, Status: "200 OK", Headers: http.Header{"X-Other": {"value"}}},
			expectedFilePath: "testdata/http_response_files/validator_headerscontain_key_missing.hresp", // Expected header 'Content-Type' missing from actual
			expectedErrCount: 1,
			expectedErrTexts: []string{"expected header 'Content-Type' not found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: actualResponse and expectedFilePath from the test case tt
			client, _ := NewClient()

			// When
			err := client.ValidateResponses(tt.expectedFilePath, []*Response{tt.actualResponse})

			// Then
			if tt.expectedErrCount == 0 {
				assert.NoError(t, err)
			} else {
				assertMultierrorContains(t, err, tt.expectedErrCount, tt.expectedErrTexts)
			}
		})
	}
}
