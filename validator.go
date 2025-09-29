package restclient

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
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
	// For {{$anyDatetime}} without args
	anyDatetimeNoArgFinder = regexp.MustCompile(`\{\{\$anyDatetime\}\}`)
	anyPlaceholderFinder   = regexp.MustCompile(`\{\{\$any\}\}`)

	// Pre-compiled regex patterns for JSON placeholder normalization
	jsonAnyGuidPlaceholderPattern      = regexp.MustCompile(`"\{\{\$anyGuid\}\}"`)
	jsonAnyTimestampPlaceholderPattern = regexp.MustCompile(`"\{\{\$anyTimestamp\}\}"`)
	jsonAnyDatetimePlaceholderPattern  = regexp.MustCompile(`"\{\{\$anyDatetime.*?\}\}"`)
	jsonAnyPlaceholderPattern          = regexp.MustCompile(`"\{\{\$any(?:\s+[^}]*)?\}\}"`)
)

const guidRegexPattern = `[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}`
const timestampRegexPattern = `\d+`

// Regex for RFC1123: e.g., Mon, 02 Jan 2006 15:04:05 MST
const rfc1123RegexPattern = `[A-Za-z]{3},\s\d{2}\s[A-Za-z]{3}\s\d{4}\s\d{2}:\d{2}:\d{2}\s[A-Z]{3}`

// Regex for a common ISO8601/RFC3339 form: e.g., 2006-01-02T15:04:05Z or 2006-01-02T15:04:05+07:00
// Added optional milliseconds
const iso8601RegexPattern = `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|([+-]\d{2}:\d{2}))`
const genericDatetimeRegexPattern = `[\w\d\s.:\-,+/TZ()]+`
const nonMatchingRegexPattern = `\z.\A` // Valid but never matches
const anyRegexPattern = `(?s).*?`       // Matches any char (incl newline), non-greedy, no outer group

// ValidateResponses compares actual HTTP responses against a set of expected responses
// parsed from the specified .hresp file. It leverages the client's configuration for variable substitution.
// The `actualResponses` parameter is variadic, allowing zero or more responses to be passed.
//
// As a method on the `Client`, it uses `c.programmaticVars` for programmatic variables and the client instance `c`
// itself for resolving system variables (e.g., {{$uuid}}) within the .hresp content.
// Variables can also be defined in the .hresp file using `@name = value` syntax.
// The precedence for variable resolution is detailed in `hresp_vars.go:resolveAndSubstitute`.
//
// It returns a consolidated error (multierror) if any discrepancies are found (e.g., status mismatch,
// header mismatch, body mismatch, or count mismatch between actual and expected responses), or nil
// if all validations pass. Errors during file reading, @define extraction, variable substitution, or
// .hresp parsing are also returned.
func (c *Client) ValidateResponses(responseFilePath string, actualResponses ...*Response) error {
	expectedResponses, errs, parseErr := c.loadAndParseExpectedResponses(responseFilePath)

	// If there was a critical error (file not found, etc.), return immediately
	if parseErr != nil && errs == nil {
		return parseErr
	}

	// Continue with validation even if parsing failed, but use empty expected responses
	if parseErr != nil {
		expectedResponses = nil
	}

	errs = c.validateResponseCounts(responseFilePath, actualResponses, expectedResponses, errs)
	errs = c.validateResponsePairs(responseFilePath, actualResponses, expectedResponses, errs)
	return errs.ErrorOrNil()
}

func (c *Client) loadAndParseExpectedResponses(
	responseFilePath string) ([]*ExpectedResponse, *multierror.Error, error) {
	var errs *multierror.Error

	hrespFileContent, err := os.ReadFile(responseFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read expected response file %s: %w", responseFilePath, err)
	}

	fileVars, contentWithoutDefines, err := extractHrespDefines(string(hrespFileContent))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract @defines from %s: %w", responseFilePath, err)
	}

	substitutedContent := resolveAndSubstitute(contentWithoutDefines, fileVars, c)

	expectedResponses, parseErr := parseExpectedResponses(strings.NewReader(substitutedContent), responseFilePath)
	if parseErr != nil {
		errs = multierror.Append(errs, fmt.Errorf(
			"failed to parse expected response file '%s' after variable substitution: %w",
			responseFilePath, parseErr))
		return nil, errs, parseErr
	}

	return expectedResponses, nil, nil
}

