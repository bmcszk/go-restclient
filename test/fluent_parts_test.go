package test_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/require"

	rc "github.com/bmcszk/go-restclient"
)

// parts is the shared state for the fluent Given/When/Then test DSL.
// The same instance is handed out as given, when and then.
type parts struct {
	*testing.T
	require   *require.Assertions
	client    *rc.Client
	servers   []*httptest.Server
	serverURL string
	// requestHits backs the requestCount assertion; a field and method cannot
	// share the name requestCount in Go.
	requestHits      atomic.Int64
	httpFilePath     string
	expectedFilePath string
	programmaticVars map[string]any
	responses        []*rc.Response
	execErr          error
	validationErr    error
	cursor           int
}

// newParts returns the given, when and then entry points of the DSL.
func newParts(t *testing.T) (given, when, then *parts) {
	t.Helper()

	p := &parts{T: t, require: require.New(t)}

	return p, p, p
}

// and keeps the fluent chain readable.
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

// --- Given ---

// aHttpServer starts a local test server that counts every request it serves.
func (p *parts) aHttpServer(h http.HandlerFunc) *parts {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.requestHits.Add(1)
		h(w, r)
	}))
	p.servers = append(p.servers, srv)
	p.serverURL = srv.URL
	p.Cleanup(srv.Close)

	return p
}

// aHttpFile writes the given .http content, replacing {{server}} with the test server URL.
func (p *parts) aHttpFile(content string) *parts {
	resolved := strings.ReplaceAll(content, "{{server}}", p.serverURL)
	path := filepath.Join(p.TempDir(), "requests.http")
	p.require.NoError(os.WriteFile(path, []byte(resolved), 0644))
	p.httpFilePath = path

	return p
}

// aClient builds the restclient under test with the given options.
func (p *parts) aClient(opts ...rc.ClientOption) *parts {
	client, err := rc.NewClient(opts...)
	p.require.NoError(err)
	p.client = client

	return p
}

// withProgrammaticVars sets programmatic variables on the client.
func (p *parts) withProgrammaticVars(vars map[string]any) *parts {
	p.require.NotNil(p.client)
	p.client.SetProgrammaticVars(vars)
	p.programmaticVars = vars

	return p
}

// withEnv sets an environment variable for the duration of the test.
func (p *parts) withEnv(k, v string) *parts {
	p.Setenv(k, v)

	return p
}

// expectedResponseFile writes the given .hresp content as the expected responses file.
func (p *parts) expectedResponseFile(content string) *parts {
	path := filepath.Join(p.TempDir(), "expected.hresp")
	p.require.NoError(os.WriteFile(path, []byte(content), 0644))
	p.expectedFilePath = path

	return p
}

// --- When ---

// executeFile runs every request in the .http file.
func (p *parts) executeFile() *parts {
	p.require.NotNil(p.client)
	p.responses, p.execErr = p.client.ExecuteFile(context.Background(), p.httpFilePath)

	return p
}

// sendingRequest executes the named request from the .http file.
func (p *parts) sendingRequest(name string) *parts {
	p.require.NotNil(p.client)

	parsed, err := p.client.ParseFile(p.httpFilePath)
	p.require.NoError(err)

	index := -1
	for i, req := range parsed.Requests {
		if req.Name == name {
			index = i

			break
		}
	}

	if index < 0 {
		p.Fatalf("request %q not found in %s", name, p.httpFilePath)
	}

	resp, execErr := p.client.ExecuteRequest(context.Background(), parsed, index)
	p.responses = append(p.responses, resp)
	p.execErr = execErr

	return p
}

// validateResponses checks the actual responses against the expected file.
func (p *parts) validateResponses() *parts {
	p.validationErr = p.client.ValidateResponses(p.expectedFilePath, p.responses...)

	return p
}

// --- Then ---

// responseAt selects the response the following assertions operate on.
func (p *parts) responseAt(i int) *parts {
	p.cursor = i

	return p
}

// current returns the response the cursor points at.
func (p *parts) current() *rc.Response {
	p.require.Greater(len(p.responses), p.cursor)

	return p.responses[p.cursor]
}

// responseCode asserts the status code of the selected response.
func (p *parts) responseCode(code int) *parts {
	p.require.Equal(code, p.current().StatusCode)

	return p
}

// responseContains asserts the body of the selected response contains s.
func (p *parts) responseContains(s string) *parts {
	p.require.Contains(p.current().BodyString, s)

	return p
}

// responseNotContains asserts the body of the selected response does not contain s.
func (p *parts) responseNotContains(s string) *parts {
	p.require.NotContains(p.current().BodyString, s)

	return p
}

// responseBodyIs asserts the exact body of the selected response.
func (p *parts) responseBodyIs(s string) *parts {
	p.require.Equal(s, p.current().BodyString)

	return p
}

// responseHeader asserts the value of a header of the selected response.
func (p *parts) responseHeader(k, v string) *parts {
	p.require.Equal(v, p.current().Headers.Get(k))

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

// errorCount asserts the number of errors returned by the execution.
func (p *parts) errorCount(n int) *parts {
	p.require.Equal(n, multierrorCount(p.execErr))

	return p
}

// requestCount asserts how many requests hit the test server.
func (p *parts) requestCount(n int64) *parts {
	p.require.Equal(n, p.requestHits.Load())

	return p
}

// validationSucceeds asserts the response validation passed.
func (p *parts) validationSucceeds() *parts {
	p.require.NoError(p.validationErr)

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

// dslSymbols keeps every DSL entry point referenced so the unused linter stays
// quiet in this definitions-only file until tests adopt the DSL.
var _ = []any{
	newParts,
	(*parts).and,
	(*parts).aHttpServer,
	(*parts).aHttpFile,
	(*parts).aClient,
	(*parts).withProgrammaticVars,
	(*parts).withEnv,
	(*parts).expectedResponseFile,
	(*parts).executeFile,
	(*parts).sendingRequest,
	(*parts).validateResponses,
	(*parts).responseAt,
	(*parts).current,
	(*parts).responseCode,
	(*parts).responseContains,
	(*parts).responseNotContains,
	(*parts).responseBodyIs,
	(*parts).responseHeader,
	(*parts).noError,
	(*parts).errorContains,
	(*parts).errorCount,
	(*parts).requestCount,
	(*parts).validationSucceeds,
	(*parts).validationFails,
}
