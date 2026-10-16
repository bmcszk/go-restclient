package restclient_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// captureParts holds the capture/tracking domain state of the fluent test DSL:
// everything recorded by the test server, the mock transport client and the
// consistency-tracking buckets.
type captureParts struct {
	// capturedRequests holds every request the test server received.
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
	// activeTrack names the bucket captured* methods append into.
	activeTrack string
}

func (p *testFixture) requestURLContains(fragments ...string) *testFixture {
	current := p.current()
	p.require.NotNil(current.Request)

	for _, fragment := range fragments {
		p.assert.Contains(current.Request.URL.String(), fragment)
	}

	return p
}

func (p *testFixture) requestHeaderIs(key, value string) *testFixture {
	current := p.current()
	p.require.NotNil(current.Request)
	p.assert.Equal(value, current.Request.Headers.Get(key))

	return p
}

func (p *testFixture) requestRawBodyIs(body string) *testFixture {
	current := p.current()
	p.require.NotNil(current.Request)
	p.assert.Equal(body, current.Request.RawBody)

	return p
}

func (p *testFixture) requestRawBodyContains(fragments ...string) *testFixture {
	current := p.current()
	p.require.NotNil(current.Request)

	for _, fragment := range fragments {
		p.assert.Contains(current.Request.RawBody, fragment)
	}

	return p
}

func (p *testFixture) requestRawBodyMatchesRegexp(pattern string) *testFixture {
	current := p.current()
	p.require.NotNil(current.Request)
	p.assert.Regexp(pattern, current.Request.RawBody)

	return p
}

// serverReceivedBodyIs asserts the exact body captured for the server hit at index.
func (p *testFixture) serverReceivedBodyIs(index int, body string) *testFixture {
	p.require.Greater(len(p.capturedBodies), index)
	p.assert.Equal(body, p.capturedBodies[index])

	return p
}

// serverReceivedBodyParsesAsJSON asserts the captured body at index is valid JSON.
func (p *testFixture) serverReceivedBodyParsesAsJSON(index int) *testFixture {
	p.require.Greater(len(p.capturedBodies), index)

	var data map[string]any
	p.require.NoError(json.Unmarshal([]byte(p.capturedBodies[index]), &data))

	return p
}

// serverReceivedLatin1BodyIs asserts the captured body at index equals the expected text
// after decoding it from ISO-8859-1.
func (p *testFixture) serverReceivedLatin1BodyIs(index int, body string) *testFixture {
	p.require.Greater(len(p.capturedBodies), index)

	decoded, _, err := transform.Bytes(charmap.ISO8859_1.NewDecoder(), []byte(p.capturedBodies[index]))
	p.require.NoError(err)
	p.assert.Equal(body, string(decoded))

	return p
}

func (p *testFixture) allResponseCodesAre(code int) *testFixture {
	p.require.NotEmpty(p.responses)

	for i, resp := range p.responses {
		p.require.Equal(code, resp.StatusCode, "response %d", i)
	}

	return p
}

// tracking starts (or resumes) a named consistency-tracking bucket for the captured*
// assertion methods that follow.
func (p *testFixture) tracking(name string) *testFixture {
	p.activeTrack = name

	if _, ok := p.trackedValues[name]; !ok {
		p.trackedValues[name] = []string{}
	}

	return p
}

func (p *testFixture) capturedURLSegment(index int) *testFixture {
	p.require.Greater(len(p.capturedRequests), index)

	segments := strings.Split(p.capturedRequests[index].URL.Path, "/")
	p.track(segments[len(segments)-1])

	return p
}

func (p *testFixture) capturedURLSegmentAt(reqIndex, segmentIndex int) *testFixture {
	p.require.Greater(len(p.capturedRequests), reqIndex)

	segments := strings.Split(p.capturedRequests[reqIndex].URL.Path, "/")
	p.require.Greater(len(segments), segmentIndex, "segment %d not present", segmentIndex)
	p.track(segments[segmentIndex])

	return p
}

func (p *testFixture) capturedHeaderField(index int, key string) *testFixture {
	p.require.Greater(len(p.capturedRequests), index)
	p.track(p.capturedRequests[index].Header.Get(key))

	return p
}

