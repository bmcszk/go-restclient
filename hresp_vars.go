package restclient

import (
	"bufio"
	"strings"
)

// extractHrespDefines parses raw .hresp content to find @name=value definitions at the beginning of lines.
// These definitions are extracted and returned as a map. The function also returns the .hresp content
// with these definition lines removed.
//
// Lines are trimmed of whitespace before checking for the "@" prefix. A valid definition requires
// an "=" sign. Example: "@token = mysecret". Lines that are successfully parsed as definitions
// are not included in the returned content string. Variable values in `@define` are treated literally;
// they are not resolved against other variables at this extraction stage.
func extractHrespDefines(hrespContent string) (map[string]string, string, error) {
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

		// Process @define line
		processDefineLineToMaps(trimmedLine, defines)
	}

	if err := scanner.Err(); err != nil {
		return nil, "", err
	}

	return defines, strings.Join(processedLines, "\n"), nil
}

// processDefineLineToMaps processes a @define line and adds it to the defines map
func processDefineLineToMaps(trimmedLine string, defines map[string]string) {
	// Line starts with "@", try to parse as a define
	parts := strings.SplitN(trimmedLine[1:], "=", 2)
	if len(parts) != 2 {
		// Malformed define (e.g., "@foo" without "="), or just an "@" symbol.
		// Current logic implies @-prefixed lines that are not valid defines are simply dropped.
		return
	}

	varName := strings.TrimSpace(parts[0])
	varValue := strings.TrimSpace(parts[1])

	if varName == "" {
		// Variable name cannot be empty.
		return
	}

	defines[varName] = varValue
}
