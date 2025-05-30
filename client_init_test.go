package restclient

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	// Given
	// No specific setup needed

	// When
	c, err := NewClient()

	// Then
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.NotNil(t, c.httpClient)
	assert.Empty(t, c.BaseURL)
	assert.NotNil(t, c.DefaultHeaders)
	assert.Empty(t, c.DefaultHeaders)
}

func TestNewClient_WithOptions(t *testing.T) {
	// Given
	customHTTPClient := &http.Client{Timeout: 15 * time.Second}
	baseURL := "https://api.example.com"
	defaultHeaderKey := "X-Default"
	defaultHeaderValue := "DefaultValue"

	// When
	c, err := NewClient(
		WithHTTPClient(customHTTPClient),
		WithBaseURL(baseURL),
		WithDefaultHeader(defaultHeaderKey, defaultHeaderValue),
	)

	// Then
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, customHTTPClient, c.httpClient)
	assert.Equal(t, baseURL, c.BaseURL)
	assert.Equal(t, defaultHeaderValue, c.DefaultHeaders.Get(defaultHeaderKey))

	// And when testing nil http client option
	c2, err2 := NewClient(WithHTTPClient(nil))

	// Then
	require.NoError(t, err2)
	require.NotNil(t, c2.httpClient, "httpClient should default if nil provided")
}
