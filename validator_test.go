package restclient

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func ptr[T any](v T) *T { return &v }

func TestValidateResponse_NilInputs(t *testing.T) {
	result := ValidateResponse(nil, &ExpectedResponse{})
	assert.False(t, result.Passed)
	assert.Contains(t, result.Mismatches[0], "Actual or Expected response is nil")

	result = ValidateResponse(&Response{}, nil)
	assert.False(t, result.Passed)
	assert.Contains(t, result.Mismatches[0], "Actual or Expected response is nil")
}

func TestValidateResponse_StatusCode(t *testing.T) {
	actual := &Response{StatusCode: 200}
	expected := &ExpectedResponse{StatusCode: ptr(200)}
	result := ValidateResponse(actual, expected)
	assert.True(t, result.Passed)

	expectedFail := &ExpectedResponse{StatusCode: ptr(400)}
	resultFail := ValidateResponse(actual, expectedFail)
	assert.False(t, resultFail.Passed)
	assert.Contains(t, resultFail.Mismatches[0], "StatusCode: expected 400, got 200")
}

func TestValidateResponse_Status(t *testing.T) {
	actual := &Response{Status: "200 OK"}
	expected := &ExpectedResponse{Status: ptr("200 OK")}
	result := ValidateResponse(actual, expected)
	assert.True(t, result.Passed)

	expectedFail := &ExpectedResponse{Status: ptr("400 Bad Request")}
	resultFail := ValidateResponse(actual, expectedFail)
	assert.False(t, resultFail.Passed)
	assert.Contains(t, resultFail.Mismatches[0], "Status: expected \"400 Bad Request\", got \"200 OK\"")
}

func TestValidateResponse_Headers_ExactMatch(t *testing.T) {
	actual := &Response{
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
			"X-Custom":     []string{"value1", "value2"},
		},
	}
	expected := &ExpectedResponse{
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
		},
	}
	result := ValidateResponse(actual, expected)
	assert.True(t, result.Passed, "Mismatches: %v", result.Mismatches)

	// Case: Expected header not found
	expectedFailNotFound := &ExpectedResponse{
		Headers: http.Header{"X-Not-Found": []string{"any"}},
	}
	resultFailNotFound := ValidateResponse(actual, expectedFailNotFound)
	assert.False(t, resultFailNotFound.Passed)
	assert.Contains(t, resultFailNotFound.Mismatches[0], "Header 'X-Not-Found': expected but not found")

	// Case: Expected header value mismatch (strict with current softer check, so should pass if subset)
	expectedMismatchValues := &ExpectedResponse{
		Headers: http.Header{"Content-Type": []string{"text/xml"}},
	}
	resultMismatchValues := ValidateResponse(actual, expectedMismatchValues)
	assert.False(t, resultMismatchValues.Passed)
	assert.Contains(t, resultMismatchValues.Mismatches[0], "Header 'Content-Type': expected values [text/xml], got [application/json]")

	// Case: Multiple values, all expected are present
	expectedMulti := &ExpectedResponse{
		Headers: http.Header{"X-Custom": []string{"value1"}}, // Softer check: value1 is in actual [value1, value2]
	}
	resultMulti := ValidateResponse(actual, expectedMulti)
	assert.True(t, resultMulti.Passed, "Mismatches: %v", resultMulti.Mismatches)

	expectedMultiFail := &ExpectedResponse{
		Headers: http.Header{"X-Custom": []string{"value3"}},
	}
	resultMultiFail := ValidateResponse(actual, expectedMultiFail)
	assert.False(t, resultMultiFail.Passed)
	assert.Contains(t, resultMultiFail.Mismatches[0], "Header 'X-Custom': expected values [value3], got [value1 value2]")
}

func TestValidateResponse_HeadersContain(t *testing.T) {
	actual := &Response{
		Headers: http.Header{
			"Content-Disposition": []string{"attachment; filename=\"file.zip\""},
			"Set-Cookie":          []string{"session=123; path=/; HttpOnly", "pref=abc; path=/;"},
		},
	}
	expected := &ExpectedResponse{
		HeadersContain: map[string]string{
			"Content-Disposition": "filename=",
			"Set-Cookie":          "HttpOnly",
		},
	}
	result := ValidateResponse(actual, expected)
	assert.True(t, result.Passed, "Mismatches: %v", result.Mismatches)

	// Case: Header not found
	expectedFailNotFound := &ExpectedResponse{
		HeadersContain: map[string]string{"X-Non-Existent": "any"},
	}
	resultFailNotFound := ValidateResponse(actual, expectedFailNotFound)
	assert.False(t, resultFailNotFound.Passed)
	assert.Contains(t, resultFailNotFound.Mismatches[0], "HeadersContain: Expected header 'X-Non-Existent' not found")

	// Case: Substring not found in any value of the header
	expectedFailSubstring := &ExpectedResponse{
		HeadersContain: map[string]string{"Set-Cookie": "Secure"},
	}
	resultFailSubstring := ValidateResponse(actual, expectedFailSubstring)
	assert.False(t, resultFailSubstring.Passed)
	assert.Contains(t, resultFailSubstring.Mismatches[0], "HeadersContain: Header 'Set-Cookie' did not contain substring 'Secure'. Values: [session=123; path=/; HttpOnly pref=abc; path=/;]")
}

