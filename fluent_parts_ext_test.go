package restclient_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/google/uuid"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// This file extends the fluent DSL (fluent_parts_test.go) with the entry points
// introduced by the batch-2b test migration. Same conventions: state lives in parts,
// assertions come from parts.require / parts.assert, .and() stays at end of line.

// --- Given ---

// anExternalFile writes content to <baseDir>/name. The content is stored verbatim; no
// {{server}} substitution happens. The written file becomes the request file under test
// (e.g. for .rest extension tests) — combine with aHttpFile when it is only a body file.
func (p *parts) anExternalFile(name, content string) *parts {
	p.writeTempFile(name, content)
	p.httpFilePath = filepath.Join(p.baseDir, name)

	return p
}

// anExternalFileBytes writes raw bytes to <baseDir>/name (e.g. latin1-encoded bodies).
func (p *parts) anExternalFileBytes(name string, data []byte) *parts {
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, name), data, 0644))

	return p
}

// aFormattedRequestFixture reads a committed request fixture containing %s placeholders,
// applies fmt.Sprintf with the given substitutions in order and writes the result to a
// temp file that becomes the .http file under test. Fixtures are looked up in
// requestFilesDir first, then in test/data/execute_file_ignore_empty_blocks.
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

// aTemplateFixture reads a committed template from test/data/<dir>/<name>, renders it
// with [[ ]] delimiters and the given data, and stores the processed file path.
// Generalizes aHttpFileFromTemplate to fixture directories beyond http_request_files.
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

// writeTempFile writes content to a file named name inside the shared per-test base dir.
func (p *parts) writeTempFile(name, content string) {
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, name), []byte(content), 0644))
}

// --- When ---

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

// --- Then ---

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

// requestURLContains asserts every given fragment appears in the URL of the selected
// response's originating request.
func (p *parts) requestURLContains(fragments ...string) *parts {
	current := p.current()
	p.require.NotNil(current.Request)

	for _, fragment := range fragments {
		p.assert.Contains(current.Request.URL.String(), fragment)
	}

	return p
}

// requestHeaderIs asserts a header value of the selected response's originating request.
func (p *parts) requestHeaderIs(key, value string) *parts {
	current := p.current()
	p.require.NotNil(current.Request)
	p.assert.Equal(value, current.Request.Headers.Get(key))

	return p
}

// requestRawBodyIs asserts the exact raw body of the selected response's originating request.
func (p *parts) requestRawBodyIs(body string) *parts {
	current := p.current()
	p.require.NotNil(current.Request)
	p.assert.Equal(body, current.Request.RawBody)

	return p
}

// requestRawBodyContains asserts every given fragment appears in the raw body of the
// selected response's originating request.
func (p *parts) requestRawBodyContains(fragments ...string) *parts {
	current := p.current()
	p.require.NotNil(current.Request)

	for _, fragment := range fragments {
		p.assert.Contains(current.Request.RawBody, fragment)
	}

	return p
}

// requestRawBodyMatchesRegexp asserts the raw body of the selected response's originating
// request matches the given regular expression.
func (p *parts) requestRawBodyMatchesRegexp(pattern string) *parts {
	current := p.current()
	p.require.NotNil(current.Request)
	p.assert.Regexp(pattern, current.Request.RawBody)

	return p
}

// serverReceivedBodyIs asserts the exact body captured for the server hit at index.
func (p *parts) serverReceivedBodyIs(index int, body string) *parts {
	p.require.Greater(len(p.capturedBodies), index)
	p.assert.Equal(body, p.capturedBodies[index])

	return p
}

// serverReceivedBodyParsesAsJSON asserts the captured body at index is valid JSON.
func (p *parts) serverReceivedBodyParsesAsJSON(index int) *parts {
	p.require.Greater(len(p.capturedBodies), index)

	var data map[string]any
	p.require.NoError(json.Unmarshal([]byte(p.capturedBodies[index]), &data))

	return p
}

// serverReceivedLatin1BodyIs asserts the captured body at index equals the expected text
// after decoding it from ISO-8859-1.
func (p *parts) serverReceivedLatin1BodyIs(index int, body string) *parts {
	p.require.Greater(len(p.capturedBodies), index)

	decoded, _, err := transform.Bytes(charmap.ISO8859_1.NewDecoder(), []byte(p.capturedBodies[index]))
	p.require.NoError(err)
	p.assert.Equal(body, string(decoded))

	return p
}

// allResponseCodesAre asserts every captured response has the given status code.
func (p *parts) allResponseCodesAre(code int) *parts {
	p.require.NotEmpty(p.responses)

	for i, resp := range p.responses {
		p.require.Equal(code, resp.StatusCode, "response %d", i)
	}

	return p
}

// tracking starts (or resumes) a named consistency-tracking bucket for the captured*
// assertion methods that follow.
func (p *parts) tracking(name string) *parts {
	p.activeTrack = name

	if _, ok := p.trackedValues[name]; !ok {
		p.trackedValues[name] = []string{}
	}

	return p
}

// capturedURLSegment appends the last "/"-segment of the captured request URL path at
// index to the active tracking bucket.
func (p *parts) capturedURLSegment(index int) *parts {
	p.require.Greater(len(p.capturedRequests), index)

	segments := strings.Split(p.capturedRequests[index].URL.Path, "/")
	p.track(segments[len(segments)-1])

	return p
}

