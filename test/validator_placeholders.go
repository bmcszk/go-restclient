package test

import (
	"net/http"
	"testing"

	rc "github.com/bmcszk/go-restclient"
	"github.com/stretchr/testify/assert"
)

func RunValidateResponses_BodyRegexpPlaceholder(t *testing.T) {
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
			name:             "SCENARIO-LIB-022-001: simple regexp match",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}},
				BodyString: `{"id": "123"}`,
			},
			expectedFilePath: "testdata/http_response_files/validator_body_regexp_simple_match.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "SCENARIO-LIB-022-002: simple regexp no match",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}},
				BodyString: `{"status": "FAILED"}`,
			},
			expectedFilePath: "testdata/http_response_files/validator_body_regexp_simple_no_match.hresp",
			expectedErrCount: 1,
			expectedErrTexts: []string{
				"body mismatch (regexp/placeholder evaluation failed)",
				"Compiled Regex: ^\\{\"status\": \"(SUCCESS)\"\\}$",
			},
		},
		{
			name:             "SCENARIO-LIB-022-003: multiple regexp match",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}},
				BodyString: `{"userId": "U-abc", "transactionId": "T-12345"}`,
			},
			expectedFilePath: "testdata/http_response_files/validator_body_regexp_multiple_match.hresp",
			expectedErrCount: 0,
		},
		// {
		// 	name:             "SCENARIO-LIB-022-004: regexp with special characters in pattern",
		// 	actualResponse:   &rc.Response{StatusCode: 200, Status: "200 OK", BodyString: `Value: 123.test`},
		// 	expectedFilePath: "testdata/http_response_files/validator_body_regexp_special_chars.hresp",
		// 	expectedErrCount: 0,
		// },
		{
			name:             "SCENARIO-LIB-022-005: invalid regexp pattern",
			actualResponse:   &rc.Response{StatusCode: 200, Status: "200 OK", BodyString: `{"data": "value"}`},
			expectedFilePath: "testdata/http_response_files/validator_body_regexp_invalid_pattern.hresp",
			expectedErrCount: 1,
			expectedErrTexts: []string{"failed to compile master regex from expected body", "error parsing regexp"},
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

func RunValidateResponses_BodyAnyGuidPlaceholder(t *testing.T) {
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
			name:             "SCENARIO-LIB-023-001: valid GUID match",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}},
				BodyString: `{"correlationId": "123e4567-e89b-12d3-a456-426614174000"}`,
			},
			expectedFilePath: "testdata/http_response_files/validator_body_anyguid_match.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "SCENARIO-LIB-023-002: not a GUID",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}},
				BodyString: `{"id": "not-a-guid"}`,
			},
			expectedFilePath: "testdata/http_response_files/validator_body_anyguid_no_match.hresp",
			expectedErrCount: 1,
			expectedErrTexts: []string{"body mismatch (regexp/placeholder evaluation failed)"},
		},
		{
			name:             "SCENARIO-LIB-023-003: GUID in larger text",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				BodyString: `Session started with ID: 123e4567-e89b-12d3-a456-426614174000. ` +
				`Please use this for subsequent requests.`,
			},
			expectedFilePath: "testdata/http_response_files/validator_body_anyguid_larger_text.hresp",
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

func RunValidateResponses_BodyAnyTimestampPlaceholder(t *testing.T) {
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
			name:             "SCENARIO-LIB-024-001: valid integer timestamp match",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}},
				BodyString: `{"createdAt": "1678886400"}`,
			},
			expectedFilePath: "testdata/http_response_files/validator_body_anytimestamp_match.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "SCENARIO-LIB-024-002: not an integer timestamp (string)",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}},
				BodyString: `{"timestamp": "not-a-timestamp"}`,
			},
			expectedFilePath: "testdata/http_response_files/validator_body_anytimestamp_no_match.hresp",
			expectedErrCount: 1,
			expectedErrTexts: []string{"body mismatch (regexp/placeholder evaluation failed)"},
		},
		{
			name:             "SCENARIO-LIB-024-003: not an integer timestamp (float)",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}},
				BodyString: `{"eventTime": "1678886400.5"}`,
			},
			expectedFilePath: "testdata/http_response_files/validator_body_anytimestamp_float.hresp",
			expectedErrCount: 1,
			expectedErrTexts: []string{"body mismatch (regexp/placeholder evaluation failed)"},
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

