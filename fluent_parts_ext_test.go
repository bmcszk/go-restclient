package restclient_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"

	rc "github.com/bmcszk/go-restclient"
)

// introduced by the batch-2b test migration. Same conventions: state lives in parts,
// assertions come from parts.require / parts.assert, .and() stays at end of line.

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

// aFixtureCopy reads the committed fixture file at repo-root-relative requestPath, applies
// under test. Used for fixtures that need per-test substitution beyond the [[ ]] templating
// (e.g. http-client.env.json handling) or plain copies of committed fixtures that must live
// next to a .env file.
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

// anEnvJsonFile reads committed env template file templatePath, replaces "{{SERVER_URL}}"
// with serverURL, writes to <baseDir>/fileName (0600). For http-client.env.json /
// http-client.private.env.json setups.
func (p *parts) anEnvJsonFile(fileName, templatePath, serverURL string) *parts {
	content, err := os.ReadFile(templatePath)
	p.require.NoError(err)
	resolved := strings.ReplaceAll(string(content), "{{SERVER_URL}}", serverURL)
	path := filepath.Join(p.baseDir, fileName)
	p.require.NoError(os.WriteFile(path, []byte(resolved), 0600))
	return p
}

// aClientWithEnvironment builds the client via rc.NewClient(rc.WithEnvironment(env)) when env
// is not empty, else rc.NewClient(); stores it in p.client. env == "" means no environment
// selected (the http-client.env.json lookup is skipped).
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

// aRequestFixtureAbs points the DSL at a repo-root-relative request fixture file
// without prepending the default requestFilesDir. Used for fixtures that live
// under subdirectories other than test/data/http_request_files.
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

func (p *parts) requestURLContains(fragments ...string) *parts {
	current := p.current()
	p.require.NotNil(current.Request)

	for _, fragment := range fragments {
		p.assert.Contains(current.Request.URL.String(), fragment)
	}

	return p
}

func (p *parts) requestHeaderIs(key, value string) *parts {
	current := p.current()
	p.require.NotNil(current.Request)
	p.assert.Equal(value, current.Request.Headers.Get(key))

	return p
}

func (p *parts) requestRawBodyIs(body string) *parts {
	current := p.current()
	p.require.NotNil(current.Request)
	p.assert.Equal(body, current.Request.RawBody)

	return p
}
func (p *parts) requestRawBodyContains(fragments ...string) *parts {
	current := p.current()
	p.require.NotNil(current.Request)

	for _, fragment := range fragments {
		p.assert.Contains(current.Request.RawBody, fragment)
	}

	return p
}

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
func (p *parts) capturedURLSegment(index int) *parts {
	p.require.Greater(len(p.capturedRequests), index)

	segments := strings.Split(p.capturedRequests[index].URL.Path, "/")
	p.track(segments[len(segments)-1])

	return p
}
func (p *parts) capturedURLSegmentAt(reqIndex, segmentIndex int) *parts {
	p.require.Greater(len(p.capturedRequests), reqIndex)

	segments := strings.Split(p.capturedRequests[reqIndex].URL.Path, "/")
	p.require.Greater(len(segments), segmentIndex, "segment %d not present", segmentIndex)
	p.track(segments[segmentIndex])

	return p
}
func (p *parts) capturedHeaderField(index int, key string) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.track(p.capturedRequests[index].Header.Get(key))

	return p
}
func (p *parts) capturedJSONField(index int, path ...string) *parts {
	leaf, ok := p.capturedJSONLeaf(index, path).(string)
	p.require.True(ok, "field %v is not a string", path)
	p.track(leaf)

	return p
}
func (p *parts) capturedJSONNumberField(index int, path ...string) *parts {
	value, ok := p.capturedJSONLeaf(index, path).(float64)
	p.require.True(ok, "field %v is not a number", path)
	p.track(strconv.FormatInt(int64(value), 10))

	return p
}

func (p *parts) allTrackedValuesEqual() *parts {
	values := p.activeBucket()
	p.require.NotEmpty(values)

	for i, value := range values {
		p.assert.Equal(values[0], value, "tracked value %d", i)
	}

	return p
}

func (p *parts) allTrackedValuesAreValidUUIDs() *parts {
	for _, value := range p.activeBucket() {
		_, err := uuid.Parse(value)
		p.require.NoError(err, "value %q should be a valid UUID", value)
	}

	return p
}
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

func (p *parts) capturedRequestURLIs(index int, rawURL string) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Equal(rawURL, p.capturedRequests[index].URL.String())

	return p
}

func (p *parts) capturedRequestPathIs(index int, path string) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Equal(path, p.capturedRequests[index].URL.Path)

	return p
}

