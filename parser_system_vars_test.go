package restclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PRD-COMMENT: FR3.1 - System Variables: Basic (UUID, Timestamp, Datetime)
// Corresponds to: http_syntax.md "System Variables", "Basic System Variables (UUID, Timestamp, Datetime)"
// This test verifies that the parser correctly preserves basic system variable placeholders
// (like {{$guid}}, {{$uuid}}, {{$timestamp}}, {{$isoTimestamp}}, {{$datetime format}}, {{$localDatetime format}})
// in the RawURLString and Headers of parsed requests. It ensures these placeholders are not prematurely evaluated.
// It uses 'testdata/system_variables/basic_system_vars.http'.
func TestParseRequestFile_BasicSystemVariables(t *testing.T) {
	t.Parallel()

	// Given: A file with basic system variables like UUID and timestamp
	filename := "testdata/system_variables/basic_system_vars.http"

	// When: We parse the request file
	parsedFile, err := parseRequestFile(filename, nil, make([]string, 0))

	// Then: We should have valid requests with preserved system variables
	require.NoError(t, err, "Failed to parse basic system variables file")
	require.Len(t, parsedFile.Requests, 3, "Expected 3 requests in basic system variables file")

	// Verify UUID/GUID request
	uuidRequest := parsedFile.Requests[0]
	assert.Equal(t, "GET", uuidRequest.Method)
	// For requests with system variables in RawURLString, URL parsing is deferred.
	assert.Nil(t, uuidRequest.URL, "uuidRequest.URL should be nil due to deferred parsing of system variables")

	// Check path components and variable placeholders in the RawURLString
	assert.Contains(t, uuidRequest.RawURLString, "/api/uuid/", "uuidRequest.RawURLString should contain /api/uuid/ path component")
	assert.Contains(t, uuidRequest.RawURLString, "{{$guid}}", "uuidRequest.RawURLString should contain {{$guid}} placeholder")

	// Check headers - note that header values are stored as []string
	require.Contains(t, uuidRequest.Headers, "X-Request-Id") // HTTP headers are normalized to Title-Case with lowercase
	require.Len(t, uuidRequest.Headers["X-Request-Id"], 1)
	assert.Equal(t, "{{$uuid}}", uuidRequest.Headers["X-Request-Id"][0])

	// Verify Timestamp request
	timestampRequest := parsedFile.Requests[1]
	assert.Equal(t, "GET", timestampRequest.Method)
	// For requests with system variables in RawURLString, URL parsing is deferred.
	assert.Nil(t, timestampRequest.URL, "timestampRequest.URL should be nil due to deferred parsing of system variables")
	assert.Contains(t, timestampRequest.RawURLString, "/api/timestamp/", "timestampRequest.RawURLString should contain /api/timestamp/ path component")
	assert.Contains(t, timestampRequest.RawURLString, "{{$timestamp}}", "timestampRequest.RawURLString should contain {{$timestamp}} placeholder")

	require.Contains(t, timestampRequest.Headers, "X-Request-Time")
	require.Len(t, timestampRequest.Headers["X-Request-Time"], 1)
	assert.Equal(t, "{{$isoTimestamp}}", timestampRequest.Headers["X-Request-Time"][0])

	// Verify Datetime Format request
	datetimeRequest := parsedFile.Requests[2]
	assert.Equal(t, "GET", datetimeRequest.Method)
	assert.Equal(t, "https://example.com/api/date", datetimeRequest.URL.String())

	require.Contains(t, datetimeRequest.Headers, "X-Date-Rfc")
	require.Len(t, datetimeRequest.Headers["X-Date-Rfc"], 1)
	assert.Equal(t, "{{$datetime rfc1123}}", datetimeRequest.Headers["X-Date-Rfc"][0])

	require.Contains(t, datetimeRequest.Headers, "X-Date-Iso")
	require.Len(t, datetimeRequest.Headers["X-Date-Iso"], 1)
	assert.Equal(t, "{{$datetime iso8601}}", datetimeRequest.Headers["X-Date-Iso"][0])

	require.Contains(t, datetimeRequest.Headers, "X-Date-Custom")
	require.Len(t, datetimeRequest.Headers["X-Date-Custom"], 1)
	assert.Equal(t, `{{$datetime "2006-01-02"}}`, datetimeRequest.Headers["X-Date-Custom"][0])

	require.Contains(t, datetimeRequest.Headers, "X-Local-Date")
	require.Len(t, datetimeRequest.Headers["X-Local-Date"], 1)
	assert.Equal(t, `{{$localDatetime "2006-01-02 15:04:05"}}`, datetimeRequest.Headers["X-Local-Date"][0])
}

