package restclient_test

import (
	"time"
)

// parsedRequestDisabled asserts the parsed request at index i has the disabled flag set to want.
func (c *parseParts) parsedRequestDisabled(i int, want bool) *parts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].Disabled)

	return p
}

// parsedRequestSleep asserts the parsed request at index i has SleepDuration set to want.
func (c *parseParts) parsedRequestSleep(i int, want time.Duration) *parts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].SleepDuration)

	return p
}

// parsedRequestDisabledExpr asserts the conditional @disabled expression (after `!`) at index i equals want.
func (c *parseParts) parsedRequestDisabledExpr(i int, want string) *parts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].DisabledExpr)

	return p
}

// parsedRequestLoopCount asserts the literal `@loop for N` count on the parsed request at index i.
func (c *parseParts) parsedRequestLoopCount(i int, want int) *parts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].LoopFor)

	return p
}

// parsedRequestLoopExpr asserts the raw expression on `@loop for {{expr}}` of the parsed request at index i.
func (c *parseParts) parsedRequestLoopExpr(i int, want string) *parts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].LoopExpr)

	return p
}

// parsedRequestLoopCollection asserts the collection name on `@loop for item of collection` at index i.
func (c *parseParts) parsedRequestLoopCollection(i int, want string) *parts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].LoopCollection)

	return p
}
