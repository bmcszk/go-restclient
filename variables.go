package restclient

import (
	cryptorand "crypto/rand"
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
			return resolveVariablePlaceholder(match, variableResolverContext{
				clientProgrammaticVars:    clientProgrammaticVars,
				fileScopedVars:            fileScopedVars,
				environmentVars:           environmentVars,
				globalVars:                globalVars,
				requestScopedSystemVars:   requestScopedSystemVars,
				osEnvGetter:               osEnvGetter,
				dotEnvVars:                dotEnvVars,
			})
		}) // End of ReplaceAllStringFunc

		if currentText == previousText {
			break // No more substitutions made in this pass
		}
		// If we are on the last iteration and still making changes, it might be a circular dependency.
		// The currentText will be returned as is, potentially with unresolved variables.
	} // End of for loop
	return currentText
}

// variableResolverContext holds all the variable sources for resolution.
type variableResolverContext struct {
	clientProgrammaticVars  map[string]any
	fileScopedVars          map[string]string
	environmentVars         map[string]string
	globalVars              map[string]string
	requestScopedSystemVars map[string]string
	osEnvGetter             func(string) (string, bool)
	dotEnvVars              map[string]string
}

// resolveVariablePlaceholder resolves a single variable placeholder.
func resolveVariablePlaceholder(match string, ctx variableResolverContext) string {
	directive := strings.TrimSpace(match[2 : len(match)-2])
	varName, fallbackValue, hasFallback := parseVariableDirective(directive)

	// Handle system variables first
	if strings.HasPrefix(varName, "$") {
		return resolveSystemVariable(varName, match, ctx.requestScopedSystemVars)
	}

	// Resolve regular variables with precedence
	if resolved := resolveRegularVariable(varName, ctx); resolved != "" {
		return resolved
	}

	// Use fallback if available
	if hasFallback {
		return fallbackValue
	}

	// Default to empty string
	return ""
}

// parseVariableDirective parses a variable directive and extracts the variable name and fallback.
func parseVariableDirective(directive string) (varName, fallbackValue string, hasFallback bool) {
	if strings.Contains(directive, "|") {
		parts := strings.SplitN(directive, "|", 2)
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
	}
	return directive, "", false
}

// resolveSystemVariable handles system variables that start with $.
func resolveSystemVariable(varName, match string, requestScopedSystemVars map[string]string) string {
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
	return match // Preserve for substituteDynamicSystemVariables
}

// resolveRegularVariable resolves regular variables using the precedence order.
func resolveRegularVariable(varName string, ctx variableResolverContext) string {
	// Try high-priority sources first
	if resolved := resolveHighPriorityVariables(varName, ctx); resolved != "" {
		return resolved
	}

	// Try low-priority sources
	return resolveLowPriorityVariables(varName, ctx)
}

// resolveHighPriorityVariables resolves from programmatic and file-scoped variables.
func resolveHighPriorityVariables(varName string, ctx variableResolverContext) string {
	// 1. Client programmatic variables
	if resolved := resolveProgrammaticVariable(varName, ctx.clientProgrammaticVars); resolved != "" {
		return resolved
	}

	// 2. File-scoped variables
	if resolved := resolveFileScopedVariable(varName, ctx); resolved != "" {
		return resolved
	}

	return ""
}

// resolveLowPriorityVariables resolves from environment and system variables.
func resolveLowPriorityVariables(varName string, ctx variableResolverContext) string {
	// 3. Environment-specific variables
	if resolved := resolveFromMap(varName, ctx.environmentVars); resolved != "" {
		return resolved
	}

	// 4. Global variables
	if resolved := resolveFromMap(varName, ctx.globalVars); resolved != "" {
		return resolved
	}

	// 5. OS environment variables
	if ctx.osEnvGetter != nil {
		if val, ok := ctx.osEnvGetter(varName); ok {
			return val
		}
	}

	// 6. .env file variables
	return resolveFromMap(varName, ctx.dotEnvVars)
}

