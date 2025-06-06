package restclient

import (
	"fmt"
	"strings"
)

// extractImportString extracts the import path from an @import line.
func extractImportString(trimmedLine string) (string, error) {
	// Extract from comment if needed
	importStr := trimmedLine
	if strings.HasPrefix(trimmedLine, commentPrefix) || strings.HasPrefix(trimmedLine, slashCommentPrefix) {
		if strings.HasPrefix(trimmedLine, commentPrefix) {
			importStr = strings.TrimPrefix(trimmedLine, commentPrefix)
		} else {
			importStr = strings.TrimPrefix(trimmedLine, slashCommentPrefix)
		}
		importStr = strings.TrimSpace(importStr)
	}

	// Find the @import part
	importIdx := strings.Index(importStr, "@import")
	if importIdx >= 0 {
		// Take everything after @import
		importPath := importStr[importIdx+len("@import"):]
		// Trim any leading/trailing whitespace and quotes
		importPath = strings.TrimSpace(importPath)
		importPath = strings.Trim(importPath, "\"'")
		return importPath, nil
	}

	return "", fmt.Errorf("no @import found in string: %s", trimmedLine)
}

// handleImportDirectiveImpl returns an error as @import directives are not supported in http_syntax.md.
// This functionality is retained for error reporting but not actual implementation.
func handleImportDirectiveImpl(p *requestParserState, trimmedLine string) error {
	// Extract import path for informational purposes only
	importFilePath, err := extractImportString(trimmedLine)
	if err != nil {
		return fmt.Errorf("invalid @import directive: %w", err)
	}

	// Return error indicating imports are not supported
	return fmt.Errorf("@import directive is not supported - the feature is not documented in http_syntax.md: %s", importFilePath)
}
