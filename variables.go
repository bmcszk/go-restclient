package restclient

import (
	crypto_rand "crypto/rand"
	"fmt"
	"log/slog"
	"math/rand"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	reRandomInt             = regexp.MustCompile(`{{\$randomInt(?:\s+(-?\d+)\s+(-?\d+))?}}`)
	reRandomDotInteger      = regexp.MustCompile(`{{\$random\.integer(?:\s+(-?\d+)\s+(-?\d+))?}}`)
	reRandomFloat           = regexp.MustCompile(`{{\$randomFloat(?:\s+(-?\d*\.?\d+)\s+(-?\d*\.?\d+))?}}`)
	reRandomDotFloat        = regexp.MustCompile(`{{\$random\.float(?:\s+(-?\d*\.?\d+)\s+(-?\d*\.?\d+))?}}`)
	reRandomHex             = regexp.MustCompile(`{{\$randomHex(?:\s+(\d+))?}}`)
	reRandomDotHexadecimal  = regexp.MustCompile(`{{\$random\.hexadecimal(?:\s+(\d+))?}}`)
	reRandomAlphaNumeric    = regexp.MustCompile(`{{\$randomAlphaNumeric(?:\s+(\d+))?}}`)
	reRandomDotAlphabetic   = regexp.MustCompile(`{{\$random\.alphabetic(?:\s+(\d+))?}}`)
	reRandomDotAlphanumeric = regexp.MustCompile(`{{\$random\.alphanumeric(?:\s+(\d+))?}}`)
	reRandomString          = regexp.MustCompile(`{{\$randomString(?:\s+(\d+))?}}`)
	reRandomPassword        = regexp.MustCompile(`{{\$randomPassword(?:\s+(\d+))?}}`)
	reDotEnv                = regexp.MustCompile(`{{\s*\$dotenv\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*}}`)
	reProcessEnv            = regexp.MustCompile(`{{\s*\$processEnv\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*}}`)
	reDateTime = regexp.MustCompile(
		`{{\s*\$datetime(?:\s+("([^"]+)"|[^}\s]+))?(?:\s+("([^"]+)"|[^}\s]+))?\s*}}`)
	reAadToken              = regexp.MustCompile(`{{\s*\$aadToken(?:\s+("([^"]+)"|[^}\s]+))*\s*}}`)
)

const (
	defaultRandomLength          = 16
	defaultRandomHexLength       = 16
	defaultRandomPasswordLength  = 12
	defaultRandomMinInt          = 0
	defaultRandomMaxInt          = 100
	defaultRandomMinFloat        = 0.0
	defaultRandomMaxFloat        = 1.0
	charsetHex                   = "0123456789abcdef"
	charsetAlphabetic            = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	charsetAlphaNumeric          = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	charsetAlphaNumericWithExtra = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
	charsetFull = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"0123456789!@#$%^&*()_+-=[]{};':\",./<>?"
)

// Word list for $randomWord
var randomWords = []string{"apple", "banana", "cherry", "date", "elderberry", "fig", "grape"}

