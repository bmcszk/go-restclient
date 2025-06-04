package restclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	// "regexp" // Unused

	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/joho/godotenv"
)

// Client is the main struct for interacting with the REST client library.
// It holds configuration like the HTTP client, base URL, default headers,
// and programmatic variables for substitution.
type Client struct {
	httpClient              *http.Client
	BaseURL                 string
	DefaultHeaders          http.Header
	currentDotEnvVars       map[string]string
	programmaticVars        map[string]interface{}
	selectedEnvironmentName string // Added for T4
}

// NewClient creates a new instance of the REST client.
// Options for customization (e.g., timeout, custom transport) can be added later.
func NewClient(options ...ClientOption) (*Client, error) {
	c := &Client{
		httpClient:     &http.Client{},
		DefaultHeaders: make(http.Header),
	}

	for _, option := range options {
		err := option(c)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

// ClientOption is a functional option for configuring the Client.
type ClientOption func(*Client) error

// WithHTTPClient allows providing a custom http.Client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) error {
		if hc == nil {
			c.httpClient = &http.Client{}
		} else {
			c.httpClient = hc
		}
		return nil
	}
}

// WithBaseURL sets a base URL for the client.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) error {
		c.BaseURL = baseURL
		return nil
	}
}

// WithDefaultHeader adds a default header to be sent with every request.
func WithDefaultHeader(key, value string) ClientOption {
	return func(c *Client) error {
		c.DefaultHeaders.Add(key, value)
		return nil
	}
}

// WithDefaultHeaders adds multiple default headers.
func WithDefaultHeaders(headers http.Header) ClientOption {
	return func(c *Client) error {
		for key, values := range headers {
			for _, value := range values {
				c.DefaultHeaders.Add(key, value)
			}
		}
		return nil
	}
}

// WithVars sets programmatic variables for the client instance.
// These variables can be used in .http and .hresp files.
// Programmatic variables have the highest precedence during substitution,
// overriding file-defined variables, environment variables, and .env variables.
// If called multiple times, the provided vars are merged with existing ones,
// with new values for existing keys overwriting old ones.
func WithVars(vars map[string]interface{}) ClientOption {
	return func(c *Client) error {
		if c.programmaticVars == nil {
			c.programmaticVars = make(map[string]interface{})
		}
		for k, v := range vars {
			c.programmaticVars[k] = v
		}
		return nil
	}
}

// WithEnvironment sets the name of the environment to be used from http-client.env.json.
func WithEnvironment(name string) ClientOption {
	return func(c *Client) error {
		c.selectedEnvironmentName = name
		return nil
	}
}

