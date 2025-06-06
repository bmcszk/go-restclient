package restclient

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseHTTPTestFile is a helper function to parse HTTP test files with error handling
func parseHTTPTestFile(t *testing.T, filename string) *ParsedFile {
	// Given
	testFilePath := filepath.Join("testdata", "request_body", filename)

	// When
	parsedFile, err := parseRequestFile(testFilePath, nil, make([]string, 0))

	// Then
	require.NoError(t, err)
	require.NotNil(t, parsedFile)
	return parsedFile
}

// assertRequestBasics verifies the common request fields (method, URL, content-type)
func assertRequestBasics(t *testing.T, req *Request, method, url, contentType string) {
	assert.Equal(t, method, req.Method)
	assert.Equal(t, url, req.URL.String())
	if contentType != "" {
		require.Contains(t, req.Headers, "Content-Type")
		require.Len(t, req.Headers["Content-Type"], 1)
		assert.Equal(t, contentType, req.Headers["Content-Type"][0])
	}
}

func TestJsonRequestBodies(t *testing.T) {
	// Parse the HTTP test file
	parsedFile := parseHTTPTestFile(t, "json_body.http")

	// Then
	require.Len(t, parsedFile.Requests, 3)

	// Basic JSON Body Test
	r1 := parsedFile.Requests[0]
	assertRequestBasics(t, r1, "POST", "https://example.com/api/users", "application/json")
	expectedBody := `{
  "name": "Test User",
  "email": "test@example.com",
  "role": "admin"
}`
	assert.Equal(t, expectedBody, r1.RawBody)

	// JSON Body with Array Test
	r2 := parsedFile.Requests[1]
	assert.Equal(t, "POST", r2.Method)
	assert.Equal(t, "https://example.com/api/users/batch", r2.URL.String())
	require.Contains(t, r2.Headers, "Content-Type")
	require.Len(t, r2.Headers["Content-Type"], 1)
	assert.Equal(t, "application/json", r2.Headers["Content-Type"][0])
	expectedBody = `[
  {
    "name": "User 1",
    "email": "user1@example.com"
  },
  {
    "name": "User 2", 
    "email": "user2@example.com"
  }
]`
	assert.Equal(t, expectedBody, r2.RawBody)

	// JSON Body with Nested Objects
	r3 := parsedFile.Requests[2]
	assertRequestBasics(t, r3, "POST", "https://example.com/api/orders", "application/json")
	// Using Contains to avoid whitespace comparison issues with deeply nested JSON
	assert.Contains(t, r3.RawBody, `"order"`)
	assert.Contains(t, r3.RawBody, `"customer"`)
	assert.Contains(t, r3.RawBody, `"items"`)
	assert.Contains(t, r3.RawBody, `"Product 1"`)
	assert.Contains(t, r3.RawBody, `"Product 2"`)
}

func TestFileBasedBodies(t *testing.T) {
	// Parse the HTTP test file
	parsedFile := parseHTTPTestFile(t, "file_body.http")

	// Then
	require.Len(t, parsedFile.Requests, 3)

	// File as Request Body Test - JSON
	r1 := parsedFile.Requests[0]
	assertRequestBasics(t, r1, "POST", "https://example.com/api/documents", "application/json")
	// The RawBody will contain the file reference, not the actual content
	assert.Equal(t, "< ./testdata/request_body/sample_payload.json", r1.RawBody)

	// File as Request Body - Text File
	r2 := parsedFile.Requests[1]
	assertRequestBasics(t, r2, "POST", "https://example.com/api/text", "text/plain")
	assert.Equal(t, "< ./testdata/request_body/sample_text.txt", r2.RawBody)

	// File as Request Body - XML
	r3 := parsedFile.Requests[2]
	assertRequestBasics(t, r3, "POST", "https://example.com/api/xml", "application/xml")
	assert.Equal(t, "< ./testdata/request_body/sample_data.xml", r3.RawBody)
}

func TestFormUrlEncodedBodies(t *testing.T) {
	// Parse the HTTP test file
	parsedFile := parseHTTPTestFile(t, "form_urlencoded.http")

	// Then
	require.Len(t, parsedFile.Requests, 3)

	// Basic Form URL Encoded
	r1 := parsedFile.Requests[0]
	assertRequestBasics(t, r1, "POST", "https://example.com/api/login", "application/x-www-form-urlencoded")
	assert.Equal(t, "username=testuser&password=testpass123", r1.RawBody)

	// Multiple Fields Form URL Encoded
	r2 := parsedFile.Requests[1]
	assertRequestBasics(t, r2, "POST", "https://example.com/api/subscribe", "application/x-www-form-urlencoded")
	assert.Equal(t, "email=user@example.com&name=Test+User&interests=sports&interests=technology&subscribe=true", r2.RawBody)

	// Special Characters Form URL Encoded
	r3 := parsedFile.Requests[2]
	assertRequestBasics(t, r3, "POST", "https://example.com/api/search", "application/x-www-form-urlencoded")
	assert.Equal(t, "query=test%20search&filter=date%3E2023-01-01&sort=relevance", r3.RawBody)
}