func (*Client) validateResponseCounts(responseFilePath string, actualResponses []*Response,
	expectedResponses []*ExpectedResponse, errs *multierror.Error) *multierror.Error {
	effectiveNumActual := countNonNilActuals(actualResponses)
	effectiveNumExpected := 0
	if expectedResponses != nil {
		effectiveNumExpected = len(expectedResponses)
	}

	if effectiveNumActual != effectiveNumExpected {
		errs = multierror.Append(errs, fmt.Errorf(
			"mismatch in number of responses: got %d actual, but expected %d from file '%s'",
			effectiveNumActual, effectiveNumExpected, responseFilePath))
	}

	return errs
}

func (c *Client) validateResponsePairs(responseFilePath string, actualResponses []*Response,
	expectedResponses []*ExpectedResponse, errs *multierror.Error) *multierror.Error {
	effectiveNumActual := countNonNilActuals(actualResponses)
	effectiveNumExpected := 0
	if expectedResponses != nil {
		effectiveNumExpected = len(expectedResponses)
	}

	// Only validate pairs where both actual and expected responses exist
	maxPairs := effectiveNumActual
	if effectiveNumExpected < maxPairs {
		maxPairs = effectiveNumExpected
	}

	for i := 0; i < maxPairs; i++ {
		actual := actualResponses[i]
		expected := expectedResponses[i]

		if actual == nil {
			errs = multierror.Append(errs, fmt.Errorf(
				"validation for response #%d ('%s'): actual response is nil",
				i+1, responseFilePath))
			continue
		}

		errs = c.validateSingleResponse(responseFilePath, i+1, actual, expected, errs)
	}

	return errs
}

func (c *Client) validateSingleResponse(responseFilePath string, responseIndex int,
	actual *Response, expected *ExpectedResponse, errs *multierror.Error) *multierror.Error {
	errs = c.validateStatusCode(responseFilePath, responseIndex, actual, expected, errs)
	errs = c.validateStatusString(responseFilePath, responseIndex, actual, expected, errs)
	errs = c.validateHeaders(responseFilePath, responseIndex, actual, expected, errs)
	errs = c.validateBody(responseFilePath, responseIndex, actual, expected, errs)
	return errs
}

func (*Client) validateStatusCode(responseFilePath string, responseIndex int,
	actual *Response, expected *ExpectedResponse, errs *multierror.Error) *multierror.Error {
	if expected.StatusCode != nil && (actual.StatusCode != *expected.StatusCode) {
		errs = multierror.Append(errs, fmt.Errorf(
			"validation for response #%d ('%s'): status code mismatch: expected %d, got %d",
			responseIndex, responseFilePath, *expected.StatusCode, actual.StatusCode))
	}
	return errs
}

func (*Client) validateStatusString(responseFilePath string, responseIndex int,
	actual *Response, expected *ExpectedResponse, errs *multierror.Error) *multierror.Error {
	if expected.Status != nil && *expected.Status != "" && (actual.Status != *expected.Status) {
		errs = multierror.Append(errs, fmt.Errorf(
			"validation for response #%d ('%s'): status string mismatch: expected '%s', got '%s'",
			responseIndex, responseFilePath, *expected.Status, actual.Status))
	}
	return errs
}

func (c *Client) validateHeaders(responseFilePath string, responseIndex int,
	actual *Response, expected *ExpectedResponse, errs *multierror.Error) *multierror.Error {
	if expected.Headers == nil {
		return errs
	}

	for key, expectedValues := range expected.Headers {
		actualValues, ok := actual.Headers[key]
		if !ok {
			errs = multierror.Append(errs, fmt.Errorf(
				"validation for response #%d ('%s'): expected header '%s' not found",
				responseIndex, responseFilePath, key))
			continue
		}

		errs = c.validateHeaderValues(responseFilePath, responseIndex, key, expectedValues, actualValues, errs)
	}

	return errs
}

func (*Client) validateHeaderValues(responseFilePath string, responseIndex int, key string,
	expectedValues, actualValues []string, errs *multierror.Error) *multierror.Error {
	for _, ev := range expectedValues {
		if !isHeaderValuePresent(ev, actualValues) {
			errs = multierror.Append(errs, fmt.Errorf(
				"validation for response #%d ('%s'): expected value '%s' for "+
					"header '%s' not found in actual values %v",
				responseIndex, responseFilePath, ev, key, actualValues))
		}
	}
	return errs
}

