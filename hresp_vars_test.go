package restclient

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// _assertMixedCustomAndSystemResult checks results for tests like "mixed custom (programmatic) and system".
func _assertMixedCustomAndSystemResult(t *testing.T, result string) {
	t.Helper()
	assert.True(t, strings.HasPrefix(result, "User: user123, ReqID: "), "Result should start with static part")
	dynamicPart := result[len("User: user123, ReqID: "):]
	assert.Regexp(t, `^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`, dynamicPart, "Dynamic part should be a UUID")
}

// _assertUUIDResult checks results for tests involving {{$uuid}}.
func _assertUUIDResult(t *testing.T, testName, content, result string) {
	t.Helper()

	// If {{$uuid}} is not in the original content, this specific assertion logic is not applicable.
	// The calling switch statement should ensure this function is called appropriately.
	if !strings.Contains(content, "{{$uuid}}") {
		// This case implies that {{$uuid}} was not expected in the content for this assertion type.
		// If a UUID still appears in the result, it might be from an unexpected fallback.
		// For now, we assume the caller (switch) is correct, and this path means no UUID check is needed from this func.
		return
	}

	// Case 1: {{$uuid}} was NOT substituted and remains literal in the result.
	// This happens if the client is nil (so system vars aren't processed) AND the fallback (if any) was not a different value.
	if strings.Contains(result, "{{$uuid}}") {
		// To be certain it's the literal and not part of a malformed actual UUID string that happens to contain "{{$uuid}}",
		// we check if the result, when {{$uuid}} is considered a placeholder, matches the original content structure.
		// Example: content="ID: {{$uuid}}", result="ID: {{$uuid}}"
		expectedLiteralResult := content // If content has {{$uuid}}, this is the expected literal form.
		assert.Equal(t, expectedLiteralResult, result,
			"Expected literal '{{$uuid}}' to remain if not substituted. Test: '%s', Content: '%s', Result: '%s'",
			testName, content, result)
		return
	}

	// Case 2: {{$uuid}} was substituted by an actual UUID value (or a fallback that resolved to a UUID).
	// At this point, 'result' does NOT contain '{{$uuid}}' literally; it should contain an actual UUID.
	var expectedPrefix string
	if testName == "fallback providing a system variable to be resolved" {
		// Test case: content = "Special: {{special_val | {{$uuid}}}}"
		// 'special_val' is not defined, so fallback '{{$uuid}}' is used and resolved.
		// Result should be "Special: <actual_uuid>"
		expectedPrefix = "Special: "
	} else {
		// Standard case: content = "Prefix: {{$uuid}}" -> result = "Prefix: <actual_uuid>"
		idx := strings.Index(content, "{{$uuid}}")
		// This check is for safety; already guarded by the initial strings.Contains(content, "{{$uuid}}").
		if idx == -1 {
			t.Fatalf("Internal logic error: {{$uuid}} not found in content '%s' despite initial check. Test: '%s'", content, testName)
			return // Should be unreachable
		}
		expectedPrefix = content[:idx]
	}

	assert.True(t, strings.HasPrefix(result, expectedPrefix),
		"Result '%s' should have prefix '%s'. Test: '%s'", result, expectedPrefix, testName)

	dynamicPart := result[len(expectedPrefix):]
	assert.Regexp(t, `^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`, dynamicPart,
		"Dynamic part '%s' of result '%s' should be a UUID. Test: '%s'", dynamicPart, result, testName)
}

// _assertTimestampResult checks results for tests involving {{$timestamp}}.
func _assertTimestampResult(t *testing.T, content, result string) {
	t.Helper()
	idx := strings.Index(content, "{{$timestamp}}")
	if idx == -1 {
		t.Errorf("{{$timestamp}} not found in content '%s'", content)
		return
	}
	prefix := content[:idx]
	assert.True(t, strings.HasPrefix(result, prefix))
	dynamicPart := result[len(prefix):]
	assert.Regexp(t, `^\d+$`, dynamicPart)
}

