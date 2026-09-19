package restclient_test

import (
	"testing"

	rc "github.com/bmcszk/go-restclient"
)

// configurations (e.g., standard http.Client, no base URL, empty default headers).

func TestNewClient(t *testing.T) {
	given, _, then := newParts(t)

	given.
		aClient()

	then.
		clientExists().and().
		clientBaseURLIsEmpty().and().
		clientDefaultHeadersEmpty()
}

func TestNewClient_WithOptions(t *testing.T) {
	given, _, then := newParts(t)

	given.
		aClientWithDefaults()

	then.
		clientExists().and().
		clientBaseURLIs("https://api.example.com").and().
		clientDefaultHeaderIs("X-Default", "DefaultValue")
}
func TestNewClient_NilHTTPClientOption(t *testing.T) {
	given, _, then := newParts(t)

	given.
		aClient(rc.WithHTTPClient(nil))

	then.
		clientExists().and().
		clientBaseURLIsEmpty()
}