func TestValidateResponse_Body_ExactMatch(t *testing.T) {
	actual := &Response{BodyString: "Hello World"}
	expected := &ExpectedResponse{Body: ptr("Hello World")}
	result := ValidateResponse(actual, expected)
	assert.True(t, result.Passed)

	expectedFail := &ExpectedResponse{Body: ptr("Goodbye World")}
	resultFail := ValidateResponse(actual, expectedFail)
	assert.False(t, resultFail.Passed)
	assert.Contains(t, resultFail.Mismatches[0], "Body: mismatch.")
}

func TestValidateResponse_BodyContains(t *testing.T) {
	actual := &Response{BodyString: "This is a test response with important data."}
	expected := &ExpectedResponse{
		BodyContains: []string{"test response", "important data"},
	}
	result := ValidateResponse(actual, expected)
	assert.True(t, result.Passed, "Mismatches: %v", result.Mismatches)

	expectedFail := &ExpectedResponse{
		BodyContains: []string{"test response", "missing_text"},
	}
	resultFail := ValidateResponse(actual, expectedFail)
	assert.False(t, resultFail.Passed)
	assert.Contains(t, resultFail.Mismatches[0], "BodyContains: expected substring not found: 'missing_text'")
}

func TestValidateResponse_BodyNotContains(t *testing.T) {
	actual := &Response{BodyString: "Allowed content without error messages."}
	expected := &ExpectedResponse{
		BodyNotContains: []string{"ERROR", "Failed"},
	}
	result := ValidateResponse(actual, expected)
	assert.True(t, result.Passed, "Mismatches: %v", result.Mismatches)

	expectedFail := &ExpectedResponse{
		BodyNotContains: []string{"ERROR", "content"}, // "content" is present
	}
	resultFail := ValidateResponse(actual, expectedFail)
	assert.False(t, resultFail.Passed)
	assert.Contains(t, resultFail.Mismatches[0], "BodyNotContains: unexpected substring found: 'content'")
}

func TestValidateResponse_JSONPathChecks(t *testing.T) {
	jsonBody := `{"user": {"id": 123, "name": "Test User", "active": true}, "tags": ["a", "b"]}`
	actual := &Response{Body: []byte(jsonBody), BodyString: jsonBody}

	expected := &ExpectedResponse{
		JSONPathChecks: map[string]interface{}{
			"$.user.id":         123.0, // JSON numbers are float64 by default when unmarshaled to interface{}
			"$.user.name":       "Test User",
			"$.user.active":     true,
			"$.tags[?(@=='a')]": "a", // This path will return a slice if used with jsonpath.Get, so check might need adjustment
		},
	}
	// Adjusting the expectation for jsonpath.Get on array filter
	// jsonpath.Get with "$.tags[?(@=='a')]" will return a slice like ["a"].
	// For a simple value check, one might use "$.tags[0]" == "a".
	// Or, if we want to check for existence from a filter: an empty slice means not found.
	// For this test, let's assume the expected value is the first element of the filtered result if not empty.
	// Or more robustly, we want to check if the value 'a' is present in the 'tags' array.
	// A better JSONPath for existence could be `$.tags[?(@ == "a")]` and check if the result is non-empty.
	// However, the current structure is `path: expectedValue`. So, if path yields a slice, expectedValue should be a slice.

	// For now, this specific test `"$.tags[?(@=='a')]": "a"` will likely fail due to type mismatch (slice vs string)
	// or the Get function returning a slice. Let's simplify for direct value assertion.

	// Let's refine the JSONPath test for tags:
	expectedWithSimplerTagCheck := &ExpectedResponse{
		JSONPathChecks: map[string]interface{}{
			"$.user.id":     123.0,
			"$.user.name":   "Test User",
			"$.user.active": true,
			"$.tags[0]":     "a",
		},
	}

	result := ValidateResponse(actual, expectedWithSimplerTagCheck)
	assert.True(t, result.Passed, "Mismatches: %v", result.Mismatches)

	// Case: Path not found
	expectedFailPath := &ExpectedResponse{
		JSONPathChecks: map[string]interface{}{"$.user.nonexistent": "any"},
	}
	resultFailPath := ValidateResponse(actual, expectedFailPath)
	assert.False(t, resultFailPath.Passed)
	assert.Contains(t, resultFailPath.Mismatches[0], "JSONPathChecks: error evaluating path '$.user.nonexistent'")

	// Case: Value mismatch
	expectedFailValue := &ExpectedResponse{
		JSONPathChecks: map[string]interface{}{"$.user.id": 456.0},
	}
	resultFailValue := ValidateResponse(actual, expectedFailValue)
	assert.False(t, resultFailValue.Passed)
	assert.Contains(t, resultFailValue.Mismatches[0], "JSONPathChecks: path '$.user.id', expected '456' (float64), got '123' (float64)")

	// Case: Malformed JSON body in actual response
	actualMalformed := &Response{Body: []byte("{not_json"), BodyString: "{not_json"}
	resultMalformed := ValidateResponse(actualMalformed, expected)
	assert.False(t, resultMalformed.Passed)
	assert.Contains(t, resultMalformed.Mismatches[0], "JSONPathChecks: failed to unmarshal actual body to JSON")
}

// TODO: Test LoadExpectedResponseFromJSONFile