// PRD-COMMENT: FR3.2 - System Variables: Random Values
// Corresponds to: http_syntax.md "System Variables", "Random Value Generators"
// This test verifies that the parser correctly preserves random value system variable placeholders
// (e.g., {{$randomInt}}, {{$random.integer(min, max)}}, {{$random.float(min, max)}}, {{$random.alphabetic(length)}},
// {{$random.alphanumeric(length)}}, {{$random.hexadecimal(length)}}) in parsed requests.
// It ensures these placeholders are not prematurely evaluated by the parser.
// It uses 'testdata/system_variables/random_values.http'.
func TestParseRequestFile_RandomSystemVariables(t *testing.T) {
	t.Parallel()

	// Given: A file with random system variables
	filename := "testdata/system_variables/random_values.http"

	// When: We parse the request file
	parsedFile, err := parseRequestFile(filename, nil, make([]string, 0))

	// Then: We should have valid requests with preserved random system variables
	require.NoError(t, err, "Failed to parse random system variables file")
	require.Len(t, parsedFile.Requests, 3, "Expected 3 requests in random system variables file")

	// Verify Random Integer request
	intRequest := parsedFile.Requests[0]
	assert.Equal(t, "GET", intRequest.Method)
	assert.Equal(t, "https://example.com/api/random", intRequest.URL.String())

	require.Contains(t, intRequest.Headers, "X-Random-Int")
	require.Len(t, intRequest.Headers["X-Random-Int"], 1)
	assert.Equal(t, "{{$randomInt}}", intRequest.Headers["X-Random-Int"][0])

	require.Contains(t, intRequest.Headers, "X-Random-Int-Range")
	require.Len(t, intRequest.Headers["X-Random-Int-Range"], 1)
	assert.Equal(t, "{{$randomInt 100 200}}", intRequest.Headers["X-Random-Int-Range"][0])

	require.Contains(t, intRequest.Headers, "X-Random-Int-Jetbrains")
	require.Len(t, intRequest.Headers["X-Random-Int-Jetbrains"], 1)
	assert.Equal(t, "{{$random.integer(300, 400)}}", intRequest.Headers["X-Random-Int-Jetbrains"][0])

	// Verify Random Float request
	floatRequest := parsedFile.Requests[1]
	assert.Equal(t, "GET", floatRequest.Method)
	assert.Equal(t, "https://example.com/api/random/float", floatRequest.URL.String())

	require.Contains(t, floatRequest.Headers, "X-Random-Float")
	require.Len(t, floatRequest.Headers["X-Random-Float"], 1)
	assert.Equal(t, "{{$random.float(1.0, 2.5)}}", floatRequest.Headers["X-Random-Float"][0])

	require.Contains(t, floatRequest.Headers, "X-Random-Float-Negative")
	require.Len(t, floatRequest.Headers["X-Random-Float-Negative"], 1)
	assert.Equal(t, "{{$random.float(-1.5, 0.5)}}", floatRequest.Headers["X-Random-Float-Negative"][0])

	// Verify Random String request (alphabetic, alphanumeric)
	stringRequest := parsedFile.Requests[2]
	assert.Equal(t, "GET", stringRequest.Method)
	assert.Equal(t, "https://example.com/api/random/string", stringRequest.URL.String())

	require.Contains(t, stringRequest.Headers, "X-Random-Alphabetic")
	require.Len(t, stringRequest.Headers["X-Random-Alphabetic"], 1)
	assert.Equal(t, "{{$random.alphabetic(10)}}", stringRequest.Headers["X-Random-Alphabetic"][0])

	require.Contains(t, stringRequest.Headers, "X-Random-Alphabetic-Zero")
	require.Len(t, stringRequest.Headers["X-Random-Alphabetic-Zero"], 1)
	assert.Equal(t, "{{$random.alphabetic(0)}}", stringRequest.Headers["X-Random-Alphabetic-Zero"][0])

	require.Contains(t, stringRequest.Headers, "X-Random-Alphanumeric")
	require.Len(t, stringRequest.Headers["X-Random-Alphanumeric"], 1)
	assert.Equal(t, "{{$random.alphanumeric(15)}}", stringRequest.Headers["X-Random-Alphanumeric"][0])

	require.Contains(t, stringRequest.Headers, "X-Random-Hexadecimal")
	require.Len(t, stringRequest.Headers["X-Random-Hexadecimal"], 1)
	assert.Equal(t, "{{$random.hexadecimal(8)}}", stringRequest.Headers["X-Random-Hexadecimal"][0])
}

// PRD-COMMENT: FR3.3 - System Variables: Environment Variable Access
// Corresponds to: http_syntax.md "System Variables", "Environment Variable Access"
// This test verifies that the parser correctly preserves environment access system variable placeholders
// (e.g., {{$processEnv VAR_NAME}}, {{$env.VAR_NAME}}) in parsed requests. It ensures these placeholders,
// which are intended for runtime resolution, are not evaluated by the parser itself.
// It uses 'testdata/system_variables/environment_access.http'.
func TestParseRequestFile_EnvironmentAccess(t *testing.T) {
	t.Parallel()

	// Given: A file with environment access system variables
	filename := "testdata/system_variables/environment_access.http"

	// When: We parse the request file
	parsedFile, err := parseRequestFile(filename, nil, make([]string, 0))

	// Then: We should have valid requests with preserved environment access variables
	require.NoError(t, err, "Failed to parse environment access file")
	require.Len(t, parsedFile.Requests, 2, "Expected 2 requests in environment access file")

	// Verify VS Code environment access
	vsRequest := parsedFile.Requests[0]
	assert.Equal(t, "GET", vsRequest.Method)
	assert.Equal(t, "https://example.com/api/env", vsRequest.URL.String())

	require.Contains(t, vsRequest.Headers, "X-Vs-Code-Env")
	require.Len(t, vsRequest.Headers["X-Vs-Code-Env"], 1)
	assert.Equal(t, "{{$processEnv PATH}}", vsRequest.Headers["X-Vs-Code-Env"][0])

	// Verify JetBrains environment access
	jbRequest := parsedFile.Requests[1]
	assert.Equal(t, "GET", jbRequest.Method)
	assert.Equal(t, "https://example.com/api/env", jbRequest.URL.String())

	require.Contains(t, jbRequest.Headers, "X-Jetbrains-Env")
	require.Len(t, jbRequest.Headers["X-Jetbrains-Env"], 1)
	assert.Equal(t, "{{$env.PATH}}", jbRequest.Headers["X-Jetbrains-Env"][0])
}
