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
	*parts
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

var _ = []any{
	captureParts{},
}

func (c *captureParts) requestURLContains(fragments ...string) *captureParts {
	p := c.parts
	current := p.current()
	p.require.NotNil(current.Request)

	for _, fragment := range fragments {
		p.assert.Contains(current.Request.URL.String(), fragment)
	}

	return c
}

func (c *captureParts) requestHeaderIs(key, value string) *captureParts {
	p := c.parts
	current := p.current()
	p.require.NotNil(current.Request)
	p.assert.Equal(value, current.Request.Headers.Get(key))

	return c
}

func (c *captureParts) requestRawBodyIs(body string) *captureParts {
	p := c.parts
	current := p.current()
	p.require.NotNil(current.Request)
	p.assert.Equal(body, current.Request.RawBody)

	return c
}

func (c *captureParts) requestRawBodyContains(fragments ...string) *captureParts {
	p := c.parts
	current := p.current()
	p.require.NotNil(current.Request)

	for _, fragment := range fragments {
		p.assert.Contains(current.Request.RawBody, fragment)
	}

	return c
}

func (c *captureParts) requestRawBodyMatchesRegexp(pattern string) *captureParts {
	p := c.parts
	current := p.current()
	p.require.NotNil(current.Request)
	p.assert.Regexp(pattern, current.Request.RawBody)

	return c
}

// serverReceivedBodyIs asserts the exact body captured for the server hit at index.
func (c *captureParts) serverReceivedBodyIs(index int, body string) *captureParts {
	p := c.parts
	p.require.Greater(len(p.capturedBodies), index)
	p.assert.Equal(body, p.capturedBodies[index])

	return c
}

// serverReceivedBodyParsesAsJSON asserts the captured body at index is valid JSON.
func (c *captureParts) serverReceivedBodyParsesAsJSON(index int) *captureParts {
	p := c.parts
	p.require.Greater(len(p.capturedBodies), index)

	var data map[string]any
	p.require.NoError(json.Unmarshal([]byte(p.capturedBodies[index]), &data))

	return c
}

// serverReceivedLatin1BodyIs asserts the captured body at index equals the expected text
// after decoding it from ISO-8859-1.
func (c *captureParts) serverReceivedLatin1BodyIs(index int, body string) *captureParts {
	p := c.parts
	p.require.Greater(len(p.capturedBodies), index)

	decoded, _, err := transform.Bytes(charmap.ISO8859_1.NewDecoder(), []byte(p.capturedBodies[index]))
	p.require.NoError(err)
	p.assert.Equal(body, string(decoded))

	return c
}

func (c *captureParts) allResponseCodesAre(code int) *captureParts {
	p := c.parts
	p.require.NotEmpty(p.responses)

	for i, resp := range p.responses {
		p.require.Equal(code, resp.StatusCode, "response %d", i)
	}

	return c
}

// tracking starts (or resumes) a named consistency-tracking bucket for the captured*
// assertion methods that follow.
func (c *captureParts) tracking(name string) *captureParts {
	p := c.parts
	p.activeTrack = name

	if _, ok := p.trackedValues[name]; !ok {
		p.trackedValues[name] = []string{}
	}

	return c
}

func (c *captureParts) capturedURLSegment(index int) *captureParts {
	p := c.parts
	p.require.Greater(len(p.capturedRequests), index)

	segments := strings.Split(p.capturedRequests[index].URL.Path, "/")
	p.track(segments[len(segments)-1])

	return c
}

func (c *captureParts) capturedURLSegmentAt(reqIndex, segmentIndex int) *captureParts {
	p := c.parts
	p.require.Greater(len(p.capturedRequests), reqIndex)

	segments := strings.Split(p.capturedRequests[reqIndex].URL.Path, "/")
	p.require.Greater(len(segments), segmentIndex, "segment %d not present", segmentIndex)
	p.track(segments[segmentIndex])

	return c
}

func (c *captureParts) capturedHeaderField(index int, key string) *captureParts {
	p := c.parts
	p.require.Greater(len(p.capturedRequests), index)
	p.track(p.capturedRequests[index].Header.Get(key))

	return c
}

func (c *captureParts) capturedJSONField(index int, path ...string) *captureParts {
	p := c.parts
	leaf, ok := p.capturedJSONLeaf(index, path).(string)
	p.require.True(ok, "field %v is not a string", path)
	p.track(leaf)

	return c
}

