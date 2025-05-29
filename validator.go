package restclient

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pmezard/go-difflib/difflib"
)

// ExpectedResponse is defined in response.go

var ( //nolint:gochecknoglobals
	regexpPlaceholderFinder       = regexp.MustCompile(`\{\{\$regexp\s+((?s).*?)\}\}`)
	anyGuidPlaceholderFinder      = regexp.MustCompile(`\{\{\$anyGuid\}\}`)
	anyTimestampPlaceholderFinder = regexp.MustCompile(`\{\{\$anyTimestamp\}\}`)
	anyDatetimePlaceholderFinder  = regexp.MustCompile(`\{\{\$anyDatetime\s+(.*?)\}\}`) // Captures format arg
	anyDatetimeNoArgFinder        = regexp.MustCompile(`\{\{\$anyDatetime\}\}`)         // For {{$anyDatetime}} without args
	anyPlaceholderFinder          = regexp.MustCompile(`\{\{\$any\}\}`)
)

const guidRegexPattern = `[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}` //nolint:gochecknoglobals
const timestampRegexPattern = `\d+`                                                                    //nolint:gochecknoglobals
// Regex for RFC1123: e.g., Mon, 02 Jan 2006 15:04:05 MST
const rfc1123RegexPattern = `[A-Za-z]{3},\s\d{2}\s[A-Za-z]{3}\s\d{4}\s\d{2}:\d{2}:\d{2}\s[A-Z]{3}` //nolint:gochecknoglobals
// Regex for a common ISO8601/RFC3339 form: e.g., 2006-01-02T15:04:05Z or 2006-01-02T15:04:05+07:00
const iso8601RegexPattern = `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|([+-]\d{2}:\d{2}))` // Added optional milliseconds //nolint:gochecknoglobals
const genericDatetimeRegexPattern = `[\w\d\s.:\-,+/TZ()]+`                                     //nolint:gochecknoglobals
const nonMatchingRegexPattern = `\z.\A`                                                        // Valid but never matches                                         //nolint:gochecknoglobals
const anyRegexPattern = `(?s).*?`                                                              // Matches any char (incl newline), non-greedy, no outer group //nolint:gochecknoglobals

// ValidateResponses compares a slice of actual HTTP responses against a set of expected responses
// parsed from the specified file. It returns a consolidated error (multierror) if any
// discrepancies are found, or nil if all validations pass.
func ValidateResponses(responseFilePath string, actualResponses ...*Response) error {
	var errs *multierror.Error

	// Attempt to parse the expected responses from the file.
	expectedResponses, parseErr := parseExpectedResponseFile(responseFilePath)
	if parseErr != nil {
		errs = multierror.Append(errs, fmt.Errorf("failed to parse expected response file '%s': %w", responseFilePath, parseErr))
	}

	// Determine effective counts for actual and expected responses.
	effectiveNumActual := countNonNilActuals(actualResponses)
	effectiveNumExpected := 0
	if expectedResponses != nil {
		effectiveNumExpected = len(expectedResponses)
	}

	// Check for count mismatch.
	if effectiveNumActual != effectiveNumExpected {
		errs = multierror.Append(errs, fmt.Errorf("mismatch in number of responses: got %d actual, but expected %d from file '%s'", effectiveNumActual, effectiveNumExpected, responseFilePath))
	}

	// If parsing failed, we cannot proceed with per-response validation. Return collected errors so far.
	if parseErr != nil {
		return errs.ErrorOrNil()
	}

	// If there was no parse error, but counts mismatched, return the count mismatch error.
	if effectiveNumActual != effectiveNumExpected { // This implies parseErr was nil to reach here
		return errs.ErrorOrNil()
	}

	// If we reach here, parsing succeeded and counts match. Proceed to validate pairs.
	for i := 0; i < effectiveNumActual; i++ {
		actual := actualResponses[i]
		expected := expectedResponses[i]

		if actual == nil {
			errs = multierror.Append(errs, fmt.Errorf("validation for response #%d ('%s'): actual response is nil", i+1, responseFilePath))
			continue
		}

		// Validate Status Code
		if expected.StatusCode != nil && (actual.StatusCode != *expected.StatusCode) {
			errs = multierror.Append(errs, fmt.Errorf("validation for response #%d ('%s'): status code mismatch: expected %d, got %d", i+1, responseFilePath, *expected.StatusCode, actual.StatusCode))
		}

		// Validate Status String
		if expected.Status != nil && *expected.Status != "" && (actual.Status != *expected.Status) {
			errs = multierror.Append(errs, fmt.Errorf("validation for response #%d ('%s'): status string mismatch: expected '%s', got '%s'", i+1, responseFilePath, *expected.Status, actual.Status))
		}

		// Validate Headers (Exact Match for specified keys)
		if expected.Headers != nil {
			for key, expectedValues := range expected.Headers {
				actualValues, ok := actual.Headers[key]
				if !ok {
					errs = multierror.Append(errs, fmt.Errorf("validation for response #%d ('%s'): expected header '%s' not found", i+1, responseFilePath, key))
					continue
				}
				for _, ev := range expectedValues {
					found := false
					for _, av := range actualValues {
						if av == ev {
							found = true
							break
						}
					}
					if !found {
						errs = multierror.Append(errs, fmt.Errorf("validation for response #%d ('%s'): expected value '%s' for header '%s' not found in actual values %v", i+1, responseFilePath, ev, key, actualValues))
					}
				}
			}
		}

		// Validate Body
		if expected.Body != nil {
			bodyErr := compareBodies(responseFilePath, i+1, *expected.Body, actual.BodyString)
			if bodyErr != nil {
				errs = multierror.Append(errs, bodyErr)
			}
		}
	}

	return errs.ErrorOrNil()
}

