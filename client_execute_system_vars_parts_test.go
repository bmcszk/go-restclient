package restclient_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func (c *captureState) allResponseCodesAre(code int) *parts {
	p := c.parts
	p.require.NotEmpty(p.responses)

	for i, resp := range p.responses {
		p.require.Equal(code, resp.StatusCode, "response %d", i)
	}

	return p
}

func (c *captureState) capturedJSONNumberField(index int, path ...string) *parts {
	p := c.parts
	value, ok := p.capturedJSONLeaf(index, path).(float64)
	p.require.True(ok, "field %v is not a number", path)
	p.track(strconv.FormatInt(int64(value), 10))

	return p
}

func (c *captureState) allTrackedValuesAreValidRFC3339Timestamps() *parts {
	p := c.parts
	for _, value := range p.activeBucket() {
		_, err := time.Parse(time.RFC3339Nano, value)
		p.require.NoError(err, "value %q should be a valid RFC3339Nano timestamp", value)
	}

	return p
}

// allTrackedDatetimeValuesAreWithin parses tracked values ("timestamp" = Unix seconds) and checks recency.
func (c *captureState) allTrackedDatetimeValuesAreWithin(threshold time.Duration, layout string) *parts {
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

	return p
}

// allTrackedDatetimeValuesHaveUTCZone asserts tracked RFC1123/RFC3339/Unix values resolve to UTC.
func (c *captureState) allTrackedDatetimeValuesHaveUTCZone() *parts {
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

	return p
}

// allTrackedDatetimeValuesHaveLocalZone asserts tracked datetime values resolve to the local zone.
func (c *captureState) allTrackedDatetimeValuesHaveLocalZone() *parts {
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

	return p
}

func (c *captureState) capturedJSONFieldMatchesRegexp(index int, pattern string, path ...string) *parts {
	p := c.parts
	leaf, ok := p.capturedJSONLeaf(index, path).(string)
	p.require.True(ok, "field %v is not a string", path)
	p.assert.Regexp(pattern, leaf)

	return p
}

func (c *captureState) capturedJSONFieldNotContains(index int, fragment string, path ...string) *parts {
	p := c.parts
	leaf, ok := p.capturedJSONLeaf(index, path).(string)
	p.require.True(ok, "field %v is not a string", path)
	p.assert.NotContains(leaf, fragment)

	return p
}

// serverReceivedBodyLengthIs asserts the captured body at index has exactly want bytes.
func (c *captureState) serverReceivedBodyLengthIs(index int, want int) *parts {
	p := c.parts
	p.require.Greater(len(p.capturedRequests), index)
	p.require.Equal(want, len(p.capturedBodies[index]))

	return p
}

// serverReceivedBodyMatches asserts the captured request body against a regexp.
func (c *captureState) serverReceivedBodyMatches(index int, pattern string) *parts {
	p := c.parts
	p.require.Greater(len(p.capturedRequests), index)
	p.assert.Regexp(pattern, p.capturedBodies[index])

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

// expectedResponseFile writes the given .hresp content as the expected responses file.
func (p *parts) expectedResponseFile(content string) *parts {
	path := filepath.Join(p.baseDir, "expected.hresp")
	p.require.NoError(os.WriteFile(path, []byte(content), 0644))
	p.expectedFilePath = path

	return p
}
