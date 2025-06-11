package restclient_test

import (
	"bufio" // Added for ExtractHrespDefines
	"strings" // Added for assertMultierrorContains
	"testing"

	"github.com/hashicorp/go-multierror" // Added for assertMultierrorContains
	"github.com/stretchr/testify/assert"  // Added for assertMultierrorContains
	"github.com/stretchr/testify/require"
)


// Helper to return a pointer to an int
func intPtr(i int) *int {
	return &i
}

// assertMultierrorContains checks if a multierror.Error contains the expected number of errors
// and if each of the expected error substrings is present in one of the actual errors.
func assertMultierrorContains(t *testing.T, err error, expectedErrCount int, expectedErrTexts []string) {
	t.Helper()
	merr := validateMultierrorFormat(t, err)
	assert.Len(t, merr.Errors, expectedErrCount, "Mismatch in expected number of errors")
	validateExpectedErrorTexts(t, merr.Errors, expectedErrTexts)
}

func validateMultierrorFormat(t *testing.T, err error) *multierror.Error {
	t.Helper()
	require.Error(t, err, "Expected an error, but got nil")
	merr, ok := err.(*multierror.Error)
	require.True(t, ok, "Expected a *multierror.Error, but got %T", err)
	return merr
}

func validateExpectedErrorTexts(t *testing.T, actualErrors []error, expectedErrTexts []string) {
	t.Helper()
	if len(expectedErrTexts) == 0 {
		return
	}
	for _, expectedText := range expectedErrTexts {
		assertErrorTextExists(t, actualErrors, expectedText)
	}
}

func assertErrorTextExists(t *testing.T, actualErrors []error, expectedText string) {
	t.Helper()
	for _, actualErr := range actualErrors {
		if strings.Contains(actualErr.Error(), expectedText) {
			return
		}
	}
	assert.Fail(t, "Expected error text not found",
		"Expected error text '%s' not found in %v", expectedText, actualErrors)
}

// ExtractHrespDefines parses raw .hresp content to find @name=value definitions at the beginning of lines.
// These definitions are extracted and returned as a map. The function also returns the .hresp content
// with these definition lines removed.
//
// Lines are trimmed of whitespace before checking for the "@" prefix. A valid definition requires
// an "=" sign. Example: "@token = mysecret". Lines that are successfully parsed as definitions
// are not included in the returned content string. Variable values in `@define` are treated literally;
// they are not resolved against other variables at this extraction stage.
func ExtractHrespDefines(hrespContent string) (map[string]string, string, error) {
	defines := make(map[string]string)
	var processedLines []string
	scanner := bufio.NewScanner(strings.NewReader(hrespContent))

	for scanner.Scan() {
		line := scanner.Text()
		processLine(line, defines, &processedLines)
	}

	if err := scanner.Err(); err != nil {
		return nil, "", err
	}

	return defines, strings.Join(processedLines, "\n"), nil
}

func processLine(line string, defines map[string]string, processedLines *[]string) {
	trimmedLine := strings.TrimSpace(line)

	if !strings.HasPrefix(trimmedLine, "@") {
		*processedLines = append(*processedLines, line)
		return
	}

	if !tryParseDefine(trimmedLine, defines) {
		// Malformed define lines are dropped
		return
	}
	// Valid @define lines are not added to processedLines
}

func tryParseDefine(trimmedLine string, defines map[string]string) bool {
	// Line starts with "@", try to parse as a define
	parts := strings.SplitN(trimmedLine[1:], "=", 2)
	if len(parts) != 2 {
		return false
	}

	varName := strings.TrimSpace(parts[0])
	varValue := strings.TrimSpace(parts[1])

	if varName == "" {
		return false
	}

	defines[varName] = varValue
	return true
}

// Ptr returns a pointer to the given value.
// This is a generic helper useful for obtaining pointers to literals in tests.
func Ptr[T any](v T) *T {
	return &v
}