func (c *captureParts) capturedJSONNumberField(index int, path ...string) *captureParts {
	p := c.parts
	value, ok := p.capturedJSONLeaf(index, path).(float64)
	p.require.True(ok, "field %v is not a number", path)
	p.track(strconv.FormatInt(int64(value), 10))

	return c
}

func (c *captureParts) allTrackedValuesEqual() *captureParts {
	p := c.parts
	values := p.activeBucket()
	p.require.NotEmpty(values)

	for i, value := range values {
		p.assert.Equal(values[0], value, "tracked value %d", i)
	}

	return c
}

func (c *captureParts) allTrackedValuesAreValidUUIDs() *captureParts {
	p := c.parts
	for _, value := range p.activeBucket() {
		_, err := uuid.Parse(value)
		p.require.NoError(err, "value %q should be a valid UUID", value)
	}

	return c
}

func (c *captureParts) allTrackedValuesArePositiveIntegers() *captureParts {
	p := c.parts
	for _, value := range p.activeBucket() {
		number, err := strconv.ParseInt(value, 10, 64)
		p.require.NoError(err, "value %q should be an integer", value)
		p.require.Greater(number, int64(0), "value %q should be positive", value)
	}

	return c
}

// firstTrackedValueIsIntegerInRange asserts the first value of the active bucket parses as
// an integer within [lowest, highest].
func (c *captureParts) firstTrackedValueIsIntegerInRange(lowest, highest int64) *captureParts {
	p := c.parts
	values := p.activeBucket()
	p.require.NotEmpty(values)

	number, err := strconv.ParseInt(values[0], 10, 64)
	p.require.NoError(err, "value %q should be an integer", values[0])
	p.require.GreaterOrEqual(number, lowest)
	p.require.LessOrEqual(number, highest)

	return c
}

// activeBucket returns the values of the currently active tracking bucket.
func (c *captureParts) activeBucket() []string {
	p := c.parts
	p.require.NotEmpty(p.activeTrack)

	values, ok := p.trackedValues[p.activeTrack]
	p.require.True(ok, "no tracking bucket named %q", p.activeTrack)

	return values
}

// track appends a captured value to the active tracking bucket.
func (c *captureParts) track(value string) {
	p := c.parts
	p.trackedValues[p.activeTrack] = append(p.trackedValues[p.activeTrack], value)
}

