package restclient_test

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	rc "github.com/bmcszk/go-restclient"
)

// serverParts holds the server domain state of the fluent test DSL: the httptest
// servers started by the server builders, the address of the most recent one,
// its scheme/host/port variables and the request counter shared with the DSL.
type serverParts struct {
	// servers holds every httptest server started by the server builders.
	servers []*httptest.Server
	// serverURL holds the address of the most recently started test server.
	serverURL string
	// share the name requestCount in Go.
	requestHits atomic.Int64
	// activeServerVars holds the scheme/host/port variables of the running test server.
	activeServerVars map[string]any
}

// newServerParts returns the given/when/then fixture for the server-domain fluent DSL.
func newServerParts(t *testing.T) (given, when, then *testFixture) {
	t.Helper()

	return newParts(t)
}

// aHttpServer counts requests; bodies are read eagerly because net/http closes them on return.
func (p *testFixture) aHttpServer(h http.HandlerFunc) *testFixture {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		p.capturedRequests = append(p.capturedRequests, r)
		p.capturedBodies = append(p.capturedBodies, string(body))
		p.capturedRequestTimes = append(p.capturedRequestTimes, time.Now())
		p.requestHits.Add(1)
		h(w, r)
	}))
	p.servers = append(p.servers, srv)
	p.serverURL = srv.URL
	p.Cleanup(srv.Close)

	return p
}

// cannedRoute is one aCannedServer route; validJSON also requires a json.Valid body.
type cannedRoute struct {
	method    string
	code      int
	body      string
	validJSON bool
}

// serveCannedRoute writes the cannedRoute response for r: 405 on method mismatch,
// 400 on validJSON with an unparseable body. The caller handled unknown paths.
func serveCannedRoute(w http.ResponseWriter, r *http.Request, route cannedRoute) {
	if r.Method != route.method {
		w.WriteHeader(http.StatusMethodNotAllowed)

		return
	}
	if route.validJSON {
		body, _ := io.ReadAll(r.Body)
		if !json.Valid(body) {
			w.WriteHeader(http.StatusBadRequest)

			return
		}
	}
	w.WriteHeader(route.code)
	_, _ = fmt.Fprint(w, route.body)
}

// aCannedServer serves a fixed routing table of canned responses: exact path lookup,
// 405 on method mismatch, 404 for unknown paths. Replaces multi-path handler vars.
func (p *testFixture) aCannedServer(routes map[string]cannedRoute) *testFixture {
	return p.aHttpServer(func(w http.ResponseWriter, r *http.Request) {
		route, ok := routes[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)

			return
		}
		serveCannedRoute(w, r, route)
	})
}

// anEchoServer echoes the request body back verbatim with 200 OK — for tests that only
// need a round-tripped body (e.g. external-file encoding round trips).
func (p *testFixture) anEchoServer() *testFixture {
	return p.aHttpServer(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})
}

// withServerAddressVars stores scheme/host/port vars and rebuilds the client with them.
func (p *testFixture) withServerAddressVars() *testFixture {
	parsed, err := url.Parse(p.serverURL)
	p.require.NoError(err)

	p.activeServerVars = map[string]any{
		"scheme": parsed.Scheme,
		"host":   parsed.Hostname(),
		"port":   parsed.Port(),
	}

	p.aClient(rc.WithVars(p.activeServerVars))

	return p
}

// aTLSServer hosts an echo handler over httptest TLS (self-signed cert).
func (p *testFixture) aTLSServer() *testFixture {
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

	return p
}

// aTLSInsecureClient builds a restclient trusting the httptest self-signed cert.
func (p *testFixture) aTLSInsecureClient() *testFixture {
	httpClient := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}

	p.aClient(rc.WithHTTPClient(httpClient))

	return p
}

// aMockTransportClient builds the client under test with a mock round tripper that
// records the outgoing request into parts.intercepted and answers 200 with a fixed body.
func (p *testFixture) aMockTransportClient() *testFixture {
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

	return p
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

// graphqlResponse is the mocked server response body for GraphQL requests.
type graphqlResponse struct {
	Data   any            `json:"data,omitempty"`
	Errors []graphqlError `json:"errors,omitempty"`
}

// graphqlError mirrors the GraphQL error entry shape.
type graphqlError struct {
	Message   string `json:"message"`
	Locations []any  `json:"locations,omitempty"`
	Path      []any  `json:"path,omitempty"`
}

// aGraphQLServer serves the given responses in order, one per request, as HTTP 200 JSON.
func (p *testFixture) aGraphQLServer(responses ...graphqlResponse) *testFixture {
	idx := 0
	return p.aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		i := idx
		if i >= len(responses) {
			i = len(responses) - 1
		}
		_ = json.NewEncoder(w).Encode(responses[i])
		idx++
	})
}

func (p *testFixture) requestCount(n int64) *testFixture {
	p.require.Equal(n, p.requestHits.Load())

	return p
}
