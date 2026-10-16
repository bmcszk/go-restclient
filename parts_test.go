package restclient_test

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rc "github.com/bmcszk/go-restclient"
)

const (
	responseFilesDir = "test/data/http_response_files"
)

// parts is the shared state for the fluent Given/When/Then test DSL.
type parts struct {
	*testing.T
	require *require.Assertions
	assert  *assert.Assertions
	client  *rc.Client
	// server holds the server domain state (the httptest servers and their
	// address variables); parts_test.go builds on it.
	serverParts
	// capture holds the capture/tracking domain state (recorded requests, bodies and
	// consistency-tracking buckets); parts_test.go builds on it.
	captureParts
	// parse holds the parse-domain state (the parsed .http file and its error);
	// parts_test.go builds on it.
	parseParts
	// file holds the file/fixture domain state (the .http file path the file
	// builders write and executeFile / parse read); parts_test.go builds on it.
	fileParts
	// env holds the env domain state (dotenv files, environment-backed client
	// builds); parts_test.go builds on it.
	envParts
	// oauth2 holds the oauth2 domain state (the OAuth2 token server address);
	// client_oauth2_test.go builds on it.
	oauth2Parts
	// cookies holds the cookie/redirect domain state (the cookie round-trip flip
	// flag); client_cookies_redirects_test.go builds on it.
	cookiesParts
	clientConfig     clientConfigSnapshot
	expectedFilePath string
	responses        []*rc.Response
	execErr          error
	validationErr    error
	cursor           int
	// baseDir is the single per-test directory every given* file-writing method uses.
	// (testing.T.TempDir creates a fresh subdirectory on every call, so external
	// files and the .http file would otherwise land in different directories.)
	baseDir string
}

// newParts returns the given, when and then entry points of the DSL.
func newParts(t *testing.T) (given, when, then *parts) {
	t.Helper()

	p := &parts{
		T:       t,
		require: require.New(t),
		assert:  assert.New(t),
		baseDir: t.TempDir(),
	}
	p.serverParts.parts = p
	p.captureParts.parts = p
	p.parseParts.parts = p
	p.fileParts.parts = p
	p.envParts.parts = p
	p.oauth2Parts.parts = p
	p.cookiesParts.parts = p
	p.trackedValues = make(map[string][]string)

	return p, p, p
}

func newCaptureParts(t *testing.T) (given, when, then *captureParts) {
	t.Helper()

	p, _, _ := newParts(t)
	c := &p.captureParts

	return c, c, c
}

func newParseParts(t *testing.T) (given, when, then *parseParts) {
	t.Helper()

	p, _, _ := newParts(t)
	c := &p.parseParts

	return c, c, c
}

func newOauth2Parts(t *testing.T) (given, when, then *oauth2Parts) {
	t.Helper()

	p, _, _ := newParts(t)
	c := &p.oauth2Parts

	return c, c, c
}

func newCookiesParts(t *testing.T) (given, when, then *cookiesParts) {
	t.Helper()

	p, _, _ := newParts(t)
	c := &p.cookiesParts

	return c, c, c
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

// anExpectedResponseFixture points the DSL at a committed expected-response fixture file.
func (p *parts) anExpectedResponseFixture(name string) *parts {
	p.expectedFilePath = filepath.Join(responseFilesDir, name)

	return p
}

// aResponseWith appends a hand-constructed rc.Response to p.responses.
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

// validationFails asserts the response validation failed with count errors mentioning every text.
func (p *parts) validationFails(count int, texts ...string) *parts {
	p.require.Error(p.validationErr)

	p.require.Equal(count, multierrorCount(p.validationErr))

	for _, text := range texts {
		p.require.Contains(p.validationErr.Error(), text)
	}

	return p
}

// validationSucceeds asserts the response validation passed.
func (p *parts) validationSucceeds() *parts {
	p.require.NoError(p.validationErr)

	return p
}