// ExecuteFile parses a request file (.http, .rest), executes all requests found, and returns their responses.
// It returns an error if the file cannot be parsed or no requests are found.
// Individual request execution errors are stored within each Response object.
//
// Variable Substitution Workflow:
// 1. File Parsing (`parseRequestFile`):
//   - Loads .env file from the request file's directory.
//   - Generates request-scoped system variables (e.g., `{{$uuid}}`) once for the entire file parsing pass.
//   - Resolves `@variable = value` definitions. The `value` itself can contain placeholders,
//     which are resolved using: Client programmatic vars > request-scoped system vars > OS env vars > .env vars.
//
// 2. Request Execution (within `ExecuteFile` loop for each request):
//   - Re-generates request-scoped system variables (e.g., `{{$uuid}}`) *once per individual request* to ensure
//     uniqueness if needed across multiple requests in the same file, but consistency within a single request.
//   - For each part of the request (URL, headers, body):
//     a. `resolveVariablesInText` is called. For {{variableName}} placeholders (where 'variableName' does not start with '$'),
//     the precedence is: Client programmatic vars > file-scoped `@vars` (rcRequest.ActiveVariables) >
//     Environment vars (parsedFile.EnvironmentVariables) > Global vars (parsedFile.GlobalVariables) >
//     OS env vars > .env vars > fallback.
//     System variables like {{$uuid}} are resolved from the request-scoped map if the placeholder is {{$systemVarName}}.
//     It resolves simple system variables like `{{$uuid}}` from the request-scoped map.
//     It leaves dynamic system variables (e.g., `{{$dotenv NAME}}`) untouched for the next step.
//     b. `substituteDynamicSystemVariables` is called: This handles system variables requiring arguments
//     (e.g., `{{$dotenv NAME}}`, `{{$processEnv NAME}}`, `{{$randomInt MIN MAX}}`).
//
// Programmatic variables for substitution can be set on the Client using `WithVars()`.
func (c *Client) ExecuteFile(ctx context.Context, requestFilePath string) ([]*Response, error) {
	parsedFile, err := parseRequestFile(requestFilePath, c)
	if err != nil {
		return nil, fmt.Errorf("failed to parse request file %s: %w", requestFilePath, err)
	}
	if len(parsedFile.Requests) == 0 {
		return nil, fmt.Errorf("no requests found in file %s", requestFilePath)
	}

	c.currentDotEnvVars = make(map[string]string)
	envFilePath := filepath.Join(filepath.Dir(requestFilePath), ".env")
	if _, err := os.Stat(envFilePath); err == nil {
		loadedVars, loadErr := godotenv.Read(envFilePath)
		if loadErr == nil {
			c.currentDotEnvVars = loadedVars
		}
	}

	responses := make([]*Response, len(parsedFile.Requests))
	var multiErr *multierror.Error

	osEnvGetter := func(key string) (string, bool) {
		return os.LookupEnv(key)
	}

	for i, restClientReq := range parsedFile.Requests {
		requestScopedSystemVars := c.generateRequestScopedSystemVariables()

		substitutedRawURL := c.resolveVariablesInText(
			restClientReq.RawURLString,
			c.programmaticVars,
			restClientReq.ActiveVariables,
			parsedFile.EnvironmentVariables, // Added for T3
			parsedFile.GlobalVariables,      // Added for T3
			requestScopedSystemVars,
			osEnvGetter,
			c.currentDotEnvVars,
		)
		substitutedRawURL = c.substituteDynamicSystemVariables(substitutedRawURL, c.currentDotEnvVars)

		finalParsedURL, parseErr := url.Parse(substitutedRawURL)

		if parseErr != nil {
			resp := &Response{Request: restClientReq}
			resp.Error = fmt.Errorf("failed to parse URL after variable substitution: %s (original: %s): %w", substitutedRawURL, restClientReq.RawURLString, parseErr)
			wrappedErr := fmt.Errorf("request %d (%s %s) failed URL parsing: %w", i+1, restClientReq.Method, restClientReq.RawURLString, resp.Error)
			multiErr = multierror.Append(multiErr, wrappedErr)
			responses[i] = resp
			continue
		}
		restClientReq.URL = finalParsedURL

		// "[DEBUG_EXECUTEFILE_HEADERS_AFTER_PARSE]", "filePath", requestFilePath, "reqName", restClientReq.Name, "initialHeaders", restClientReq.Headers)

		if restClientReq.Headers != nil {
			for key, values := range restClientReq.Headers {
				newValues := make([]string, len(values))
				for j, val := range values {
					resolvedVal := c.resolveVariablesInText(
						val,
						c.programmaticVars,
						restClientReq.ActiveVariables,
						parsedFile.EnvironmentVariables, // Added for T3
						parsedFile.GlobalVariables,      // Added for T3
						requestScopedSystemVars,
						osEnvGetter,
						c.currentDotEnvVars,
					)
					newValues[j] = c.substituteDynamicSystemVariables(resolvedVal, c.currentDotEnvVars)
					// "[DEBUG_HEADER_FINAL]", "key", key, "index", j, "finalValue", newValues[j]) // Added for debugging header values
				}
				restClientReq.Headers[key] = newValues
			}
		}

		// "[DEBUG_EXECUTEFILE_HEADERS_BEFORE_EXECREQ]", "filePath", requestFilePath, "reqName", restClientReq.Name, "headers", restClientReq.Headers)

		if restClientReq.RawBody != "" {
			resolvedBody := c.resolveVariablesInText(
				restClientReq.RawBody,
				c.programmaticVars,
				restClientReq.ActiveVariables,
				parsedFile.EnvironmentVariables, // Added for T3
				parsedFile.GlobalVariables,      // Added for T3
				requestScopedSystemVars,
				osEnvGetter,
				c.currentDotEnvVars,
			)
			finalBody := c.substituteDynamicSystemVariables(resolvedBody, c.currentDotEnvVars)
			// finalBody is now correctly substituted
			restClientReq.RawBody = finalBody // Update RawBody with the substituted content
			restClientReq.Body = strings.NewReader(finalBody)
			restClientReq.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(strings.NewReader(finalBody)), nil
			}
		}

		resp, execErr := c.executeRequest(ctx, restClientReq)
		if execErr != nil {
			if resp == nil {
				resp = &Response{Request: restClientReq}
			}
			currentErr := resp.Error
			if currentErr == nil {
				resp.Error = execErr
			} else {
				resp.Error = fmt.Errorf("execution error: %w (prior error: %s)", execErr, currentErr)
			}
			urlForError := restClientReq.RawURLString
			if restClientReq.URL != nil {
				urlForError = restClientReq.URL.String()
			}
			wrappedExecErr := fmt.Errorf("request %d (%s %s) failed with critical error: %w", i+1, restClientReq.Method, urlForError, execErr)
			multiErr = multierror.Append(multiErr, wrappedExecErr)
		} else if resp != nil && resp.Error != nil {
			urlForError := restClientReq.RawURLString
			if restClientReq.URL != nil {
				urlForError = restClientReq.URL.String()
			}
			wrappedRespErr := fmt.Errorf("request %d (%s %s) processing resulted in error: %w", i+1, restClientReq.Method, urlForError, resp.Error)
			multiErr = multierror.Append(multiErr, wrappedRespErr)
		}
		responses[i] = resp
	}

	return responses, multiErr.ErrorOrNil()
}

