package restclient_test

import (
	"net/http"
	"testing"
	"time"

	rc "github.com/bmcszk/go-restclient"
)

// aClientWithDefaults builds a client from fixed defaults recorded in parts.clientConfig.
func (p *parts) aClientWithDefaults() *parts {
	p.clientConfig = clientConfigSnapshot{
		baseURL:     "https://api.example.com",
		headerKey:   "X-Default",
		headerValue: "DefaultValue",
		httpTimeout: 15 * time.Second,
	}

	return p.aClient(
		rc.WithHTTPClient(&http.Client{Timeout: p.clientConfig.httpTimeout}),
		rc.WithBaseURL(p.clientConfig.baseURL),
		rc.WithDefaultHeader(p.clientConfig.headerKey, p.clientConfig.headerValue),
	)
}

// clientBaseURLIs asserts the client's BaseURL matches the recorded config snapshot.
func (p *parts) clientBaseURLIs(_ string) *parts {
	p.require.Equal(p.clientConfig.baseURL, p.client.BaseURL)

	return p
}

// clientBaseURLIsEmpty asserts the client's BaseURL is empty.
func (p *parts) clientBaseURLIsEmpty() *parts {
	p.assert.Empty(p.client.BaseURL)

	return p
}

// clientDefaultHeaderIs asserts the client's DefaultHeaders value for key equals the
// recorded config snapshot.
func (p *parts) clientDefaultHeaderIs(key, _ string) *parts {
	p.require.Equal(p.clientConfig.headerValue, p.client.DefaultHeaders.Get(key))

	return p
}

// clientExists asserts the client is non-nil.
func (p *parts) clientExists() *parts {
	p.require.NotNil(p.client)

	return p
}

// clientDefaultHeadersEmpty asserts the client's DefaultHeaders map is non-nil and empty.
func (p *parts) clientDefaultHeadersEmpty() *parts {
	p.require.NotNil(p.client.DefaultHeaders)
	p.assert.Empty(p.client.DefaultHeaders)

	return p
}

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
