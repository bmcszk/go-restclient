package restclient

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
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

// resolveAndSubstitute performs variable substitution in a given string content using multiple sources.
// It replaces placeholders in the format `{{variableName}}` or `{{variableName | fallbackValue}}`.
//
// The order of precedence for variable resolution is:
//  1. Request-scoped System Variables (e.g. `{{$uuid}}`, `{{$timestamp}}`,
//     if the placeholder is a direct system variable like `{{$uuid}}`).
//     These are generated once per call by `client.generateRequestScopedSystemVariables()` if a `client` is provided.
//  2. Client Programmatic variables (from `client.programmaticVars`, map[string]any)
//  3. `fileVars` (variables defined with `@name=value` in the .hresp file itself, map[string]string)
//  4. OS Environment variables (looked up by `variableName`)
//  5. `fallbackValue` (if provided in the placeholder like `{{variableName | fallbackValue}}`)
//
// After the above substitutions, a final pass is made using
// `client.substituteDynamicSystemVariables` if a `client` is provided.
// This second pass handles system variables that require argument parsing
// from their placeholder (e.g., `{{$dotenv VAR}}`, `{{$randomInt MIN MAX}}`),
// and also resolves simple system variables (like `{{$uuid}}`) that might have been
// exposed if a fallback value itself was a system variable placeholder
// (e.g., `{{undefined | {{$uuid}}}}`).
//
// If a variable is not found in any source and no fallback is specified, the placeholder remains unchanged.
// The `client` parameter is optional; if nil, client-side programmatic variable substitution and all system
// variable substitutions will not occur.
func resolveAndSubstitute(content string, fileVars map[string]string, client *Client) string {
	re := regexp.MustCompile(`{{\s*(.*?)\s*}}`)
	requestScopedSystemVars := getRequestScopedSystemVars(client)

	resolvedContent := re.ReplaceAllStringFunc(content, 
		createFirstPassReplacer(requestScopedSystemVars, fileVars, client))
	resolvedContent = performSecondPass(re, resolvedContent, requestScopedSystemVars)
	resolvedContent = performFinalPass(resolvedContent, client)

	return resolvedContent
}

// getRequestScopedSystemVars gets system variables from client if available
func getRequestScopedSystemVars(client *Client) map[string]string {
	if client != nil {
		return client.generateRequestScopedSystemVariables()
	}
	return nil
}

// createFirstPassReplacer creates the replacement function for the first pass
func createFirstPassReplacer(
	requestScopedSystemVars map[string]string,
	fileVars map[string]string,
	client *Client,
) func(string) string {
	return func(match string) string {
		directive := strings.TrimSpace(match[2 : len(match)-2])

		if handleSystemVariable(directive, requestScopedSystemVars, match) != match {
			return handleSystemVariable(directive, requestScopedSystemVars, match)
		}

		varName, fallbackValue, hasFallback := parseDirective(directive)
		
		if result := resolveVariable(varName, client, fileVars); result != "" {
			return result
		}

		if hasFallback {
			return fallbackValue
		}

		return match
	}
}

// handleSystemVariable handles system variables (starting with $)
func handleSystemVariable(directive string, requestScopedSystemVars map[string]string, match string) string {
	if strings.HasPrefix(directive, "$") {
		if requestScopedSystemVars != nil {
			if val, ok := requestScopedSystemVars[directive]; ok {
				return val
			}
		}
		return match // Leave for substituteDynamicSystemVariables
	}
	return match // Not a system variable, continue processing
}

// parseDirective parses a directive to extract variable name and fallback
func parseDirective(directive string) (varName, fallbackValue string, hasFallback bool) {
	if strings.Contains(directive, "|") {
		parts := strings.SplitN(directive, "|", 2)
		varName = strings.TrimSpace(parts[0])
		fallbackValue = strings.TrimSpace(parts[1])
		hasFallback = true
	} else {
		varName = directive
	}
	return varName, fallbackValue, hasFallback
}

// resolveVariable attempts to resolve a variable from various sources
func resolveVariable(varName string, client *Client, fileVars map[string]string) string {
	if val := tryProgrammaticVars(varName, client); val != "" {
		return val
	}
	if val := tryFileVars(varName, fileVars); val != "" {
		return val
	}
	if val := tryEnvironmentVars(varName); val != "" {
		return val
	}
	return "" // Not found
}

// tryProgrammaticVars checks client programmatic variables
func tryProgrammaticVars(varName string, client *Client) string {
	if client != nil && client.programmaticVars != nil {
		if val, ok := client.programmaticVars[varName]; ok {
			return fmt.Sprintf("%v", val)
		}
	}
	return ""
}

// tryFileVars checks file-scoped variables
func tryFileVars(varName string, fileVars map[string]string) string {
	if fileVars != nil {
		if val, ok := fileVars[varName]; ok {
			return val
		}
	}
	return ""
}

// tryEnvironmentVars checks OS environment variables
func tryEnvironmentVars(varName string) string {
	if envVal, ok := os.LookupEnv(varName); ok {
		return envVal
	}
	return ""
}

// performSecondPass handles the second pass for system variables
func performSecondPass(re *regexp.Regexp, content string, requestScopedSystemVars map[string]string) string {
	if requestScopedSystemVars == nil {
		return content
	}

	return re.ReplaceAllStringFunc(content, func(match string) string {
		directive := strings.TrimSpace(match[2 : len(match)-2])
		if strings.HasPrefix(directive, "$") {
			if val, ok := requestScopedSystemVars[directive]; ok {
				return val
			}
		}
		return match
	})
}

// performFinalPass handles dynamic system variables
func performFinalPass(content string, client *Client) string {
	if client != nil {
		return substituteDynamicSystemVariables(
			content, client.currentDotEnvVars, client.programmaticVars)
	}
	return content
}
