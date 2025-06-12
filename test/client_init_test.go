package test

import (
	rc "github.com/bmcszk/go-restclient"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PRD-COMMENT: FR_CLIENT_INIT_DEFAULT - Client Initialization: Default
// Corresponds to: The ability to create a new HTTP client instance with default
// configurations (e.g., standard http.Client, no base URL, empty default headers).
// This test verifies that `rc.NewClient()` without options returns a valid client with expected default values.
func TestNewClient(t *testing.T) {
	// Given
	// No specific setup needed

	// When
	c, err := rc.NewClient()

	// Then
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Empty(t, c.BaseURL)
	assert.NotNil(t, c.DefaultHeaders)
	assert.Empty(t, c.DefaultHeaders)
}

// PRD-COMMENT: FR_CLIENT_INIT_OPTIONS - Client Initialization: With Options
// Corresponds to: The ability to create a new HTTP client instance configured with
// specific options, such as a custom underlying `http.Client` (FR_CLIENT_CONFIG_HTTPCLIENT),
// a base URL (FR_CLIENT_CONFIG_BASEURL), and default headers (FR_CLIENT_CONFIG_HEADERS).
// This test verifies that `rc.NewClient()` with options (e.g., `WithHTTPClient`,
// `WithBaseURL`, `WithDefaultHeader`) correctly applies these configurations to the new
// client instance. It also checks that providing a nil http.Client results in a default
// client being used.
func TestNewClient_WithOptions(t *testing.T) {
	// Given
	customHTTPClient := &http.Client{Timeout: 15 * time.Second}
	baseURL := "https://api.example.com"
	defaultHeaderKey := "X-Default"
	defaultHeaderValue := "DefaultValue"

	// When
	c, err := rc.NewClient(
		rc.WithHTTPClient(customHTTPClient),
		rc.WithBaseURL(baseURL),
		rc.WithDefaultHeader(defaultHeaderKey, defaultHeaderValue),
	)

	// Then
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, baseURL, c.BaseURL)
	assert.Equal(t, defaultHeaderValue, c.DefaultHeaders.Get(defaultHeaderKey))

	// And when testing nil http client option
	c2, err2 := rc.NewClient(rc.WithHTTPClient(nil))

	// Then
	require.NoError(t, err2)
	require.NotNil(t, c2)
}