// resolveVariablesInText is the primary substitution engine for non-system
// variables and request-scoped system variables.
// It iterates through placeholders like `{{varName | fallback}}` and resolves them based on a defined precedence.
// Dynamic system variables (like {{$dotenv NAME}}) are left untouched by this
// function for substituteDynamicSystemVariables.
//
// Precedence for {{variableName}} placeholders (where 'variableName' does not start with '$'):
// 1. Client programmatic variables (clientProgrammaticVars)
// 2. Request file-defined variables (fileScopedVars, from @name=value, effectively rcRequest.ActiveVariables)
// 3. Environment variables (environmentVars, from selected environment like http-client.env.json)
// 4. Global variables (globalVars, from http-client.private.env.json or similar)
// 5. OS Environment variables (via osEnvGetter)
// 6. Variables from .env file (dotEnvVars)
// 7. Fallback value provided in the placeholder itself.
// System variables (e.g., {{$uuid}}, {{$timestamp}}) are handled if the
// placeholder is like {{$systemVarName}} (i.e. varName starts with '$').
func resolveVariablesInText(
	text string,
	clientProgrammaticVars map[string]any,
	fileScopedVars map[string]string,
	environmentVars map[string]string,
	globalVars map[string]string,
	requestScopedSystemVars map[string]string,
	osEnvGetter func(string) (string, bool),
	dotEnvVars map[string]string,
) string {
	const maxIterations = 10 // Safety break for circular dependencies
	currentText := text

	for i := 0; i < maxIterations; i++ {
		previousText := currentText
		re := regexp.MustCompile(`{{\s*(.*?)\s*}}`)

		currentText = re.ReplaceAllStringFunc(previousText, func(match string) string {
			directive := strings.TrimSpace(match[2 : len(match)-2])

			var varName string
			var fallbackValue string
			var hasFallback bool

			if strings.Contains(directive, "|") {
				parts := strings.SplitN(directive, "|", 2)
				varName = strings.TrimSpace(parts[0])
				fallbackValue = strings.TrimSpace(parts[1])
				hasFallback = true
			} else {
				varName = directive
			}

			// 1. Request-scoped System Variables (if varName starts with $)
			// These are simple, pre-generated variables like $uuid, $timestamp, $randomInt (no-args).
			if strings.HasPrefix(varName, "$") {
				if val, ok := requestScopedSystemVars[varName]; ok {
					slog.Debug(
					"resolveVariablesInText: Found system var in requestScopedSystemVars",
					"varName", varName, "value", val)
					return val
				}
				slog.Debug(
				"resolveVariablesInText: System var not in requestScopedSystemVars, "+
					"returning original match for later dynamic processing",
				"varName", varName, "match", match)
				return match // Preserve for substituteDynamicSystemVariables if it's like {{$dotenv NAME}}
			}

			// Precedence for {{variableName}} placeholders:

			// 1. Client programmatic variables (clientProgrammaticVars)
			if clientProgrammaticVars != nil {
				slog.Debug(
				"resolveVariablesInText: Checking clientProgrammaticVars",
				"varName", varName, "found", clientProgrammaticVars[varName] != nil)
				if val, ok := clientProgrammaticVars[varName]; ok {
					slog.Debug(
					"resolveVariablesInText: Found in clientProgrammaticVars",
					"varName", varName, "value", val)
					return fmt.Sprintf("%v", val)
				}
			}

			// 2. File-scoped variables (e.g., @name from .rest file)
			// These are stored with an '@' prefix in the map, so we must prepend it for lookup.
			fileScopedVarNameToTry := "@" + varName
			if val, ok := fileScopedVars[fileScopedVarNameToTry]; ok {
				slog.Debug("resolveVariablesInText: Found in fileScopedVars", "varNameLookup", fileScopedVarNameToTry, "value", val)
				// Check if the resolved file-scoped variable's value is itself a dynamic system variable placeholder
				if isDynamicSystemVariablePlaceholder(val, requestScopedSystemVars) {
					slog.Debug("resolveVariablesInText: File-scoped var value is a dynamic system variable placeholder. Evaluating and caching.", "varNameLookup", fileScopedVarNameToTry, "placeholderValue", val)
					// Pass clientProgrammaticVars and dotEnvVars to substituteDynamicSystemVariables
					evaluatedVal := substituteDynamicSystemVariables(val, dotEnvVars, clientProgrammaticVars)
					fileScopedVars[fileScopedVarNameToTry] = evaluatedVal // Cache the evaluated value
					slog.Debug("resolveVariablesInText: Cached evaluated dynamic system variable from file-scoped var", "varNameLookup", fileScopedVarNameToTry, "evaluatedValue", evaluatedVal)
					return evaluatedVal
				}
				return val // Return the original value if not a dynamic placeholder needing evaluation
			} else {
				slog.Debug("resolveVariablesInText: Not found in fileScopedVars", "varNameLookup", fileScopedVarNameToTry, "originalVarNameFromPlaceholder", varName)
			}

			// The block for fileScopedVars lookup (originally here as step 2) has been moved up
			// and consolidated to correctly handle the '@' prefix for file-scoped variable names.
			// See lines around 101-109 for the correct implementation.

			// 3. Try environment-specific variables (from http-client.env.json)
			if environmentVars != nil {
				if val, ok := environmentVars[varName]; ok {
					return val
				}
			}

			// 4. Try global variables (from http-client.private.env.json)
			if globalVars != nil {
				if val, ok := globalVars[varName]; ok {
					return val
				}
			}

			// 5. Try OS environment variables
			if osEnvGetter != nil {
				if val, ok := osEnvGetter(varName); ok {
					return val
				}
			}

			// 6. Try .env file variables
			if dotEnvVars != nil {
				if val, ok := dotEnvVars[varName]; ok {
					return val
				}
			}

			// 7. Use fallback value if provided in the placeholder itself
			if hasFallback {
				return fallbackValue
			}

			// No resolution found, and no fallback in placeholder. Default to empty string.
			return ""
		}) // End of ReplaceAllStringFunc

		if currentText == previousText {
			break // No more substitutions made in this pass
		}
		// If we are on the last iteration and still making changes, it might be a circular dependency.
		// The currentText will be returned as is, potentially with unresolved variables.
	} // End of for loop
	return currentText
}

