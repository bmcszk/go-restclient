package restclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"golang.org/x/text/transform"

	rc "github.com/bmcszk/go-restclient"
)

// introduced by the batch-2b test migration. Same conventions: state lives in parts,
// assertions come from parts.require / parts.assert, .and() stays at end of line.

// anExternalFile writes the request file under test verbatim to <baseDir>/name.
func (p *parts) anExternalFile(name, content string) *parts {
	p.writeTempFile(name, content)
	p.httpFilePath = filepath.Join(p.baseDir, name)

	return p
}

// aFormattedRequestFixture renders a committed %s-placeholder fixture with fmt.Sprintf.
func (p *parts) aFormattedRequestFixture(name string, substitutions ...string) *parts {
	baseContent, err := os.ReadFile(filepath.Join(requestFilesDir, name))
	if err != nil {
		baseContent, err = os.ReadFile(filepath.Join("test", "data", "execute_file_ignore_empty_blocks", name))
		p.require.NoError(err)
	}

	args := make([]any, len(substitutions))
	for i, s := range substitutions {
		args[i] = s
	}

	return p.aHttpFile(fmt.Sprintf(string(baseContent), args...))
}

// aTemplateFixture renders a committed test/data/<dir> template with [[ ]] delimiters.
func (p *parts) aTemplateFixture(dir, name string, data any) *parts {
	tmplContent, err := os.ReadFile(filepath.Join("test", "data", dir, name))
	p.require.NoError(err)

	tmpl, err := template.New(name).Delims("[[", "]]").Parse(string(tmplContent))
	p.require.NoError(err)

	path := filepath.Join(p.baseDir, name)

	file, err := os.Create(path)
	p.require.NoError(err)
	p.require.NoError(tmpl.Execute(file, data))
	p.require.NoError(file.Close())

	p.httpFilePath = path

	return p
}

// aMissingFile points httpFilePath at a path that does not exist, for file-not-found tests.
func (p *parts) aMissingFile(path string) *parts {
	p.httpFilePath = path

	return p
}

// aDotEnvFile writes content to <baseDir>/.env. The parser resolves .env relative to the
// so this works for aTempDir-based .http files. No-op safety: writing empty content is allowed.
func (p *parts) aDotEnvFile(content string) *parts {
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, ".env"), []byte(content), 0644))
	return p
}

// aDotEnvFileRemoved removes <baseDir>/.env if present (os.Remove, ignore not-exists) so a
// previous scenario's .env cannot leak into the next subtest.
func (p *parts) aDotEnvFileRemoved() *parts {
	_ = os.Remove(filepath.Join(p.baseDir, ".env"))
	return p
}

// aFixtureCopy copies a committed fixture into baseDir, applying text replacements.
func (p *parts) aFixtureCopy(requestPath, destName string, replacements map[string]string) *parts {
	content, err := os.ReadFile(requestPath)
	p.require.NoError(err)
	result := string(content)
	for old, newVal := range replacements {
		result = strings.ReplaceAll(result, old, newVal)
	}
	path := filepath.Join(p.baseDir, destName)
	p.require.NoError(os.WriteFile(path, []byte(result), 0644))
	p.httpFilePath = path
	return p
}

// anEnvJsonFile writes a committed env-json template with {{SERVER_URL}} resolved.
func (p *parts) anEnvJsonFile(fileName, templatePath, serverURL string) *parts {
	content, err := os.ReadFile(templatePath)
	p.require.NoError(err)
	resolved := strings.ReplaceAll(string(content), "{{SERVER_URL}}", serverURL)
	path := filepath.Join(p.baseDir, fileName)
	p.require.NoError(os.WriteFile(path, []byte(resolved), 0600))
	return p
}

// aClientWithEnvironment builds the client with WithEnvironment unless env is empty.
func (p *parts) aClientWithEnvironment(env string) *parts {
	var client *rc.Client
	var err error
	if env != "" {
		client, err = rc.NewClient(rc.WithEnvironment(env))
	} else {
		client, err = rc.NewClient()
	}
	p.require.NoError(err)
	p.client = client
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

// anEchoServer echoes the request body back verbatim with 200 OK — for tests that only
// need a round-tripped body (e.g. external-file encoding round trips).
func (p *parts) anEchoServer() *parts {
	return p.aHttpServer(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})
}

