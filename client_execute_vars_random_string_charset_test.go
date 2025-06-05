package restclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRandomStringFromCharset tests the randomStringFromCharset helper function.
func TestRandomStringFromCharset(t *testing.T) {
	tests := []struct {
		name     string
		length   int
		charset  string
		wantLen  int
		assertFn func(t *testing.T, s string, charset string)
	}{
		{
			name:    "alphabetic_10",
			length:  10,
			charset: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ",
			wantLen: 10,
			assertFn: func(t *testing.T, s string, charset string) {
				for _, r := range s {
					assert.Contains(t, charset, string(r))
				}
			},
		},
		{
			name:    "numeric_5",
			length:  5,
			charset: "0123456789",
			wantLen: 5,
			assertFn: func(t *testing.T, s string, charset string) {
				for _, r := range s {
					assert.Contains(t, charset, string(r))
				}
			},
		},
		{
			name:    "empty_charset",
			length:  5,
			charset: "",
			wantLen: 0,
		},
		{
			name:    "zero_length",
			length:  0,
			charset: "abc",
			wantLen: 0,
			assertFn: func(t *testing.T, s string, charset string) {
			},
		},
		{
			name:    "negative_length",
			length:  -5,
			charset: "abc",
			wantLen: 0,
			assertFn: func(t *testing.T, s string, charset string) {
			},
		},
	}

	for _, testCase := range tests {
		capturedTC := testCase // Explicitly capture the current test case
		t.Run(capturedTC.name, func(t *testing.T) {
			if capturedTC.charset == "" && capturedTC.length > 0 {
				s := randomStringFromCharset(capturedTC.length, capturedTC.charset)
				assert.Len(t, s, capturedTC.wantLen, "String length mismatch")

			} else {
				s := randomStringFromCharset(capturedTC.length, capturedTC.charset)
				assert.Len(t, s, capturedTC.wantLen, "String length mismatch")
				if capturedTC.assertFn != nil && capturedTC.wantLen > 0 {
					capturedTC.assertFn(t, s, capturedTC.charset)
				}
			}
		})
	}
}
