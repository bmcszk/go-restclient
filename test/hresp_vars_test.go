package test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type extractHrespDefinesTestCase struct {
	name              string
	hrespContent      string
	expectedDefines   map[string]string
	expectedRemaining string
	expectError       bool
}

func getBasicDefineTestCases() []extractHrespDefinesTestCase {
	return []extractHrespDefinesTestCase{
		{
			name: "no defines",
			hrespContent: `HTTP/1.1 200 OK
Content-Type: application/json

{
  "message": "hello"
}`,
			expectedDefines: map[string]string{},
			expectedRemaining: `HTTP/1.1 200 OK
Content-Type: application/json

{
  "message": "hello"
}`,
			expectError: false,
		},
		{
			name: "simple defines",
			hrespContent: `@name = John Doe
@age = 30
HTTP/1.1 200 OK
Content-Type: application/json

{
  "user": "{{name}}",
  "age": {{age}}
}`,
			expectedDefines: map[string]string{
				"name": "John Doe",
				"age":  "30",
			},
			expectedRemaining: `HTTP/1.1 200 OK
Content-Type: application/json

{
  "user": "{{name}}",
  "age": {{age}}
}`,
			expectError: false,
		},
		{
			name: "defines with extra spaces",
			hrespContent: `  @name  =   John Doe  
@age=30
HTTP/1.1 200 OK`,
			expectedDefines: map[string]string{
				"name": "John Doe",
				"age":  "30",
			},
			expectedRemaining: `HTTP/1.1 200 OK`,
			expectError:       false,
		},
	}
}

func getMixedDefineTestCases() []extractHrespDefinesTestCase {
	return []extractHrespDefinesTestCase{
		{
			name: "defines mixed with comments and blank lines",
			hrespContent: `// This is a comment
@name = Value1

  @key2 = Another Value  

HTTP/1.1 200 OK
Body content here.`,
			expectedDefines: map[string]string{
				"name": "Value1",
				"key2": "Another Value",
			},
			expectedRemaining: `// This is a comment


HTTP/1.1 200 OK
Body content here.`,
			expectError: false,
		},
	}
}

func getMalformedAndEdgeDefineTestCases() []extractHrespDefinesTestCase {
	return []extractHrespDefinesTestCase{
		{
			name: "malformed define - no equals",
			hrespContent: `@name
HTTP/1.1 200 OK`,
			expectedDefines:   map[string]string{},
			expectedRemaining: `HTTP/1.1 200 OK`,
			expectError:       false,
		},
		{
			name: "malformed define - empty name",
			hrespContent: `@ = value
HTTP/1.1 200 OK`,
			expectedDefines:   map[string]string{},
			expectedRemaining: `HTTP/1.1 200 OK`,
			expectError:       false,
		},
		{
			name: "define with empty value",
			hrespContent: `@name = 
HTTP/1.1 200 OK`,
			expectedDefines: map[string]string{
				"name": "",
			},
			expectedRemaining: `HTTP/1.1 200 OK`,
			expectError:       false,
		},
	}
}

func getTestExtractHrespDefinesTestCases() []extractHrespDefinesTestCase {
	var tests []extractHrespDefinesTestCase
	tests = append(tests, getBasicDefineTestCases()...)
	tests = append(tests, getMixedDefineTestCases()...)
	tests = append(tests, getMalformedAndEdgeDefineTestCases()...)
	return tests
}

func TestExtractHrespDefines(t *testing.T) {
	tests := getTestExtractHrespDefinesTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defines, remaining, err := ExtractHrespDefines(tt.hrespContent)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedDefines, defines)
				assert.Equal(t, strings.ReplaceAll(tt.expectedRemaining, "\r\n", "\n"),
				strings.ReplaceAll(remaining, "\r\n", "\n"))
			}
		})
	}
}
