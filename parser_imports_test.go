package restclient

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestImports_SimpleImport tests importing a single file
func TestImports_SimpleImport(t *testing.T) {
	// Given: A request file that imports another file
	requestFilePath := "testdata/parser/import_tests/main_simple_import.http"
	client, err := NewClient()
	require.NoError(t, err, "Should create client without error")

	// When: We parse the request file
	parsedFile, err := parseRequestFile(requestFilePath, client, []string{})

	// Then: The parsing should succeed and contain requests from both files
	require.NoError(t, err, "Should parse file without error")
	require.NotNil(t, parsedFile, "Parsed file should not be nil")

	// Check variable scoping - variables from the imported file should be accessible
	assert.Equal(t, "imported_value", parsedFile.FileVariables["imported_var"],
		"Variable from imported file should be accessible")
	assert.Equal(t, "http://localhost:8080", parsedFile.FileVariables["host"],
		"Host variable should be accessible")

	// Check that we have both requests
	require.Len(t, parsedFile.Requests, 2, "Should have two requests (imported and main)")

	// Check the request from the imported file
	importedRequest := parsedFile.Requests[0]
	assert.Equal(t, "Request from imported_file_1", importedRequest.Name,
		"First request should be from imported file")
	assert.Equal(t, "GET", importedRequest.Method, "Method should be GET")
	assert.Equal(t, "{{host}}/imported_request_1", importedRequest.RawURLString,
		"URL should match imported file")

	// Check request body
	bodyReader, err := importedRequest.GetBody()
	require.NoError(t, err, "Should get body without error")
	require.NotNil(t, bodyReader, "Body reader should not be nil")

	bodyBytes, err := io.ReadAll(bodyReader)
	require.NoError(t, err, "Should read body without error")
	bodyString := string(bodyBytes)
	assert.Contains(t, bodyString, "\"key\": \"{{imported_var}}\"",
		"Body should contain variable reference")

	// Check the request from the main file
	mainRequest := parsedFile.Requests[1]
	assert.Equal(t, "Request from main file", mainRequest.Name,
		"Second request should be from main file")
	assert.Equal(t, "GET", mainRequest.Method, "Method should be GET")
	assert.Equal(t, "{{host}}/main_request", mainRequest.RawURLString,
		"URL should match main file")
}

// TestImports_NestedImport tests importing files that import other files
func TestImports_NestedImport(t *testing.T) {
	// Given: A request file that imports another file which imports a third file
	requestFilePath := "testdata/parser/import_tests/main_nested_import.http"
	client, err := NewClient()
	require.NoError(t, err, "Should create client without error")

	// When: We parse the request file
	parsedFile, err := parseRequestFile(requestFilePath, client, []string{})

	// Then: The parsing should succeed and contain requests from all files
	require.NoError(t, err, "Should parse file without error")
	require.NotNil(t, parsedFile, "Parsed file should not be nil")

	// Check variables from all levels
	assert.Equal(t, "http://localhost:8080", parsedFile.FileVariables["host"],
		"Host variable should be accessible")
	assert.Equal(t, "level1_value", parsedFile.FileVariables["level1_var"],
		"Level 1 variable should be accessible")
	assert.Equal(t, "level2_value", parsedFile.FileVariables["level2_var"],
		"Level 2 variable should be accessible")

	// Check that we have all three requests
	require.Len(t, parsedFile.Requests, 3, "Should have three requests (from all levels)")

	// Check the requests in order they should appear
	// Level 2 request (innermost imported file)
	level2Request := parsedFile.Requests[0]
	assert.Equal(t, "Request from imported_file_3_level_2", level2Request.Name,
		"First request should be from level 2 imported file")
	assert.Equal(t, "{{host}}/level2_request", level2Request.RawURLString,
		"URL should match level 2 file")

	// Level 1 request
	level1Request := parsedFile.Requests[1]
	assert.Equal(t, "Request from imported_file_2_level_1", level1Request.Name,
		"Second request should be from level 1 imported file")
	assert.Equal(t, "{{host}}/level1_request", level1Request.RawURLString,
		"URL should match level 1 file")

	// Main request
	mainRequest := parsedFile.Requests[2]
	assert.Equal(t, "Request from main_nested_import", mainRequest.Name,
		"Third request should be from main file")
	assert.Equal(t, "{{host}}/main_nested_request", mainRequest.RawURLString,
		"URL should match main file")
}

// TestImports_VariableOverride tests variables from imported files and host overrides
func TestImports_VariableOverride(t *testing.T) {
	// Given: A request file that imports another file with variables
	requestFilePath := "testdata/parser/import_tests/main_variable_override.http"
	client, err := NewClient()
	require.NoError(t, err, "Should create client without error")

	// When: We parse the request file
	parsedFile, err := parseRequestFile(requestFilePath, client, []string{})

	// Then: The parsing should succeed with variables correctly set
	require.NoError(t, err, "Should parse file without error")
	require.NotNil(t, parsedFile, "Parsed file should not be nil")

	// Check variables from imported file
	assert.Equal(t, "value1_from_imported", parsedFile.FileVariables["var1"],
		"var1 should come from imported file")
	assert.Equal(t, "value2_from_imported", parsedFile.FileVariables["var2"],
		"var2 should come from imported file")

	// Check that main file overrides the host variable
	assert.Equal(t, "http://main-override.com", parsedFile.FileVariables["host"],
		"host should be overridden by main file")

	// Check variable defined only in main file
	assert.Equal(t, "main_value", parsedFile.FileVariables["var_from_main"],
		"var_from_main should be defined in main file")
}

// TestImports_CircularImport tests handling of circular imports (should error)
func TestImports_CircularImport(t *testing.T) {
	// Given: Files with circular imports
	requestFilePath := "testdata/parser/import_tests/main_circular_import_a.http"
	client, err := NewClient()
	require.NoError(t, err, "Should create client without error")

	// When: We parse the request file
	_, err = parseRequestFile(requestFilePath, client, []string{})

	// Then: The parsing should fail with a circular import error
	require.Error(t, err, "Should return error for circular import")
	assert.Contains(t, err.Error(), "circular import",
		"Error should mention circular import")
}

// TestImports_ImportNotFound tests handling of missing import files (should error)
func TestImports_ImportNotFound(t *testing.T) {
	// Given: A file that imports a non-existent file
	requestFilePath := "testdata/parser/import_tests/main_import_not_found.http"
	client, err := NewClient()
	require.NoError(t, err, "Should create client without error")

	// When: We parse the request file
	_, err = parseRequestFile(requestFilePath, client, []string{})

	// Then: The parsing should fail with a file not found error
	require.Error(t, err, "Should return error for file not found")
	assert.Contains(t, err.Error(), "no such file",
		"Error should mention file not found")
}