// resolveProgrammaticVariable resolves from client programmatic variables.
func resolveProgrammaticVariable(varName string, clientProgrammaticVars map[string]any) string {
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
	return ""
}

// resolveFileScopedVariable resolves from file-scoped variables with dynamic evaluation.
func resolveFileScopedVariable(varName string, ctx variableResolverContext) string {
	fileScopedVarNameToTry := "@" + varName
	val, ok := ctx.fileScopedVars[fileScopedVarNameToTry]
	if !ok {
		slog.Debug(
			"resolveVariablesInText: Not found in fileScopedVars",
			"varNameLookup", fileScopedVarNameToTry,
			"originalVarNameFromPlaceholder", varName)
		return ""
	}

	slog.Debug(
		"resolveVariablesInText: Found in fileScopedVars",
		"varNameLookup", fileScopedVarNameToTry, "value", val)

	// Check if the resolved file-scoped variable's value is itself a dynamic system variable placeholder
	if isDynamicSystemVariablePlaceholder(val, ctx.requestScopedSystemVars) {
		slog.Debug(
			"resolveVariablesInText: File-scoped var value is a dynamic system "+
				"variable placeholder. Evaluating and caching.",
			"varNameLookup", fileScopedVarNameToTry, "placeholderValue", val)
		// Pass clientProgrammaticVars and dotEnvVars to substituteDynamicSystemVariables
		evaluatedVal := substituteDynamicSystemVariables(val, ctx.dotEnvVars, ctx.clientProgrammaticVars)
		ctx.fileScopedVars[fileScopedVarNameToTry] = evaluatedVal // Cache the evaluated value
		slog.Debug(
			"resolveVariablesInText: Cached evaluated dynamic system variable "+
				"from file-scoped var",
			"varNameLookup", fileScopedVarNameToTry, "evaluatedValue", evaluatedVal)
		return evaluatedVal
	}
	return val // Return the original value if not a dynamic placeholder needing evaluation
}

// resolveFromMap resolves a variable from a simple string map.
func resolveFromMap(varName string, varMap map[string]string) string {
	if varMap != nil {
		if val, ok := varMap[varName]; ok {
			return val
		}
	}
	return ""
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
	if !isPlaceholderPattern(value) {
		return isDirectSystemVarKey(value, requestScopedSystemVars)
	}

	innerDirective := strings.TrimSpace(value[2 : len(value)-2])
	if !strings.HasPrefix(innerDirective, "$") {
		return false
	}

	if isPreEvaluatedSystemVar(innerDirective, requestScopedSystemVars, value) {
		return false
	}

	return matchesDynamicPattern(value)
}

// isPlaceholderPattern checks if value is in {{...}} format
func isPlaceholderPattern(value string) bool {
	return strings.HasPrefix(value, "{{") && strings.HasSuffix(value, "}}")
}

// isDirectSystemVarKey checks for direct system var keys like $uuid
func isDirectSystemVarKey(value string, requestScopedSystemVars map[string]string) bool {
	if strings.HasPrefix(value, "$") {
		if _, ok := requestScopedSystemVars[value]; ok {
			slog.Debug(
				"isDynamicSystemVariablePlaceholder: Direct value is a "+
					"pre-evaluated system variable key",
				"value", value)
			return false // It's a key like $uuid, not a placeholder like {{$uuid}}
		}
	}
	return false
}

// isPreEvaluatedSystemVar checks if directive is already pre-evaluated
func isPreEvaluatedSystemVar(innerDirective string, requestScopedSystemVars map[string]string, value string) bool {
	if _, ok := requestScopedSystemVars[innerDirective]; ok {
		slog.Debug(
			"isDynamicSystemVariablePlaceholder: Placeholder's inner directive "+
				"is a pre-evaluated system variable",
			"value", value, "innerDirective", innerDirective)
		return true
	}
	return false
}