func (c *captureParts) capturedJSONLeaf(index int, path []string) any {
	p := c.parts
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

func (c *captureParts) capturedRequestURLIs(index int, rawURL string) *captureParts {
	p := c.parts
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Equal(rawURL, p.capturedRequests[index].URL.String())

	return c
}

func (c *captureParts) capturedRequestPathIs(index int, path string) *captureParts {
	p := c.parts
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Equal(path, p.capturedRequests[index].URL.Path)

	return c
}

// serverReceivedHostIs asserts p.capturedRequests[index].Host equals host.
func (c *captureParts) serverReceivedHostIs(index int, host string) *captureParts {
	p := c.parts
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Equal(host, p.capturedRequests[index].Host)

	return c
}

// capturedJSONStringMapIs decodes the captured body at index as a flat string
// map and asserts each key/value equals.
func (c *captureParts) capturedJSONStringMapIs(index int, expected map[string]string) *captureParts {
	p := c.parts
	p.require.Greater(len(p.capturedBodies), index)

	var data map[string]string
	p.require.NoError(json.Unmarshal([]byte(p.capturedBodies[index]), &data))

	for key, want := range expected {
		p.assert.Equal(want, data[key], "key %q mismatch", key)
	}

	return c
}

func (c *captureParts) requestPathMatchesCapturedPath() *captureParts {
	p := c.parts
	current := p.current()
	p.require.NotNil(current.Request)
	p.require.Greater(len(p.capturedRequests), 0)
	p.assert.Equal(p.capturedRequests[0].URL.Path, current.Request.URL.Path)

	return c
}

func (c *captureParts) requestHeadersMatchCaptured(keys ...string) *captureParts {
	p := c.parts
	current := p.current()
	p.require.NotNil(current.Request)
	p.require.Greater(len(p.capturedRequests), 0)

	for _, key := range keys {
		p.assert.Equal(p.capturedRequests[0].Header.Get(key), current.Request.Headers.Get(key), "header %q", key)
	}

	return c
}

// requestRawBodyMatchesCapturedBody asserts the request RawBody equals the first captured body.
func (c *captureParts) requestRawBodyMatchesCapturedBody() *captureParts {
	p := c.parts
	current := p.current()
	p.require.NotNil(current.Request)
	p.require.Greater(len(p.capturedBodies), 0)
	p.assert.Equal(p.capturedBodies[0], current.Request.RawBody)

	return c
}

func (c *captureParts) allTrackedValuesAreValidRFC3339Timestamps() *captureParts {
	p := c.parts
	for _, value := range p.activeBucket() {
		_, err := time.Parse(time.RFC3339Nano, value)
		p.require.NoError(err, "value %q should be a valid RFC3339Nano timestamp", value)
	}

	return c
}

// allTrackedDatetimeValuesAreWithin parses tracked values ("timestamp" = Unix seconds) and checks recency.
func (c *captureParts) allTrackedDatetimeValuesAreWithin(threshold time.Duration, layout string) *captureParts {
	p := c.parts
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

	return c
}

// allTrackedDatetimeValuesHaveUTCZone asserts tracked RFC1123/RFC3339/Unix values resolve to UTC.
func (c *captureParts) allTrackedDatetimeValuesHaveUTCZone() *captureParts {
	p := c.parts
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

	return c
}

// allTrackedDatetimeValuesHaveLocalZone asserts tracked datetime values resolve to the local zone.
func (c *captureParts) allTrackedDatetimeValuesHaveLocalZone() *captureParts {
	p := c.parts
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

	return c
}

func (c *captureParts) capturedJSONFieldIs(index int, want string, path ...string) *captureParts {
	p := c.parts
	leaf := p.capturedJSONLeaf(index, path)
	switch v := leaf.(type) {
	case string:
		p.assert.Equal(want, v)
	case float64:
		p.assert.Equal(want, strconv.FormatInt(int64(v), 10))
	default:
		p.assert.Equal(want, fmt.Sprint(v))
	}

	return c
}

func (c *captureParts) capturedJSONFieldContains(index int, fragment string, path ...string) *captureParts {
	p := c.parts
	leaf, ok := p.capturedJSONLeaf(index, path).(string)
	p.require.True(ok, "field %v is not a string", path)
	p.assert.Contains(leaf, fragment)

	return c
}

func (c *captureParts) capturedJSONFieldMatchesRegexp(index int, pattern string, path ...string) *captureParts {
	p := c.parts
	leaf, ok := p.capturedJSONLeaf(index, path).(string)
	p.require.True(ok, "field %v is not a string", path)
	p.assert.Regexp(pattern, leaf)

	return c
}

func (c *captureParts) capturedJSONFieldNotContains(index int, fragment string, path ...string) *captureParts {
	p := c.parts
	leaf, ok := p.capturedJSONLeaf(index, path).(string)
	p.require.True(ok, "field %v is not a string", path)
	p.assert.NotContains(leaf, fragment)

	return c
}

func (c *captureParts) capturedRequestPathMatchesRegexp(index int, pattern string) *captureParts {
	p := c.parts
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Regexp(pattern, p.capturedRequests[index].URL.Path)

	return c
}

// gapBetweenCapturedRequestTimesAtLeast asserts captured-request times[ j] - [i] >= minGap.
func (c *captureParts) gapBetweenCapturedRequestTimesAtLeast(i, j int, minGap time.Duration) *captureParts {
	p := c.parts
	p.require.Greater(len(p.capturedRequestTimes), i, "no captured-request time at index %d", i)
	p.require.Greater(len(p.capturedRequestTimes), j, "no captured-request time at index %d", j)
	p.assert.GreaterOrEqual(p.capturedRequestTimes[j].Sub(p.capturedRequestTimes[i]), minGap,
		"gap between captured-request times %d→%d was below %s", i, j, minGap)

	return c
}

// firstTrackedValueIsFloatInRange asserts the first value of the active bucket parses as
// a float within [lowest, highest].
func (c *captureParts) firstTrackedValueIsFloatInRange(lowest, highest float64) *captureParts {
	p := c.parts
	values := p.activeBucket()
	p.require.NotEmpty(values)

	number, err := strconv.ParseFloat(values[0], 64)
	p.require.NoError(err, "value %q should be a float", values[0])
	p.assert.GreaterOrEqual(number, lowest)
	p.assert.LessOrEqual(number, highest)

	return c
}

func (c *captureParts) allTrackedValuesMatchRegexp(pattern string) *captureParts {
	p := c.parts
	re := regexp.MustCompile(pattern)
	for _, value := range p.activeBucket() {
		p.assert.True(re.MatchString(value), "value %q should match %q", value, pattern)
	}

	return c
}

func (c *captureParts) capturedBodyContains(index int, fragment string) *captureParts {
	p := c.parts
	p.require.Greater(len(p.capturedBodies), index)
	p.assert.Contains(p.capturedBodies[index], fragment)

	return c
}

func (c *captureParts) capturedBodyMatchesRegexp(index int, pattern string) *captureParts {
	p := c.parts
	p.require.Greater(len(p.capturedBodies), index)
	p.assert.Regexp(pattern, p.capturedBodies[index])

	return c
}

func (c *captureParts) serverReceivedBodyLengthIs(index int, want int) *captureParts {
	p := c.parts
	p.require.Greater(len(p.capturedRequests), index)
	p.require.Equal(want, len(p.capturedBodies[index]))

	return c
}

// serverReceivedBodyMatches asserts the captured request body against a regexp.
func (c *captureParts) serverReceivedBodyMatches(index int, pattern string) *captureParts {
	p := c.parts
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Regexp(pattern, p.capturedBodies[index])

	return c
}

// trackedValueIs asserts the most recent tracked value equals want.
func (c *captureParts) trackedValueIs(want string) *captureParts {
	p := c.parts
	values, ok := p.trackedValues[p.activeTrack]
	p.require.True(ok, "no tracking bucket %q active", p.activeTrack)
	p.require.NotEmpty(values)
	p.assert.Equal(want, values[len(values)-1])

	return c
}

// clientRequestInterceptorCaptures asserts the request captured by aMockTransportClient
// has the given method and full URL.
func (c *captureParts) clientRequestInterceptorCaptures(method, wantURL string) *captureParts {
	p := c.parts
	p.require.NotNil(p.intercepted)
	p.assert.Equal(method, p.intercepted.Method)
	p.assert.Equal(wantURL, p.intercepted.URL.String())

	return c
}

// clientSentNoHeaders asserts the request captured by aMockTransportClient sent no headers.
func (c *captureParts) clientSentNoHeaders() *captureParts {
	p := c.parts
	p.require.NotNil(p.intercepted)
	p.assert.Empty(p.intercepted.Header)

	return c
}

// serverReceivedMethodAndPath asserts the request at the given server-hit index used the
// expected HTTP method and path.
func (c *captureParts) serverReceivedMethodAndPath(index int, method, path string) *captureParts {
	p := c.parts
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Equal(method, p.capturedRequests[index].Method)
	p.assert.Equal(path, p.capturedRequests[index].URL.Path)

	return c
}

// serverReceivedJSONBody asserts the request body at the given server-hit index equals the
// expected JSON regardless of key order or whitespace.
func (c *captureParts) serverReceivedJSONBody(index int, expectedJSON string) *captureParts {
	p := c.parts
	p.require.Greater(len(p.capturedRequests), index)

	body, err := io.ReadAll(p.capturedRequests[index].Body)
	p.require.NoError(err)
	p.assert.JSONEq(expectedJSON, string(body))

	return c
}

// serverReceivedHeaderValue asserts a request header value at the given server-hit index.
func (c *captureParts) serverReceivedHeaderValue(index int, key, value string) *captureParts {
	p := c.parts
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Equal(value, p.capturedRequests[index].Header.Get(key))

	return c
}

// requestRawURLContains asserts the current request's RawURLString contains every fragment.
func (c *captureParts) requestRawURLContains(fragments ...string) *captureParts {
	p := c.parts
	current := p.current()
	p.require.NotNil(current.Request)

	for _, fragment := range fragments {
		p.assert.Contains(current.Request.RawURLString, fragment)
	}

	return c
}

// capturedRequestCount asserts how many requests the test server received.
func (c *captureParts) capturedRequestCount(n int) *captureParts {
	p := c.parts
	p.require.Equal(n, len(p.capturedRequests))

	return c
}

// requestSkipped asserts the response at index i is marked as skipped.
func (c *captureParts) requestSkipped(i int) *captureParts {
	p := c.parts
	p.require.Greater(len(p.responses), i)
	p.assert.True(p.responses[i].Skipped)

	return c
}

// requestExecuted asserts the response at index i is not marked as skipped.
func (c *captureParts) requestExecuted(i int) *captureParts {
	p := c.parts
	p.require.Greater(len(p.responses), i)
	p.assert.False(p.responses[i].Skipped)

	return c
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
