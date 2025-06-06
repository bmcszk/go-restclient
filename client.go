package restclient

import (
	"context"
	crypto_rand "crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	rand "math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/joho/godotenv"
)

// ResolveOptions controls the behavior of variable substitution.
// If both FallbackToOriginal and FallbackToEmpty are false, and a variable is not found,
// an error or specific handling might occur (though current implementation defaults to empty string if not original).
type ResolveOptions struct {
	FallbackToOriginal bool // If true, an unresolved placeholder {{var}} becomes "{{var}}"
	FallbackToEmpty    bool // If true, an unresolved placeholder {{var}} becomes "" (empty string)
}

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

// SetProgrammaticVar sets or updates a single programmatic variable for the client instance.
// These variables can be used in .http and .hresp files and have the highest precedence.
func (c *Client) SetProgrammaticVar(key string, value interface{}) error {
	if c.programmaticVars == nil {
		c.programmaticVars = make(map[string]interface{})
	}
	c.programmaticVars[key] = value
	return nil
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
	slog.Debug("ExecuteFile: Entered function", "requestFilePath", requestFilePath)
	parsedFile, err := parseRequestFile(requestFilePath, c, make([]string, 0)) // Pass initial empty import stack
	if err != nil {
		return nil, fmt.Errorf("failed to parse request file %s: %w", requestFilePath, err)
	}
	if len(parsedFile.Requests) == 0 {
		return nil, fmt.Errorf("no requests found in file %s", requestFilePath)
	}
	slog.Debug("ExecuteFile: parseRequestFile completed successfully", "numRequests", len(parsedFile.Requests), "requestFilePath", requestFilePath)

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

		var parsedURLPath, parsedURLScheme, parsedURLHost, parsedURLOpaque string
		var isURLNil bool
		if restClientReq.URL != nil {
			parsedURLPath = restClientReq.URL.Path
			parsedURLScheme = restClientReq.URL.Scheme
			parsedURLHost = restClientReq.URL.Host
			parsedURLOpaque = restClientReq.URL.Opaque
			isURLNil = false
		} else {
			slog.Warn("ExecuteFile: restClientReq.URL is nil before substituteRequestVariables loop iteration", "method", restClientReq.Method, "rawURL", restClientReq.RawURLString, "requestName", restClientReq.Name)
			isURLNil = true
		}
		slog.Debug("ExecuteFile: Before substituteRequestVariables", "method", restClientReq.Method, "rawURL", restClientReq.RawURLString, "parsedURLPath", parsedURLPath, "parsedURLScheme", parsedURLScheme, "parsedURLHost", parsedURLHost, "parsedURLOpaque", parsedURLOpaque, "isURLNil", isURLNil)
		finalParsedURL, err := c.substituteRequestVariables(restClientReq, parsedFile, requestScopedSystemVars, osEnvGetter)
		if err != nil {
			resp := &Response{Request: restClientReq}
			resp.Error = err // Already wrapped in substituteRequestVariables
			wrappedErr := fmt.Errorf("request %d (%s %s) failed variable substitution: %w", i+1, restClientReq.Method, restClientReq.RawURLString, resp.Error)
			multiErr = multierror.Append(multiErr, wrappedErr)
			responses[i] = resp
			continue
		}
		restClientReq.URL = finalParsedURL

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
				nil, // Use default options (fallback to empty for unresolved)
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

		urlForError := restClientReq.RawURLString
		if restClientReq.URL != nil {
			urlForError = restClientReq.URL.String()
		}

		if execErr != nil { // Critical error from executeRequest, resp from executeRequest is nil
			resp = &Response{Request: restClientReq, Error: execErr} // Initialize resp and set its error
			wrappedErr := fmt.Errorf("request %d (%s %s) failed with critical error: %w", i+1, restClientReq.Method, urlForError, execErr)
			multiErr = multierror.Append(multiErr, wrappedErr)
		} else if resp.Error != nil { // Non-critical error, execErr was nil, resp is non-nil from executeRequest
			wrappedErr := fmt.Errorf("request %d (%s %s) processing resulted in error: %w", i+1, restClientReq.Method, urlForError, resp.Error)
			multiErr = multierror.Append(multiErr, wrappedErr)
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
func (c *Client) resolveVariablesInText(text string, clientProgrammaticVars map[string]interface{}, fileScopedVars map[string]string, environmentVars map[string]string, globalVars map[string]string, requestScopedSystemVars map[string]string, osEnvGetter func(string) (string, bool), dotEnvVars map[string]string, options *ResolveOptions) string {
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

			// Handle unresolved based on options
			if options != nil {
				if options.FallbackToOriginal {
					return match // Return original placeholder {{varName}}
				}
				if options.FallbackToEmpty {
					return "" // Return empty string
				}
			}

			// Default behavior if no options specify otherwise, or if options is nil:
			// For backward compatibility and current test expectations in some areas,
			// this often means returning an empty string. However, to ensure that
			// placeholders that are *meant* to be preserved (like {{client_var}} in a file var)
			// are not lost if options are accidentally nil, returning 'match' is safer default
			// when no explicit fallback option is hit.
			// The caller (e.g. substituteRequestVariables vs resolveAndSetFileVariable) will decide the options.
			// If options is nil, it implies the caller expects a certain default, which prior to this change
			// was effectively 'fallback to empty' in many cases due to how tests were written or how
			// unresolved variables were handled implicitly.
			// Let's default to empty string if options is nil and no other condition met, to mimic old behavior for existing calls.
			if options == nil {
				return "" // Default to empty string if no options and not found
			}

			return match // Should ideally not be reached if options are well-defined
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

// substituteRequestVariables handles the substitution of variables in the request's URL and headers.
// It returns the final parsed URL or an error if substitution/parsing fails.
func (c *Client) substituteRequestVariables(rcRequest *Request, parsedFile *ParsedFile, requestScopedSystemVars map[string]string, osEnvGetter func(string) (string, bool)) (*url.URL, error) {
	substitutedRawURL := c.resolveVariablesInText(
		rcRequest.RawURLString,
		c.programmaticVars,
		rcRequest.ActiveVariables,
		parsedFile.EnvironmentVariables,
		parsedFile.GlobalVariables,
		requestScopedSystemVars,
		osEnvGetter,
		c.currentDotEnvVars,
		nil, // Use default options (fallback to empty for unresolved)
	)
	substitutedRawURL = c.substituteDynamicSystemVariables(substitutedRawURL, c.currentDotEnvVars)

	finalParsedURL, parseErr := url.Parse(substitutedRawURL)
	if parseErr != nil {
		return nil, fmt.Errorf("failed to parse URL after variable substitution: %s (original: %s): %w", substitutedRawURL, rcRequest.RawURLString, parseErr)
	}
	rcRequest.URL = finalParsedURL // Assign here as it's successfully parsed

	if rcRequest.Headers != nil {
		for key, values := range rcRequest.Headers {
			newValues := make([]string, len(values))
			for j, val := range values {
				resolvedVal := c.resolveVariablesInText(
					val,
					c.programmaticVars,
					rcRequest.ActiveVariables,
					parsedFile.EnvironmentVariables,
					parsedFile.GlobalVariables,
					requestScopedSystemVars,
					osEnvGetter,
					c.currentDotEnvVars,
					nil, // Use default options (fallback to empty for unresolved)
				)
				newValues[j] = c.substituteDynamicSystemVariables(resolvedVal, c.currentDotEnvVars)
			}
			rcRequest.Headers[key] = newValues
		}
	}
	return finalParsedURL, nil
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

	text = c.substituteRandomVariables(text)

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

// substituteRandomVariables handles the substitution of $random.* variables.
func (c *Client) substituteRandomVariables(text string) string {
	// Substitute {{$randomInt}}, {{$randomInt MIN MAX}}
	text = regexp.MustCompile(`\{\{\$randomInt(?:\s+(-?\d+)\s+(-?\d+))?\}\}`).ReplaceAllStringFunc(text, func(match string) string {
		parts := regexp.MustCompile(`\{\{\$randomInt(?:\s+(-?\d+)\s+(-?\d+))?\}\}`).FindStringSubmatch(match)
		min, max := 0, 100 // Default range
		if len(parts) == 3 && parts[1] != "" && parts[2] != "" {
			var errMin, errMax error
			min, errMin = strconv.Atoi(parts[1])
			max, errMax = strconv.Atoi(parts[2])
			if errMin != nil || errMax != nil || min > max {
				return match
			}
		}
		return strconv.Itoa(rand.Intn(max-min+1) + min)
	})

	// Substitute {{$randomFloat}}, {{$randomFloat MIN MAX}}
	text = regexp.MustCompile(`\{\{\$randomFloat(?:\s+(-?\d*\.?\d+)\s+(-?\d*\.?\d+))?\}\}`).ReplaceAllStringFunc(text, func(match string) string {
		parts := regexp.MustCompile(`\{\{\$randomFloat(?:\s+(-?\d*\.?\d+)\s+(-?\d*\.?\d+))?\}\}`).FindStringSubmatch(match)
		min, max := 0.0, 1.0 // Default range
		if len(parts) == 3 && parts[1] != "" && parts[2] != "" {
			var errMin, errMax error
			min, errMin = strconv.ParseFloat(parts[1], 64)
			max, errMax = strconv.ParseFloat(parts[2], 64)
			if errMin != nil || errMax != nil || min > max {
				return match
			}
		}
		return fmt.Sprintf("%f", min+rand.Float64()*(max-min))
	})

	// Substitute {{$randomBoolean}}
	text = strings.ReplaceAll(text, "{{$randomBoolean}}", strconv.FormatBool(rand.Intn(2) == 0))

	// Substitute {{$randomHex}}
	text = regexp.MustCompile(`\{\{\$randomHex(?:\s+(\d+))?\}\}`).ReplaceAllStringFunc(text, _substituteRandomHexFunc)

	// Substitute {{$randomAlphaNumeric}}
	text = regexp.MustCompile(`\{\{\$randomAlphaNumeric(?:\s+(\d+))?\}\}`).ReplaceAllStringFunc(text, _substituteRandomAlphaNumericFunc)

	// Substitute {{$randomString}}
	text = regexp.MustCompile(`\{\{\$randomString(?:\s+(\d+))?\}\}`).ReplaceAllStringFunc(text, _substituteRandomStringFunc)

	// Substitute {{$randomEmail}}
	text = strings.ReplaceAll(text, "{{$randomEmail}}",
		fmt.Sprintf("%s@%s.com",
			randomStringFromCharset(10, "abcdefghijklmnopqrstuvwxyz"),
			randomStringFromCharset(7, "abcdefghijklmnopqrstuvwxyz")))

	// Substitute {{$randomDomain}}
	text = strings.ReplaceAll(text, "{{$randomDomain}}",
		fmt.Sprintf("%s.com", randomStringFromCharset(10, "abcdefghijklmnopqrstuvwxyz")))

	// Substitute {{$randomIPv4}}
	text = strings.ReplaceAll(text, "{{$randomIPv4}}",
		fmt.Sprintf("%d.%d.%d.%d", rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256)))

	// Substitute {{$randomIPv6}}
	text = strings.ReplaceAll(text, "{{$randomIPv6}}", func() string {
		segments := make([]string, 8)
		for i := 0; i < 8; i++ {
			segments[i] = fmt.Sprintf("%x", rand.Intn(0x10000))
		}
		return strings.Join(segments, ":")
	}())

	// Substitute {{$randomMacAddress}}
	text = strings.ReplaceAll(text, "{{$randomMacAddress}}", func() string {
		segments := make([]string, 6)
		for i := 0; i < 6; i++ {
			segments[i] = fmt.Sprintf("%02x", rand.Intn(0x100))
		}
		return strings.Join(segments, ":")
	}())

	// Substitute {{$randomUUID}}
	text = strings.ReplaceAll(text, "{{$randomUUID}}", uuid.New().String()) // Different from {{$uuid}} which is request-scoped

	// Substitute {{$randomPassword}}
	text = regexp.MustCompile(`\{\{\$randomPassword(?:\s+(\d+))?\}\}`).ReplaceAllStringFunc(text, _substituteRandomPasswordFunc)

	// Substitute {{$randomColor}}
	text = strings.ReplaceAll(text, "{{$randomColor}}",
		fmt.Sprintf("#%02x%02x%02x", rand.Intn(256), rand.Intn(256), rand.Intn(256)))

	// Substitute {{$randomWord}}
	words := []string{"apple", "banana", "cherry", "date", "elderberry", "fig", "grape"}
	text = strings.ReplaceAll(text, "{{$randomWord}}", words[rand.Intn(len(words))])

	return text
}

func _substituteRandomHexFunc(match string) string {
	parts := regexp.MustCompile(`\{\{\$randomHex(?:\s+(\d+))?\}\}`).FindStringSubmatch(match)
	length := 16 // Default length
	if len(parts) == 2 && parts[1] != "" {
		var err error
		length, err = strconv.Atoi(parts[1])
		if err != nil || length <= 0 {
			return match // Malformed length
		}
	}
	b := make([]byte, length/2+length%2)
	if _, err := crypto_rand.Read(b); err != nil { // Changed to crypto_rand.Read
		return match // Error generating random bytes
	}
	hexStr := fmt.Sprintf("%x", b)
	return hexStr[:length]
}

func _substituteRandomAlphaNumericFunc(match string) string {
	parts := regexp.MustCompile(`\{\{\$randomAlphaNumeric(?:\s+(\d+))?\}\}`).FindStringSubmatch(match)
	length := 16 // Default length
	if len(parts) == 2 && parts[1] != "" {
		var err error
		length, err = strconv.Atoi(parts[1])
		if err != nil || length <= 0 {
			return match // Malformed length
		}
	}
	return randomStringFromCharset(length, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
}

func _substituteRandomStringFunc(match string) string {
	parts := regexp.MustCompile(`\{\{\$randomString(?:\s+(\d+))?\}\}`).FindStringSubmatch(match)
	length := 16 // Default length
	if len(parts) == 2 && parts[1] != "" {
		var err error
		length, err = strconv.Atoi(parts[1])
		if err != nil || length <= 0 {
			return match // Malformed length
		}
	}
	return randomStringFromCharset(length, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+-=[]{};':\",./<>?")
}

func _substituteRandomPasswordFunc(match string) string {
	parts := regexp.MustCompile(`\{\{\$randomPassword(?:\s+(\d+))?\}\}`).FindStringSubmatch(match)
	length := 12 // Default length
	if len(parts) == 2 && parts[1] != "" {
		var err error
		length, err = strconv.Atoi(parts[1])
		if err != nil || length <= 0 {
			return match // Malformed length
		}
	}
	if length < 4 {
		return randomStringFromCharset(length, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*")
	}
	return randomStringFromCharset(length, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+-=[]{};':\",./<>?")
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

// _resolveRequestURL resolves the final request URL based on the client's BaseURL and the request's URL.
// It returns the resolved URL or an error if the BaseURL is invalid or requestURL is nil.
func (c *Client) _resolveRequestURL(baseURLStr string, requestURL *url.URL) (*url.URL, error) {
	slog.Debug("_resolveRequestURL: Entered function", "baseURL", baseURLStr)

	if requestURL == nil {
		return nil, fmt.Errorf("request URL is unexpectedly nil")
	}

	// Sanitize the incoming requestURL
	requestURLStr := requestURL.String()
	freshRequestURL, err := url.Parse(requestURLStr)
	if err != nil {
		return nil, fmt.Errorf("failed to re-parse incoming requestURL string '%s': %w", requestURLStr, err)
	}

	// If freshRequestURL is absolute, use it directly
	if freshRequestURL.IsAbs() {
		return freshRequestURL, nil
	}

	// If no BaseURL, return freshRequestURL as is
	if baseURLStr == "" {
		return freshRequestURL, nil
	}

	// Parse and sanitize the base URL
	base, err := url.Parse(baseURLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid BaseURL %s: %w", baseURLStr, err)
	}

	baseStr := base.String()
	freshBase, err := url.Parse(baseStr)
	if err != nil {
		return nil, fmt.Errorf("failed to re-parse base URL string '%s': %w", baseStr, err)
	}

	// Handle special path joining for absolute paths
	if strings.HasPrefix(freshRequestURL.Path, "/") && freshBase.Path != "" && freshBase.Path != "/" {
		finalResolvedURL := joinURLPaths(freshBase, freshRequestURL)
		if finalResolvedURL == nil {
			return nil, fmt.Errorf("failed to join URL paths: %s and %s", freshBase.Path, freshRequestURL.Path)
		}
		return finalResolvedURL, nil
	}

	// Default behavior for other cases
	return freshBase.ResolveReference(freshRequestURL), nil
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

	slog.Debug("executeRequest: Before _resolveRequestURL", "baseURL", c.BaseURL, "rcRequestURLPath", rcRequest.URL.Path, "rcRequestURLScheme", rcRequest.URL.Scheme, "rcRequestURLHost", rcRequest.URL.Host, "rcRequestURLOpaque", rcRequest.URL.Opaque)
	urlToUse, urlErr := c._resolveRequestURL(c.BaseURL, rcRequest.URL)
	if urlErr != nil {
		// An error from _resolveRequestURL implies a bad BaseURL or nil rcRequest.URL.
		// Per original logic for bad BaseURL, return nil for *Response.
		return nil, urlErr
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
		if httpResp == nil {
			return clientResponse, nil // Early return if httpResp is nil, error is already set
		}

		// At this point, httpResp is non-nil. Attempt to process its body and populate details.
		var bodyBytes []byte
		var readErr error
		if httpResp.Body != nil {
			defer func() { _ = httpResp.Body.Close() }() // Defer close if body exists
			bodyBytes, readErr = io.ReadAll(httpResp.Body)
		}
		// If httpResp.Body was nil, bodyBytes and readErr remain nil.
		// _populateResponseDetails is expected to handle this gracefully.
		c._populateResponseDetails(clientResponse, httpResp, bodyBytes, readErr)
		return clientResponse, nil // Return response with error and any populated details
	}

	// Success path: ensure body is closed after reading.
	// httpResp and httpResp.Body are guaranteed non-nil here if err is nil.
	defer func() { _ = httpResp.Body.Close() }()

	bodyBytes, readErr := io.ReadAll(httpResp.Body)
	c._populateResponseDetails(clientResponse, httpResp, bodyBytes, readErr)

	return clientResponse, nil
}

// _populateResponseDetails copies relevant information from an *http.Response and body to our *Response.
func (c *Client) _populateResponseDetails(resp *Response, httpResp *http.Response, bodyBytes []byte, bodyReadErr error) {
	if httpResp == nil {
		return
	}

	resp.Status = httpResp.Status
	resp.StatusCode = httpResp.StatusCode
	resp.Proto = httpResp.Proto
	resp.Headers = httpResp.Header
	resp.Size = httpResp.ContentLength // This might be -1 if chunked, actual size is len(bodyBytes)

	if bodyReadErr != nil {
		readErrWrapped := fmt.Errorf("failed to read response body: %w", bodyReadErr)
		resp.Error = multierror.Append(resp.Error, readErrWrapped).ErrorOrNil()
	} else {
		resp.Body = bodyBytes
		resp.BodyString = string(bodyBytes)
		if resp.Size == -1 || (resp.Size == 0 && len(bodyBytes) > 0) { // Update size if not set or if chunked and body was read
			resp.Size = int64(len(bodyBytes))
		}
	}

	if httpResp.TLS != nil {
		resp.IsTLS = true
		switch httpResp.TLS.Version {
		case tls.VersionTLS10:
			resp.TLSVersion = "TLS 1.0"
		case tls.VersionTLS11:
			resp.TLSVersion = "TLS 1.1"
		case tls.VersionTLS12:
			resp.TLSVersion = "TLS 1.2"
		case tls.VersionTLS13:
			resp.TLSVersion = "TLS 1.3"
		default:
			resp.TLSVersion = fmt.Sprintf("TLS unknown (0x%04x)", httpResp.TLS.Version)
		}
		resp.TLSCipherSuite = tls.CipherSuiteName(httpResp.TLS.CipherSuite)
		// TODO: Add more TLS details like server name, peer certificates if needed
	}
}

// TODO: Add other public methods as needed, e.g.:
// - Execute(ctx context.Context, request *Request, options ...RequestOption) (*Response, error)
// - A method to validate a single response if users construct ExpectedResponse manually.
//
