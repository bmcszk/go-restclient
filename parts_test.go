package restclient_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rc "github.com/bmcszk/go-restclient"
)

const (
	responseFilesDir = "test/data/http_response_files"
)

// parts is the core state of the fluent Given/When/Then test DSL: the test
// handle, the assertion clients, the client under test and the response ledger.
type parts struct {
	*testing.T
	require          *require.Assertions
	assert           *assert.Assertions
	client           *rc.Client
	clientConfig     clientConfigSnapshot
	expectedFilePath string
	responses        []*rc.Response
	execErr          error
	validationErr    error
	cursor           int
	// baseDir is the single per-test directory every given* file-writing method uses.
	// (testing.T.TempDir creates a fresh subdirectory on every call, so external
	// files and the .http file would otherwise land in different directories.)
	baseDir          string
	capturedRequests []*http.Request
	// capturedRequestTimes records time.Now() when the test server received each request.
	capturedRequestTimes []time.Time
	// capturedBodies holds request bodies read eagerly, because net/http closes the
	// original body when the handler returns.
	capturedBodies []string
	// intercepted holds the outgoing request captured by aMockTransportClient.
	intercepted *http.Request
	// trackedValues holds consistency-tracking buckets keyed by tracking name.
	trackedValues map[string][]string
	activeTrack   string
	servers       []*httptest.Server
	serverURL     string
	// requestHits shares its JSON/URL var name with requestCount in Go.
	requestHits atomic.Int64
	// activeServerVars holds the scheme/host/port variables of the running test server.
	activeServerVars map[string]any
	parsedFile       *rc.ParsedFile
	parseErr         error
	// httpFilePath holds the path of the .http file under test that the file
	// and fixture builders write and the executeFile / parse builders read back.
	httpFilePath   string
	dotenvFiles    []string
	builtResponses []*rc.Response
	tokenServerURL string
	cookieCheck    bool
}

func newParts(t *testing.T) (given, when, then *parts) {
	t.Helper()

	p := &parts{
		T:             t,
		require:       require.New(t),
		assert:        assert.New(t),
		baseDir:       t.TempDir(),
		trackedValues: make(map[string][]string),
	}

	return p, p, p
}

type clientConfigSnapshot struct {
	baseURL     string
	headerKey   string
	headerValue string
	httpTimeout time.Duration
}

func (p *parts) and() *parts { return p }

func multierrorCount(err error) int {
	if err == nil {
		return 0
	}

	if merr, ok := err.(*multierror.Error); ok {
		return len(merr.Errors)
	}

	return 1
}

func (p *parts) aClient(opts ...rc.ClientOption) *parts {
	client, err := rc.NewClient(opts...)
	p.require.NoError(err)
	p.client = client

	return p
}

func (p *parts) validateResponses() *parts {
	p.validationErr = p.client.ValidateResponses(p.expectedFilePath, p.responses...)

	return p
}

func (p *parts) responseAt(i int) *parts {
	p.cursor = i

	return p
}

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

func (p *parts) responseHasNoError() *parts {
	p.require.NoError(p.current().Error)

	return p
}

func (p *parts) responseHasError(texts ...string) *parts {
	p.require.Error(p.current().Error)

	for _, text := range texts {
		p.require.Contains(p.current().Error.Error(), text)
	}

	return p
}

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

func (p *parts) errorContains(texts ...string) *parts {
	p.require.Error(p.execErr)

	for _, text := range texts {
		p.require.Contains(p.execErr.Error(), text)
	}

	return p
}

func (p *parts) anExpectedResponseFixture(name string) *parts {
	p.expectedFilePath = filepath.Join(responseFilesDir, name)

	return p
}

func (p *parts) aResponseWith(statusCode int, statusText, body string, headers http.Header) *parts {
	resp := &rc.Response{
		StatusCode: statusCode,
		Status:     statusText,
		Headers:    headers,
	}
	if body != "" {
		resp.BodyString = body
		resp.Body = []byte(body)
	}
	p.responses = append(p.responses, resp)
	return p
}

func (p *parts) validationFails(count int, texts ...string) *parts {
	p.require.Error(p.validationErr)

	p.require.Equal(count, multierrorCount(p.validationErr))

	for _, text := range texts {
		p.require.Contains(p.validationErr.Error(), text)
	}

	return p
}

func (p *parts) validationSucceeds() *parts {
	p.require.NoError(p.validationErr)

	return p
}

// clientConfigSnapshot records the client options asserted by clientBaseURLIs /

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

func (p *parts) clientBaseURLIs(_ string) *parts {
	p.require.Equal(p.clientConfig.baseURL, p.client.BaseURL)

	return p
}

func (p *parts) clientBaseURLIsEmpty() *parts {
	p.assert.Empty(p.client.BaseURL)

	return p
}

// clientDefaultHeaderIs asserts the client's DefaultHeaders value for key
// equals the recorded config snapshot.
func (p *parts) clientDefaultHeaderIs(key, _ string) *parts {
	p.require.Equal(p.clientConfig.headerValue, p.client.DefaultHeaders.Get(key))

	return p
}

func (p *parts) clientExists() *parts {
	p.require.NotNil(p.client)

	return p
}

func (p *parts) clientDefaultHeadersEmpty() *parts {
	p.require.NotNil(p.client.DefaultHeaders)
	p.assert.Empty(p.client.DefaultHeaders)

	return p
}
