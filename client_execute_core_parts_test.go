package restclient_test

import (
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"time"

	rc "github.com/bmcszk/go-restclient"
)

// aTLSServer hosts an echo handler over httptest TLS (self-signed cert).
func (s *serverParts) aTLSServer() *serverParts {
	p := s.parts
	srv := httptest.NewTLSServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		p.capturedRequests = append(p.capturedRequests, r)
		p.capturedBodies = append(p.capturedBodies, string(body))
		p.capturedRequestTimes = append(p.capturedRequestTimes, time.Now())
		p.requestHits.Add(1)
	}))
	p.servers = append(p.servers, srv)
	p.serverURL = srv.URL
	p.Cleanup(srv.Close)

	return s
}

// aTLSInsecureClient builds a restclient trusting the httptest self-signed cert.
func (s *serverParts) aTLSInsecureClient() *serverParts {
	p := s.parts
	httpClient := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}

	p.aClient(rc.WithHTTPClient(httpClient))

	return s
}

// aMockTransportClient builds the client under test with a mock round tripper that
// records the outgoing request into parts.intercepted and answers 200 with a fixed body.
func (s *serverParts) aMockTransportClient() *serverParts {
	p := s.parts
	transport := &mockRoundTripper{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			p.intercepted = req.Clone(req.Context())

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("mocked response")),
				Header:     make(http.Header),
			}, nil
		},
	}

	p.aClient(rc.WithHTTPClient(&http.Client{Transport: transport}))

	return s
}

// mockRoundTripper adapts a function into an http.RoundTripper for the DSL.
type mockRoundTripper struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

// RoundTrip delegates to the configured function.
func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.RoundTripFunc != nil {
		return m.RoundTripFunc(req)
	}

	return nil, errors.New("RoundTripFunc not set")
}

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
