package restclient

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/PaesslerAG/jsonpath"
)

// ValidateResponse compares an actual Response against an ExpectedResponse.
func ValidateResponse(actual *Response, expected *ExpectedResponse) *ValidationResult {
	if actual == nil || expected == nil {
		return &ValidationResult{ // Or return an error / specific result type
			Passed:      false,
			Mismatches:  []string{"Actual or Expected response is nil"},
			RawActual:   actual,
			RawExpected: expected,
		}
	}

	result := &ValidationResult{
		Passed:      true, // Assume pass until a mismatch is found
		Mismatches:  []string{},
		RawActual:   actual,
		RawExpected: expected,
	}

	// 1. Validate Status Code
	if expected.StatusCode != nil {
		if actual.StatusCode != *expected.StatusCode {
			result.Passed = false
			result.Mismatches = append(result.Mismatches, fmt.Sprintf("StatusCode: expected %d, got %d", *expected.StatusCode, actual.StatusCode))
		}
	}

	// 2. Validate Status (e.g., "200 OK") - less common, but supported
	if expected.Status != nil {
		if actual.Status != *expected.Status {
			result.Passed = false
			result.Mismatches = append(result.Mismatches, fmt.Sprintf("Status: expected \"%s\", got \"%s\"", *expected.Status, actual.Status))
		}
	}

	// 3. Validate Headers (presence and exact match for specified headers)
	// This checks if all headers in expected.Headers are present in actual.Headers with the same values.
	// It does NOT check if actual.Headers contains extra headers not in expected.Headers.
	for key, expectedValues := range expected.Headers {
		actualValues, ok := actual.Headers[key]
		if !ok {
			result.Passed = false
			result.Mismatches = append(result.Mismatches, fmt.Sprintf("Header '%s': expected but not found", key))
			continue
		}
		// For simplicity, if a header is present, we check if all expected values are present.
		// More complex logic (e.g. exact match of value counts, order) could be added.
		if !reflect.DeepEqual(expectedValues, actualValues) { // This is strict: order and count must match
			// A softer check: ensure all expectedValues are within actualValues
			match := true
			for _, ev := range expectedValues {
				found := false
				for _, av := range actualValues {
					if ev == av {
						found = true
						break
					}
				}
				if !found {
					match = false
					break
				}
			}
			if !match {
				result.Passed = false
				result.Mismatches = append(result.Mismatches, fmt.Sprintf("Header '%s': expected values %v, got %v", key, expectedValues, actualValues))
			}
		}
	}

	// 4. Validate HeadersContain (check if actual headers contain specific key-value substrings)
	for expectedKey, expectedValueSubstring := range expected.HeadersContain {
		actualHeaderValues := actual.Headers.Values(expectedKey)
		if len(actualHeaderValues) == 0 {
			result.Passed = false
			result.Mismatches = append(result.Mismatches, fmt.Sprintf("HeadersContain: Expected header '%s' not found", expectedKey))
			continue
		}
		foundMatch := false
		for _, actualValue := range actualHeaderValues {
			if strings.Contains(actualValue, expectedValueSubstring) {
				foundMatch = true
				break
			}
		}
		if !foundMatch {
			result.Passed = false
			result.Mismatches = append(result.Mismatches, fmt.Sprintf("HeadersContain: Header '%s' did not contain substring '%s'. Values: %v", expectedKey, expectedValueSubstring, actualHeaderValues))
		}
	}

	// 5. Validate Body (exact match, if specified)
	if expected.Body != nil {
		if actual.BodyString != *expected.Body {
			result.Passed = false
			// For long bodies, showing a diff or snippet might be better than full bodies.
			result.Mismatches = append(result.Mismatches, fmt.Sprintf("Body: mismatch. Expected:\n%s\nGot:\n%s", *expected.Body, actual.BodyString))
		}
	}

	// 6. Validate BodyContains (substrings)
	for _, sub := range expected.BodyContains {
		if !strings.Contains(actual.BodyString, sub) {
			result.Passed = false
			result.Mismatches = append(result.Mismatches, fmt.Sprintf("BodyContains: expected substring not found: '%s'", sub))
		}
	}

	// 7. Validate BodyNotContains (substrings)
	for _, sub := range expected.BodyNotContains {
		if strings.Contains(actual.BodyString, sub) {
			result.Passed = false
			result.Mismatches = append(result.Mismatches, fmt.Sprintf("BodyNotContains: unexpected substring found: '%s'", sub))
		}
	}

	// 8. Validate JSONPathChecks
	if len(expected.JSONPathChecks) > 0 {
		var jsonData interface{}
		err := json.Unmarshal(actual.Body, &jsonData)
		if err != nil {
			result.Passed = false
			result.Mismatches = append(result.Mismatches, fmt.Sprintf("JSONPathChecks: failed to unmarshal actual body to JSON: %v", err))
		} else {
			for path, expectedValue := range expected.JSONPathChecks {
				actualValue, err := jsonpath.Get(path, jsonData)
				if err != nil {
					result.Passed = false
					result.Mismatches = append(result.Mismatches, fmt.Sprintf("JSONPathChecks: error evaluating path '%s': %v", path, err))
				} else {
					// DeepEqual might be too strict for numbers (e.g. int vs float64)
					// Convert expectedValue to type of actualValue for more robust comparison, or use a library for this.
					if !reflect.DeepEqual(actualValue, expectedValue) {
						// Attempt a more tolerant comparison for numeric types
						actualValStr := fmt.Sprintf("%v", actualValue)
						expectedValStr := fmt.Sprintf("%v", expectedValue)
						if actualValStr != expectedValStr { // Fallback to string comparison if types are tricky
							result.Passed = false
							result.Mismatches = append(result.Mismatches, fmt.Sprintf("JSONPathChecks: path '%s', expected '%v' (%T), got '%v' (%T)", path, expectedValue, expectedValue, actualValue, actualValue))
						}
					}
				}
			}
		}
	}

	return result
}

// Helper to load ExpectedResponse from a JSON file (example)
func LoadExpectedResponseFromJSONFile(filePath string) (*ExpectedResponse, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read expected response file %s: %w", filePath, err)
	}
	exp := &ExpectedResponse{}
	if err := json.Unmarshal(data, exp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal expected response file %s: %w", filePath, err)
	}
	return exp, nil
}

// TODO: Add LoadExpectedResponseFromYAMLFile
// TODO: Add LoadExpectedResponseFromHTTPFile (parsing a simplified .http format for expected responses)
