package restclient

import (
	"strings"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
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