// matchesDynamicPattern checks if value matches dynamic system variable patterns
func matchesDynamicPattern(value string) bool {
	dynamicRegexes := []*regexp.Regexp{
		reRandomInt, reRandomDotInteger, reRandomFloat, reRandomDotFloat,
		reRandomHex, reRandomDotHexadecimal, reRandomAlphaNumeric,
		reRandomDotAlphabetic,
		reRandomDotAlphanumeric, reRandomString, reRandomPassword,
		reDotEnv, reProcessEnv, reDateTime, reAadToken,
	}

	for _, re := range dynamicRegexes {
		if re.MatchString(value) {
			slog.Debug(
				"isDynamicSystemVariablePlaceholder: Placeholder matches a "+
					"dynamic system variable pattern",
				"value", value, "regex", re.String())
			return true
		}
	}
	slog.Debug(
		"isDynamicSystemVariablePlaceholder: Placeholder does not match any "+
			"known dynamic pattern or is not a dynamic system variable "+
			"requiring further evaluation",
		"value", value)
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
// variableMaps holds the different types of variable maps
type variableMaps struct {
	fileScopedVars     map[string]string
	envVarsFromFile    map[string]string
	globalVarsFromFile map[string]string
}

// It returns the final parsed URL or an error if substitution/parsing fails.
func substituteRequestVariables(
	rcRequest *Request,
	parsedFile *ParsedFile,
	requestScopedSystemVars map[string]string,
	osEnvGetter func(string) (string, bool),
	programmaticVars map[string]any,
	currentDotEnvVars map[string]string,
	clientBaseURL string,
) (*url.URL, error) {
	fileScopedVars, envVarsFromFile, globalVarsFromFile := initializeVariableMaps(parsedFile)
	mergeRequestActiveVariables(rcRequest, fileScopedVars)
	
	varMaps := variableMaps{
		fileScopedVars:     fileScopedVars,
		envVarsFromFile:    envVarsFromFile,
		globalVarsFromFile: globalVarsFromFile,
	}
	
	finalParsedURL, err := processURLSubstitution(rcRequest, varMaps,
		requestScopedSystemVars, osEnvGetter, programmaticVars, currentDotEnvVars, clientBaseURL)
	if err != nil {
		return nil, err
	}
	
	processHeaderSubstitution(rcRequest, varMaps,
		requestScopedSystemVars, osEnvGetter, programmaticVars, currentDotEnvVars)
	
	return finalParsedURL, nil
}

// initializeVariableMaps sets up the variable maps based on parsed file context
func initializeVariableMaps(parsedFile *ParsedFile) (fileScopedVars, envVarsFromFile, 
	globalVarsFromFile map[string]string) {
	if parsedFile != nil {
		fileScopedVars = make(map[string]string, len(parsedFile.FileVariables))
		for k, v := range parsedFile.FileVariables {
			fileScopedVars[k] = v
		}
		envVarsFromFile = parsedFile.EnvironmentVariables
		globalVarsFromFile = parsedFile.GlobalVariables
	} else {
		fileScopedVars = make(map[string]string)
		envVarsFromFile = make(map[string]string)
		globalVarsFromFile = make(map[string]string)
	}
	
	return fileScopedVars, envVarsFromFile, globalVarsFromFile
}

// mergeRequestActiveVariables merges request-specific variables into file-scoped vars
func mergeRequestActiveVariables(rcRequest *Request, fileScopedVars map[string]string) {
	if rcRequest != nil && rcRequest.ActiveVariables != nil {
		for k, v := range rcRequest.ActiveVariables {
			fileScopedVars[k] = v
		}
	}
}

// processURLSubstitution handles URL variable substitution and parsing
func processURLSubstitution(rcRequest *Request, varMaps variableMaps,
	requestScopedSystemVars map[string]string, osEnvGetter func(string) (string, bool),
	programmaticVars map[string]any, currentDotEnvVars map[string]string, clientBaseURL string) (*url.URL, error) {
	substitutedRawURL := resolveVariablesInText(
		rcRequest.RawURLString, programmaticVars, varMaps.fileScopedVars, varMaps.envVarsFromFile, 
		varMaps.globalVarsFromFile, requestScopedSystemVars, osEnvGetter, currentDotEnvVars)
	substitutedRawURL = substituteDynamicSystemVariables(substitutedRawURL, currentDotEnvVars, programmaticVars)

	if strings.TrimSpace(substitutedRawURL) == "" {
		return nil, fmt.Errorf("URL is empty after variable substitution (original: %s)", rcRequest.RawURLString)
	}

	substitutedRawURL = _applyBaseURLIfNeeded(substitutedRawURL, clientBaseURL)

	finalParsedURL, parseErr := url.Parse(substitutedRawURL)
	if parseErr != nil {
		return nil, fmt.Errorf(
			"failed to parse URL after variable substitution: %s (original: %s): %w",
			substitutedRawURL, rcRequest.RawURLString, parseErr)
	}
	
	return finalParsedURL, nil
}

// processHeaderSubstitution handles header variable substitution
func processHeaderSubstitution(rcRequest *Request, varMaps variableMaps,
	requestScopedSystemVars map[string]string, osEnvGetter func(string) (string, bool),
	programmaticVars map[string]any, currentDotEnvVars map[string]string) {
	if rcRequest.Headers == nil {
		return
	}
	
	for key, values := range rcRequest.Headers {
		newValues := make([]string, len(values))
		for j, val := range values {
			resolvedVal := resolveVariablesInText(val, programmaticVars, varMaps.fileScopedVars,
				varMaps.envVarsFromFile, varMaps.globalVarsFromFile, requestScopedSystemVars, 
				osEnvGetter, currentDotEnvVars)
			newValues[j] = substituteDynamicSystemVariables(resolvedVal, currentDotEnvVars, programmaticVars)
		}
		rcRequest.Headers[key] = newValues
	}
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
func _parseRangeInt(match string, re *regexp.Regexp, defaultMin, defaultMax int) (minVal, maxVal int, ok bool) {
	parts := re.FindStringSubmatch(match)
	if len(parts) == 3 && parts[1] != "" && parts[2] != "" {
		minVal, errMin := strconv.Atoi(parts[1])
		maxVal, errMax := strconv.Atoi(parts[2])
		if errMin != nil || errMax != nil || minVal > maxVal {
			return 0, 0, false // Invalid range
		}
		return minVal, maxVal, true
	}
	return defaultMin, defaultMax, true
}

// _parseRangeFloat extracts optional min and max float arguments.
func _parseRangeFloat(
	match string,
	re *regexp.Regexp,
	defaultMin, defaultMax float64,
) (minVal, maxVal float64, ok bool) {
	parts := re.FindStringSubmatch(match)
	if len(parts) == 3 && parts[1] != "" && parts[2] != "" {
		minVal, errMin := strconv.ParseFloat(parts[1], 64)
		maxVal, errMax := strconv.ParseFloat(parts[2], 64)
		if errMin != nil || errMax != nil || minVal > maxVal {
			return 0, 0, false // Invalid range
		}
		return minVal, maxVal, true
	}
	return defaultMin, defaultMax, true
}

// _substituteRandomIntFunc returns a function for ReplaceAllStringFunc to generate random integers.
func _substituteRandomIntFunc(re *regexp.Regexp, defaultMin, defaultMax int) func(string) string {
	return func(match string) string {
		minVal, maxVal, ok := _parseRangeInt(match, re, defaultMin, defaultMax)
		if !ok {
			return match // Malformed range
		}
		return strconv.Itoa(rand.Intn(maxVal-minVal+1) + minVal)
	}
}

// _substituteRandomFloatFunc returns a function for ReplaceAllStringFunc to generate random floats.
func _substituteRandomFloatFunc(re *regexp.Regexp, defaultMin, defaultMax float64) func(string) string {
	return func(match string) string {
		minVal, maxVal, ok := _parseRangeFloat(match, re, defaultMin, defaultMax)
		if !ok {
			return match // Malformed range
		}
		return fmt.Sprintf("%f", minVal+rand.Float64()*(maxVal-minVal))
	}
}

// _substituteRandomLengthCharsetFunc returns a function for ReplaceAllStringFunc to generate
// random strings from a charset.
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
		if !ok || length < 0 {
			return match
		}
		if length == 0 {
			return ""
		}
		return generateRandomHexString(length, match)
	}
}