func (p *testFixture) capturedJSONField(index int, path ...string) *testFixture {
	leaf, ok := p.capturedJSONLeaf(index, path).(string)
	p.require.True(ok, "field %v is not a string", path)
	p.track(leaf)

	return p
}

func (p *testFixture) capturedJSONNumberField(index int, path ...string) *testFixture {
	value, ok := p.capturedJSONLeaf(index, path).(float64)
	p.require.True(ok, "field %v is not a number", path)
	p.track(strconv.FormatInt(int64(value), 10))

	return p
}

func (p *testFixture) allTrackedValuesEqual() *testFixture {
	values := p.activeBucket()
	p.require.NotEmpty(values)

	for i, value := range values {
		p.assert.Equal(values[0], value, "tracked value %d", i)
	}

	return p
}

func (p *testFixture) allTrackedValuesAreValidUUIDs() *testFixture {
	for _, value := range p.activeBucket() {
		_, err := uuid.Parse(value)
		p.require.NoError(err, "value %q should be a valid UUID", value)
	}

	return p
}

func (p *testFixture) allTrackedValuesArePositiveIntegers() *testFixture {
	for _, value := range p.activeBucket() {
		number, err := strconv.ParseInt(value, 10, 64)
		p.require.NoError(err, "value %q should be an integer", value)
		p.require.Greater(number, int64(0), "value %q should be positive", value)
	}

	return p
}

// firstTrackedValueIsIntegerInRange asserts the first value of the active bucket parses as
// an integer within [lowest, highest].
func (p *testFixture) firstTrackedValueIsIntegerInRange(lowest, highest int64) *testFixture {
	values := p.activeBucket()
	p.require.NotEmpty(values)

	number, err := strconv.ParseInt(values[0], 10, 64)
	p.require.NoError(err, "value %q should be an integer", values[0])
	p.require.GreaterOrEqual(number, lowest)
	p.require.LessOrEqual(number, highest)

	return p
}

// activeBucket returns the values of the currently active tracking bucket.
func (p *testFixture) activeBucket() []string {
	p.require.NotEmpty(p.activeTrack)

	values, ok := p.trackedValues[p.activeTrack]
	p.require.True(ok, "no tracking bucket named %q", p.activeTrack)

	return values
}

// track appends a captured value to the active tracking bucket.
func (p *testFixture) track(value string) {
	p.trackedValues[p.activeTrack] = append(p.trackedValues[p.activeTrack], value)
}

