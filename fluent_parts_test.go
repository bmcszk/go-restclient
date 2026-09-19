package restclient_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rc "github.com/bmcszk/go-restclient"
)

const (
	requestFilesDir = "test/data/http_request_files"

	responseFilesDir = "test/data/http_response_files"
)

// clientConfigSnapshot records the client options asserted by clientBaseURLIs /
// clientDefaultHeaderIs after aClientWithDefaults runs.
type clientConfigSnapshot struct {
	baseURL     string
	headerKey   string
	headerValue string
	httpTimeout time.Duration
}

// parts is the shared state for the fluent Given/When/Then test DSL.
type parts struct {
	*testing.T
	require   *require.Assertions
	assert    *assert.Assertions
	client    *rc.Client
	servers   []*httptest.Server
	serverURL string
	// share the name requestCount in Go.
	requestHits      atomic.Int64
	capturedRequests []*http.Request
	// intercepted holds the outgoing request captured by aMockTransportClient.
	intercepted *http.Request
	// cookieCheck records whether the cookie test server received its cookie back.
	cookieCheck  bool
	clientConfig clientConfigSnapshot
	// activeServerVars holds the scheme/host/port variables of the running test server.
	activeServerVars map[string]any
	httpFilePath     string
	expectedFilePath string
	responses        []*rc.Response
	execErr          error
	validationErr    error
	cursor           int
	// parsedFile holds the result of parsingFile for parsed-file assertions.
	parsedFile *rc.ParsedFile
	// parseErr holds the error returned by parsingFile.
	parseErr error
	// handler runs, because net/http closes the original body when the handler returns.
	capturedBodies []string
	// trackedValues holds consistency-tracking buckets keyed by tracking name.
	trackedValues map[string][]string
	// activeTrack names the bucket captured* methods append into.
	activeTrack string
	// baseDir is the single per-test directory every given* file-writing method uses.
	// (testing.T.TempDir creates a fresh subdirectory on every call, so external
	// files and the .http file would otherwise land in different directories.)
	baseDir string
}

// newParts returns the given, when and then entry points of the DSL.
func newParts(t *testing.T) (given, when, then *parts) {
	t.Helper()

	p := &parts{
		T:             t,
		require:       require.New(t),
		assert:        assert.New(t),
		trackedValues: make(map[string][]string),
		baseDir:       t.TempDir(),
	}

	return p, p, p
}

func (p *parts) and() *parts { return p }

// multierrorCount counts the wrapped errors inside err.
func multierrorCount(err error) int {
	if err == nil {
		return 0
	}

	if merr, ok := err.(*multierror.Error); ok {
		return len(merr.Errors)
	}

	return 1
}

// aHttpServer counts requests; bodies are read eagerly because net/http closes them on return.
func (p *parts) aHttpServer(h http.HandlerFunc) *parts {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		p.capturedRequests = append(p.capturedRequests, r)
		p.capturedBodies = append(p.capturedBodies, string(body))
		p.requestHits.Add(1)
		h(w, r)
	}))
	p.servers = append(p.servers, srv)
	p.serverURL = srv.URL
	p.Cleanup(srv.Close)

	return p
}

// aClient builds the restclient under test with the given options.
func (p *parts) aClient(opts ...rc.ClientOption) *parts {
	client, err := rc.NewClient(opts...)
	p.require.NoError(err)
	p.client = client

	return p
}

// executeFile runs every request in the .http file.
func (p *parts) executeFile() *parts {
	p.require.NotNil(p.client)
	p.responses, p.execErr = p.client.ExecuteFile(context.Background(), p.httpFilePath)

	return p
}

// validateResponses checks the actual responses against the expected file.
func (p *parts) validateResponses() *parts {
	p.validationErr = p.client.ValidateResponses(p.expectedFilePath, p.responses...)

	return p
}

func (p *parts) responseAt(i int) *parts {
	p.cursor = i

	return p
}

// current returns the response the cursor points at.
func (p *parts) current() *rc.Response {
	p.require.Greater(len(p.responses), p.cursor)

	return p.responses[p.cursor]
}

func (p *parts) responseCode(code int) *parts {
	p.require.Equal(code, p.current().StatusCode)

	return p
}

func (p *parts) responseContains(s string) *parts {
	p.require.Contains(p.current().BodyString, s)

	return p
}

func (p *parts) responseBodyIs(s string) *parts {
	p.require.Equal(s, p.current().BodyString)

	return p
}

func (p *parts) responseHeader(k, v string) *parts {
	p.require.Equal(v, p.current().Headers.Get(k))

	return p
}

func (p *parts) responseCount(n int) *parts {
	p.require.Len(p.responses, n)

	return p
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

// responseHasNoError asserts the current response has no error.
func (p *parts) responseHasNoError() *parts {
	p.require.NoError(p.current().Error)

	return p
}

// responseHasError asserts the current response carries an error mentioning every text.
func (p *parts) responseHasError(texts ...string) *parts {
	p.require.Error(p.current().Error)

	for _, text := range texts {
		p.require.Contains(p.current().Error.Error(), text)
	}

	return p
}

// noError asserts the execution succeeded and no response carries an error.
func (p *parts) noError() *parts {
	p.require.NoError(p.execErr)

	for i, resp := range p.responses {
		if resp == nil {
			p.require.Failf("nil response", "response at index %d is nil", i)

			continue
		}

		p.require.NoError(resp.Error, "response %d", i)
	}

	return p
}

// errorContains asserts the execution failed and the error mentions every text.
func (p *parts) errorContains(texts ...string) *parts {
	p.require.Error(p.execErr)

	for _, text := range texts {
		p.require.Contains(p.execErr.Error(), text)
	}

	return p
}

func (p *parts) requestCount(n int64) *parts {
	p.require.Equal(n, p.requestHits.Load())

	return p
}