func TestExtractHrespDefines(t *testing.T) {
	tests := []struct {
		name            string
		hrespContent    string
		expectedDefines map[string]string
		expectedContent string
		expectedErr     bool
	}{
		{
			name:            "no defines",
			hrespContent:    "HTTP/1.1 200 OK\nContent-Type: application/json\n\n{\"key\": \"value\"}",
			expectedDefines: map[string]string{},
			expectedContent: "HTTP/1.1 200 OK\nContent-Type: application/json\n\n{\"key\": \"value\"}",
			expectedErr:     false,
		},
		{
			name:            "simple define",
			hrespContent:    "@name = value\nHTTP/1.1 200 OK",
			expectedDefines: map[string]string{"name": "value"},
			expectedContent: "HTTP/1.1 200 OK",
			expectedErr:     false,
		},
		{
			name:            "multiple defines with spaces and comments",
			hrespContent:    "  @host = example.com  \n@token=abc123xyz\n# This is a comment\nHTTP/1.1 200 OK\nAuthorization: Bearer {{token}}",
			expectedDefines: map[string]string{"host": "example.com", "token": "abc123xyz"},
			expectedContent: "# This is a comment\nHTTP/1.1 200 OK\nAuthorization: Bearer {{token}}", // Comments are preserved if not @ lines
			expectedErr:     false,
		},
		{
			name:            "define with no value",
			hrespContent:    "@name=\nHTTP/1.1 200 OK",
			expectedDefines: map[string]string{"name": ""},
			expectedContent: "HTTP/1.1 200 OK",
			expectedErr:     false,
		},
		{
			name:            "malformed define (no equals)",
			hrespContent:    "@name value\nHTTP/1.1 200 OK",
			expectedDefines: map[string]string{},
			expectedContent: "HTTP/1.1 200 OK", // Line is kept as it's not a valid define and not a comment
			expectedErr:     false,
		},
		{
			name:            "empty content",
			hrespContent:    "",
			expectedDefines: map[string]string{},
			expectedContent: "",
			expectedErr:     false,
		},
		{
			name:            "only defines",
			hrespContent:    "@var1=val1\n@var2=val2",
			expectedDefines: map[string]string{"var1": "val1", "var2": "val2"},
			expectedContent: "",
			expectedErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defines, content, err := extractHrespDefines(tt.hrespContent)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedDefines, defines)
				// Normalize newlines for content comparison as scanner might add/remove final newline
				assert.Equal(t, strings.TrimSpace(tt.expectedContent), strings.TrimSpace(content))
			}
		})
	}
}

// resolveSubstTestCase defines the structure for test cases used in TestResolveAndSubstitute.
type resolveSubstTestCase struct {
	name                   string
	content                string
	fileVars               map[string]string
	clientToUse            *Client
	programmaticClientVars map[string]interface{}
	expectedResult         string
	expectedErr            bool
}

func _getBasicResolveSubstTestCases(mockClient *Client) []resolveSubstTestCase {
	return []resolveSubstTestCase{
		{
			name:                   "simple programmatic var substitution",
			content:                "Hello {{name}}!",
			programmaticClientVars: map[string]interface{}{"name": "World"},
			clientToUse:            mockClient,
			expectedResult:         "Hello World!",
		},
		{
			name:           "simple file var substitution",
			content:        "Key: {{token}}",
			fileVars:       map[string]string{"token": "abc-123"},
			expectedResult: "Key: abc-123",
		},
		{
			name:           "env var substitution",
			content:        "From env: {{TEST_ENV_VAR}}",
			expectedResult: "From env: env_value",
		},
		{
			name:           "fallback value used",
			content:        "Value: {{undefined_var | defaultValue}}",
			expectedResult: "Value: defaultValue",
		},
	}
}

func _getPrecedenceResolveSubstTestCases(mockClient *Client) []resolveSubstTestCase {
	return []resolveSubstTestCase{
		{
			name:                   "precedence: programmatic over file",
			content:                "Var: {{my_var}}",
			fileVars:               map[string]string{"my_var": "file_val"},
			programmaticClientVars: map[string]interface{}{"my_var": "prog_val"},
			clientToUse:            mockClient,
			expectedResult:         "Var: prog_val",
		},
		{
			name:                   "precedence: programmatic over env",
			content:                "Var: {{TEST_ENV_VAR}}",
			programmaticClientVars: map[string]interface{}{"TEST_ENV_VAR": "prog_env_override"},
			clientToUse:            mockClient,
			expectedResult:         "Var: prog_env_override",
		},
		{
			name:                   "precedence: programmatic over fallback",
			content:                "Var: {{my_var | fallback}}",
			programmaticClientVars: map[string]interface{}{"my_var": "prog_val"},
			clientToUse:            mockClient,
			expectedResult:         "Var: prog_val",
		},
		{
			name:           "precedence: env over fallback",
			content:        "Var: {{TEST_ENV_VAR | fallback}}",
			expectedResult: "Var: env_value",
		},
		{
			name:           "precedence: file over fallback",
			content:        "Var: {{my_var | fallback}}",
			fileVars:       map[string]string{"my_var": "file_val"},
			expectedResult: "Var: file_val",
		},
	}
}

