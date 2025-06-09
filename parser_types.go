package restclient

import "strings"

// lineType represents the different types of lines in an HTTP request file
type lineType int

const (
	lineTypeSeparator = iota
	lineTypeVariableDefinition
	lineTypeImportDirective
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
		// Specifically check for "@import" at the beginning of the line.
		if strings.HasPrefix(trimmedLine, "@import ") { // Note the space for robustness
			return lineTypeImportDirective
		}
		// Any other line starting with "@" is considered a variable definition.
		// The handleVariableDefinition function will validate its format (e.g., presence of "=").
		return lineTypeVariableDefinition
	}

	// If none of the above, it's general content (request line, header, body part).
	return lineTypeContent
}
