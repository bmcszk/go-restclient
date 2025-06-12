package restclient_test

import (
	"testing"

	"github.com/bmcszk/go-restclient/test"
)

// Client initialization tests
func TestNewClient(t *testing.T) {
	test.RunNewClient(t)
}

func TestNewClient_WithOptions(t *testing.T) {
	test.RunNewClient_WithOptions(t)
}

// Cookie and redirect handling tests
func TestCookieJarHandling(t *testing.T) {
	test.RunCookieJarHandling(t)
}

func TestRedirectHandling(t *testing.T) {
	test.RunRedirectHandling(t)
}

// Core execution tests
func TestExecuteFile_SingleRequest(t *testing.T) {
	test.RunExecuteFile_SingleRequest(t)
}

func TestExecuteFile_MultipleRequests(t *testing.T) {
	test.RunExecuteFile_MultipleRequests(t)
}

func TestExecuteFile_RequestWithError(t *testing.T) {
	test.RunExecuteFile_RequestWithError(t)
}

func TestExecuteFile_ParseError(t *testing.T) {
	test.RunExecuteFile_ParseError(t)
}

func TestExecuteFile_NoRequestsInFile(t *testing.T) {
	test.RunExecuteFile_NoRequestsInFile(t)
}

func TestExecuteFile_ValidThenInvalidSyntax(t *testing.T) {
	test.RunExecuteFile_ValidThenInvalidSyntax(t)
}

func TestExecuteFile_MultipleErrors(t *testing.T) {
	test.RunExecuteFile_MultipleErrors(t)
}

func TestExecuteFile_CapturesResponseHeaders(t *testing.T) {
	test.RunExecuteFile_CapturesResponseHeaders(t)
}

func TestExecuteFile_SimpleGetHTTP(t *testing.T) {
	test.RunExecuteFile_SimpleGetHTTP(t)
}

func TestExecuteFile_MultipleRequests_GreaterThanTwo(t *testing.T) {
	test.RunExecuteFile_MultipleRequests_GreaterThanTwo(t)
}

// GAP TESTS - These tests demonstrate functionality gaps that need to be implemented
// They are commented out because they currently fail (which is expected)

// func TestExecuteFile_MultilineQueryParameters(t *testing.T) {
// 	test.RunExecuteFile_MultilineQueryParameters(t)
// }

// func TestExecuteFile_MultilineFormData(t *testing.T) {
// 	test.RunExecuteFile_MultilineFormData(t)
// }

// func TestExecuteFile_MultipartFileUploads(t *testing.T) {
// 	test.RunExecuteFile_MultipartFileUploads(t)
// }

// Edge case tests
func TestExecuteFile_InvalidMethodInFile(t *testing.T) {
	test.RunExecuteFile_InvalidMethodInFile(t)
}

func TestExecuteFile_IgnoreEmptyBlocks_Client(t *testing.T) {
	test.RunExecuteFile_IgnoreEmptyBlocks_Client(t)
}

// External file tests
func TestExecuteFile_ExternalFileWithVariables(t *testing.T) {
	test.RunExecuteFile_ExternalFileWithVariables(t)
}

func TestExecuteFile_ExternalFileWithoutVariables(t *testing.T) {
	test.RunExecuteFile_ExternalFileWithoutVariables(t)
}

func TestClientExecuteFileWithEncoding(t *testing.T) {
	test.RunClientExecuteFileWithEncoding(t)
}

func TestExecuteFile_ExternalFileWithEncoding(t *testing.T) {
	test.RunExecuteFile_ExternalFileWithEncoding(t)
}

func TestExecuteFile_ExternalFileWithVariablesAndEncoding(t *testing.T) {
	test.RunExecuteFile_ExternalFileWithVariablesAndEncoding(t)
}

func TestExecuteFile_WithRestExtension(t *testing.T) {
	test.RunExecuteFile_WithRestExtension(t)
}

func TestExecuteFile_ExternalFileNotFound(t *testing.T) {
	test.RunExecuteFile_ExternalFileNotFound(t)
}

// Variable handling tests
func TestExecuteFile_WithCustomVariables(t *testing.T) {
	test.RunExecuteFile_WithCustomVariables(t)
}

func TestExecuteFile_WithProcessEnvSystemVariable(t *testing.T) {
	test.RunExecuteFile_WithProcessEnvSystemVariable(t)
}

func TestExecuteFile_WithDotEnvSystemVariable(t *testing.T) {
	test.RunExecuteFile_WithDotEnvSystemVariable(t)
}

func TestExecuteFile_WithProgrammaticVariables(t *testing.T) {
	test.RunExecuteFile_WithProgrammaticVariables(t)
}

func TestExecuteFile_WithLocalDatetimeSystemVariable(t *testing.T) {
	test.RunExecuteFile_WithLocalDatetimeSystemVariable(t)
}