// generateRandomHexString generates a hex string of the specified length
func generateRandomHexString(length int, fallbackMatch string) string {
	byteCount := length/2 + length%2
	b := make([]byte, byteCount)
	if _, err := cryptorand.Read(b); err != nil {
		slog.Error("Failed to generate random bytes for hex string", "error", err)
		return fallbackMatch
	}
	hexStr := fmt.Sprintf("%x", b)
	return hexStr[:length]
}

// _substituteDateTimeVariables handles the substitution of $datetime and $localDatetime variables.
func _substituteDateTimeVariables(text string) string {
	reDateTimeRelated := regexp.MustCompile(`{{\$(datetime|localDatetime)((?:\s*(?:\"[^\"]*\"|[^\"\s}]+))*)\s*}}`)
	return reDateTimeRelated.ReplaceAllStringFunc(text, processDateTimeMatch)
}

// processDateTimeMatch processes a single datetime variable match
func processDateTimeMatch(match string) string {
	reDateTimeRelated := regexp.MustCompile(`{{\$(datetime|localDatetime)((?:\s*(?:\"[^\"]*\"|[^\"\s}]+))*)\s*}}`)
	captures := reDateTimeRelated.FindStringSubmatch(match)
	if len(captures) < 3 {
		slog.Warn("Could not parse datetime/localDatetime variable, captures unexpected",
			"match", match, "capturesCount", len(captures))
		return match
	}

	varType := captures[1]
	argsStr := strings.TrimSpace(captures[2])
	formatStr := extractFormatString(argsStr)
	now := getTimeForType(varType)

	return formatTimeString(now, formatStr, match)
}