// aRequestFixtureAbs points the DSL at a repo-root-relative request fixture path.
func (p *parts) aRequestFixtureAbs(name string) *parts {
	p.httpFilePath = name

	return p
}

// writeTempFile writes content to a file named name inside the shared per-test base dir.
func (p *parts) writeTempFile(name, content string) {
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, name), []byte(content), 0644))
}

// parsingFile parses the .http file without executing any request.
func (p *parts) parsingFile() *parts {
	p.require.NotNil(p.client)
	p.parsedFile, p.parseErr = p.client.ParseFile(p.httpFilePath)

	return p
}

// executingRequestAt parses the .http file and executes only the request at the given index.
func (p *parts) executingRequestAt(index int) *parts {
	p.require.NotNil(p.client)

	parsed, err := p.client.ParseFile(p.httpFilePath)
	p.require.NoError(err)

	resp, execErr := p.client.ExecuteRequest(context.Background(), parsed, index)
	p.responses = append(p.responses, resp)
	p.execErr = execErr

	return p
}

// parseSucceeded asserts the file was parsed without error.
func (p *parts) parseSucceeded() *parts {
	p.require.NoError(p.parseErr)

	return p
}

// parseErrorContains asserts parsing failed and the error mentions every given text.
// Called without texts it only asserts the error itself.
func (p *parts) parseErrorContains(texts ...string) *parts {
	p.require.Error(p.parseErr)

	for _, text := range texts {
		p.require.Contains(p.parseErr.Error(), text)
	}

	return p
}

// parsedRequestCount asserts how many requests the parse produced.
func (p *parts) parsedRequestCount(n int) *parts {
	p.require.NotNil(p.parsedFile)
	p.require.Len(p.parsedFile.Requests, n)

	return p
}

// parsedRequestMethod asserts the HTTP method of the parsed request at index i.
func (p *parts) parsedRequestMethod(i int, method string) *parts {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(method, p.parsedFile.Requests[i].Method)

	return p
}

// parsedRequestName asserts the name of the parsed request at index i.
func (p *parts) parsedRequestName(i int, name string) *parts {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(name, p.parsedFile.Requests[i].Name)

	return p
}

// parsedRequestRawURLIs asserts the raw (pre-substitution) URL of the parsed request
// at index i.
func (p *parts) parsedRequestRawURLIs(i int, rawURL string) *parts {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(rawURL, p.parsedFile.Requests[i].RawURLString)

	return p
}

// parsedRequestDisabled asserts the parsed request at index i has the disabled flag set to want.
func (p *parts) parsedRequestDisabled(i int, want bool) *parts {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].Disabled)

	return p
}

// parsedRequestRefs asserts the parsed request at index i declares a @ref/@forceRef entry.
func (p *parts) parsedRequestRefs(i int, wantName string, wantForce bool) *parts {
	p.require.Greater(len(p.parsedFile.Requests), i)
	req := p.parsedFile.Requests[i]
	p.require.NotEmpty(req.Refs, "request %d has no refs", i)

	found := false
	for _, ref := range req.Refs {
		if ref.Name == wantName {
			p.assert.Equal(wantForce, ref.Force,
				"ref %q force flag (expected %v)", wantName, wantForce)
			found = true

			break
		}
	}
	p.require.True(found, "request %d missing ref %q", i, wantName)

	return p
}

// urlHost returns the host portion of the given URL string. Used to compute the
// httptest server's host for assertions in http-client.env.json subtests.
func urlHost(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	return parsed.Host
}

// aHttpFileWithExternalFileEncoding writes the external file in the given encoding and a <@ request for it.
func (p *parts) aHttpFileWithExternalFileEncoding(
	encodingName, externalFileName string, rawUTF8Content []byte, encoder transform.Transformer,
) *parts {
	var encoded []byte
	if encoder != nil {
		var err error
		encoded, _, err = transform.Bytes(encoder, rawUTF8Content)
		p.require.NoError(err)
	} else {
		encoded = rawUTF8Content
	}

	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, externalFileName), encoded, 0644))

	return p.aHttpFileFromTemplateWithData("external_file_encoding.http",
		struct {
			ServerURL    string
			EncodingName string
			ExternalName string
		}{p.serverURL, encodingName, externalFileName})
}