// capturedHeaderField appends a captured request header value at index to the active
// tracking bucket.
func (p *parts) capturedHeaderField(index int, key string) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.track(p.capturedRequests[index].Header.Get(key))

	return p
}

// capturedJSONField walks the given path through the captured JSON body at index and
// appends the string leaf to the active tracking bucket.
func (p *parts) capturedJSONField(index int, path ...string) *parts {
	leaf, ok := p.capturedJSONLeaf(index, path).(string)
	p.require.True(ok, "field %v is not a string", path)
	p.track(leaf)

	return p
}

// capturedJSONNumberField walks the given path through the captured JSON body at index,
// expects a numeric leaf and appends it formatted as an integer to the active bucket.
func (p *parts) capturedJSONNumberField(index int, path ...string) *parts {
	value, ok := p.capturedJSONLeaf(index, path).(float64)
	p.require.True(ok, "field %v is not a number", path)
	p.track(strconv.FormatInt(int64(value), 10))

	return p
}

// allTrackedValuesEqual asserts every value tracked in the active bucket equals the first.
func (p *parts) allTrackedValuesEqual() *parts {
	values := p.activeBucket()
	p.require.NotEmpty(values)

	for i, value := range values {
		p.assert.Equal(values[0], value, "tracked value %d", i)
	}

	return p
}

// allTrackedValuesAreValidUUIDs asserts every value in the active bucket parses as a UUID.
func (p *parts) allTrackedValuesAreValidUUIDs() *parts {
	for _, value := range p.activeBucket() {
		_, err := uuid.Parse(value)
		p.require.NoError(err, "value %q should be a valid UUID", value)
	}

	return p
}

// allTrackedValuesArePositiveIntegers asserts every value in the active bucket parses as
// an integer greater than zero.
func (p *parts) allTrackedValuesArePositiveIntegers() *parts {
	for _, value := range p.activeBucket() {
		number, err := strconv.ParseInt(value, 10, 64)
		p.require.NoError(err, "value %q should be an integer", value)
		p.require.Greater(number, int64(0), "value %q should be positive", value)
	}

	return p
}

// firstTrackedValueIsIntegerInRange asserts the first value of the active bucket parses as
// an integer within [lowest, highest].
func (p *parts) firstTrackedValueIsIntegerInRange(lowest, highest int64) *parts {
	values := p.activeBucket()
	p.require.NotEmpty(values)

	number, err := strconv.ParseInt(values[0], 10, 64)
	p.require.NoError(err, "value %q should be an integer", values[0])
	p.require.GreaterOrEqual(number, lowest)
	p.require.LessOrEqual(number, highest)

	return p
}

// activeBucket returns the values of the currently active tracking bucket.
func (p *parts) activeBucket() []string {
	p.require.NotEmpty(p.activeTrack)

	values, ok := p.trackedValues[p.activeTrack]
	p.require.True(ok, "no tracking bucket named %q", p.activeTrack)

	return values
}

// track appends a captured value to the active tracking bucket.
func (p *parts) track(value string) {
	p.trackedValues[p.activeTrack] = append(p.trackedValues[p.activeTrack], value)
}

// capturedJSONLeaf walks the given path through the captured JSON body at index and
// returns the leaf value, requiring every step to resolve.
func (p *parts) capturedJSONLeaf(index int, path []string) any {
	p.require.Greater(len(p.capturedBodies), index)

	var data map[string]any
	p.require.NoError(json.Unmarshal([]byte(p.capturedBodies[index]), &data))

	var current any = data
	for _, step := range path {
		asMap, ok := current.(map[string]any)
		p.require.True(ok, "field %v not found in body of request %d", path, index)

		current, ok = asMap[step]
		p.require.True(ok, "field %v not found in body of request %d", path, index)
	}

	return current
}

// dslSymbolsExt keeps the batch-2b DSL extensions referenced alongside dslSymbols
// in fluent_parts_test.go.
var _ = []any{
	(*parts).anExternalFile,
	(*parts).anExternalFileBytes,
	(*parts).aFormattedRequestFixture,
	(*parts).aTemplateFixture,
	(*parts).aMissingFile,
	(*parts).parsingFile,
	(*parts).executingRequestAt,
	(*parts).parseSucceeded,
	(*parts).parseErrorContains,
	(*parts).parsedRequestCount,
	(*parts).parsedRequestMethod,
	(*parts).parsedRequestName,
	(*parts).parsedRequestRawURLIs,
	(*parts).requestURLContains,
	(*parts).requestHeaderIs,
	(*parts).requestRawBodyIs,
	(*parts).requestRawBodyContains,
	(*parts).requestRawBodyMatchesRegexp,
	(*parts).serverReceivedBodyIs,
	(*parts).serverReceivedBodyParsesAsJSON,
	(*parts).serverReceivedLatin1BodyIs,
	(*parts).allResponseCodesAre,
	(*parts).tracking,
	(*parts).capturedURLSegment,
	(*parts).capturedHeaderField,
	(*parts).capturedJSONField,
	(*parts).capturedJSONNumberField,
	(*parts).allTrackedValuesEqual,
	(*parts).allTrackedValuesAreValidUUIDs,
	(*parts).allTrackedValuesArePositiveIntegers,
	(*parts).firstTrackedValueIsIntegerInRange,
}