// resolveVariablesInText is the primary substitution engine for non-system variables and request-scoped system variables.
// It iterates through placeholders like `{{varName | fallback}}` and resolves them based on a defined precedence.
// Dynamic system variables (like {{$dotenv NAME}}) are left untouched by this function for substituteDynamicSystemVariables.
//
// Precedence for {{variableName}} placeholders (where 'variableName' does not start with '$'):
// 1. Client programmatic variables (clientProgrammaticVars)
// 2. Request file-defined variables (fileScopedVars, from @name=value, effectively rcRequest.ActiveVariables)
// 3. Environment variables (environmentVars, from selected environment like http-client.env.json)
// 4. Global variables (globalVars, from http-client.private.env.json or similar)
// 5. OS Environment variables (via osEnvGetter)
// 6. Variables from .env file (dotEnvVars)
// 7. Fallback value provided in the placeholder itself.
// System variables (e.g., {{$uuid}}, {{$timestamp}}) are handled if the placeholder is like {{$systemVarName}} (i.e. varName starts with '$').
func (c *Client) resolveVariablesInText(text string, clientProgrammaticVars map[string]interface{}, fileScopedVars map[string]string, environmentVars map[string]string, globalVars map[string]string, requestScopedSystemVars map[string]string, osEnvGetter func(string) (string, bool), dotEnvVars map[string]string) string {
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
				if requestScopedSystemVars != nil {
					if val, ok := requestScopedSystemVars[varName]; ok {
						return val
					}
				}
				// or an unknown one. These are left for substituteDynamicSystemVariables.
				// So, we return the original 'match' here to preserve the placeholder for the next stage.
				return match
			}

			// 2. Client Programmatic Variables (map[string]interface{})
			if clientProgrammaticVars != nil {
				if val, ok := clientProgrammaticVars[varName]; ok {
					return fmt.Sprintf("%v", val)
				}
			}

			// 3. File-scoped Variables (map[string]string, from @name=value or rcRequest.ActiveVariables)
			if fileScopedVars != nil {
				if val, ok := fileScopedVars[varName]; ok {
					return val
				}
			}

			// 4. Environment Variables (from http-client.env.json or selected environment)
			if environmentVars != nil {
				if val, ok := environmentVars[varName]; ok {
					return val
				}
			}

			// 5. Global Variables (from http-client.private.env.json or similar)
			if globalVars != nil {
				if val, ok := globalVars[varName]; ok {
					return val
				}
			}

			// 6. OS Environment Variables
			if osEnvGetter != nil {
				if envVal, ok := osEnvGetter(varName); ok {
					return envVal
				}
			}

			// 7. .env file variables
			if dotEnvVars != nil {
				if val, ok := dotEnvVars[varName]; ok {
					return val
				}
				// If not in dotEnvVars, continue to fallback or return match
			}

			// 8. Fallback Value
			// Must be checked AFTER all other potential sources for varName.
			if hasFallback {
				return fallbackValue
			}

			return match // Return original placeholder if not found and no fallback
		}) // End of ReplaceAllStringFunc

		if currentText == previousText {
			break // No more substitutions made in this pass
		}
		// If we are on the last iteration and still making changes, it might be a circular dependency.
		// The currentText will be returned as is, potentially with unresolved variables.
	} // End of for loop
	return currentText
} // End of function resolveVariablesInText