// _applyBaseURLIfNeeded attempts to prepend a base URL to a raw URL string
// if the raw URL doesn't have a scheme and a non-empty clientBaseURL is provided.
func _applyBaseURLIfNeeded(rawURL string, clientBaseURL string) string {
	if strings.Contains(rawURL, "://") || clientBaseURL == "" {
		return rawURL // No need to apply base URL
	}

	// At this point, rawURL has no scheme, and clientBaseURL is not empty.

	if !strings.HasPrefix(rawURL, "/") {
		// It's a relative path, simply prepend base URL + "/"
		return strings.TrimRight(clientBaseURL, "/") + "/" + rawURL
	}

	// It's an absolute path (starts with "/").
	// We need to take scheme and host from clientBaseURL.
	baseURLParsed, err := url.Parse(clientBaseURL)
	if err == nil && baseURLParsed.Scheme != "" && baseURLParsed.Host != "" {
		return baseURLParsed.Scheme + "://" + baseURLParsed.Host + rawURL
	}

	// If clientBaseURL couldn't be parsed properly or is not a full URL,
	// return the original rawURL (which is an absolute path).
	return rawURL
} // End of function resolveVariablesInText

// isDynamicSystemVariablePlaceholder checks if a given string value is a placeholder
// for a dynamic system variable that requires on-the-fly evaluation.
// It returns true if the value is a system variable placeholder (e.g., "{{$randomInt}}", "{{$dotenv VAR}}")
// AND is not already pre-evaluated in requestScopedSystemVars (like "$uuid").
func isDynamicSystemVariablePlaceholder(value string, requestScopedSystemVars map[string]string) bool {
	if !strings.HasPrefix(value, "{{") || !strings.HasSuffix(value, "}}") {
		// Check for $... without {{}} for direct system var names like $uuid
		if strings.HasPrefix(value, "$") {
			if _, ok := requestScopedSystemVars[value]; ok {
				slog.Debug("isDynamicSystemVariablePlaceholder: Direct value is a pre-evaluated system variable key", "value", value)
				return false // It's a key like $uuid, not a placeholder like {{$uuid}}
			}
		}
		return false // Not a {{...}} placeholder pattern
	}

	// Extract the potential system variable name/directive from within {{...}}
	// e.g., "$randomInt 10 20" from "{{$randomInt 10 20}}"
	innerDirective := strings.TrimSpace(value[2 : len(value)-2])

	if !strings.HasPrefix(innerDirective, "$") {
		return false // Inner part doesn't start with $, so not a system variable placeholder
	}

	// Check if this exact inner directive is already a pre-evaluated simple system variable (like $uuid, $timestamp, or no-arg $randomInt)
	if _, ok := requestScopedSystemVars[innerDirective]; ok {
		slog.Debug("isDynamicSystemVariablePlaceholder: Placeholder's inner directive is a pre-evaluated system variable", "value", value, "innerDirective", innerDirective)
		return false
	}

	dynamicRegexes := []*regexp.Regexp{
		reRandomInt, reRandomDotInteger, reRandomFloat, reRandomDotFloat,
		reRandomHex, reRandomDotHexadecimal, reRandomAlphaNumeric, reRandomDotAlphabetic,
		reRandomDotAlphanumeric, reRandomString, reRandomPassword,
		reDotEnv, reProcessEnv, reDateTime, reAadToken,
	}

	for _, re := range dynamicRegexes {
		if re.MatchString(value) {
			slog.Debug("isDynamicSystemVariablePlaceholder: Placeholder matches a dynamic system variable pattern", "value", value, "regex", re.String())
			return true
		}
	}
	slog.Debug("isDynamicSystemVariablePlaceholder: Placeholder does not match any known dynamic pattern or is not a dynamic system variable requiring further evaluation", "value", value)
	return false
}