// isHeaderValuePresent checks if an expected header value is present in the actual values.
func isHeaderValuePresent(expectedValue string, actualValues []string) bool {
	for _, av := range actualValues {
		if av == expectedValue {
			return true
		}
	}
	return false
}

func (*Client) validateBody(responseFilePath string, responseIndex int,
	actual *Response, expected *ExpectedResponse, errs *multierror.Error) *multierror.Error {
	if expected.Body != nil {
		bodyErr := compareBodies(responseFilePath, responseIndex, *expected.Body, actual.BodyString)
		if bodyErr != nil {
			errs = multierror.Append(errs, bodyErr)
		}
	}
	return errs
}

// compareBodies compares the expected body string with the actual body string,
// supporting placeholders like {{$regexp pattern}}, {{$anyGuid}}, {{$anyTimestamp}}, and {{$anyDatetime format}}.
// placeholderInfo holds details about a supported placeholder type.
type placeholderInfo struct {
	name    string         // e.g., "regexp", "anyGuid"
	finder  *regexp.Regexp // Regex to find the placeholder itself, e.g., "{{$anyGuid}}" or "{{$regexp ...}}"
	pattern string         // Regex pattern to insert for this placeholder, e.g., guidRegexPattern (if no arg)
	// True if the placeholder takes an argument (e.g., {{$regexp `pattern`}} or {{$anyDatetime "format"}})
	hasArgument bool
	// True if the argument itself is the regex pattern to use (e.g., for {{$regexp `pattern`}})
	isArgPattern bool
	// For placeholders like {{$anyDatetime "format"}}, specific logic is needed
	// to derive the pattern from the argument.
}

// buildRegexFromExpectedBody constructs a complete regular expression string
// from an expected body string containing placeholders.
func buildRegexFromExpectedBody(normalizedExpectedBody string) string {
	var finalRegexPattern strings.Builder
	_, _ = finalRegexPattern.WriteString("^")

	remainingExpectedBody := normalizedExpectedBody
	placeholders := getKnownPlaceholders()

	for len(remainingExpectedBody) > 0 {
		earliestMatchIndices, bestPlaceholder := findEarliestPlaceholder(remainingExpectedBody, placeholders)

		if earliestMatchIndices == nil {
			_, _ = finalRegexPattern.WriteString(regexp.QuoteMeta(remainingExpectedBody))
			break
		}

		appendLiteralPart(&finalRegexPattern, remainingExpectedBody, earliestMatchIndices)
		appendPlaceholderPattern(&finalRegexPattern, remainingExpectedBody, earliestMatchIndices, bestPlaceholder)
		remainingExpectedBody = remainingExpectedBody[earliestMatchIndices[1]:]
	}

	_, _ = finalRegexPattern.WriteString("$")
	return finalRegexPattern.String()
}

// getKnownPlaceholders returns all known placeholder definitions.
func getKnownPlaceholders() []placeholderInfo {
	return []placeholderInfo{
		{name: "regexp", finder: regexpPlaceholderFinder, hasArgument: true, isArgPattern: true},
		{name: "anyGuid", finder: anyGuidPlaceholderFinder, pattern: guidRegexPattern},
		{name: "anyTimestamp", finder: anyTimestampPlaceholderFinder, pattern: timestampRegexPattern},
		{name: "anyDatetimeWithArg", finder: anyDatetimePlaceholderFinder, hasArgument: true},
		{name: "anyDatetimeNoArg", finder: anyDatetimeNoArgFinder, pattern: nonMatchingRegexPattern},
		{name: "any", finder: anyPlaceholderFinder, pattern: anyRegexPattern},
	}
}

// findEarliestPlaceholder finds the earliest occurring placeholder in the text.
func findEarliestPlaceholder(text string, placeholders []placeholderInfo) ([]int, placeholderInfo) {
	var earliestMatchIndices []int
	var bestPlaceholder placeholderInfo
	currentMatchPos := len(text) + 1

	for _, ph := range placeholders {
		matchIndices := ph.finder.FindStringSubmatchIndex(text)
		if matchIndices != nil && matchIndices[0] < currentMatchPos {
			currentMatchPos = matchIndices[0]
			earliestMatchIndices = matchIndices
			bestPlaceholder = ph
		}
	}

	return earliestMatchIndices, bestPlaceholder
}

// appendLiteralPart appends the literal text before a placeholder to the regex pattern.
func appendLiteralPart(finalRegexPattern *strings.Builder, text string, matchIndices []int) {
	literalPart := text[:matchIndices[0]]
	_, _ = finalRegexPattern.WriteString(regexp.QuoteMeta(literalPart))
}