func TestMultipartFormDataBodies(t *testing.T) {
	// Parse the HTTP test file
	parsedFile := parseHTTPTestFile(t, "multipart_form_data.http")

	// Then
	require.Len(t, parsedFile.Requests, 2)

	// Basic Multipart Form Data Test
	r1 := parsedFile.Requests[0]
	assertRequestBasics(t, r1, "POST", "https://example.com/api/profile", "")
	require.Contains(t, r1.Headers, "Content-Type")
	require.Len(t, r1.Headers["Content-Type"], 1)
	assert.True(t, strings.HasPrefix(r1.Headers["Content-Type"][0], "multipart/form-data; boundary="))
	bodyStr := r1.RawBody
	assert.Contains(t, bodyStr, "Content-Disposition: form-data; name=\"username\"")
	assert.Contains(t, bodyStr, "testuser")
	assert.Contains(t, bodyStr, "Content-Disposition: form-data; name=\"email\"")
	assert.Contains(t, bodyStr, "test@example.com")

	// Complex Multipart Form Data
	r2 := parsedFile.Requests[1]
	assertRequestBasics(t, r2, "POST", "https://example.com/api/registration", "")
	require.Contains(t, r2.Headers, "Content-Type")
	require.Len(t, r2.Headers["Content-Type"], 1)
	assert.True(t, strings.HasPrefix(r2.Headers["Content-Type"][0], "multipart/form-data; boundary="))
	bodyStr = r2.RawBody
	assert.Contains(t, bodyStr, "Content-Disposition: form-data; name=\"username\"")
	assert.Contains(t, bodyStr, "newuser123")
	assert.Contains(t, bodyStr, "Content-Disposition: form-data; name=\"email\"")
	assert.Contains(t, bodyStr, "newuser@example.com")
	assert.Contains(t, bodyStr, "Content-Disposition: form-data; name=\"age\"")
	assert.Contains(t, bodyStr, "25")
	assert.Contains(t, bodyStr, "Content-Disposition: form-data; name=\"subscribe\"")
	assert.Contains(t, bodyStr, "true")
}

func TestFileUploadBodies(t *testing.T) {
	// Parse the HTTP test file
	parsedFile := parseHTTPTestFile(t, "file_uploads.http")

	// Then
	require.Len(t, parsedFile.Requests, 2)

	// Single File Upload Test
	r1 := parsedFile.Requests[0]
	assertRequestBasics(t, r1, "POST", "https://example.com/api/upload", "")
	require.Contains(t, r1.Headers, "Content-Type")
	require.Len(t, r1.Headers["Content-Type"], 1)
	assert.True(t, strings.HasPrefix(r1.Headers["Content-Type"][0], "multipart/form-data; boundary="))
	bodyStr := r1.RawBody
	// File references are preserved in RawBody, not the actual content
	assert.Contains(t, bodyStr, "Content-Disposition: form-data; name=\"file\"; filename=\"image.jpg\"")
	assert.Contains(t, bodyStr, "Content-Type: image/jpeg")
	assert.Contains(t, bodyStr, "< ./testdata/request_body/sample_image.jpg")

	// Multiple File Upload Test
	r2 := parsedFile.Requests[1]
	assertRequestBasics(t, r2, "POST", "https://example.com/api/uploads", "")
	require.Contains(t, r2.Headers, "Content-Type")
	require.Len(t, r2.Headers["Content-Type"], 1)
	assert.True(t, strings.HasPrefix(r2.Headers["Content-Type"][0], "multipart/form-data; boundary="))
	bodyStr = r2.RawBody
	assert.Contains(t, bodyStr, "Content-Disposition: form-data; name=\"files\"; filename=\"document.pdf\"")
	assert.Contains(t, bodyStr, "Content-Type: application/pdf")
	assert.Contains(t, bodyStr, "< ./testdata/request_body/sample_document.pdf")

	assert.Contains(t, bodyStr, "Content-Disposition: form-data; name=\"files\"; filename=\"image.png\"")
	assert.Contains(t, bodyStr, "Content-Type: image/png")
	assert.Contains(t, bodyStr, "< ./testdata/request_body/sample_image.png")

	assert.Contains(t, bodyStr, "Content-Disposition: form-data; name=\"description\"")
	assert.Contains(t, bodyStr, "Test files upload")
}