// compareBodies compares the expected body string with the actual body string,
// supporting placeholders like {{$regexp pattern}}, {{$anyGuid}}, {{$anyTimestamp}}, and {{$anyDatetime format}}.
func compareBodies(responseFilePath string, responseIndex int, expectedBody, actualBody string) error {
	normalizedExpectedBody := strings.TrimSpace(strings.ReplaceAll(expectedBody, "\\r\\n", "\\n"))
	normalizedActualBody := strings.TrimSpace(strings.ReplaceAll(actualBody, "\\r\\n", "\\n"))

	// Check if any placeholders are present. If not, do a simple string comparison.
	hasRegexpPlaceholder := strings.Contains(normalizedExpectedBody, "{{$regexp")
	hasAnyGuidPlaceholder := strings.Contains(normalizedExpectedBody, "{{$anyGuid}}")
	hasAnyTimestampPlaceholder := strings.Contains(normalizedExpectedBody, "{{$anyTimestamp}}")
	hasAnyDatetimePlaceholder := strings.Contains(normalizedExpectedBody, "{{$anyDatetime") // Checks for start of {{$anyDatetime...}}
	hasAnyPlaceholder := strings.Contains(normalizedExpectedBody, "{{$any}}")

	if !hasRegexpPlaceholder && !hasAnyGuidPlaceholder && !hasAnyTimestampPlaceholder && !hasAnyDatetimePlaceholder && !hasAnyPlaceholder {
		if normalizedActualBody != normalizedExpectedBody {
			diff := difflib.UnifiedDiff{
				A:        difflib.SplitLines(normalizedExpectedBody),
				B:        difflib.SplitLines(normalizedActualBody),
				FromFile: "Expected Body",
				ToFile:   "Actual Body",
				Context:  3,
			}
			diffText, _ := difflib.GetUnifiedDiffString(diff)
			return fmt.Errorf("validation for response #%d ('%s'): body mismatch:\\n%s", responseIndex, responseFilePath, diffText)
		}
		return nil
	}

	// Placeholder-based comparison
	var finalRegexPattern strings.Builder
	finalRegexPattern.WriteString("^")

	remainingExpectedBody := normalizedExpectedBody

	for len(remainingExpectedBody) > 0 {
		// Find the earliest occurrence of any known placeholder
		regexpMatchIndices := regexpPlaceholderFinder.FindStringSubmatchIndex(remainingExpectedBody)
		anyGuidMatchIndices := anyGuidPlaceholderFinder.FindStringSubmatchIndex(remainingExpectedBody)
		anyTimestampMatchIndices := anyTimestampPlaceholderFinder.FindStringSubmatchIndex(remainingExpectedBody)
		anyDatetimeMatchIndices := anyDatetimePlaceholderFinder.FindStringSubmatchIndex(remainingExpectedBody) // With arg
		anyDatetimeNoArgMatchIndices := anyDatetimeNoArgFinder.FindStringSubmatchIndex(remainingExpectedBody)  // No arg
		anyMatchIndices := anyPlaceholderFinder.FindStringSubmatchIndex(remainingExpectedBody)

		// Determine which placeholder is next (if any)
		var earliestMatchIndices []int
		var placeholderType string
		var placeholderArg string // Initialize here, to be filled IF the chosen type has an arg

		currentMatchPos := len(remainingExpectedBody) + 1

		if regexpMatchIndices != nil && regexpMatchIndices[0] < currentMatchPos {
			currentMatchPos = regexpMatchIndices[0]
			earliestMatchIndices = regexpMatchIndices
			placeholderType = "regexp"
		}
		if anyGuidMatchIndices != nil && anyGuidMatchIndices[0] < currentMatchPos {
			currentMatchPos = anyGuidMatchIndices[0]
			earliestMatchIndices = anyGuidMatchIndices
			placeholderType = "anyGuid"
		}
		if anyTimestampMatchIndices != nil && anyTimestampMatchIndices[0] < currentMatchPos {
			currentMatchPos = anyTimestampMatchIndices[0]
			earliestMatchIndices = anyTimestampMatchIndices
			placeholderType = "anyTimestamp"
		}
		if anyDatetimeMatchIndices != nil && anyDatetimeMatchIndices[0] < currentMatchPos { // With arg
			currentMatchPos = anyDatetimeMatchIndices[0]
			earliestMatchIndices = anyDatetimeMatchIndices
			placeholderType = "anyDatetimeWithArg"
		}
		if anyDatetimeNoArgMatchIndices != nil && anyDatetimeNoArgMatchIndices[0] < currentMatchPos { // No arg
			earliestMatchIndices = anyDatetimeNoArgMatchIndices
			placeholderType = "anyDatetimeNoArg"
		}
		if anyMatchIndices != nil && anyMatchIndices[0] < currentMatchPos {
			earliestMatchIndices = anyMatchIndices
			placeholderType = "any"
		}

		if earliestMatchIndices == nil { // No more placeholders found
			finalRegexPattern.WriteString(regexp.QuoteMeta(remainingExpectedBody))
			break
		}

		// Append literal part before the current placeholder
		literalPart := remainingExpectedBody[:earliestMatchIndices[0]]
		finalRegexPattern.WriteString(regexp.QuoteMeta(literalPart))

		// Append regex for the current placeholder
		switch placeholderType {
		case "regexp":
			placeholderArg = remainingExpectedBody[earliestMatchIndices[2]:earliestMatchIndices[3]]
			userPattern := placeholderArg
			if len(userPattern) >= 2 && userPattern[0] == '`' && userPattern[len(userPattern)-1] == '`' {
				userPattern = userPattern[1 : len(userPattern)-1]
			}
			finalRegexPattern.WriteString("(")
			finalRegexPattern.WriteString(userPattern)
			finalRegexPattern.WriteString(")")
		case "anyGuid":
			finalRegexPattern.WriteString("(")
			finalRegexPattern.WriteString(guidRegexPattern)
			finalRegexPattern.WriteString(")")
		case "anyTimestamp":
			finalRegexPattern.WriteString("(")
			finalRegexPattern.WriteString(timestampRegexPattern)
			finalRegexPattern.WriteString(")")
		case "anyDatetimeWithArg":
			placeholderArg = remainingExpectedBody[earliestMatchIndices[2]:earliestMatchIndices[3]]
			formatArg := strings.TrimSpace(placeholderArg)
			selectedPattern := nonMatchingRegexPattern // Default

			if formatArg == "rfc1123" {
				selectedPattern = rfc1123RegexPattern
			} else if formatArg == "iso8601" {
				selectedPattern = iso8601RegexPattern
			} else if len(formatArg) >= 2 && formatArg[0] == '"' && formatArg[len(formatArg)-1] == '"' {
				customLayout := formatArg[1 : len(formatArg)-1]
				if customLayout != "" { // e.g. "2006-01-02"
					selectedPattern = genericDatetimeRegexPattern
				} // If customLayout is "", selectedPattern remains nonMatchingRegexPattern
			}
			// If formatArg is an unrecognized keyword, selectedPattern remains nonMatchingRegexPattern
			finalRegexPattern.WriteString("(")
			finalRegexPattern.WriteString(selectedPattern)
			finalRegexPattern.WriteString(")")
		case "anyDatetimeNoArg":
			finalRegexPattern.WriteString("(")
			finalRegexPattern.WriteString(nonMatchingRegexPattern)
			finalRegexPattern.WriteString(")")
		case "any":
			finalRegexPattern.WriteString("(")
			finalRegexPattern.WriteString(anyRegexPattern)
			finalRegexPattern.WriteString(")")
		}

		// Advance position in remainingExpectedBody
		remainingExpectedBody = remainingExpectedBody[earliestMatchIndices[1]:]
	}

	finalRegexPattern.WriteString("$")

	compiledRegex, err := regexp.Compile(finalRegexPattern.String())
	if err != nil {
		// If the pattern is our nonMatchingRegexPattern, this compilation itself shouldn't fail.
		// Failure here means a user-supplied regexp was bad, or a chosen const pattern is bad.
		return fmt.Errorf("validation for response #%d ('%s'): failed to compile master regex from expected body: %w. Pattern: %s", responseIndex, responseFilePath, err, finalRegexPattern.String())
	}

	if !compiledRegex.MatchString(normalizedActualBody) {
		diff := difflib.UnifiedDiff{
			A:        difflib.SplitLines(normalizedExpectedBody), // Show original expected for diff
			B:        difflib.SplitLines(normalizedActualBody),
			FromFile: "Expected Body (with placeholders)",
			ToFile:   "Actual Body",
			Context:  3,
		}
		diffText, _ := difflib.GetUnifiedDiffString(diff)
		return fmt.Errorf("validation for response #%d ('%s'): body mismatch (regexp/placeholder evaluation failed):\\n%s\\nCompiled Regex: %s", responseIndex, responseFilePath, diffText, finalRegexPattern.String())
	}

	return nil
}

// Helper function to count non-nil actual responses
func countNonNilActuals(responses []*Response) int {
	count := 0
	for _, r := range responses {
		if r != nil {
			count++
		}
	}
	return count
}

// TODO: Add LoadExpectedResponseFromHTTPFile (parsing a simplified .http format for expected responses)