// appendPlaceholderPattern appends the regex pattern for a placeholder to the final regex.
func appendPlaceholderPattern(finalRegexPattern *strings.Builder, text string,
	matchIndices []int, placeholder placeholderInfo) {
	_, _ = finalRegexPattern.WriteString("(")

	placeholderArg := extractPlaceholderArgument(text, matchIndices, placeholder)
	pattern := getPlaceholderPattern(placeholder, placeholderArg)
	_, _ = finalRegexPattern.WriteString(pattern)

	_, _ = finalRegexPattern.WriteString(")")
}

// extractPlaceholderArgument extracts the argument from a placeholder match.
func extractPlaceholderArgument(text string, matchIndices []int, placeholder placeholderInfo) string {
	if !placeholder.hasArgument || len(matchIndices) < 4 ||
		matchIndices[2] == -1 || matchIndices[3] == -1 {
		return ""
	}
	return text[matchIndices[2]:matchIndices[3]]
}

// getPlaceholderPattern returns the regex pattern for a specific placeholder.
func getPlaceholderPattern(placeholder placeholderInfo, arg string) string {
	switch placeholder.name {
	case "regexp":
		return processRegexpPlaceholder(arg)
	case "anyDatetimeWithArg":
		return processDatetimePlaceholder(arg)
	default:
		return placeholder.pattern
	}
}

// processRegexpPlaceholder processes a {{$regexp}} placeholder argument.
func processRegexpPlaceholder(userPattern string) string {
	// Strip backticks if present
	if len(userPattern) >= 2 && userPattern[0] == '`' && userPattern[len(userPattern)-1] == '`' {
		userPattern = userPattern[1 : len(userPattern)-1]
	}
	return userPattern
}

// processDatetimePlaceholder processes a {{$anyDatetime}} placeholder argument.
func processDatetimePlaceholder(formatArg string) string {
	formatArg = strings.TrimSpace(formatArg)

	if formatArg == "rfc1123" {
		return rfc1123RegexPattern
	}
	if formatArg == "iso8601" {
		return iso8601RegexPattern
	}
	if len(formatArg) >= 2 && formatArg[0] == '"' && formatArg[len(formatArg)-1] == '"' {
		customLayout := formatArg[1 : len(formatArg)-1]
		if customLayout != "" {
			return genericDatetimeRegexPattern
		}
	}
	return nonMatchingRegexPattern
}

// isJSONContent checks if the given body string contains valid JSON content.
// It attempts to parse the string as JSON and returns true if successful.
// This handles all JSON types: objects, arrays, strings, numbers, booleans, and null.
func isJSONContent(body string) bool {
	// Attempt to parse as JSON - this handles all valid JSON types including:
	// - Objects: {"key": "value"}
	// - Arrays: [1, 2, 3]
	// - Strings: "hello"
	// - Numbers: 42, 3.14
	// - Booleans: true, false
	// - Null: null
	var jsonData any
	err := json.Unmarshal([]byte(body), &jsonData)
	return err == nil
}

// normalizeJSON parses JSON content and re-serializes it with consistent formatting.
// This ensures that JSON with different whitespace, indentation, or line breaks
// will be normalized to the same string representation.
func normalizeJSON(jsonStr string) (string, error) {
	var jsonData any

	// Parse the JSON
	if err := json.Unmarshal([]byte(jsonStr), &jsonData); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Re-serialize with consistent formatting (no extra whitespace)
	normalized, err := json.Marshal(jsonData)
	if err != nil {
		return "", fmt.Errorf("failed to serialize JSON: %w", err)
	}

	return string(normalized), nil
}

// replacePlaceholdersWithTempValues replaces JSON placeholders with temporary valid JSON values
// and returns a mapping from temp values to original placeholders for restoration.
// Uses random numbers as keys to avoid string quoting issues and ensure uniqueness.
func replacePlaceholdersWithTempValues(jsonStr string) (string, map[int]string) {
	result := jsonStr
	placeholderMap := make(map[int]string)

	// Replace all placeholder patterns with unique random number keys using pre-compiled regex patterns
	result = replacePatternPlaceholders(result, jsonAnyGuidPlaceholderPattern, placeholderMap)
	result = replacePatternPlaceholders(result, jsonAnyTimestampPlaceholderPattern, placeholderMap)
	result = replacePatternPlaceholders(result, jsonAnyDatetimePlaceholderPattern, placeholderMap)
	result = replacePatternPlaceholders(result, jsonAnyPlaceholderPattern, placeholderMap)

	return result, placeholderMap
}

