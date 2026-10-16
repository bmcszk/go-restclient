package restclient_test

import (
	"net/http"
	"path/filepath"
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
	baseDir string
}

// testFixture composes the core parts with every domain state struct; all DSL
// methods are declared on *testFixture in their domain files.
type testFixture struct {
	parts
	serverParts
	captureParts
	parseParts
	fileParts
	envParts
	validatorParts
	oauth2Parts
	cookiesParts
}

// newParts returns the given, when and then entry points of the DSL.
// newFixture constructs the base parts core and returns the shared fixture.
func newFixture(t *testing.T) *testFixture {
	t.Helper()

	fx := &testFixture{
		parts: parts{
			T:       t,
			require: require.New(t),
			assert:  assert.New(t),
			baseDir: t.TempDir(),
		},
	}
	fx.trackedValues = make(map[string][]string)

	return fx
}

// newParts returns the given/when/then fixture for base-domain tests.
func newParts(t *testing.T) (given, when, then *testFixture) {
	t.Helper()

	fx := newFixture(t)

	return fx, fx, fx
}

func (p *testFixture) and() *testFixture { return p }

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
func (p *testFixture) aClient(opts ...rc.ClientOption) *testFixture {
	client, err := rc.NewClient(opts...)
	p.require.NoError(err)
	p.client = client

	return p
}

// validateResponses checks the actual responses against the expected file.
func (p *testFixture) validateResponses() *testFixture {
	p.validationErr = p.client.ValidateResponses(p.expectedFilePath, p.responses...)

	return p
}

func (p *testFixture) responseAt(i int) *testFixture {
	p.cursor = i

	return p
}

// current returns the response the cursor points at.
func (p *testFixture) current() *rc.Response {
	p.require.Greater(len(p.responses), p.cursor)

	return p.responses[p.cursor]
}

func (p *testFixture) responseCode(code int) *testFixture {
	p.require.Equal(code, p.current().StatusCode)

	return p
}

func (p *testFixture) responseContains(s string) *testFixture {
	p.require.Contains(p.current().BodyString, s)

	return p
}

func (p *testFixture) responseBodyIs(s string) *testFixture {
	p.require.Equal(s, p.current().BodyString)

	return p
}

func (p *testFixture) responseHeader(k, v string) *testFixture {
	p.require.Equal(v, p.current().Headers.Get(k))

	return p
}

func (p *testFixture) responseCount(n int) *testFixture {
	p.require.Len(p.responses, n)

	return p
}

// responseHasNoError asserts the current response has no error.
func (p *testFixture) responseHasNoError() *testFixture {
	p.require.NoError(p.current().Error)

	return p
}

// responseHasError asserts the current response carries an error mentioning every text.
func (p *testFixture) responseHasError(texts ...string) *testFixture {
	p.require.Error(p.current().Error)

	for _, text := range texts {
		p.require.Contains(p.current().Error.Error(), text)
	}

	return p
}

// noError asserts the execution succeeded and no response carries an error.
func (p *testFixture) noError() *testFixture {
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
func (p *testFixture) errorContains(texts ...string) *testFixture {
	p.require.Error(p.execErr)

	for _, text := range texts {
		p.require.Contains(p.execErr.Error(), text)
	}

	return p
}

// anExpectedResponseFixture points the DSL at a committed expected-response fixture file.
func (p *testFixture) anExpectedResponseFixture(name string) *testFixture {
	p.expectedFilePath = filepath.Join(responseFilesDir, name)

	return p
}

// aResponseWith appends a hand-constructed rc.Response to p.responses.
func (p *testFixture) aResponseWith(statusCode int, statusText, body string, headers http.Header) *testFixture {
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
func (p *testFixture) validationFails(count int, texts ...string) *testFixture {
	p.require.Error(p.validationErr)

	p.require.Equal(count, multierrorCount(p.validationErr))

	for _, text := range texts {
		p.require.Contains(p.validationErr.Error(), text)
	}

	return p
}

// validationSucceeds asserts the response validation passed.
func (p *testFixture) validationSucceeds() *testFixture {
	p.require.NoError(p.validationErr)

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
func (p *testFixture) aClientWithDefaults() *testFixture {
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
func (p *testFixture) clientBaseURLIs(_ string) *testFixture {
	p.require.Equal(p.clientConfig.baseURL, p.client.BaseURL)

	return p
}

// clientBaseURLIsEmpty asserts the client's BaseURL is empty.
func (p *testFixture) clientBaseURLIsEmpty() *testFixture {
	p.assert.Empty(p.client.BaseURL)

	return p
}

// clientDefaultHeaderIs after aClientWithDefaults runs.
func (p *testFixture) clientDefaultHeaderIs(key, _ string) *testFixture {
	p.require.Equal(p.clientConfig.headerValue, p.client.DefaultHeaders.Get(key))

	return p
}

// clientExists asserts the client is non-nil.
func (p *testFixture) clientExists() *testFixture {
	p.require.NotNil(p.client)

	return p
}

// clientDefaultHeadersEmpty asserts the client's DefaultHeaders map is non-nil and empty.
func (p *testFixture) clientDefaultHeadersEmpty() *testFixture {
	p.require.NotNil(p.client.DefaultHeaders)
	p.assert.Empty(p.client.DefaultHeaders)

	return p
}