// serverReceivedHostIs asserts p.capturedRequests[index].Host equals host.
func (p *parts) serverReceivedHostIs(index int, host string) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Equal(host, p.capturedRequests[index].Host)

	return p
}

// .NoError) and asserts each key/value equals.
func (p *parts) capturedJSONStringMapIs(index int, expected map[string]string) *parts {
	p.require.Greater(len(p.capturedBodies), index)

	var data map[string]string
	p.require.NoError(json.Unmarshal([]byte(p.capturedBodies[index]), &data))

	for key, want := range expected {
		p.assert.Equal(want, data[key], "key %q mismatch", key)
	}

	return p
}
func (p *parts) requestPathMatchesCapturedPath() *parts {
	current := p.current()
	p.require.NotNil(current.Request)
	p.require.Greater(len(p.capturedRequests), 0)
	p.assert.Equal(p.capturedRequests[0].URL.Path, current.Request.URL.Path)

	return p
}
func (p *parts) requestHeadersMatchCaptured(keys ...string) *parts {
	current := p.current()
	p.require.NotNil(current.Request)
	p.require.Greater(len(p.capturedRequests), 0)

	for _, key := range keys {
		p.assert.Equal(p.capturedRequests[0].Header.Get(key), current.Request.Headers.Get(key), "header %q", key)
	}

	return p
}

// p.capturedBodies[0]. Together with server-side tracked equality this gives full
// 1:1 parity for body fields.
func (p *parts) requestRawBodyMatchesCapturedBody() *parts {
	current := p.current()
	p.require.NotNil(current.Request)
	p.require.Greater(len(p.capturedBodies), 0)
	p.assert.Equal(p.capturedBodies[0], current.Request.RawBody)

	return p
}
func (p *parts) allTrackedValuesAreValidRFC3339Timestamps() *parts {
	for _, value := range p.activeBucket() {
		_, err := time.Parse(time.RFC3339Nano, value)
		p.require.NoError(err, "value %q should be a valid RFC3339Nano timestamp", value)
	}

	return p
}

// the given layout (layout "timestamp" means Unix seconds; time.Unix is used) and falls
// within threshold of time.Now(). For non-timestamp layouts time.Parse is used.
func (p *parts) allTrackedDatetimeValuesAreWithin(threshold time.Duration, layout string) *parts {
	now := time.Now()
	values := p.activeBucket()
	p.require.NotEmpty(values)

	for _, value := range values {
		var parsed time.Time
		if layout == "timestamp" {
			ts, err := strconv.ParseInt(value, 10, 64)
			p.require.NoError(err, "value %q should parse as integer", value)
			parsed = time.Unix(ts, 0)
		} else {
			parsedTime, err := time.Parse(layout, value)
			p.require.NoError(err, "value %q should parse with layout %q", value, layout)
			parsed = parsedTime
		}

		p.assert.WithinDuration(now, parsed, threshold,
			"datetime %s not within %s of now %s", parsed, threshold, now)
	}

	return p
}

// time.UTC. The bucket values may be RFC1123, RFC3339 or Unix-seconds timestamps; all of
// these resolve to a location whose UTC offset matches time.UTC.
func (p *parts) allTrackedDatetimeValuesHaveUTCZone() *parts {
	values := p.activeBucket()
	p.require.NotEmpty(values)

	for _, value := range values {
		// Try RFC1123, RFC3339 and Unix-seconds in turn.
		var parsed time.Time
		var err error
		if parsed, err = time.Parse(time.RFC1123, value); err != nil {
			if parsed, err = time.Parse(time.RFC3339, value); err != nil {
				ts, tsErr := strconv.ParseInt(value, 10, 64)
				p.require.NoError(tsErr, "value %q should parse as integer, RFC3339 or RFC1123", value)
				parsed = time.Unix(ts, 0).UTC()
			}
		}

		p.assert.Equal(time.UTC, parsed.Location(), "value %q expected UTC zone", value)
	}

	return p
}

// to the local timezone offset. Layout "timestamp" uses time.Unix().In(time.Local) and
// accepts any offset.
func (p *parts) allTrackedDatetimeValuesHaveLocalZone() *parts {
	values := p.activeBucket()
	p.require.NotEmpty(values)
	_, wantOffset := time.Now().In(time.Local).Zone()

	for _, value := range values {
		var parsed time.Time
		var err error
		if parsed, err = time.Parse(time.RFC1123, value); err != nil {
			if parsed, err = time.Parse(time.RFC3339, value); err != nil {
				ts, tsErr := strconv.ParseInt(value, 10, 64)
				p.require.NoError(tsErr, "value %q should parse as integer, RFC3339 or RFC1123", value)
				parsed = time.Unix(ts, 0).In(time.Local)
			}
		}

		_, gotOffset := parsed.Zone()
		p.assert.Equal(wantOffset, gotOffset,
			"value %q expected local offset %d, got %d", value, wantOffset, gotOffset)
	}

	return p
}

