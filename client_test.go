package restclient_test

import (
	"testing"

	"github.com/bmcszk/go-restclient/test"
)

// Edge case tests

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

func TestExecuteFile_WithFakerPersonData(t *testing.T) {
	test.RunExecuteFile_WithFakerPersonData(t)
}

func TestExecuteFile_WithContactAndInternetFakerData(t *testing.T) {
	test.RunExecuteFile_WithContactAndInternetFakerData(t)
}

func TestExecuteFile_WithIndirectEnvironmentVariables(t *testing.T) {
	test.RunExecuteFile_WithIndirectEnvironmentVariables(t)
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

// GraphQL tests
func TestExecuteFile_GraphQLBasicQuery(t *testing.T) {
	test.RunExecuteFile_GraphQLBasicQuery(t)
}

func TestExecuteFile_GraphQLQueryWithVariables(t *testing.T) {
	test.RunExecuteFile_GraphQLQueryWithVariables(t)
}

func TestExecuteFile_GraphQLMutation(t *testing.T) {
	test.RunExecuteFile_GraphQLMutation(t)
}

func TestExecuteFile_GraphQLFragments(t *testing.T) {
	test.RunExecuteFile_GraphQLFragments(t)
}

func TestExecuteFile_GraphQLIntrospection(t *testing.T) {
	test.RunExecuteFile_GraphQLIntrospection(t)
}

func TestExecuteFile_GraphQLErrorHandling(t *testing.T) {
	test.RunExecuteFile_GraphQLErrorHandling(t)
}

func TestExecuteFile_GraphQLBatchQueries(t *testing.T) {
	test.RunExecuteFile_GraphQLBatchQueries(t)
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

// GraphQL tests

// Test helper tests
func TestCreateTestFileFromTemplate_DebugOutput(t *testing.T) {
	test.RunCreateTestFileFromTemplate_DebugOutput(t)
}