// extractFormatString extracts the format string from datetime arguments
func extractFormatString(argsStr string) string {
	argPartsRegex := regexp.MustCompile(`(?:\"([^\"]*)\"|([^\"\s}]+))`)
	parsedArgsMatches := argPartsRegex.FindAllStringSubmatch(argsStr, -1)

	for _, m := range parsedArgsMatches {
		if m[1] != "" {
			return m[1] // Quoted argument
		}
		if m[2] != "" {
			return m[2] // Unquoted argument
		}
	}
	return "iso8601" // Default format
}

// getTimeForType returns the appropriate time based on the variable type
func getTimeForType(varType string) time.Time {
	if varType == "datetime" {
		return time.Now().UTC()
	}
	return time.Now() // localDatetime
}

// formatTimeString formats the time according to the specified format
func formatTimeString(now time.Time, formatStr, originalMatch string) string {
	switch strings.ToLower(formatStr) {
	case "rfc1123":
		return now.Format(time.RFC1123)
	case "iso8601":
		return now.Format(time.RFC3339)
	case "timestamp":
		return strconv.FormatInt(now.Unix(), 10)
	default:
		return originalMatch // Unsupported format
	}
}

// substituteDynamicSystemVariables handles system variables that require argument
// parsing or dynamic evaluation at substitution time.
// These are typically {{$processEnv VAR}}, {{$dotenv VAR}}, and {{$randomInt MIN MAX}}.
// This also handles JetBrains HTTP client syntax variables like {{$random.integer MIN MAX}},
// {{$random.alphabetic LENGTH}}, etc.
// Other simple system variables like {{$uuid}} or {{$timestamp}}
// should have been pre-resolved and substituted by resolveVariablesInText via the
// requestScopedSystemVars map.
func substituteDynamicSystemVariables(
	text string,
	activeDotEnvVars map[string]string,
	programmaticVars map[string]any,
) string {
	text = substituteRandomVariables(text, programmaticVars)
	text = substituteSystemEnvVariables(text)
	text = substituteDotEnvVariables(text, activeDotEnvVars)
	text = substituteProcessEnvVariables(text)
	text = _substituteDateTimeVariables(text)
	return text
}