// randomStringFromCharset generates a random string of a given length using characters from the provided charset.
func randomStringFromCharset(length int, charset string) string {
	if length <= 0 || len(charset) == 0 { // Added len(charset) == 0 check
		return ""
	}
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rng.Intn(len(charset))]
	}
	return string(b)
}

// substituteRequestVariables handles the substitution of variables in the request's URL and headers.
// It returns the final parsed URL or an error if substitution/parsing fails.
func substituteRequestVariables(rcRequest *Request, parsedFile *ParsedFile, requestScopedSystemVars map[string]string, osEnvGetter func(string) (string, bool), programmaticVars map[string]any, currentDotEnvVars map[string]string, clientBaseURL string) (*url.URL, error) {

	var fileScopedVars map[string]string // Declare fileScopedVars at function scope
	var envVarsFromFile map[string]string
	var globalVarsFromFile map[string]string

	if parsedFile != nil {
		// Create a mutable copy for this request's substitution pass, to prevent modifying the original ParsedFile.FileVariables
		fileScopedVars = make(map[string]string, len(parsedFile.FileVariables))
		for k, v := range parsedFile.FileVariables {
			fileScopedVars[k] = v
		}
		envVarsFromFile = parsedFile.EnvironmentVariables // These are typically not modified by substitution, so direct assignment is fine
		globalVarsFromFile = parsedFile.GlobalVariables   // Same for these
	} else {
		// For direct execution without a file context (e.g. client.Execute called directly or URL substitution in executeRequest)
		fileScopedVars = make(map[string]string)     // Ensure not nil
		envVarsFromFile = make(map[string]string)    // Ensure not nil
		globalVarsFromFile = make(map[string]string) // Ensure not nil
	}

	// Merge request-specific variables (@name=value defined in the request block)
	// These override file-scoped variables for the current request.
	if rcRequest != nil && rcRequest.ActiveVariables != nil {
		for k, v := range rcRequest.ActiveVariables {
			fileScopedVars[k] = v
		}
	}

	substitutedRawURL := resolveVariablesInText(
		rcRequest.RawURLString,  // text
		programmaticVars,        // clientProgrammaticVars
		fileScopedVars,          // fileScopedVars (from @name=value in file, or empty if no file)
		envVarsFromFile,         // environmentVars (from http-client.env.json for selected env)
		globalVarsFromFile,      // globalVars (from http-client.private.env.json)
		requestScopedSystemVars, // requestScopedSystemVars (e.g. {{$uuid}}, {{$timestamp}})
		osEnvGetter,             // osEnvGetter
		currentDotEnvVars,       // dotEnvVars (from .env file in request's dir, or client's current if no file)
	)
	substitutedRawURL = substituteDynamicSystemVariables(substitutedRawURL, currentDotEnvVars, programmaticVars)

	// Handle empty URLs after variable substitution - this is a common cause of failures
	if strings.TrimSpace(substitutedRawURL) == "" {
		return nil, fmt.Errorf("URL is empty after variable substitution (original: %s)", rcRequest.RawURLString)
	}

	substitutedRawURL = _applyBaseURLIfNeeded(substitutedRawURL, clientBaseURL)

	finalParsedURL, parseErr := url.Parse(substitutedRawURL)
	if parseErr != nil {
		return nil, fmt.Errorf("failed to parse URL after variable substitution: %s (original: %s): %w", substitutedRawURL, rcRequest.RawURLString, parseErr)
	}
	// rcRequest.URL = finalParsedURL // Assign here as it's successfully parsed - redundant, caller should assign

	if rcRequest.Headers != nil {
		for key, values := range rcRequest.Headers {
			newValues := make([]string, len(values))
			for j, val := range values {
				resolvedVal := resolveVariablesInText(
					val,
					programmaticVars,
					fileScopedVars, // Use the common map that includes file and request-scoped vars, and allows caching
					envVarsFromFile,
					globalVarsFromFile,
					requestScopedSystemVars,
					osEnvGetter,
					currentDotEnvVars,
				)
				newValues[j] = substituteDynamicSystemVariables(resolvedVal, currentDotEnvVars, programmaticVars)
			}
			rcRequest.Headers[key] = newValues
		}
	}
	return finalParsedURL, nil
}