// replacePatternPlaceholders replaces placeholders matching a specific pattern with unique random keys
func replacePatternPlaceholders(jsonStr string, regex *regexp.Regexp, placeholderMap map[int]string) string {
	result := jsonStr
	matches := regex.FindAllString(result, -1)

	for _, match := range matches {
		// Generate unique random number key
		tempKey := generateUniqueRandomKey(placeholderMap)
		result = strings.ReplaceAll(result, match, fmt.Sprintf("%d", tempKey))
		placeholderMap[tempKey] = match
	}

	return result
}

// generateUniqueRandomKey generates a unique random number key that doesn't exist in the placeholderMap
// Uses range -2147483648 + 1000 to -2147483648 + 10000 to avoid collision with real data
func generateUniqueRandomKey(placeholderMap map[int]string) int {
	minRange := -2147483648 + 1000  // int.Min + 1000
	maxRange := -2147483648 + 10000 // int.Min + 10000

	for {
		tempKey := minRange + rand.Intn(maxRange-minRange)
		if _, exists := placeholderMap[tempKey]; !exists {
			return tempKey
		}
	}
}

// restorePlaceholdersInNormalizedJSON restores placeholders in a normalized JSON string
// by finding the temporary values and replacing them back with placeholders using the mapping.
func restorePlaceholdersInNormalizedJSON(normalizedJSON string, placeholderMap map[int]string) string {
	result := normalizedJSON
	for tempKey, originalPlaceholder := range placeholderMap {
		result = strings.ReplaceAll(result, fmt.Sprintf("%d", tempKey), originalPlaceholder)
	}
	return result
}

// compareJSONWithPlaceholders compares JSON bodies that contain placeholders.
// It attempts to normalize JSON structure while preserving placeholders, but falls back to original behavior if needed.
func compareJSONWithPlaceholders(responseFilePath string, responseIndex int, expectedBody, actualBody string) error {
	// First, try the JSON-aware placeholder handling
	// Normalize the actual JSON first
	normalizedActual, err := normalizeJSON(actualBody)
	if err != nil {
		// If actual body isn't valid JSON, fall back to original placeholder handling
		return compareBodiesOriginal(responseFilePath, responseIndex, expectedBody, actualBody)
	}

	// For JSON with placeholders, we need to:
	// 1. Create a version where we replace placeholders with temporary values that can be parsed as JSON
	// 2. Normalize this temporary JSON to get the structure
	// 3. Build regex pattern from the normalized structure with placeholders restored

	// Replace placeholders with temporary valid JSON values
	tempExpectedBody, placeholderMap := replacePlaceholdersWithTempValues(expectedBody)

	// Try to normalize the temporary JSON to get the structure
	normalizedTemp, err := normalizeJSON(tempExpectedBody)
	if err != nil {
		// If we can't normalize the temporary JSON (e.g., malformed JSON with placeholders),
		// fall back to original behavior
		return compareBodiesOriginal(responseFilePath, responseIndex, expectedBody, actualBody)
	}

	// Restore placeholders in the normalized JSON
	normalizedExpectedWithPlaceholders := restorePlaceholdersInNormalizedJSON(normalizedTemp, placeholderMap)

	// Build regex pattern from the normalized JSON with placeholders
	regexPatternString := buildRegexFromExpectedBody(normalizedExpectedWithPlaceholders)

	compiledRegex, err := regexp.Compile(regexPatternString)
	if err != nil {
		// If regex compilation fails, fall back to original behavior
		return compareBodiesOriginal(responseFilePath, responseIndex, expectedBody, actualBody)
	}

	// Match the normalized actual JSON against the regex pattern
	if compiledRegex.MatchString(normalizedActual) {
		return nil // Success!
	}

	// If JSON-aware placeholder matching failed, fall back to original behavior
	return compareBodiesOriginal(responseFilePath, responseIndex, expectedBody, actualBody)
}

