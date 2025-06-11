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
	// Regex to find {{variable}}, {{variable | fallback}}, or {{$systemVar}}
	re := regexp.MustCompile(`{{\s*(.*?)\s*}}`)

	var requestScopedSystemVars map[string]string
	if client != nil {
		requestScopedSystemVars = client.generateRequestScopedSystemVariables()
	}

	resolvedContent := re.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the directive inside {{ }}
		directive := strings.TrimSpace(match[2 : len(match)-2])

		var varName string
		var fallbackValue string
		hasFallback := false

		// Handle system variables that should be pre-resolved if client is available
		if strings.HasPrefix(directive, "$") {
			if requestScopedSystemVars != nil {
				if val, ok := requestScopedSystemVars[directive]; ok {
					return val
				}
			}
			// If not in requestScopedSystemVars, or client is nil,
			// it might be a dynamic one (e.g. {{$dotenv NAME}} or {{$randomInt MIN MAX}}).
			// These will be handled by the client.substituteDynamicSystemVariables pass later.
			// Or, if client is nil, they remain as placeholders.
			return match // Leave for substituteDynamicSystemVariables or as placeholder
		}

		if strings.Contains(directive, "|") {
			parts := strings.SplitN(directive, "|", 2)
			varName = strings.TrimSpace(parts[0])
			fallbackValue = strings.TrimSpace(parts[1])
			hasFallback = true
		} else {
			varName = directive
		}

		// 1. Check Client Programmatic Variables (map[string]any)
		if client != nil && client.programmaticVars != nil {
			if val, ok := client.programmaticVars[varName]; ok {
				return fmt.Sprintf("%v", val)
			}
		}

		// 2. Check fileVars (map[string]string)
		if fileVars != nil {
			if val, ok := fileVars[varName]; ok {
				return val
			}
		}

		// 3. Check Environment Variables
		if envVal, ok := os.LookupEnv(varName); ok {
			return envVal
		}

		// 4. Use Fallback Value
		if hasFallback {
			return fallbackValue
		}

		// If not found and no fallback, return the original placeholder.
		// This function is for substitution; erroring for missing variables is a higher-level decision.
		return match
	})

	// Second pass for simple system variables that might have been introduced by fallbacks.
	if client != nil && requestScopedSystemVars != nil {
		resolvedContent = re.ReplaceAllStringFunc(resolvedContent, func(match2 string) string {
			directive2 := strings.TrimSpace(match2[2 : len(match2)-2])
			// We only care about substituting $-prefixed directives that are in our request-scoped map.
			// Fallbacks were handled in the first pass.
			if strings.HasPrefix(directive2, "$") {
				if val, ok := requestScopedSystemVars[directive2]; ok {
					return val
				}
			}
			return match2
		})
	}

	// Final pass for dynamic system variables (e.g., {{$dotenv VAR}}) or any system vars exposed by fallbacks.
	if client != nil {
		resolvedContent = substituteDynamicSystemVariables(
			resolvedContent, client.currentDotEnvVars, client.programmaticVars)
	}

	return resolvedContent
}