func (p *parts) capturedJSONFieldIs(index int, want string, path ...string) *parts {
	leaf := p.capturedJSONLeaf(index, path)
	switch v := leaf.(type) {
	case string:
		p.assert.Equal(want, v)
	case float64:
		p.assert.Equal(want, strconv.FormatInt(int64(v), 10))
	default:
		p.assert.Equal(want, fmt.Sprint(v))
	}

	return p
}

func (p *parts) capturedJSONFieldContains(index int, fragment string, path ...string) *parts {
	leaf, ok := p.capturedJSONLeaf(index, path).(string)
	p.require.True(ok, "field %v is not a string", path)
	p.assert.Contains(leaf, fragment)

	return p
}
func (p *parts) capturedJSONFieldMatchesRegexp(index int, pattern string, path ...string) *parts {
	leaf, ok := p.capturedJSONLeaf(index, path).(string)
	p.require.True(ok, "field %v is not a string", path)
	p.assert.Regexp(pattern, leaf)

	return p
}
func (p *parts) capturedJSONFieldNotContains(index int, fragment string, path ...string) *parts {
	leaf, ok := p.capturedJSONLeaf(index, path).(string)
	p.require.True(ok, "field %v is not a string", path)
	p.assert.NotContains(leaf, fragment)

	return p
}

// serverReceivedHeaderMatchesRegexp asserts capturedRequests[index].Header.Get(key) matches pattern.
func (p *parts) serverReceivedHeaderMatchesRegexp(index int, key, pattern string) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Regexp(pattern, p.capturedRequests[index].Header.Get(key))

	return p
}

// serverReceivedHeaderNotContains asserts capturedRequests[index].Header.Get(key) does not
// contain fragment.
func (p *parts) serverReceivedHeaderNotContains(index int, key, fragment string) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.NotContains(p.capturedRequests[index].Header.Get(key), fragment)

	return p
}

// serverReceivedHeaderFieldCountIs asserts the captured header value splits into n
// whitespace-separated fields.
func (p *parts) serverReceivedHeaderFieldCountIs(index int, key string, n int) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Len(strings.Fields(p.capturedRequests[index].Header.Get(key)), n)

	return p
}

// serverReceivedHeaderContains asserts capturedRequests[index].Header.Get(key) contains fragment.
func (p *parts) serverReceivedHeaderContains(index int, key, fragment string) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Contains(p.capturedRequests[index].Header.Get(key), fragment)

	return p
}

func (p *parts) capturedRequestPathMatchesRegexp(index int, pattern string) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Regexp(pattern, p.capturedRequests[index].URL.Path)

	return p
}
func (p *parts) responsesValidateAgainst(expectedPath string) *parts {
	p.require.NotNil(p.client)
	p.assert.NoError(p.client.ValidateResponses(expectedPath, p.responses...))

	return p
}

// firstTrackedValueIsFloatInRange asserts the first value of the active bucket parses as
// a float within [lowest, highest].
func (p *parts) firstTrackedValueIsFloatInRange(lowest, highest float64) *parts {
	values := p.activeBucket()
	p.require.NotEmpty(values)

	number, err := strconv.ParseFloat(values[0], 64)
	p.require.NoError(err, "value %q should be a float", values[0])
	p.assert.GreaterOrEqual(number, lowest)
	p.assert.LessOrEqual(number, highest)

	return p
}

func (p *parts) allTrackedValuesMatchRegexp(pattern string) *parts {
	re := regexp.MustCompile(pattern)
	for _, value := range p.activeBucket() {
		p.assert.True(re.MatchString(value), "value %q should match %q", value, pattern)
	}

	return p
}

func (p *parts) capturedBodyContains(index int, fragment string) *parts {
	p.require.Greater(len(p.capturedBodies), index)
	p.assert.Contains(p.capturedBodies[index], fragment)

	return p
}

// aHttpFileWithExternalFileEncoding writes a short request body that points at the named
// external file with the given encoding directive (e.g. "latin1", "utf-8", "ascii"), and
// stores the raw encoded content in an external file of the given name. The fixture request
// body is rendered with [[.ServerURL]] substituted for the running test server. Encapsulating
// the transform.Bytes encoding step here keeps the test func free of bytes/encoder logic.
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
// file via <@, and stores the given JSON content in the external file. Replaces the older
// inline fmt.Sprintf(...) + inline JSON literal in test funcs (see PR review comments 4/5).
func (p *parts) aHttpFileWithExternalFile(externalFileName, externalFileContent string) *parts {
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, externalFileName), []byte(externalFileContent), 0644))

	return p.aHttpFileFromTemplate("external_file_with_vars.http")
}

