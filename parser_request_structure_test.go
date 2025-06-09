package restclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PRD-COMMENT: FR1.3 - Request Naming via Separator
// Corresponds to: http_syntax.md "Request Separator", "Named Requests"
// This test verifies that requests can be explicitly named using the '### Request Name' syntax.
// It ensures that the parser correctly extracts and assigns these names to the parsed Request objects.
// The test uses 'testdata/http_request_files/request_name_separator.http'.
func TestParserRequestNaming(t *testing.T) {
	client, err := NewClient()
	require.NoError(t, err)
	require.NotNil(t, client)

	parsed, err := parseRequestFile("testdata/http_request_files/request_name_separator.http", client, nil)
	require.NoError(t, err)
	require.NotNil(t, parsed)

	// Verify FR1.3: Request naming via "### Request Name"
	require.Len(t, parsed.Requests, 2)
	assert.Equal(t, "First Request", parsed.Requests[0].Name)
	assert.Equal(t, "Second Request with Data", parsed.Requests[1].Name)
}

// PRD-COMMENT: FR1.4 - Support for Multiple Comment Styles
// Corresponds to: http_syntax.md "Comments"
// This test verifies the parser's ability to correctly handle both '#' and '//' style comments.
// It ensures that comments are ignored for parsing request lines but that directives within
// comments (e.g., @name, @no-redirect) are still processed correctly regardless of comment style.
// The test uses 'testdata/http_request_files/comment_styles.http'.
func TestParserCommentStyles(t *testing.T) {
	client, err := NewClient()
	require.NoError(t, err)
	require.NotNil(t, client)

	parsed, err := parseRequestFile("testdata/http_request_files/comment_styles.http", client, nil)
	require.NoError(t, err)
	require.NotNil(t, parsed)

	// Verify FR1.4: Support for both # and // comments
	require.Len(t, parsed.Requests, 2)

	// First request should have @name directive from hash comment and @no-redirect
	// directive from slash comment processed
	assert.Equal(t, "HashDirective", parsed.Requests[0].Name)
	assert.True(t, parsed.Requests[0].NoRedirect)

	// Method should be parsed correctly after comments
	assert.Equal(t, "GET", parsed.Requests[0].Method)

	// Second request should parse correctly after both comment styles
	assert.Equal(t, "POST", parsed.Requests[1].Method)
}