// compareJSONBodies compares two JSON bodies with whitespace-agnostic comparison.
// It processes placeholders in the expected body, then normalizes both JSON strings and compares them.
func compareJSONBodies(responseFilePath string, responseIndex int, expectedBody, actualBody string) error {
	// First, check if the expected body contains placeholders
	normalizedExpectedBody := strings.TrimSpace(strings.ReplaceAll(expectedBody, "\\r\\n", "\\n"))

	if strings.Contains(normalizedExpectedBody, "{{$") {
		// Use placeholder-based comparison for JSON with placeholders
		return compareJSONWithPlaceholders(responseFilePath, responseIndex, expectedBody, actualBody)
	}

	// No placeholders, use direct JSON normalization and comparison
	normalizedExpected, err := normalizeJSON(expectedBody)
	if err != nil {
		return fmt.Errorf("validation for response #%d ('%s'): failed to normalize expected JSON: %w",
			responseIndex, responseFilePath, err)
	}

	normalizedActual, err := normalizeJSON(actualBody)
	if err != nil {
		return fmt.Errorf("validation for response #%d ('%s'): failed to normalize actual JSON: %w",
			responseIndex, responseFilePath, err)
	}

	// Compare the normalized JSON strings
	if normalizedActual != normalizedExpected {
		diff := difflib.UnifiedDiff{
			A:        difflib.SplitLines(normalizedExpected),
			B:        difflib.SplitLines(normalizedActual),
			FromFile: "Expected JSON (normalized)",
			ToFile:   "Actual JSON (normalized)",
			Context:  3,
		}
		diffText, _ := difflib.GetUnifiedDiffString(diff)
		return fmt.Errorf("validation for response #%d ('%s'): JSON content mismatch:\\n%s",
			responseIndex, responseFilePath, diffText)
	}

	return nil
}

// compareBodiesOriginal compares the expected body string with the actual body string,
// supporting placeholders like {{$regexp pattern}}, {{$anyGuid}}, {{$anyTimestamp}}, and {{$anyDatetime format}}.
// This is the original placeholder logic without JSON-specific handling.
func compareBodiesOriginal(responseFilePath string, responseIndex int, expectedBody, actualBody string) error {
	normalizedExpectedBody := strings.TrimSpace(strings.ReplaceAll(expectedBody, "\\r\\n", "\\n"))
	normalizedActualBody := strings.TrimSpace(strings.ReplaceAll(actualBody, "\\r\\n", "\\n"))

	// Quick check for placeholders to determine if the fast path (direct string comparison) can be taken.
	// The robust placeholder handling is done by buildRegexFromExpectedBody.
	if !strings.Contains(normalizedExpectedBody, "{{$") {
		if normalizedActualBody != normalizedExpectedBody {
			diff := difflib.UnifiedDiff{
				A:        difflib.SplitLines(normalizedExpectedBody),
				B:        difflib.SplitLines(normalizedActualBody),
				FromFile: "Expected Body",
				ToFile:   "Actual Body",
				Context:  3,
			}
			diffText, _ := difflib.GetUnifiedDiffString(diff)
			return fmt.Errorf("validation for response #%d ('%s'): body mismatch:\\n%s",
				responseIndex, responseFilePath, diffText)
		}
		return nil
	}

	// Placeholder-based comparison
	regexPatternString := buildRegexFromExpectedBody(normalizedExpectedBody)

	compiledRegex, err := regexp.Compile(regexPatternString)
	if err != nil {
		return fmt.Errorf(
			"validation for response #%d ('%s'): failed to compile master regex from expected body: %w. Pattern: %s",
			responseIndex, responseFilePath, err, regexPatternString)
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
		return fmt.Errorf(
			"validation for response #%d ('%s'): body mismatch "+
				"(regexp/placeholder evaluation failed):\\n%s\\nCompiled Regex: %s",
			responseIndex, responseFilePath, diffText, regexPatternString)
	}

	return nil
}

// compareBodies compares the expected body string with the actual body string,
// supporting placeholders like {{$regexp pattern}}, {{$anyGuid}}, {{$anyTimestamp}}, and {{$anyDatetime format}}.
// For JSON content, it performs whitespace-agnostic comparison by normalizing JSON formatting.
func compareBodies(responseFilePath string, responseIndex int, expectedBody, actualBody string) error {
	// Check if both bodies are JSON content - if so, use JSON-specific comparison
	if isJSONContent(expectedBody) && isJSONContent(actualBody) {
		return compareJSONBodies(responseFilePath, responseIndex, expectedBody, actualBody)
	}

	// For non-JSON content, use the original placeholder logic
	return compareBodiesOriginal(responseFilePath, responseIndex, expectedBody, actualBody)
}

// countNonNilActuals counts non-nil responses in a slice.
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
