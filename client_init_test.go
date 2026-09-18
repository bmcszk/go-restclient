// Code in this file is migrated from test/client_init.go (RunNewClient, RunNewClient_WithOptions).
package restclient_test

import (
	"testing"

	rc "github.com/bmcszk/go-restclient"
)

// PRD-COMMENT: FR_CLIENT_INIT_DEFAULT - Client Initialization: Default
// Corresponds to: The ability to create a new HTTP client instance with default
// configurations (e.g., standard http.Client, no base URL, empty default headers).
// This test verifies that `rc.NewClient()` without options returns a valid client with expected default values.
func TestNewClient(t *testing.T) {
	given, _, then := newParts(t)

	given.
		aClient()

	then.
		clientExists().and().
		clientBaseURLIsEmpty().and().
		clientDefaultHeadersEmpty()
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
	given, _, then := newParts(t)

	given.
		aClientWithDefaults()

	then.
		clientExists().and().
		clientBaseURLIs("https://api.example.com").and().
		clientDefaultHeaderIs("X-Default", "DefaultValue")

	t.Run("nil http client option falls back to default client", func(t *testing.T) {
		givenInner, _, thenInner := newParts(t)

		givenInner.
			aClient(rc.WithHTTPClient(nil))

		thenInner.
			clientExists().and().
			clientBaseURLIsEmpty()
	})
}
