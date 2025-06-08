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

	variableParts := strings.Split(trimmedLine, "=")
	if len(variableParts) > 1 && strings.HasPrefix(trimmedLine, "@") {
		return lineTypeVariableDefinition
	}

	// Check for @import directive (can be at beginning of line or in a comment)
	if strings.Contains(trimmedLine, "@import") {
		return lineTypeImportDirective
	}

	if strings.HasPrefix(trimmedLine, commentPrefix) || strings.HasPrefix(trimmedLine, slashCommentPrefix) {
		return lineTypeComment
	}

	return lineTypeContent
}