// substituteSystemEnvVariables handles {{$env.VAR_NAME}} placeholders
func substituteSystemEnvVariables(text string) string {
	reSystemEnvVar := regexp.MustCompile(`{{\$env\.([A-Za-z_][A-Za-z0-9_]*?)}}`)
	return reSystemEnvVar.ReplaceAllStringFunc(text, func(match string) string {
		parts := reSystemEnvVar.FindStringSubmatch(match)
		if len(parts) == 2 {
			return os.Getenv(parts[1])
		}
		slog.Warn("Failed to parse $env.VAR_NAME, returning original match", "match", match, "parts_len", len(parts))
		return match
	})
}

// substituteDotEnvVariables handles {{$dotenv VAR}} placeholders
func substituteDotEnvVariables(text string, activeDotEnvVars map[string]string) string {
	text = reDotEnv.ReplaceAllStringFunc(text, dotEnvReplacer(activeDotEnvVars))
	text = substituteDotEnvEncoded(text, activeDotEnvVars)
	return text
}

// dotEnvReplacer returns a replacement function for dotenv variables
func dotEnvReplacer(activeDotEnvVars map[string]string) func(string) string {
	return func(match string) string {
		parts := reDotEnv.FindStringSubmatch(match)
		if len(parts) == 2 {
			varName := parts[1]
			if val, ok := activeDotEnvVars[varName]; ok {
				return val
			}
			return ""
		}
		slog.Warn("Failed to parse $dotenv, returning original match", "match", match, "parts_len", len(parts))
		return match
	}
}

// substituteDotEnvEncoded handles URL-encoded dotenv variables
func substituteDotEnvEncoded(text string, activeDotEnvVars map[string]string) string {
	reDotEnvEncoded := regexp.MustCompile(`%7B%7B\$dotenv\s+([a-zA-Z_][a-zA-Z0-9_]*)%7D%7D`)
	return reDotEnvEncoded.ReplaceAllStringFunc(text, func(match string) string {
		parts := reDotEnvEncoded.FindStringSubmatch(match)
		if len(parts) == 2 {
			varName := parts[1]
			if val, ok := activeDotEnvVars[varName]; ok {
				return val
			}
			return ""
		}
		slog.Warn("Failed to parse URL-encoded $dotenv, returning original match",
			"match", match, "parts_len", len(parts))
		return match
	})
}

// substituteProcessEnvVariables handles {{$processEnv VAR}} placeholders
func substituteProcessEnvVariables(text string) string {
	text = reProcessEnv.ReplaceAllStringFunc(text, processEnvReplacer())
	text = substituteProcessEnvEncoded(text)
	return text
}

// processEnvReplacer returns a replacement function for process env variables
func processEnvReplacer() func(string) string {
	return func(match string) string {
		parts := reProcessEnv.FindStringSubmatch(match)
		if len(parts) == 2 {
			varName := parts[1]
			if val, ok := os.LookupEnv(varName); ok {
				return val
			}
			return match
		}
		slog.Warn("Failed to parse $processEnv, returning original match", "match", match, "parts_len", len(parts))
		return match
	}
}