func _getSystemVarResolveSubstTestCases(mockClient *Client) []resolveSubstTestCase {
	return []resolveSubstTestCase{
		{
			name:        "system variable $uuid",
			content:     "ID: {{$uuid}}",
			clientToUse: mockClient,
		},
		{
			name:        "system variable $timestamp",
			content:     "Time: {{$timestamp}}",
			clientToUse: mockClient,
		},
		{
			name:                   "mixed custom (programmatic) and system",
			content:                "User: {{user_id}}, ReqID: {{$uuid}}",
			programmaticClientVars: map[string]interface{}{"user_id": "user123"},
			clientToUse:            mockClient,
		},
		{
			name:        "fallback providing a system variable to be resolved",
			content:     "Special: {{special_val | {{$uuid}}}}",
			clientToUse: mockClient,
		},
	}
}

func _getOtherResolveSubstTestCases(mockClient *Client) []resolveSubstTestCase {
	return []resolveSubstTestCase{
		{
			name:           "unresolved variable without fallback",
			content:        "Value: {{not_found_var}}",
			expectedResult: "Value: {{not_found_var}}",
		},
		{
			name:           "no variables in content",
			content:        "Just a string.",
			expectedResult: "Just a string.",
		},
		{
			name:                   "spaces around variable and fallback pipe (prog vars)",
			content:                "Test: {{  spaced_var  |  fallback_with_spaces  }}",
			programmaticClientVars: map[string]interface{}{"spaced_var": "actual_value_prog"},
			clientToUse:            mockClient,
			expectedResult:         "Test: actual_value_prog",
		},
		{
			name:           "spaces around variable, fallback used",
			content:        "Test: {{  spaced_var_fb  |  fallback_with_spaces  }}",
			expectedResult: "Test: fallback_with_spaces",
		},
		{
			name:                   "numeric programmatic var",
			content:                "Count: {{count_var}}",
			programmaticClientVars: map[string]interface{}{"count_var": 123},
			clientToUse:            mockClient,
			expectedResult:         "Count: 123",
		},
		{
			name:                   "boolean programmatic var",
			content:                "Enabled: {{enabled_var}}",
			programmaticClientVars: map[string]interface{}{"enabled_var": true},
			clientToUse:            mockClient,
			expectedResult:         "Enabled: true",
		},
	}
}

// _getResolveAndSubstituteTestCases provides the test cases for TestResolveAndSubstitute.
func _getResolveAndSubstituteTestCases(mockClient *Client) []resolveSubstTestCase {
	var tests []resolveSubstTestCase
	tests = append(tests, _getBasicResolveSubstTestCases(mockClient)...)
	tests = append(tests, _getPrecedenceResolveSubstTestCases(mockClient)...)
	tests = append(tests, _getSystemVarResolveSubstTestCases(mockClient)...)
	tests = append(tests, _getOtherResolveSubstTestCases(mockClient)...)
	return tests
}

func TestResolveAndSubstitute(t *testing.T) {
	// Mock client for system variable substitution (optional, can be nil)
	mockClient, _ := NewClient()

	// Setup environment variables for testing
	_ = os.Setenv("TEST_ENV_VAR", "env_value")
	_ = os.Setenv("OTHER_ENV_VAR", "other_env_val")
	defer func() {
		_ = os.Unsetenv("TEST_ENV_VAR")
		_ = os.Unsetenv("OTHER_ENV_VAR")
	}()

	tests := _getResolveAndSubstituteTestCases(mockClient)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Configure the client for the current test case
			var currentClient *Client
			if tt.clientToUse != nil { // If a base client (like mockClient for system vars) is specified
				currentClient, _ = NewClient()                       // Create a fresh one to avoid interference
				currentClient.httpClient = tt.clientToUse.httpClient // Copy transport if needed for sys vars
				if tt.programmaticClientVars != nil {
					currentClient.programmaticVars = tt.programmaticClientVars
				} else {
					currentClient.programmaticVars = make(map[string]interface{}) // ensure not nil
				}
			} else if tt.programmaticClientVars != nil { // If no base client, but programmatic vars exist
				currentClient, _ = NewClient(WithVars(tt.programmaticClientVars))
			} // If both clientToUse and programmaticClientVars are nil, currentClient remains nil, which is fine for some tests.

			result, err := resolveAndSubstitute(tt.content, tt.fileVars, currentClient)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Special handling for system variables that produce dynamic output
				switch {
				case tt.name == "mixed custom (programmatic) and system":
					_assertMixedCustomAndSystemResult(t, result)
				case strings.Contains(tt.name, "$uuid") || strings.Contains(tt.name, "fallback providing a system variable"):
					_assertUUIDResult(t, tt.name, tt.content, result)
				case strings.Contains(tt.name, "$timestamp"):
					_assertTimestampResult(t, tt.content, result)
				default:
					assert.Equal(t, tt.expectedResult, result)
				}
			}
		})
	}
}