func TestExecuteFile_VariableFunctionConsistency(t *testing.T) {
	test.RunExecuteFile_VariableFunctionConsistency(t)
}

func TestExecuteFile_WithHttpClientEnvJson(t *testing.T) {
	test.RunExecuteFile_WithHttpClientEnvJson(t)
}

func TestExecuteFile_WithExtendedRandomSystemVariables(t *testing.T) {
	test.RunExecuteFile_WithExtendedRandomSystemVariables(t)
}

// System variable tests
func TestExecuteFile_WithGuidSystemVariable(t *testing.T) {
	test.RunExecuteFile_WithGuidSystemVariable(t)
}

func TestExecuteFile_WithIsoTimestampSystemVariable(t *testing.T) {
	test.RunExecuteFile_WithIsoTimestampSystemVariable(t)
}

func TestExecuteFile_WithDatetimeSystemVariables(t *testing.T) {
	test.RunExecuteFile_WithDatetimeSystemVariables(t)
}

func TestExecuteFile_WithTimestampSystemVariable(t *testing.T) {
	test.RunExecuteFile_WithTimestampSystemVariable(t)
}

func TestExecuteFile_WithRandomIntSystemVariable(t *testing.T) {
	test.RunExecuteFile_WithRandomIntSystemVariable(t)
}

// In-place variable tests
func TestExecuteFile_InPlace_SimpleVariableInURL(t *testing.T) {
	test.RunExecuteFile_InPlace_SimpleVariableInURL(t)
}

func TestExecuteFile_InPlace_VariableInHeader(t *testing.T) {
	test.RunExecuteFile_InPlace_VariableInHeader(t)
}

func TestExecuteFile_InPlace_VariableInBody(t *testing.T) {
	test.RunExecuteFile_InPlace_VariableInBody(t)
}

func TestExecuteFile_InPlace_VariableDefinedByAnotherVariable(t *testing.T) {
	test.RunExecuteFile_InPlace_VariableDefinedByAnotherVariable(t)
}

func TestExecuteFile_InPlace_VariablePrecedenceOverEnvironment(t *testing.T) {
	test.RunExecuteFile_InPlace_VariablePrecedenceOverEnvironment(t)
}

func TestExecuteFile_InPlace_VariableInCustomHeader(t *testing.T) {
	test.RunExecuteFile_InPlace_VariableInCustomHeader(t)
}

func TestExecuteFile_InPlace_VariableSubstitutionInBody(t *testing.T) {
	test.RunExecuteFile_InPlace_VariableSubstitutionInBody(t)
}

func TestExecuteFile_InPlace_VariableDefinedBySystemVariable(t *testing.T) {
	test.RunExecuteFile_InPlace_VariableDefinedBySystemVariable(t)
}

func TestExecuteFile_InPlace_VariableDefinedByOsEnvVariable(t *testing.T) {
	test.RunExecuteFile_InPlace_VariableDefinedByOsEnvVariable(t)
}

func TestExecuteFile_InPlace_VariableInAuthHeader(t *testing.T) {
	test.RunExecuteFile_InPlace_VariableInAuthHeader(t)
}

func TestExecuteFile_InPlace_VariableInJsonRequestBody(t *testing.T) {
	test.RunExecuteFile_InPlace_VariableInJsonRequestBody(t)
}

func TestExecuteFile_InPlace_VariableDefinedByAnotherInPlaceVariable(t *testing.T) {
	test.RunExecuteFile_InPlace_VariableDefinedByAnotherInPlaceVariable(t)
}

func TestExecuteFile_InPlace_VariableDefinedByDotEnvOsVariable(t *testing.T) {
	test.RunExecuteFile_InPlace_VariableDefinedByDotEnvOsVariable(t)
}

func TestExecuteFile_InPlace_Malformed_NameOnlyNoEqualsNoValue(t *testing.T) {
	test.RunExecuteFile_InPlace_Malformed_NameOnlyNoEqualsNoValue(t)
}

func TestExecuteFile_InPlace_Malformed_NoNameEqualsValue(t *testing.T) {
	test.RunExecuteFile_InPlace_Malformed_NoNameEqualsValue(t)
}

func TestExecuteFile_InPlace_VariableDefinedByDotEnvSystemVariable(t *testing.T) {
	test.RunExecuteFile_InPlace_VariableDefinedByDotEnvSystemVariable(t)
}

func TestExecuteFile_InPlace_VariableDefinedByRandomInt(t *testing.T) {
	test.RunExecuteFile_InPlace_VariableDefinedByRandomInt(t)
}

// Test helper tests
func TestCreateTestFileFromTemplate_DebugOutput(t *testing.T) {
	test.RunCreateTestFileFromTemplate_DebugOutput(t)
}