// substituteProcessEnvEncoded handles URL-encoded process env variables
func substituteProcessEnvEncoded(text string) string {
	reProcessEnvEncoded := regexp.MustCompile(`%7B%7B\$processEnv\s+([A-Za-z_][A-Za-z0-9_]*)%7D%7D`)
	return reProcessEnvEncoded.ReplaceAllStringFunc(text, func(match string) string {
		parts := reProcessEnvEncoded.FindStringSubmatch(match)
		if len(parts) == 2 {
			varName := parts[1]
			if val, ok := os.LookupEnv(varName); ok {
				return val
			}
			return match
		}
		slog.Warn("Failed to parse URL-encoded $processEnv, returning original match",
			"match", match, "parts_len", len(parts))
		return match
	})
}

// substituteRandomVariables handles the substitution of $random.* variables.
func substituteRandomVariables(text string, programmaticVars map[string]any) string {
	// Integer types
	text = reRandomInt.ReplaceAllStringFunc(text,
		_substituteRandomIntFunc(reRandomInt, defaultRandomMinInt, defaultRandomMaxInt))
	text = reRandomDotInteger.ReplaceAllStringFunc(text,
		_substituteRandomIntFunc(reRandomDotInteger, defaultRandomMinInt, defaultRandomMaxInt))

	// Float types
	text = reRandomFloat.ReplaceAllStringFunc(text,
		_substituteRandomFloatFunc(reRandomFloat, defaultRandomMinFloat, defaultRandomMaxFloat))
	text = reRandomDotFloat.ReplaceAllStringFunc(text,
		_substituteRandomFloatFunc(reRandomDotFloat, defaultRandomMinFloat, defaultRandomMaxFloat))

	// Boolean
	text = strings.ReplaceAll(text, "{{$randomBoolean}}", strconv.FormatBool(rand.Intn(2) == 0))

	// Hexadecimal
	text = reRandomHex.ReplaceAllStringFunc(text, _substituteRandomHexHelper(reRandomHex, defaultRandomHexLength))
	text = reRandomDotHexadecimal.ReplaceAllStringFunc(text,
		_substituteRandomHexHelper(reRandomDotHexadecimal, defaultRandomHexLength))

	// Alphabetic / Alphanumeric
	text = reRandomDotAlphabetic.ReplaceAllStringFunc(text,
		_substituteRandomLengthCharsetFunc(reRandomDotAlphabetic, charsetAlphabetic))
	// Uses underscore
	text = reRandomAlphaNumeric.ReplaceAllStringFunc(text,
		_substituteRandomLengthCharsetFunc(reRandomAlphaNumeric, charsetAlphaNumericWithExtra))
	// No underscore
	text = reRandomDotAlphanumeric.ReplaceAllStringFunc(text,
		_substituteRandomLengthCharsetFunc(reRandomDotAlphanumeric, charsetAlphaNumeric))

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
	length := parsePasswordLength(match)
	if length < 0 {
		return match // Malformed length
	}
	if length == 0 {
		return ""
	}

	charset := getPasswordCharset(programmaticVars)
	return randomStringFromCharset(length, charset)
}

// parsePasswordLength extracts and validates the length parameter from a password match
func parsePasswordLength(match string) int {
	parts := reRandomPassword.FindStringSubmatch(match)
	length := defaultRandomPasswordLength
	if len(parts) >= 2 && parts[1] != "" {
		parsedLen, err := strconv.Atoi(parts[1])
		if err != nil || parsedLen < 0 {
			return -1 // Invalid length
		}
		length = parsedLen
	}
	return length
}

// getPasswordCharset determines the charset to use for password generation
func getPasswordCharset(programmaticVars map[string]any) string {
	if psVars, ok := programmaticVars["password"]; ok {
		if psMap, ok := psVars.(map[string]string); ok {
			if charset, ok := psMap["charset"]; ok && charset != "" {
				return charset
			}
		}
	}
	return charsetFull
}
