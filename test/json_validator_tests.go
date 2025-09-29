package test

import (
	"testing"

	rc "github.com/bmcszk/go-restclient"

	"github.com/stretchr/testify/assert"
)

// RunValidateResponses_JSON_WhitespaceComparison tests JSON comparison with different whitespace
func RunValidateResponses_JSON_WhitespaceComparison(t *testing.T) {
	t.Helper()

	response := &rc.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		BodyString: "{\"key\":\"value\"}",
	}

	client, err := rc.NewClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	responses := []*rc.Response{response}
	validationErr := client.ValidateResponses(
		"test/data/http_response_files/validator_json_indentation_different.hresp",
		responses...,
	)

	assert.NoError(t, validationErr, "JSON with different whitespace should validate successfully")
}

// RunValidateResponses_JSON_WithPlaceholders tests JSON comparison with placeholders
func RunValidateResponses_JSON_WithPlaceholders(t *testing.T) {
	t.Helper()

	response := &rc.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		BodyString: "{\"id\": \"550e8400-e29b-41d4-a716-446655440000\"}",
	}

	client, err := rc.NewClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	responses := []*rc.Response{response}
	validationErr := client.ValidateResponses(
		"test/data/http_response_files/validator_json_placeholders_formatting.hresp",
		responses...,
	)

	assert.NoError(t, validationErr, "JSON with placeholders should validate successfully")
}
