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
	"testing"
	"time"

	rc "github.com/bmcszk/go-restclient"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type serverParts struct {
	parts
}

func newServerParts(t *testing.T) (given, when, then *serverParts) {
	t.Helper()

	s := &serverParts{parts: parts{
		trackedValues: make(map[string][]string),
		T:             t,
		require:       require.New(t),
		assert:        assert.New(t),
		baseDir:       t.TempDir(),
	}}

	return s, s, s
}

// aHttpServer counts requests; bodies are read eagerly because net/http closes them on return.
type cannedRoute struct {
	method    string
	code      int
	body      string
	validJSON bool
}

type mockRoundTripper struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

type graphqlResponse struct {
	Data   any            `json:"data,omitempty"`
	Errors []graphqlError `json:"errors,omitempty"`
}

type graphqlError struct {
	Message   string `json:"message"`
	Locations []any  `json:"locations,omitempty"`
	Path      []any  `json:"path,omitempty"`
}

func (p *parts) aHttpServer(h http.HandlerFunc) *parts {
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
// 405 on method mismatch, 404 for unknown paths.
func (p *parts) aCannedServer(routes map[string]cannedRoute) *parts {
	return p.aHttpServer(func(w http.ResponseWriter, r *http.Request) {
		route, ok := routes[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)

			return
		}
		serveCannedRoute(w, r, route)
	})
}

func (p *parts) anEchoServer() *parts {
	return p.aHttpServer(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})
}

func (p *parts) withServerAddressVars() *parts {
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

func (p *parts) aTLSServer() *parts {
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

func (p *parts) aTLSInsecureClient() *parts {
	httpClient := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}

	p.aClient(rc.WithHTTPClient(httpClient))

	return p
}

// aMockTransportClient builds the client under test with a mock round tripper that
// records the outgoing request into parts.intercepted and answers 200 with a fixed body.
func (p *parts) aMockTransportClient() *parts {
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

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.RoundTripFunc != nil {
		return m.RoundTripFunc(req)
	}

	return nil, errors.New("RoundTripFunc not set")
}

func (p *parts) aGraphQLServer(responses ...graphqlResponse) *parts {
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

func (p *parts) requestCount(n int64) *parts {
	p.require.Equal(n, p.requestHits.Load())

	return p
}