func (p *testFixture) capturedJSONLeaf(index int, path []string) any {
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

func (p *testFixture) capturedRequestURLIs(index int, rawURL string) *testFixture {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Equal(rawURL, p.capturedRequests[index].URL.String())

	return p
}

func (p *testFixture) capturedRequestPathIs(index int, path string) *testFixture {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Equal(path, p.capturedRequests[index].URL.Path)

	return p
}

// serverReceivedHostIs asserts p.capturedRequests[index].Host equals host.
func (p *testFixture) serverReceivedHostIs(index int, host string) *testFixture {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Equal(host, p.capturedRequests[index].Host)

	return p
}

// capturedJSONStringMapIs decodes the captured body at index as a flat string
// map and asserts each key/value equals.
func (p *testFixture) capturedJSONStringMapIs(index int, expected map[string]string) *testFixture {
	p.require.Greater(len(p.capturedBodies), index)

	var data map[string]string
	p.require.NoError(json.Unmarshal([]byte(p.capturedBodies[index]), &data))

	for key, want := range expected {
		p.assert.Equal(want, data[key], "key %q mismatch", key)
	}

	return p
}

func (p *testFixture) requestPathMatchesCapturedPath() *testFixture {
	current := p.current()
	p.require.NotNil(current.Request)
	p.require.Greater(len(p.capturedRequests), 0)
	p.assert.Equal(p.capturedRequests[0].URL.Path, current.Request.URL.Path)

	return p
}

func (p *testFixture) requestHeadersMatchCaptured(keys ...string) *testFixture {
	current := p.current()
	p.require.NotNil(current.Request)
	p.require.Greater(len(p.capturedRequests), 0)

	for _, key := range keys {
		p.assert.Equal(p.capturedRequests[0].Header.Get(key), current.Request.Headers.Get(key), "header %q", key)
	}

	return p
}

// requestRawBodyMatchesCapturedBody asserts the request RawBody equals the first captured body.
func (p *testFixture) requestRawBodyMatchesCapturedBody() *testFixture {
	current := p.current()
	p.require.NotNil(current.Request)
	p.require.Greater(len(p.capturedBodies), 0)
	p.assert.Equal(p.capturedBodies[0], current.Request.RawBody)

	return p
}

func (p *testFixture) allTrackedValuesAreValidRFC3339Timestamps() *testFixture {
	for _, value := range p.activeBucket() {
		_, err := time.Parse(time.RFC3339Nano, value)
		p.require.NoError(err, "value %q should be a valid RFC3339Nano timestamp", value)
	}

	return p
}

// allTrackedDatetimeValuesAreWithin parses tracked values ("timestamp" = Unix seconds) and checks recency.
func (p *testFixture) allTrackedDatetimeValuesAreWithin(threshold time.Duration, layout string) *testFixture {
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

// allTrackedDatetimeValuesHaveUTCZone asserts tracked RFC1123/RFC3339/Unix values resolve to UTC.
func (p *testFixture) allTrackedDatetimeValuesHaveUTCZone() *testFixture {
	values := p.activeBucket()
	p.require.NotEmpty(values)

	for _, value := range values {
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

// allTrackedDatetimeValuesHaveLocalZone asserts tracked datetime values resolve to the local zone.
func (p *testFixture) allTrackedDatetimeValuesHaveLocalZone() *testFixture {
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

func (p *testFixture) capturedJSONFieldIs(index int, want string, path ...string) *testFixture {
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

func (p *testFixture) capturedJSONFieldContains(index int, fragment string, path ...string) *testFixture {
	leaf, ok := p.capturedJSONLeaf(index, path).(string)
	p.require.True(ok, "field %v is not a string", path)
	p.assert.Contains(leaf, fragment)

	return p
}

func (p *testFixture) capturedJSONFieldMatchesRegexp(index int, pattern string, path ...string) *testFixture {
	leaf, ok := p.capturedJSONLeaf(index, path).(string)
	p.require.True(ok, "field %v is not a string", path)
	p.assert.Regexp(pattern, leaf)

	return p
}

func (p *testFixture) capturedJSONFieldNotContains(index int, fragment string, path ...string) *testFixture {
	leaf, ok := p.capturedJSONLeaf(index, path).(string)
	p.require.True(ok, "field %v is not a string", path)
	p.assert.NotContains(leaf, fragment)

	return p
}

func (p *testFixture) capturedRequestPathMatchesRegexp(index int, pattern string) *testFixture {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Regexp(pattern, p.capturedRequests[index].URL.Path)

	return p
}

// gapBetweenCapturedRequestTimesAtLeast asserts captured-request times[ j] - [i] >= minGap.
func (p *testFixture) gapBetweenCapturedRequestTimesAtLeast(i, j int, minGap time.Duration) *testFixture {
	p.require.Greater(len(p.capturedRequestTimes), i, "no captured-request time at index %d", i)
	p.require.Greater(len(p.capturedRequestTimes), j, "no captured-request time at index %d", j)
	p.assert.GreaterOrEqual(p.capturedRequestTimes[j].Sub(p.capturedRequestTimes[i]), minGap,
		"gap between captured-request times %d→%d was below %s", i, j, minGap)

	return p
}

// firstTrackedValueIsFloatInRange asserts the first value of the active bucket parses as
// a float within [lowest, highest].
func (p *testFixture) firstTrackedValueIsFloatInRange(lowest, highest float64) *testFixture {
	values := p.activeBucket()
	p.require.NotEmpty(values)

	number, err := strconv.ParseFloat(values[0], 64)
	p.require.NoError(err, "value %q should be a float", values[0])
	p.assert.GreaterOrEqual(number, lowest)
	p.assert.LessOrEqual(number, highest)

	return p
}

func (p *testFixture) allTrackedValuesMatchRegexp(pattern string) *testFixture {
	re := regexp.MustCompile(pattern)
	for _, value := range p.activeBucket() {
		p.assert.True(re.MatchString(value), "value %q should match %q", value, pattern)
	}

	return p
}

func (p *testFixture) capturedBodyContains(index int, fragment string) *testFixture {
	p.require.Greater(len(p.capturedBodies), index)
	p.assert.Contains(p.capturedBodies[index], fragment)

	return p
}

func (p *testFixture) capturedBodyMatchesRegexp(index int, pattern string) *testFixture {
	p.require.Greater(len(p.capturedBodies), index)
	p.assert.Regexp(pattern, p.capturedBodies[index])

	return p
}

func (p *testFixture) serverReceivedBodyLengthIs(index int, want int) *testFixture {
	p.require.Greater(len(p.capturedRequests), index)
	p.require.Equal(want, len(p.capturedBodies[index]))

	return p
}

// serverReceivedBodyMatches asserts the captured request body against a regexp.
func (p *testFixture) serverReceivedBodyMatches(index int, pattern string) *testFixture {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Regexp(pattern, p.capturedBodies[index])

	return p
}

// trackedValueIs asserts the most recent tracked value equals want.
func (p *testFixture) trackedValueIs(want string) *testFixture {
	values, ok := p.trackedValues[p.activeTrack]
	p.require.True(ok, "no tracking bucket %q active", p.activeTrack)
	p.require.NotEmpty(values)
	p.assert.Equal(want, values[len(values)-1])

	return p
}

// clientRequestInterceptorCaptures asserts the request captured by aMockTransportClient
// has the given method and full URL.
func (p *testFixture) clientRequestInterceptorCaptures(method, wantURL string) *testFixture {
	p.require.NotNil(p.intercepted)
	p.assert.Equal(method, p.intercepted.Method)
	p.assert.Equal(wantURL, p.intercepted.URL.String())

	return p
}

// clientSentNoHeaders asserts the request captured by aMockTransportClient sent no headers.
func (p *testFixture) clientSentNoHeaders() *testFixture {
	p.require.NotNil(p.intercepted)
	p.assert.Empty(p.intercepted.Header)

	return p
}

// serverReceivedMethodAndPath asserts the request at the given server-hit index used the
// expected HTTP method and path.
func (p *testFixture) serverReceivedMethodAndPath(index int, method, path string) *testFixture {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Equal(method, p.capturedRequests[index].Method)
	p.assert.Equal(path, p.capturedRequests[index].URL.Path)

	return p
}

// serverReceivedJSONBody asserts the request body at the given server-hit index equals the
// expected JSON regardless of key order or whitespace.
func (p *testFixture) serverReceivedJSONBody(index int, expectedJSON string) *testFixture {
	p.require.Greater(len(p.capturedRequests), index)

	body, err := io.ReadAll(p.capturedRequests[index].Body)
	p.require.NoError(err)
	p.assert.JSONEq(expectedJSON, string(body))

	return p
}

// serverReceivedHeaderValue asserts a request header value at the given server-hit index.
func (p *testFixture) serverReceivedHeaderValue(index int, key, value string) *testFixture {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Equal(value, p.capturedRequests[index].Header.Get(key))

	return p
}

// requestRawURLContains asserts the current request's RawURLString contains every fragment.
func (p *testFixture) requestRawURLContains(fragments ...string) *testFixture {
	current := p.current()
	p.require.NotNil(current.Request)

	for _, fragment := range fragments {
		p.assert.Contains(current.Request.RawURLString, fragment)
	}

	return p
}

// capturedRequestCount asserts how many requests the test server received.
func (p *testFixture) capturedRequestCount(n int) *testFixture {
	p.require.Equal(n, len(p.capturedRequests))

	return p
}

// requestSkipped asserts the response at index i is marked as skipped.
func (p *testFixture) requestSkipped(i int) *testFixture {
	p.require.Greater(len(p.responses), i)
	p.assert.True(p.responses[i].Skipped)

	return p
}

// requestExecuted asserts the response at index i is not marked as skipped.
func (p *testFixture) requestExecuted(i int) *testFixture {
	p.require.Greater(len(p.responses), i)
	p.assert.False(p.responses[i].Skipped)

	return p
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
func (p *testFixture) serverReceivedFakerHeaders(reqIndex int, rules []fakerHeaderRule) *testFixture {
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