// _parseLength extracts an optional length argument from a regex match.
func _parseLength(match string, re *regexp.Regexp, defaultLength int) (int, bool) {
	parts := re.FindStringSubmatch(match)
	if len(parts) > 1 && parts[1] != "" {
		parsedLen, err := strconv.Atoi(parts[1])
		if err != nil || parsedLen < 0 { // Allow 0 for empty string, but not negative
			return 0, false // Invalid length
		}
		return parsedLen, true
	}
	return defaultLength, true
}

// _parseRangeInt extracts optional min and max integer arguments.
func _parseRangeInt(match string, re *regexp.Regexp, defaultMin, defaultMax int) (int, int, bool) {
	parts := re.FindStringSubmatch(match)
	if len(parts) == 3 && parts[1] != "" && parts[2] != "" {
		min, errMin := strconv.Atoi(parts[1])
		max, errMax := strconv.Atoi(parts[2])
		if errMin != nil || errMax != nil || min > max {
			return 0, 0, false // Invalid range
		}
		return min, max, true
	}
	return defaultMin, defaultMax, true
}

// _parseRangeFloat extracts optional min and max float arguments.
func _parseRangeFloat(match string, re *regexp.Regexp, defaultMin, defaultMax float64) (float64, float64, bool) {
	parts := re.FindStringSubmatch(match)
	if len(parts) == 3 && parts[1] != "" && parts[2] != "" {
		min, errMin := strconv.ParseFloat(parts[1], 64)
		max, errMax := strconv.ParseFloat(parts[2], 64)
		if errMin != nil || errMax != nil || min > max {
			return 0, 0, false // Invalid range
		}
		return min, max, true
	}
	return defaultMin, defaultMax, true
}

// _substituteRandomIntFunc returns a function for ReplaceAllStringFunc to generate random integers.
func _substituteRandomIntFunc(re *regexp.Regexp, defaultMin, defaultMax int) func(string) string {
	return func(match string) string {
		min, max, ok := _parseRangeInt(match, re, defaultMin, defaultMax)
		if !ok {
			return match // Malformed range
		}
		return strconv.Itoa(rand.Intn(max-min+1) + min)
	}
}

// _substituteRandomFloatFunc returns a function for ReplaceAllStringFunc to generate random floats.
func _substituteRandomFloatFunc(re *regexp.Regexp, defaultMin, defaultMax float64) func(string) string {
	return func(match string) string {
		min, max, ok := _parseRangeFloat(match, re, defaultMin, defaultMax)
		if !ok {
			return match // Malformed range
		}
		return fmt.Sprintf("%f", min+rand.Float64()*(max-min))
	}
}

// _substituteRandomLengthCharsetFunc returns a function for ReplaceAllStringFunc to generate random strings from a charset.
func _substituteRandomLengthCharsetFunc(re *regexp.Regexp, charset string) func(string) string {
	return func(match string) string {
		length, ok := _parseLength(match, re, defaultRandomLength)
		if !ok { // Invalid length format
			return match
		}
		if length == 0 { // Explicit request for empty string
			return ""
		}
		if length < 0 { // Should be caught by _parseLength, but defensive
			return match
		}
		return randomStringFromCharset(length, charset)
	}
}

// _substituteRandomHexHelper is a specific helper for $randomHex and $random.hexadecimal.
func _substituteRandomHexHelper(re *regexp.Regexp, defaultLength int) func(string) string {
	return func(match string) string {
		length, ok := _parseLength(match, re, defaultLength)
		if !ok { // Invalid length format
			return match
		}
		if length == 0 {
			return ""
		}
		if length < 0 {
			return match
		}

		// Each byte becomes two hex characters.
		// For odd length, we need one extra byte and then truncate.
		byteCount := length/2 + length%2
		b := make([]byte, byteCount)
		if _, err := crypto_rand.Read(b); err != nil {
			slog.Error("Failed to generate random bytes for hex string", "error", err)
			return match // Error generating random bytes
		}
		hexStr := fmt.Sprintf("%x", b)
		return hexStr[:length] // Truncate if original length was odd
	}
}