// aHttpFileWithExternalFile writes a short request body that points at the named external
// file via <@, and stores the given JSON content in the external file.
func (p *parts) aHttpFileWithExternalFile(externalFileName, externalFileContent string) *parts {
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, externalFileName), []byte(externalFileContent), 0644))

	return p.aHttpFileFromTemplate("external_file_with_vars.http")
}

// aHttpFileWithExternalFileStatic writes a short request body that points at the named
// external file via < (no @: NO variable substitution).
func (p *parts) aHttpFileWithExternalFileStatic(externalFileName, externalFileContent string) *parts {
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, externalFileName), []byte(externalFileContent), 0644))

	return p.aHttpFileFromTemplate("external_file_static.http")
}

const externalFilesFixtureDir = "test/data/external_files"

// aHttpFileWithExternalFileFixture loads the committed external-file fixture via <@.
func (p *parts) aHttpFileWithExternalFileFixture(externalFileName string) *parts {
	src := filepath.Join(externalFilesFixtureDir, externalFileName)
	content, err := os.ReadFile(src)
	p.require.NoError(err)

	return p.aHttpFileWithExternalFile(externalFileName, string(content))
}

// aHttpFileWithExternalFileStaticFixture is the no-substitution < counterpart.
func (p *parts) aHttpFileWithExternalFileStaticFixture(externalFileName string) *parts {
	src := filepath.Join(externalFilesFixtureDir, externalFileName)
	content, err := os.ReadFile(src)
	p.require.NoError(err)

	return p.aHttpFileWithExternalFileStatic(externalFileName, string(content))
}

// fakerHeaderRule is one faker-header rule; containsCheck wins over pattern+fieldCount.
type fakerHeaderRule struct {
	key           string
	pattern       string
	fieldCount    int
	containsCheck string
}

// serverReceivedFakerHeaders applies the standard set of faker-header assertions to the
// captured request at reqIndex, for every rule in rules.
func (p *parts) serverReceivedFakerHeaders(reqIndex int, rules []fakerHeaderRule) *parts {
	p.require.Greater(len(p.capturedRequests), reqIndex)

	for _, rule := range rules {
		got := p.capturedRequests[reqIndex].Header.Get(rule.key)
		if rule.containsCheck != "" {
			p.assert.Contains(got, rule.containsCheck)
			p.assert.NotContains(got, "{{")
			continue
		}

		p.assert.Regexp(rule.pattern, got)
		p.assert.NotContains(got, "{{")
		if rule.fieldCount > 0 {
			p.assert.Len(strings.Fields(got), rule.fieldCount)
		}
	}

	return p
}

// aHttpFile writes the given .http content, replacing {{server}} with the test server URL.
func (p *parts) aHttpFile(content string) *parts {
	resolved := strings.ReplaceAll(content, "{{server}}", p.serverURL)
	path := filepath.Join(p.baseDir, "requests.http")
	p.require.NoError(os.WriteFile(path, []byte(resolved), 0644))
	p.httpFilePath = path

	return p
}

// aHttpFileFromTemplate renders a committed request fixture template with the test server
// URL ([[.ServerURL]] delimiters) and stores the processed file path for execution.
func (p *parts) aHttpFileFromTemplate(templateName string) *parts {
	p.require.NotEmpty(p.serverURL)

	return p.aHttpFileFromTemplateWithData(templateName, struct{ ServerURL string }{ServerURL: p.serverURL})
}

// aHttpFileFromTemplateWithData renders a committed request fixture template with the given
// template data ([[ ]] delimiters) and stores the processed file path for execution.
func (p *parts) aHttpFileFromTemplateWithData(templateName string, data any) *parts {
	tmplContent, err := os.ReadFile(filepath.Join(requestFilesDir, templateName))
	p.require.NoError(err)

	tmpl, err := template.New(templateName).Delims("[[", "]]").Parse(string(tmplContent))
	p.require.NoError(err)

	path := filepath.Join(p.baseDir, templateName)

	file, err := os.Create(path)
	p.require.NoError(err)
	p.require.NoError(tmpl.Execute(file, data))
	p.require.NoError(file.Close())

	p.httpFilePath = path

	return p
}

// aRequestFixture points the DSL at a committed request fixture file without templating.
func (p *parts) aRequestFixture(name string) *parts {
	p.httpFilePath = filepath.Join(requestFilesDir, name)

	return p
}

