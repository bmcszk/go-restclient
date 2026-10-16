package restclient_test

import (
	"net/http"
	"path/filepath"
	"time"

	rc "github.com/bmcszk/go-restclient"
)

// responseHeaderValues asserts the Headers value for key equals values.
func (p *parts) responseHeaderValues(key string, values ...string) *parts {
	p.require.Equal(values, p.current().Headers[key])

	return p
}

// responseHeaderEmpty asserts the Headers value for key is empty.
func (p *parts) responseHeaderEmpty(key string) *parts {
	p.require.Empty(p.current().Headers.Get(key))

	return p
}

// responseIsTLS asserts the current response used TLS.
func (p *parts) responseIsTLS() *parts {
	p.require.True(p.current().IsTLS, "expected TLS response, got plain")

	return p
}

// responseTLSVersionMatches asserts the current response TLS version against a regexp.
func (p *parts) responseTLSVersionMatches(pattern string) *parts {
	p.assert.Regexp(pattern, p.current().TLSVersion)

	return p
}

// clientConfigSnapshot records the client options asserted by clientBaseURLIs /
// clientDefaultHeaderIs after aClientWithDefaults runs.
type clientConfigSnapshot struct {
	baseURL     string
	headerKey   string
	headerValue string
	httpTimeout time.Duration
}

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

// responsesValidateAgainstFixture validates responses against a committed response-files fixture.
func (p *parts) responsesValidateAgainstFixture(name string) *parts {
	path := filepath.Join(responseFilesDir, name)
	p.assert.NoError(p.client.ValidateResponses(path, p.responses...))

	return p
}
