package restclient_test

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

type captureParts struct {
	parts
}

func newCaptureParts(t *testing.T) (given, when, then *captureParts) {
	t.Helper()

	fx := &captureParts{parts: parts{
		T:             t,
		require:       require.New(t),
		assert:        assert.New(t),
		baseDir:       t.TempDir(),
		trackedValues: make(map[string][]string),
	}}

	return fx, fx, fx
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

func (p *parts) serverReceivedBodyIs(index int, body string) *parts {
	p.require.Greater(len(p.capturedBodies), index)
	p.assert.Equal(body, p.capturedBodies[index])

	return p
}

func (p *parts) serverReceivedBodyParsesAsJSON(index int) *parts {
	p.require.Greater(len(p.capturedBodies), index)

	var data map[string]any
	p.require.NoError(json.Unmarshal([]byte(p.capturedBodies[index]), &data))

	return p
}

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

func (p *parts) firstTrackedValueIsIntegerInRange(lowest, highest int64) *parts {
	values := p.activeBucket()
	p.require.NotEmpty(values)

	number, err := strconv.ParseInt(values[0], 10, 64)
	p.require.NoError(err, "value %q should be an integer", values[0])
	p.require.GreaterOrEqual(number, lowest)
	p.require.LessOrEqual(number, highest)

	return p
}

func (p *parts) activeBucket() []string {
	p.require.NotEmpty(p.activeTrack)

	values, ok := p.trackedValues[p.activeTrack]
	p.require.True(ok, "no tracking bucket named %q", p.activeTrack)

	return values
}

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

func (p *parts) serverReceivedHostIs(index int, host string) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Equal(host, p.capturedRequests[index].Host)

	return p
}

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

func (p *parts) allTrackedDatetimeValuesHaveUTCZone() *parts {
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

func (p *parts) capturedRequestPathMatchesRegexp(index int, pattern string) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Regexp(pattern, p.capturedRequests[index].URL.Path)

	return p
}

func (p *parts) gapBetweenCapturedRequestTimesAtLeast(i, j int, minGap time.Duration) *parts {
	p.require.Greater(len(p.capturedRequestTimes), i, "no captured-request time at index %d", i)
	p.require.Greater(len(p.capturedRequestTimes), j, "no captured-request time at index %d", j)
	p.assert.GreaterOrEqual(p.capturedRequestTimes[j].Sub(p.capturedRequestTimes[i]), minGap,
		"gap between captured-request times %d→%d was below %s", i, j, minGap)

	return p
}

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

func (p *parts) capturedBodyMatchesRegexp(index int, pattern string) *parts {
	p.require.Greater(len(p.capturedBodies), index)
	p.assert.Regexp(pattern, p.capturedBodies[index])

	return p
}

func (p *parts) serverReceivedBodyLengthIs(index int, want int) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.require.Equal(want, len(p.capturedBodies[index]))

	return p
}

func (p *parts) serverReceivedBodyMatches(index int, pattern string) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Regexp(pattern, p.capturedBodies[index])

	return p
}

func (p *parts) trackedValueIs(want string) *parts {
	values, ok := p.trackedValues[p.activeTrack]
	p.require.True(ok, "no tracking bucket %q active", p.activeTrack)
	p.require.NotEmpty(values)
	p.assert.Equal(want, values[len(values)-1])

	return p
}

func (p *parts) clientRequestInterceptorCaptures(method, wantURL string) *parts {
	p.require.NotNil(p.intercepted)
	p.assert.Equal(method, p.intercepted.Method)
	p.assert.Equal(wantURL, p.intercepted.URL.String())

	return p
}

func (p *parts) clientSentNoHeaders() *parts {
	p.require.NotNil(p.intercepted)
	p.assert.Empty(p.intercepted.Header)

	return p
}

func (p *parts) serverReceivedMethodAndPath(index int, method, path string) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Equal(method, p.capturedRequests[index].Method)
	p.assert.Equal(path, p.capturedRequests[index].URL.Path)

	return p
}

func (p *parts) serverReceivedJSONBody(index int, expectedJSON string) *parts {
	p.require.Greater(len(p.capturedRequests), index)

	body, err := io.ReadAll(p.capturedRequests[index].Body)
	p.require.NoError(err)
	p.assert.JSONEq(expectedJSON, string(body))

	return p
}

func (p *parts) serverReceivedHeaderValue(index int, key, value string) *parts {
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Equal(value, p.capturedRequests[index].Header.Get(key))

	return p
}

func (p *parts) requestRawURLContains(fragments ...string) *parts {
	current := p.current()
	p.require.NotNil(current.Request)

	for _, fragment := range fragments {
		p.assert.Contains(current.Request.RawURLString, fragment)
	}

	return p
}

func (p *parts) capturedRequestCount(n int) *parts {
	p.require.Equal(n, len(p.capturedRequests))

	return p
}

func (p *parts) requestSkipped(i int) *parts {
	p.require.Greater(len(p.responses), i)
	p.assert.True(p.responses[i].Skipped)

	return p
}

func (p *parts) requestExecuted(i int) *parts {
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
