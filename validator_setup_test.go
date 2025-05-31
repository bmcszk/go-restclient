package restclient

import (
	// Used by actualResp in TestValidateResponses_FileErrors
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateResponses_NilAndEmptyActuals(t *testing.T) {
	// Given
	testFilePath := "testdata/http_response_files/validator_nil_empty_actuals_expected.hresp"

	t.Run("nil actual response slice", func(t *testing.T) {
		// Given
		var nilActuals []*Response // nil slice
		client, _ := NewClient()

		// When
		err := client.ValidateResponses(testFilePath, nilActuals)

		// Then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "mismatch in number of responses: got 0 actual, but expected 1")
	})

	t.Run("empty actual response slice", func(t *testing.T) {
		// Given
		emptyActuals := []*Response{} // empty slice
		client, _ := NewClient()

		// When
		err := client.ValidateResponses(testFilePath, emptyActuals)

		// Then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "mismatch in number of responses: got 0 actual, but expected 1")
	})

	t.Run("slice with one nil actual response", func(t *testing.T) {
		// Given
		oneNilActual := []*Response{nil}
		client, _ := NewClient()

		// When
		err := client.ValidateResponses(testFilePath, oneNilActual)

		// Then
		require.Error(t, err)
		merr, ok := err.(*multierror.Error)
		require.True(t, ok, "Expected a multierror.Error")
		require.Len(t, merr.Errors, 1)
		assert.Contains(t, merr.Errors[0].Error(), "mismatch in number of responses: got 0 actual, but expected 1")
	})
}

func TestValidateResponses_FileErrors(t *testing.T) {
	// Given
	actualResp := &Response{StatusCode: 200}
	client, _ := NewClient()

	t.Run("missing expected response file", func(t *testing.T) {
		// Given: actualResp defined above, expected file path is "nonexistent.hresp"

		// When
		err := client.ValidateResponses("nonexistent.hresp", []*Response{actualResp})

		// Then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read expected response file")
		assert.Contains(t, err.Error(), "nonexistent.hresp")
	})

	t.Run("empty expected response file", func(t *testing.T) {
		// Given: actualResp defined above
		emptyFilePath := "testdata/http_response_files/validator_empty_expected.hresp"

		// When
		err := client.ValidateResponses(emptyFilePath, []*Response{actualResp})

		// Then
		assertMultierrorContains(t, err, 1, []string{
			"mismatch in number of responses: got 1 actual, but expected 0",
		})
	})

	t.Run("malformed expected response file", func(t *testing.T) {
		// Given: actualResp defined above
		malformedFilePath := "testdata/http_response_files/validator_malformed_status.hresp"

		// When
		err := client.ValidateResponses(malformedFilePath, []*Response{actualResp})

		// Then
		assertMultierrorContains(t, err, 2, []string{
			"failed to parse expected response file",
			"invalid status code",
			"mismatch in number of responses: got 1 actual, but expected 0",
		})
	})
}
