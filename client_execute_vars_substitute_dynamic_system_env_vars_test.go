package restclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSubstituteDynamicSystemVariables_EnvVars tests the {{$env.VAR_NAME}} substitution.
func TestSubstituteDynamicSystemVariables_EnvVars(t *testing.T) {
	client, _ := NewClient()
	tests := []struct {
		name    string
		input   string
		setup   func(t *testing.T) // For setting env vars
		want    string
		wantErr bool // If we expect a parsing warning (though not directly testable here without log capture)
	}{
		{
			name:  "existing env var",
			input: "Hello {{$env.MY_TEST_VAR}}!",
			setup: func(t *testing.T) { t.Setenv("MY_TEST_VAR", "World") },
			want:  "Hello World!",
		},
		{
			name:  "non-existing env var",
			input: "Value: {{$env.NON_EXISTENT_VAR}}",
			setup: func(t *testing.T) {},
			want:  "Value: ",
		},
		{
			name:  "multiple env vars",
			input: "{{$env.FIRST_VAR}} and {{$env.SECOND_VAR}}",
			setup: func(t *testing.T) {
				t.Setenv("FIRST_VAR", "Apple")
				t.Setenv("SECOND_VAR", "Banana")
			},
			want: "Apple and Banana",
		},
		{
			name:  "env var with underscore and numbers",
			input: "{{$env.MY_VAR_123}}",
			setup: func(t *testing.T) { t.Setenv("MY_VAR_123", "Test123") },
			want:  "Test123",
		},
		{
			name:  "empty env var value",
			input: "Prefix{{$env.EMPTY_VAR}}Suffix",
			setup: func(t *testing.T) { t.Setenv("EMPTY_VAR", "") },
			want:  "PrefixSuffix",
		},
		{
			name:    "malformed - no var name",
			input:   "{{$env.}}",
			setup:   func(t *testing.T) {},
			want:    "{{$env.}}",
			wantErr: true, // Expect original match due to regex non-match or parse fail
		},
		{
			name:    "malformed - invalid char in var name",
			input:   "{{$env.MY-VAR}}", // Hyphen is not allowed by regex
			setup:   func(t *testing.T) { t.Setenv("MY-VAR", "ShouldNotBeUsed") },
			want:    "{{$env.MY-VAR}}",
			wantErr: true,
		},
		{
			name:  "var name starting with underscore",
			input: "{{$env._MY_VAR}}",
			setup: func(t *testing.T) { t.Setenv("_MY_VAR", "StartsWithUnderscore") },
			want:  "StartsWithUnderscore",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup(t) // Set/unset env vars for this test case
			output := client.substituteDynamicSystemVariables(tc.input, client.currentDotEnvVars)
			assert.Equal(t, tc.want, output)
			// Note: Testing for slog.Warn would require log capture, which is out of scope here.
			// We rely on the fact that if tc.wantErr is true, the output should be the original input.
		})
	}
}