// _substituteDateTimeVariables handles the substitution of $datetime and $localDatetime variables.
func _substituteDateTimeVariables(text string) string {
	// REQ-LIB-029 & REQ-LIB-030: Datetime variables ($datetime, $localDatetime)
	reDateTimeRelated := regexp.MustCompile(`{{\$(datetime|localDatetime)((?:\s*(?:\"[^\"]*\"|[^\"\s}]+))*)\s*}}`)
	text = reDateTimeRelated.ReplaceAllStringFunc(text, func(match string) string {
		captures := reDateTimeRelated.FindStringSubmatch(match)
		if len(captures) < 3 { // Should have full match, type, and args part
			slog.Warn("Could not parse datetime/localDatetime variable, captures unexpected", "match", match, "capturesCount", len(captures))
			return match // Safety return
		}
		varType := captures[1] // "datetime" or "localDatetime"
		argsStr := strings.TrimSpace(captures[2])

		var formatStr string
		// var offsetStr string // TODO: Implement offset parsing in a subsequent step

		// Regex to parse arguments: captures quoted strings or unquoted non-space sequences
		argPartsRegex := regexp.MustCompile(`(?:\"([^\"]*)\"|([^\"\s}]+))`)
		parsedArgsMatches := argPartsRegex.FindAllStringSubmatch(argsStr, -1)

		parsedArgs := []string{}
		for _, m := range parsedArgsMatches {
			if m[1] != "" { // Quoted argument
				parsedArgs = append(parsedArgs, m[1])
			} else if m[2] != "" { // Unquoted argument
				parsedArgs = append(parsedArgs, m[2])
			}
		}

		if len(parsedArgs) > 0 {
			formatStr = parsedArgs[0]
		} else {
			// Default format if none provided, as per common expectations (e.g., ISO8601)
			formatStr = "iso8601"
		}
		// "[DEBUG_DATETIME] Determined formatStr", "varType", varType, "argsStr", argsStr, "parsedFormatStr", formatStr, "originalMatch", match)
		// if len(parsedArgs) > 1 {
		// offsetStr = parsedArgs[1] // TODO: Implement offset parsing
		// }

		var now time.Time
		if varType == "datetime" {
			now = time.Now().UTC()
		} else { // localDatetime
			now = time.Now() // System's local time zone
		}

		// TODO: Implement offset application using offsetStr and 'now' in a subsequent step

		var resultStr string
		switch strings.ToLower(formatStr) {
		case "rfc1123":
			resultStr = now.Format(time.RFC1123)
		case "iso8601":
			resultStr = now.Format(time.RFC3339)
		case "timestamp":
			resultStr = strconv.FormatInt(now.Unix(), 10)
		default:
			// TODO: Implement custom Java format string translation to Go layout in a subsequent step
			// slog.Warn("Unsupported or custom datetime format, returning original match for now", "format", formatStr, "variableType", varType, "originalMatch", match)
			return match
		}

		return resultStr
	})
	return text
}

