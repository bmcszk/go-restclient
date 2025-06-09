package restclient

import "strings"

// lineType represents the different types of lines in an HTTP request file
type lineType int

const (
	lineTypeSeparator = iota
	lineTypeVariableDefinition
	lineTypeComment
	lineTypeContent // For request lines, headers, or body
)

func determineLineType(trimmedLine string) lineType {
	if strings.HasPrefix(trimmedLine, requestSeparator) {
		return lineTypeSeparator
	}

	// Handle comments first, as directives like @import might appear in comments
	// but should not be treated as active directives if the line is a comment.
	if strings.HasPrefix(trimmedLine, commentPrefix) || strings.HasPrefix(trimmedLine, slashCommentPrefix) {
		return lineTypeComment
	}

	// Check for lines starting with "@"
	if strings.HasPrefix(trimmedLine, "@") {
		return lineTypeVariableDefinition
	}

	// If none of the above, it's general content (request line, header, body part).
	return lineTypeContent
}
