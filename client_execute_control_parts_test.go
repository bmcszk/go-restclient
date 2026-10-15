package restclient_test

import (
	"time"
)

// parsedRequestDisabled asserts the parsed request at index i has the disabled flag set to want.
func (c *parseState) parsedRequestDisabled(i int, want bool) *parts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].Disabled)

	return p
}

// parsedRequestSleep asserts the parsed request at index i has SleepDuration set to want.
func (c *parseState) parsedRequestSleep(i int, want time.Duration) *parts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].SleepDuration)

	return p
}

// parsedRequestDisabledExpr asserts the conditional @disabled expression (after `!`) at index i equals want.
func (c *parseState) parsedRequestDisabledExpr(i int, want string) *parts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].DisabledExpr)

	return p
}

// parsedRequestLoopCount asserts the literal `@loop for N` count on the parsed request at index i.
func (c *parseState) parsedRequestLoopCount(i int, want int) *parts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].LoopFor)

	return p
}

// parsedRequestLoopExpr asserts the raw expression on `@loop for {{expr}}` of the parsed request at index i.
func (c *parseState) parsedRequestLoopExpr(i int, want string) *parts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].LoopExpr)

	return p
}

// parsedRequestLoopCollection asserts the collection name on `@loop for item of collection` at index i.
func (c *parseState) parsedRequestLoopCollection(i int, want string) *parts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].LoopCollection)

	return p
}

// requestSkipped asserts the response at index i is marked as skipped.
func (c *captureState) requestSkipped(i int) *parts {
	p := c.parts
	p.require.Greater(len(p.responses), i)
	p.assert.True(p.responses[i].Skipped)

	return p
}

// requestExecuted asserts the response at index i is not marked as skipped.
func (c *captureState) requestExecuted(i int) *parts {
	p := c.parts
	p.require.Greater(len(p.responses), i)
	p.assert.False(p.responses[i].Skipped)

	return p
}

// gapBetweenCapturedRequestTimesAtLeast asserts captured-request times[ j] - [i] >= minGap.
func (c *captureState) gapBetweenCapturedRequestTimesAtLeast(i, j int, minGap time.Duration) *parts {
	p := c.parts
	p.require.Greater(len(p.capturedRequestTimes), i, "no captured-request time at index %d", i)
	p.require.Greater(len(p.capturedRequestTimes), j, "no captured-request time at index %d", j)
	p.assert.GreaterOrEqual(p.capturedRequestTimes[j].Sub(p.capturedRequestTimes[i]), minGap,
		"gap between captured-request times %d→%d was below %s", i, j, minGap)

	return p
}