// randomStringFromCharset generates a random string of a given length using characters from the provided charset.
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

// substituteDynamicSystemVariables handles system variables that require argument parsing or dynamic evaluation at substitution time.
// These are typically {{$processEnv VAR}}, {{$dotenv VAR}}, and {{$randomInt MIN MAX}}.
// Other simple system variables like {{$uuid}} or {{$timestamp}}
// should have been pre-resolved and substituted by resolveVariablesInText via the
// requestScopedSystemVars map.
func (c *Client) substituteDynamicSystemVariables(text string, activeDotEnvVars map[string]string) string {
	// "[DEBUG_DYN_VARS_INPUT]", "inputText", text)
	originalTextForLogging := text // Keep a copy for logging. Used if we add more complex types with logging.
	_ = originalTextForLogging     // Avoid unused variable error if no logging exists below.

	// Handle {{$randomInt min max}}
	reRandomIntWithArgs := regexp.MustCompile(`{{\$randomInt\s+(-?\d+)\s+(-?\d+)}}`)
	text = reRandomIntWithArgs.ReplaceAllStringFunc(text, func(match string) string {
		parts := reRandomIntWithArgs.FindStringSubmatch(match)
		if len(parts) == 3 {
			min, errMin := strconv.Atoi(parts[1])
			max, errMax := strconv.Atoi(parts[2])
			if errMin == nil && errMax == nil && min <= max {
				return strconv.Itoa(rand.Intn(max-min+1) + min)
			}
		}
		return match // Return original match if parsing fails or min > max
	})

	// Handle {{$random.integer min max}}
	reRandomDotInteger := regexp.MustCompile(`{{\$random\.integer\s+(-?\d+)\s+(-?\d+)}}`)
	text = reRandomDotInteger.ReplaceAllStringFunc(text, func(match string) string {
		parts := reRandomDotInteger.FindStringSubmatch(match)
		if len(parts) == 3 {
			min, errMin := strconv.Atoi(parts[1])
			max, errMax := strconv.Atoi(parts[2])
			if errMin == nil && errMax == nil && min <= max {
				return strconv.Itoa(rand.Intn(max-min+1) + min)
			}
		}
		slog.Warn("Failed to parse $random.integer, returning original match", "match", match)
		return match
	})

	// Handle {{$random.float min max}}
	reRandomDotFloat := regexp.MustCompile(`{{\$random\.float\s+(-?\d*\.?\d+)\s+(-?\d*\.?\d+)}}`)
	text = reRandomDotFloat.ReplaceAllStringFunc(text, func(match string) string {
		parts := reRandomDotFloat.FindStringSubmatch(match)
		if len(parts) == 3 {
			min, errMin := strconv.ParseFloat(parts[1], 64)
			max, errMax := strconv.ParseFloat(parts[2], 64)
			if errMin == nil && errMax == nil && min <= max {
				return strconv.FormatFloat(min+rand.Float64()*(max-min), 'f', -1, 64)
			}
		}
		slog.Warn("Failed to parse $random.float, returning original match", "match", match)
		return match
	})

	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const alphanumeric = letters + "0123456789_"
	const hexChars = "0123456789abcdef"

	// Handle {{$random.alphabetic length}}
	reRandomDotAlphabetic := regexp.MustCompile(`{{\$random\.alphabetic\s+(\d+)}}`)
	text = reRandomDotAlphabetic.ReplaceAllStringFunc(text, func(match string) string {
		parts := reRandomDotAlphabetic.FindStringSubmatch(match)
		if len(parts) == 2 {
			length, err := strconv.Atoi(parts[1])
			if err == nil && length >= 0 {
				return randomStringFromCharset(length, letters)
			}
		}
		slog.Warn("Failed to parse $random.alphabetic, returning original match", "match", match)
		return match
	})

	// Handle {{$random.alphanumeric length}}
	reRandomDotAlphanumeric := regexp.MustCompile(`{{\$random\.alphanumeric\s+(\d+)}}`)
	text = reRandomDotAlphanumeric.ReplaceAllStringFunc(text, func(match string) string {
		parts := reRandomDotAlphanumeric.FindStringSubmatch(match)
		if len(parts) == 2 {
			length, err := strconv.Atoi(parts[1])
			if err == nil && length >= 0 {
				return randomStringFromCharset(length, alphanumeric)
			}
		}
		slog.Warn("Failed to parse $random.alphanumeric, returning original match", "match", match)
		return match
	})

	// Handle {{$random.hexadecimal length}}
	reRandomDotHexadecimal := regexp.MustCompile(`{{\$random\.hexadecimal\s+(\d+)}}`)
	text = reRandomDotHexadecimal.ReplaceAllStringFunc(text, func(match string) string {
		parts := reRandomDotHexadecimal.FindStringSubmatch(match)
		if len(parts) == 2 {
			length, err := strconv.Atoi(parts[1])
			if err == nil && length >= 0 {
				return randomStringFromCharset(length, hexChars)
			}
		}
		slog.Warn("Failed to parse $random.hexadecimal, returning original match", "match", match)
		return match
	})

	// Handle {{$random.email}}
	reRandomDotEmail := regexp.MustCompile(`{{\$random\.email}}`)
	text = reRandomDotEmail.ReplaceAllStringFunc(text, func(match string) string {
		// Simple email pattern: user@domain.tld
		user := randomStringFromCharset(rand.Intn(10)+5, alphanumeric) // 5-14 chars for user
		domain := randomStringFromCharset(rand.Intn(5)+5, letters)     // 5-9 chars for domain
		tld := randomStringFromCharset(rand.Intn(2)+2, letters)        // 2-3 chars for tld
		return fmt.Sprintf("%s@%s.%s", user, domain, tld)
	})

	// Handle {{$env.VAR_NAME}}
	reSystemEnvVar := regexp.MustCompile(`{{\$env\.([A-Za-z_][A-Za-z0-9_]*?)}}`)
	text = reSystemEnvVar.ReplaceAllStringFunc(text, func(match string) string {
		parts := reSystemEnvVar.FindStringSubmatch(match)
		if len(parts) == 2 {
			varName := parts[1]
			// os.Getenv returns empty string if var is not set, which is desired.
			return os.Getenv(varName)
		}
		slog.Warn("Failed to parse $env.VAR_NAME, returning original match", "match", match)
		return match
	})

	// NOTE: {{$guid}}, {{$uuid}}, {{$timestamp}}, {{$randomInt (no-args)}}
	// are EXCLUDED here as they are now handled by generateRequestScopedSystemVariables
	// and substituted in resolveVariablesInText.

	// REQ-LIB-011: $dotenv MY_VARIABLE_NAME
	reDotEnv := regexp.MustCompile(`{{\$dotenv\s+([a-zA-Z_][a-zA-Z0-9_]*)}}`)
	text = reDotEnv.ReplaceAllStringFunc(text, func(match string) string {
		parts := reDotEnv.FindStringSubmatch(match)
		if len(parts) == 2 {
			varName := parts[1]
			if val, ok := activeDotEnvVars[varName]; ok {
				return val
			}
			return "" // Variable not found in .env, return empty string
		}
		return match // Should not happen with a valid regex, but good for safety
	})

	// REQ-LIB-012: $processEnv MY_ENV_VAR
	reProcessEnv := regexp.MustCompile(`{{\$processEnv\s+([a-zA-Z_][a-zA-Z0-9_]*)}}`)
	text = reProcessEnv.ReplaceAllStringFunc(text, func(match string) string {
		parts := reProcessEnv.FindStringSubmatch(match)
		if len(parts) == 2 {
			varName := parts[1]
			if val, ok := os.LookupEnv(varName); ok {
				return val
			}
			return match // Variable not found in process env, return original placeholder (PRD)
		}
		return match // Should not happen
	})

	// REQ-LIB-029 & REQ-LIB-030: Datetime variables ($datetime, $localDatetime)
	reDateTimeRelated := regexp.MustCompile(`{{\$(datetime|localDatetime)((?:\s*(?:"[^"]*"|[^"\s}]+))*)\s*}}`)
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
		argPartsRegex := regexp.MustCompile(`(?:\"([^\"]*)\"|([^\"\\s}]+))`)
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

// generateRequestScopedSystemVariables creates a map of system variables that are generated once per request.
// This ensures that if, for example, {{$uuid}} is used multiple times within the same request
// (e.g., in the URL and a header), it resolves to the same value for that specific request.
func (c *Client) generateRequestScopedSystemVariables() map[string]string {
	vars := make(map[string]string)
	vars["$uuid"] = uuid.NewString()
	vars["$guid"] = vars["$uuid"]        // Alias $guid to $uuid
	vars["$random.uuid"] = vars["$uuid"] // Add $random.uuid as alias
	vars["$timestamp"] = strconv.FormatInt(time.Now().UTC().Unix(), 10)
	vars["$isoTimestamp"] = time.Now().UTC().Format(time.RFC3339) // Add $isoTimestamp
	vars["$randomInt"] = strconv.Itoa(rand.Intn(1001))            // 0-1000 inclusive as per PRD
	// Add other simple, no-argument system variables here if any

	// For logging/debugging purposes, to see what was generated once per request
	// fmt.Printf("[DEBUG] Generated request-scoped system variables: %v\n", vars)
	return vars
}

// executeRequest sends a given Request and returns the Response.
// Errors during execution (e.g. network, body read) are captured in Response.Error.
// A non-nil error is returned by this function only for critical pre-execution failures (e.g. nil request, bad BaseURL).
func (c *Client) executeRequest(ctx context.Context, rcRequest *Request) (*Response, error) {
	if rcRequest == nil {
		// For a nil request, we can't even populate a Response struct meaningfully.
		return nil, fmt.Errorf("cannot execute a nil request")
	}

	// Initialize a response object upfront to hold results or errors
	clientResponse := &Response{
		Request: rcRequest, // Link the request early
	}

	urlToUse := rcRequest.URL
	if !urlToUse.IsAbs() && c.BaseURL != "" {
		base, err := url.Parse(c.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("invalid BaseURL %s: %w", c.BaseURL, err)
		}
		if rcRequest.URL.Scheme == "" && rcRequest.URL.Host == "" {
			if base.Path != "" && !strings.HasSuffix(base.Path, "/") && !strings.HasPrefix(rcRequest.URL.Path, "/") {
				base.Path += "/"
			}
			urlToUse = base.ResolveReference(rcRequest.URL)
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, rcRequest.Method, urlToUse.String(), rcRequest.Body)
	if err != nil {
		clientResponse.Error = fmt.Errorf("failed to create http request: %w", err)
		return clientResponse, nil
	}

	for key, values := range c.DefaultHeaders {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}
	for key, values := range rcRequest.Headers {
		// "[DEBUG_HEADER_LOOP_ITERATION]", "original_key", key, "canonicalKey", textproto.CanonicalMIMEHeaderKey(key), "values_from_rcRequest", values)
		// "[DEBUG_HEADER_STATE_BEFORE_DEL]", "canonicalKey", canonicalKey, "slice_in_httpReq", httpReq.Header[canonicalKey])

		httpReq.Header.Del(key) // Uses canonicalKey internally
		// "[DEBUG_HEADER_STATE_AFTER_DEL]", "canonicalKey", textproto.CanonicalMIMEHeaderKey(key), "slice_in_httpReq", httpReq.Header[textproto.CanonicalMIMEHeaderKey(key)])

		for _, value := range values {
			httpReq.Header.Add(key, value) // Uses canonicalKey internally
			// "[DEBUG_HEADER_STATE_AFTER_ADD]", "canonicalKey", canonicalKey, "slice_in_httpReq", httpReq.Header[canonicalKey], "value_idx", i)
		}
		// "[DEBUG_HEADER_STATE_AFTER_ALL_ADDS_FOR_KEY]", "canonicalKey", canonicalKey, "slice_in_httpReq", httpReq.Header[canonicalKey])
	}

	// "[DEBUG_SPECIFIC_HEADER_AFTER_LOOP]",
	// 	"key_original", "X-Datetime-RFC1123",
	// 	"value_get", httpReq.Header.Get("X-Datetime-RFC1123"),
	// 	"key_canonical", "X-Datetime-Rfc1123",
	// 	"value_slice_direct", httpReq.Header["X-Datetime-Rfc1123"],
	// )

	if httpReq.Header.Get("Host") == "" && httpReq.URL.Host != "" {
		httpReq.Host = httpReq.URL.Host
	}

	// "[DEBUG_HTTPREQ_FINAL_HEADERS]", "headers", httpReq.Header) // Added for debugging
	startTime := time.Now()
	httpResp, err := c.httpClient.Do(httpReq)
	duration := time.Since(startTime)
	clientResponse.Duration = duration // Set duration regardless of http error

	if err != nil {
		clientResponse.Error = fmt.Errorf("http request failed: %w", err)
		// Attempt to get some info from httpResp even if err != nil (e.g. if it's a redirect error httpClient is configured not to follow)
		if httpResp != nil {
			clientResponse.Status = httpResp.Status
			clientResponse.StatusCode = httpResp.StatusCode
			clientResponse.Proto = httpResp.Proto
			clientResponse.Headers = httpResp.Header
			// Don't try to read body if there was an error from Do(), as httpResp.Body might be nil or invalid.
			// But ensure it's closed if non-nil to prevent resource leaks.
			defer func() { _ = httpResp.Body.Close() }()
		}
		return clientResponse, nil // Return response with error populated
	}
	defer func() { _ = httpResp.Body.Close() }()

	// 4. Capture response details into clientResponse
	clientResponse.Status = httpResp.Status
	clientResponse.StatusCode = httpResp.StatusCode
	clientResponse.Proto = httpResp.Proto
	clientResponse.Headers = httpResp.Header
	clientResponse.Size = httpResp.ContentLength

	bodyBytes, readErr := io.ReadAll(httpResp.Body)
	if readErr != nil {
		clientResponse.Error = fmt.Errorf("failed to read response body: %w", readErr)
		// BodyBytes will be nil or partial, BodyString will be empty or partial
		// Still return clientResponse with the error
	} else {
		clientResponse.Body = bodyBytes
		clientResponse.BodyString = string(bodyBytes)
	}

	// TODO: Populate TLS details if applicable
	// (Requires inspecting httpResp.TLS which is *tls.ConnectionState)
	if httpResp.TLS != nil {
		clientResponse.IsTLS = true
		// Basic TLS info, more can be added from httpResp.TLS
		switch httpResp.TLS.Version {
		case tls.VersionTLS10:
			clientResponse.TLSVersion = "TLS 1.0"
		case tls.VersionTLS11:
			clientResponse.TLSVersion = "TLS 1.1"
		case tls.VersionTLS12:
			clientResponse.TLSVersion = "TLS 1.2"
		case tls.VersionTLS13:
			clientResponse.TLSVersion = "TLS 1.3"
		default:
			clientResponse.TLSVersion = "unknown"
		}
		clientResponse.TLSCipherSuite = tls.CipherSuiteName(httpResp.TLS.CipherSuite)
	}

	return clientResponse, nil
}

// TODO: Add other public methods as needed, e.g.:
// - Execute(ctx context.Context, request *Request, options ...RequestOption) (*Response, error)
// - A method to validate a single response if users construct ExpectedResponse manually.