// substituteDynamicSystemVariables handles system variables that require argument parsing or dynamic evaluation at substitution time.
// These are typically {{$processEnv VAR}}, {{$dotenv VAR}}, and {{$randomInt MIN MAX}}.
// This also handles JetBrains HTTP client syntax variables like {{$random.integer MIN MAX}},
// {{$random.alphabetic LENGTH}}, etc.
// Other simple system variables like {{$uuid}} or {{$timestamp}}
// should have been pre-resolved and substituted by resolveVariablesInText via the
// requestScopedSystemVars map.
func substituteDynamicSystemVariables(text string, activeDotEnvVars map[string]string, programmaticVars map[string]any) string {
	// "[DEBUG_DYN_VARS_INPUT]", "inputText", text)
	originalTextForLogging := text // Keep a copy for logging. Used if we add more complex types with logging.
	_ = originalTextForLogging     // Avoid unused variable error if no logging exists below.

	text = substituteRandomVariables(text, programmaticVars)

	// Handle {{$env.VAR_NAME}}
	reSystemEnvVar := regexp.MustCompile(`{{\$env\.([A-Za-z_][A-Za-z0-9_]*?)}}`)
	text = reSystemEnvVar.ReplaceAllStringFunc(text, func(match string) string {
		parts := reSystemEnvVar.FindStringSubmatch(match)
		if len(parts) == 2 {
			varName := parts[1]
			value := os.Getenv(varName)
			return value
		}
		slog.Warn("Failed to parse $env.VAR_NAME, returning original match", "match", match, "parts_len", len(parts))
		return match
	})

	// REQ-LIB-011: $dotenv MY_VARIABLE_NAME
	text = reDotEnv.ReplaceAllStringFunc(text, func(match string) string {
		parts := reDotEnv.FindStringSubmatch(match)
		if len(parts) == 2 {
			varName := parts[1]
			if val, ok := activeDotEnvVars[varName]; ok {
				return val
			}
			return "" // Variable not found in .env, return empty string
		}
		slog.Warn("Failed to parse $dotenv, returning original match", "match", match, "parts_len", len(parts))
		return match // Should not happen with a valid regex, but good for safety
	})

	// Also handle URL encoded version
	reDotEnvEncoded := regexp.MustCompile(`%7B%7B\$dotenv\s+([a-zA-Z_][a-zA-Z0-9_]*)%7D%7D`)
	text = reDotEnvEncoded.ReplaceAllStringFunc(text, func(match string) string {
		parts := reDotEnvEncoded.FindStringSubmatch(match)
		if len(parts) == 2 {
			varName := parts[1]
			if val, ok := activeDotEnvVars[varName]; ok {
				return val
			}
			return "" // Variable not found in .env, return empty string
		}
		slog.Warn("Failed to parse URL-encoded $dotenv, returning original match", "match", match, "parts_len", len(parts))
		return match
	})

	// REQ-LIB-012: $processEnv MY_ENV_VAR
	// First, try with double braces
	// reProcessEnv is now a package-level variable
	text = reProcessEnv.ReplaceAllStringFunc(text, func(match string) string {
		parts := reProcessEnv.FindStringSubmatch(match)
		if len(parts) == 2 {
			varName := parts[1]
			if val, ok := os.LookupEnv(varName); ok {
				return val
			}
			return match // Variable not found, return original placeholder
		}
		slog.Warn("Failed to parse $processEnv, returning original match", "match", match, "parts_len", len(parts))
		return match
	})

	// Also handle URL encoded version
	reProcessEnvEncoded := regexp.MustCompile(`%7B%7B\$processEnv\s+([A-Za-z_][A-Za-z0-9_]*)%7D%7D`)
	text = reProcessEnvEncoded.ReplaceAllStringFunc(text, func(match string) string {
		parts := reProcessEnvEncoded.FindStringSubmatch(match)
		if len(parts) == 2 {
			varName := parts[1]
			if val, ok := os.LookupEnv(varName); ok {
				return val
			}
			return match // Variable not found, return original placeholder
		}
		slog.Warn("Failed to parse URL-encoded $processEnv, returning original match", "match", match, "parts_len", len(parts))
		return match
	})

	text = _substituteDateTimeVariables(text)
	return text
}