// aHttpFileWithExternalFileStatic writes a short request body that points at the named
// external file via < (no @: NO variable substitution). Used by tests that need the raw
// external file bytes verbatim (PR review comment 4 generalization).
func (p *parts) aHttpFileWithExternalFileStatic(externalFileName, externalFileContent string) *parts {
	p.require.NoError(os.WriteFile(filepath.Join(p.baseDir, externalFileName), []byte(externalFileContent), 0644))

	return p.aHttpFileFromTemplate("external_file_static.http")
}

// fakerHeaderRule describes one header validation rule for the faker tests.
// fieldCount == -1 (or 0) skips the field-count check. containsCheck != "" switches the
// check from regex-match to substring-contain (and skips the field-count check too).
type fakerHeaderRule struct {
	key           string
	pattern       string
	fieldCount    int
	containsCheck string
}

// serverReceivedFakerHeaders applies the standard set of faker-header assertions (regex
// match + NotContains "{{" + optional field-count or contains check) to the captured request
// at reqIndex, for every rule in rules. This is the single DSL entry point for faker header
// validation across all faker-data tests; tests pass an inline []fakerHeaderRule literal.
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

func (p *parts) capturedBodyMatchesRegexp(index int, pattern string) *parts {
	p.require.Greater(len(p.capturedBodies), index)
	p.assert.Regexp(pattern, p.capturedBodies[index])

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

// dslSymbolsExt keeps the batch-2b/2c DSL extensions referenced alongside dslSymbols
// in fluent_parts_test.go.
var _ = []any{
	(*parts).anExternalFile,
	(*parts).anExternalFileBytes,
	(*parts).aFormattedRequestFixture,
	(*parts).aTemplateFixture,
	(*parts).aRequestFixtureAbs,
	(*parts).aMissingFile,
	(*parts).aDotEnvFile,
	(*parts).aDotEnvFileRemoved,
	(*parts).aFixtureCopy,
	(*parts).anEnvJsonFile,
	(*parts).aClientWithEnvironment,
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
	(*parts).requestPathMatchesCapturedPath,
	(*parts).requestHeadersMatchCaptured,
	(*parts).requestRawBodyMatchesCapturedBody,
	(*parts).serverReceivedBodyIs,
	(*parts).serverReceivedBodyParsesAsJSON,
	(*parts).serverReceivedLatin1BodyIs,
	(*parts).allResponseCodesAre,
	(*parts).capturedRequestURLIs,
	(*parts).capturedRequestPathIs,
	(*parts).serverReceivedHostIs,
	(*parts).capturedJSONStringMapIs,
	(*parts).tracking,
	(*parts).capturedURLSegment,
	(*parts).capturedURLSegmentAt,
	(*parts).capturedHeaderField,
	(*parts).capturedJSONField,
	(*parts).capturedJSONNumberField,
	(*parts).allTrackedValuesEqual,
	(*parts).allTrackedValuesAreValidUUIDs,
	(*parts).allTrackedValuesArePositiveIntegers,
	(*parts).firstTrackedValueIsIntegerInRange,
	(*parts).firstTrackedValueIsFloatInRange,
	(*parts).allTrackedValuesMatchRegexp,
	(*parts).allTrackedValuesAreValidRFC3339Timestamps,
	(*parts).allTrackedDatetimeValuesAreWithin,
	(*parts).allTrackedDatetimeValuesHaveUTCZone,
	(*parts).allTrackedDatetimeValuesHaveLocalZone,
	(*parts).capturedJSONFieldIs,
	(*parts).capturedJSONFieldContains,
	(*parts).capturedJSONFieldMatchesRegexp,
	(*parts).capturedJSONFieldNotContains,
	(*parts).serverReceivedHeaderMatchesRegexp,
	(*parts).serverReceivedHeaderNotContains,
	(*parts).serverReceivedHeaderFieldCountIs,
	(*parts).serverReceivedHeaderContains,
	(*parts).capturedRequestPathMatchesRegexp,
	(*parts).responsesValidateAgainst,
	(*parts).capturedBodyContains,
	(*parts).capturedBodyMatchesRegexp,
	(*parts).aHttpFileWithExternalFileEncoding,
	(*parts).aHttpFileWithExternalFile,
	(*parts).aHttpFileWithExternalFileStatic,
	(*parts).serverReceivedFakerHeaders,
}
