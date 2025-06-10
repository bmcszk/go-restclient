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
	require.Error(t, err, "Expected an error, but got nil")

	merr, ok := err.(*multierror.Error)
	require.True(t, ok, "Expected a *multierror.Error, but got %T", err)

	assert.Len(t, merr.Errors, expectedErrCount, "Mismatch in expected number of errors")

	if len(expectedErrTexts) > 0 {
		for _, expectedText := range expectedErrTexts {
			found := false
			for _, actualErr := range merr.Errors {
				if strings.Contains(actualErr.Error(), expectedText) {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected error text '%s' not found in %v", expectedText, merr.Errors)
		}
	}
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
		trimmedLine := strings.TrimSpace(line)

		if !strings.HasPrefix(trimmedLine, "@") {
			processedLines = append(processedLines, line)
			continue
		}

		// Line starts with "@", try to parse as a define
		parts := strings.SplitN(trimmedLine[1:], "=", 2)
		if len(parts) != 2 {
			// Malformed define (e.g., "@foo" without "="), or just an "@" symbol.
			// Treat as a regular line if it should be kept, or skip if @-lines not part of content.
			// Current logic implies @-prefixed lines that are not valid defines are simply dropped.
			continue
		}

		varName := strings.TrimSpace(parts[0])
		varValue := strings.TrimSpace(parts[1])

		if varName == "" {
			// Variable name cannot be empty.
			continue
		}

		defines[varName] = varValue
		// Valid @define lines are not added to processedLines
	}

	if err := scanner.Err(); err != nil {
		return nil, "", err
	}

	return defines, strings.Join(processedLines, "\n"), nil
}

// Ptr returns a pointer to the given value.
// This is a generic helper useful for obtaining pointers to literals in tests.
func Ptr[T any](v T) *T {
	return &v
}