// substituteRandomVariables handles the substitution of $random.* variables.
func substituteRandomVariables(text string, programmaticVars map[string]any) string {
	// Integer types
	text = reRandomInt.ReplaceAllStringFunc(text, _substituteRandomIntFunc(reRandomInt, defaultRandomMinInt, defaultRandomMaxInt))
	text = reRandomDotInteger.ReplaceAllStringFunc(text, _substituteRandomIntFunc(reRandomDotInteger, defaultRandomMinInt, defaultRandomMaxInt))

	// Float types
	text = reRandomFloat.ReplaceAllStringFunc(text, _substituteRandomFloatFunc(reRandomFloat, defaultRandomMinFloat, defaultRandomMaxFloat))
	text = reRandomDotFloat.ReplaceAllStringFunc(text, _substituteRandomFloatFunc(reRandomDotFloat, defaultRandomMinFloat, defaultRandomMaxFloat))

	// Boolean
	text = strings.ReplaceAll(text, "{{$randomBoolean}}", strconv.FormatBool(rand.Intn(2) == 0))

	// Hexadecimal
	text = reRandomHex.ReplaceAllStringFunc(text, _substituteRandomHexHelper(reRandomHex, defaultRandomHexLength))
	text = reRandomDotHexadecimal.ReplaceAllStringFunc(text, _substituteRandomHexHelper(reRandomDotHexadecimal, defaultRandomHexLength))

	// Alphabetic / Alphanumeric
	text = reRandomDotAlphabetic.ReplaceAllStringFunc(text, _substituteRandomLengthCharsetFunc(reRandomDotAlphabetic, charsetAlphabetic))
	text = reRandomAlphaNumeric.ReplaceAllStringFunc(text, _substituteRandomLengthCharsetFunc(reRandomAlphaNumeric, charsetAlphaNumericWithExtra)) // Uses underscore
	text = reRandomDotAlphanumeric.ReplaceAllStringFunc(text, _substituteRandomLengthCharsetFunc(reRandomDotAlphanumeric, charsetAlphaNumeric))    // No underscore

	// General Random String
	text = reRandomString.ReplaceAllStringFunc(text, _substituteRandomLengthCharsetFunc(reRandomString, charsetFull))

	// Email
	emailGenerator := func() string {
		return fmt.Sprintf("%s@%s.com",
			randomStringFromCharset(10, charsetAlphaNumeric),
			randomStringFromCharset(7, charsetAlphabetic))
	}
	text = strings.ReplaceAll(text, "{{$randomEmail}}", emailGenerator())
	text = strings.ReplaceAll(text, "{{$random.email}}", emailGenerator())

	// Domain
	text = strings.ReplaceAll(text, "{{$randomDomain}}",
		fmt.Sprintf("%s.com", randomStringFromCharset(10, charsetAlphabetic)))

	// IP Addresses
	text = strings.ReplaceAll(text, "{{$randomIPv4}}",
		fmt.Sprintf("%d.%d.%d.%d", rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256)))

	text = strings.ReplaceAll(text, "{{$randomIPv6}}", func() string {
		segments := make([]string, 8)
		for i := 0; i < 8; i++ {
			segments[i] = fmt.Sprintf("%x", rand.Intn(0x10000))
		}
		return strings.Join(segments, ":")
	}())

	// UUID
	text = strings.ReplaceAll(text, "{{$randomUUID}}", uuid.New().String())

	// Password (uses programmaticVars, so it calls the existing _substituteRandomPasswordFunc with modification)
	text = reRandomPassword.ReplaceAllStringFunc(text, func(match string) string {
		return _substituteRandomPasswordFunc(match, programmaticVars)
	})

	// Color
	text = strings.ReplaceAll(text, "{{$randomColor}}",
		fmt.Sprintf("#%02x%02x%02x", rand.Intn(256), rand.Intn(256), rand.Intn(256)))

	// Word
	if len(randomWords) > 0 { // Prevent panic on empty slice
		text = strings.ReplaceAll(text, "{{$randomWord}}", randomWords[rand.Intn(len(randomWords))])
	}

	return text
}

// _substituteRandomPasswordFunc handles the substitution of $randomPassword.* variables.
// It now accepts programmaticVars to allow charset overrides.
func _substituteRandomPasswordFunc(match string, programmaticVars map[string]any) string {
	parts := reRandomPassword.FindStringSubmatch(match) // Use the global regex
	length := defaultRandomPasswordLength
	if len(parts) >= 2 && parts[1] != "" { // parts[0] is full match, parts[1] is optional length
		parsedLen, err := strconv.Atoi(parts[1])
		if err != nil || parsedLen < 0 { // Allow 0 for empty string
			return match // Malformed length
		}
		length = parsedLen
	}

	if length == 0 {
		return ""
	}

	// Check for programmatic override for password character set
	if psVars, ok := programmaticVars["password"]; ok {
		if psMap, ok := psVars.(map[string]string); ok {
			if charset, ok := psMap["charset"]; ok && charset != "" {
				return randomStringFromCharset(length, charset)
			}
		}
	}

	// Default password generation logic (complex charset)
	// Ensure it's cryptographically secure if this is for actual passwords
	// For now, using the full charset as a default placeholder
	return randomStringFromCharset(length, charsetFull)
}