// withServerAddressVars stores scheme/host/port vars and rebuilds the client with them.
func (p *parts) withServerAddressVars() *parts {
	parsed, err := url.Parse(p.serverURL)
	p.require.NoError(err)

	p.activeServerVars = map[string]any{
		"scheme": parsed.Scheme,
		"host":   parsed.Hostname(),
		"port":   parsed.Port(),
	}

	return p.aClient(rc.WithVars(p.activeServerVars))
}

// withEnv sets an environment variable for the duration of the test.
func (p *parts) withEnv(k, v string) *parts {
	p.Setenv(k, v)

	return p
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

	return p.aClient(rc.WithHTTPClient(&http.Client{Transport: transport}))
}

// aCookieRedirectFixture renders a committed cookie/redirect template into baseDir.
func (p *parts) aCookieRedirectFixture(name string) *parts {
	tmplContent, err := os.ReadFile(filepath.Join("test", "data", "cookies_redirects", name))
	p.require.NoError(err)

	tmpl, err := template.New(name).Delims("[[", "]]").Parse(string(tmplContent))
	p.require.NoError(err)

	path := filepath.Join(p.baseDir, name)

	file, err := os.Create(path)
	p.require.NoError(err)
	p.require.NoError(tmpl.Execute(file, nil))
	p.require.NoError(file.Close())

	p.httpFilePath = path

	return p
}

// anUploadsFixtureCopy copies the multipart fixture with server URL and < paths resolved.
func (p *parts) anUploadsFixtureCopy() *parts {
	p.require.NotEmpty(p.serverURL)

	content, err := os.ReadFile(filepath.Join("test", "data", "http_request_files", "multipart_file_uploads.http"))
	p.require.NoError(err)

	resolved := strings.ReplaceAll(string(content), "< ./test/data/request_body/", "< test/data/request_body/")
	resolved = strings.ReplaceAll(resolved, "[[.ServerURL]]", p.serverURL)
	path := filepath.Join(p.baseDir, "multipart_file_uploads.http")

	p.require.NoError(os.WriteFile(path, []byte(resolved), 0644))
	p.httpFilePath = path

	return p
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

// clientRequestInterceptorCaptures asserts the request captured by aMockTransportClient
// has the given method and full URL.
func (p *parts) clientRequestInterceptorCaptures(method, wantURL string) *parts {
	p.require.NotNil(p.intercepted)
	p.assert.Equal(method, p.intercepted.Method)
	p.assert.Equal(wantURL, p.intercepted.URL.String())

	return p
}

// clientSentNoHeaders asserts the request captured by aMockTransportClient sent no headers.
func (p *parts) clientSentNoHeaders() *parts {
	p.require.NotNil(p.intercepted)
	p.assert.Empty(p.intercepted.Header)

	return p
}

// serverReceivedMethodAndPath asserts the request at the given server-hit index used the
// expected HTTP method and path.
func (p *parts) serverReceivedMethodAndPath(index int, method, path string) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Equal(method, p.capturedRequests[index].Method)
	p.assert.Equal(path, p.capturedRequests[index].URL.Path)

	return p
}

// serverReceivedJSONBody asserts the request body at the given server-hit index equals the
// expected JSON regardless of key order or whitespace.
func (p *parts) serverReceivedJSONBody(index int, expectedJSON string) *parts {
	p.require.Greater(len(p.capturedRequests), index)

	body, err := io.ReadAll(p.capturedRequests[index].Body)
	p.require.NoError(err)
	p.assert.JSONEq(expectedJSON, string(body))

	return p
}

// serverReceivedHeaderValue asserts a request header value at the given server-hit index.
func (p *parts) serverReceivedHeaderValue(index int, key, value string) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Equal(value, p.capturedRequests[index].Header.Get(key))

	return p
}

// requestRawURLContains asserts the current request's RawURLString contains every fragment.
func (p *parts) requestRawURLContains(fragments ...string) *parts {
	current := p.current()
	p.require.NotNil(current.Request)

	for _, fragment := range fragments {
		p.assert.Contains(current.Request.RawURLString, fragment)
	}

	return p
}

// capturedRequestCount asserts how many requests the test server received.
func (p *parts) capturedRequestCount(n int) *parts {
	p.require.Equal(n, len(p.capturedRequests))

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
