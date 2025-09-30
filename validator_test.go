package restclient_test

import (
	"testing"

	"github.com/bmcszk/go-restclient/test"
)

// General validator tests
func TestValidateResponses_WithSampleFile(t *testing.T) {
	test.RunValidateResponses_WithSampleFile(t)
}

func TestValidateResponses_PartialExpected(t *testing.T) {
	test.RunValidateResponses_PartialExpected(t)
}

// Setup and error handling tests
func TestValidateResponses_NilAndEmptyActuals(t *testing.T) {
	test.RunValidateResponses_NilAndEmptyActuals(t)
}

func TestValidateResponses_FileErrors(t *testing.T) {
	test.RunValidateResponses_FileErrors(t)
}

// Status validation tests
func TestValidateResponses_StatusString(t *testing.T) {
	test.RunValidateResponses_StatusString(t)
}

func TestValidateResponses_StatusCode(t *testing.T) {
	test.RunValidateResponses_StatusCode(t)
}

// Header validation tests
func TestValidateResponses_Headers(t *testing.T) {
	test.RunValidateResponses_Headers(t)
}

func TestValidateResponses_HeadersContain(t *testing.T) {
	test.RunValidateResponses_HeadersContain(t)
}

// Body validation tests
func TestValidateResponses_Body_ExactMatch(t *testing.T) {
	test.RunValidateResponses_Body_ExactMatch(t)
}

func TestValidateResponses_BodyContains(t *testing.T) {
	test.RunValidateResponses_BodyContains(t)
}

func TestValidateResponses_BodyNotContains(t *testing.T) {
	test.RunValidateResponses_BodyNotContains(t)
}

// Placeholder validation tests
func TestValidateResponses_BodyRegexpPlaceholder(t *testing.T) {
	test.RunValidateResponses_BodyRegexpPlaceholder(t)
}

func TestValidateResponses_BodyAnyGuidPlaceholder(t *testing.T) {
	test.RunValidateResponses_BodyAnyGuidPlaceholder(t)
}

func TestValidateResponses_BodyAnyTimestampPlaceholder(t *testing.T) {
	test.RunValidateResponses_BodyAnyTimestampPlaceholder(t)
}

func TestValidateResponses_BodyAnyDatetimePlaceholder(t *testing.T) {
	test.RunValidateResponses_BodyAnyDatetimePlaceholder(t)
}

func TestValidateResponses_BodyAnyPlaceholder(t *testing.T) {
	test.RunValidateResponses_BodyAnyPlaceholder(t)
}

// JSON comparison tests
func TestValidateResponses_JSON_WhitespaceComparison(t *testing.T) {
	test.RunValidateResponses_JSON_WhitespaceComparison(t)
}

func TestValidateResponses_JSON_WithPlaceholders(t *testing.T) {
	test.RunValidateResponses_JSON_WithPlaceholders(t)
}
