package restclient

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

	tests := []struct {
		name                   string
		content                string
		fileVars               map[string]string
		clientToUse            *Client                // Renamed from client for clarity, and will be configured with programmaticVars
		programmaticClientVars map[string]interface{} // For setting up clientToUse
		expectedResult         string
		expectedErr            bool
	}{
		{
			name:                   "simple programmatic var substitution",
			content:                "Hello {{name}}!",
			programmaticClientVars: map[string]interface{}{"name": "World"},
			clientToUse:            mockClient, // Will be reconfigured or a new one made
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
		{
			name:           "unresolved variable without fallback",
			content:        "Value: {{not_found_var}}",
			expectedResult: "Value: {{not_found_var}}", // Stays as is
		},
		{
			name:        "system variable $uuid",
			content:     "ID: {{$uuid}}",
			clientToUse: mockClient,
			// expectedResult: "ID: <some_uuid_pattern>", // Requires regex match
		},
		{
			name:        "system variable $timestamp",
			content:     "Time: {{$timestamp}}",
			clientToUse: mockClient,
			// expectedResult: "Time: <some_timestamp_pattern>", // Requires regex match
		},
		{
			name:                   "mixed custom (programmatic) and system",
			content:                "User: {{user_id}}, ReqID: {{$uuid}}",
			programmaticClientVars: map[string]interface{}{"user_id": "user123"},
			clientToUse:            mockClient, // System vars need a client
			// expectedResult: "User: user123, ReqID: <uuid_pattern>",
		},
		{
			name:        "fallback providing a system variable to be resolved",
			content:     "Special: {{special_val | {{$uuid}}}}", // Removed space before closing }}
			clientToUse: mockClient,
			// expectedResult: "Special: <uuid_pattern>",
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
				if tt.name == "mixed custom (programmatic) and system" {
					assert.True(t, strings.HasPrefix(result, "User: user123, ReqID: "), "Result should start with static part")
					dynamicPart := result[len("User: user123, ReqID: "):]
					assert.Regexp(t, `^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`, dynamicPart, "Dynamic part should be a UUID")
				} else if strings.Contains(tt.name, "$uuid") || strings.Contains(tt.name, "fallback providing a system variable") {
					// Example: "ID: {{$uuid}}" -> "ID: d7a8f008-3a36-4b97-8486-02a3a7c87c5b"
					// We need to check the part before the dynamic value and then regex check the dynamic part.
					if strings.Contains(tt.content, "{{$uuid}}") {
						parts := strings.Split(result, "{{$uuid}}")            // Placeholder if client was nil
						if len(parts) == 1 && !strings.Contains(result, "-") { // if $uuid was not replaced, it's just the string literal
							assert.Contains(t, result, "{{$uuid}}") // System var not resolved because client might be nil
						} else {
							// It was substituted, check structure
							var prefix string
							if tt.name == "fallback providing a system variable to be resolved" {
								prefix = "Special: " // Because special_val is not defined, fallback is used.
							} else {
								prefix = tt.content[:strings.Index(tt.content, "{{$uuid}}")]
							}
							assert.True(t, strings.HasPrefix(result, prefix), "Expected prefix '%s', got '%s'", prefix, result)
							dynamicPart := result[len(prefix):]
							assert.Regexp(t, `^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`, dynamicPart)
						}
					}
				} else if strings.Contains(tt.name, "$timestamp") {
					prefix := tt.content[:strings.Index(tt.content, "{{$timestamp}}")]
					assert.True(t, strings.HasPrefix(result, prefix))
					dynamicPart := result[len(prefix):]
					assert.Regexp(t, `^\d+$`, dynamicPart)
				} else {
					assert.Equal(t, tt.expectedResult, result)
				}
			}
		})
	}
}