func RunValidateResponses_BodyAnyDatetimePlaceholder(t *testing.T) {
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
			name:             "SCENARIO-LIB-025-001: rfc1123 match",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}},
				BodyString: `{"lastModified": "Tue, 15 Mar 2023 12:00:00 GMT"}`,
			},
			expectedFilePath: "testdata/http_response_files/validator_body_anydatetime_rfc1123_match.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "SCENARIO-LIB-025-002: iso8601 match (RFC3339)",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}},
				BodyString: `{"eventTime": "2023-03-15T12:00:00Z"}`,
			},
			expectedFilePath: "testdata/http_response_files/validator_body_anydatetime_iso8601_match.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "SCENARIO-LIB-025-002b: iso8601 match with offset",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}},
				BodyString: `{"eventTime": "2023-03-15T12:00:00+01:00"}`,
			},
			expectedFilePath: "testdata/http_response_files/validator_body_anydatetime_iso8601_match.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "SCENARIO-LIB-025-002c: iso8601 match with milliseconds and Z",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}},
				BodyString: `{"eventTime": "2023-03-15T12:00:00.123Z"}`,
			},
			expectedFilePath: "testdata/http_response_files/validator_body_anydatetime_iso8601_match.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "SCENARIO-LIB-025-003: custom Go layout match",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}},
				BodyString: `{"date": "2023-03-15"}`,
			},
			expectedFilePath: "testdata/http_response_files/validator_body_anydatetime_custom_match.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "SCENARIO-LIB-025-004: format mismatch (rfc1123 expected, actual is YYYY-MM-DD)",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}},
				BodyString: `{"timestamp": "2023-03-15"}`,
			},
			expectedFilePath: "testdata/http_response_files/validator_body_anydatetime_format_mismatch.hresp",
			expectedErrCount: 1,
			expectedErrTexts: []string{"body mismatch (regexp/placeholder evaluation failed)"},
		},
		{
			name:             "SCENARIO-LIB-025-005: invalid format keyword",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}},
				BodyString: `{"time": "12:34:56"}`,
			},
			expectedFilePath: "testdata/http_response_files/validator_body_anydatetime_invalid_keyword.hresp",
			expectedErrCount: 1,
			expectedErrTexts: []string{"body mismatch (regexp/placeholder evaluation failed)", "(\\z.\\A)"},
		},
		{
			name:             "SCENARIO-LIB-025-006: missing format argument",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}},
				BodyString: `{"time": "12:34:56"}`,
			},
			expectedFilePath: "testdata/http_response_files/validator_body_anydatetime_missing_format.hresp",
			expectedErrCount: 1,
			expectedErrTexts: []string{"body mismatch (regexp/placeholder evaluation failed)", "(\\z.\\A)"},
		},
		{
			name:             "custom format empty string literal \"\" - should fail",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}},
				BodyString: `{"date": ""}`,
			},
			expectedFilePath: "testdata/http_response_files/validator_body_anydatetime_custom_empty_format.hresp",
			expectedErrCount: 1,
			expectedErrTexts: []string{"body mismatch (regexp/placeholder evaluation failed)", "(\\z.\\A)"},
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

func RunValidateResponses_BodyAnyPlaceholder(t *testing.T) {
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
			name:             "SCENARIO-LIB-026-001: $any matching a simple string",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}},
				BodyString: `{"key": "some value"}`,
			},
			expectedFilePath: "testdata/http_response_files/validator_body_any_simple_match.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "SCENARIO-LIB-026-002: $any matching special characters and spaces",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"text/plain"}},
				BodyString: `Value: !@#$%^&*()_+{}[];':\",./<>?         end`,
			},
			expectedFilePath: "testdata/http_response_files/validator_body_any_special_chars_match.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "SCENARIO-LIB-026-003: $any matching an empty string segment",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}},
				BodyString: `{"prefix": "", "data": "", "suffix": ""}`,
			},
			expectedFilePath: "testdata/http_response_files/validator_body_any_empty_segment_match.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "SCENARIO-LIB-026-004: $any matching a multi-line string",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"text/plain"}},
				BodyString: "Start:\nThis is line 1.\nThis is line 2.\nAnd line 3.\nEnd.",
			},
			expectedFilePath: "testdata/http_response_files/validator_body_any_multiline_match.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "SCENARIO-LIB-026-005: multiple $any placeholders",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}},
				BodyString: `{"field1": "value1", "field2": "constant", "field3": "value3"}`,
			},
			expectedFilePath: "testdata/http_response_files/validator_body_any_multiple_placeholders_match.hresp",
			expectedErrCount: 0,
		},
		{
			name:             "$any fails if preceding literal doesn't match",
			actualResponse: &rc.Response{
				StatusCode: 200, Status: "200 OK",
				Headers: http.Header{"Content-Type": {"application/json"}},
				BodyString: `{"wrong_key": "some value"}`,
			},
			// Expects {"key": ...}
			expectedFilePath: "testdata/http_response_files/validator_body_any_simple_match.hresp",
			expectedErrCount: 1,
			expectedErrTexts: []string{"body mismatch"},
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
