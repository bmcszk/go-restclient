package restclient_test

import (
	"context"
	"time"

	rc "github.com/bmcszk/go-restclient"
)

// parseParts holds the parse-domain state of the fluent test DSL: the result of
// parsing the .http file under test plus the parse error asserted by the
// parsed* builders.
type parseParts struct {
	*parts
	parsedFile *rc.ParsedFile
	parseErr   error
}

// parsingFile parses the .http file without executing any request.
func (c *parseParts) parsingFile() *parseParts {
	c.require.NotNil(c.client)
	c.parsedFile, c.parseErr = c.client.ParseFile(c.httpFilePath)

	return c
}

func (c *parseParts) parseSucceeded() *parseParts {
	c.require.NoError(c.parseErr)

	return c
}

// parseErrorContains asserts parsing failed and the error mentions every given text.
// Called without texts it only asserts the error itself.
func (c *parseParts) parseErrorContains(texts ...string) *parseParts {
	c.require.Error(c.parseErr)

	for _, text := range texts {
		c.require.Contains(c.parseErr.Error(), text)
	}

	return c
}

func (c *parseParts) parsedRequestCount(n int) *parseParts {
	c.require.NotNil(c.parsedFile)
	c.require.Len(c.parsedFile.Requests, n)

	return c
}

func (c *parseParts) parsedRequestBodyIs(i int, want string) *parseParts {
	c.require.Greater(len(c.parsedFile.Requests), i)
	c.assert.Equal(want, c.parsedFile.Requests[i].RawBody)

	return c
}

func (c *parseParts) parsedRequestName(i int, name string) *parseParts {
	c.require.Greater(len(c.parsedFile.Requests), i)
	c.assert.Equal(name, c.parsedFile.Requests[i].Name)

	return c
}

func (c *parseParts) parsedRequestMethod(i int, method string) *parseParts {
	c.require.Greater(len(c.parsedFile.Requests), i)
	c.assert.Equal(method, c.parsedFile.Requests[i].Method)

	return c
}

// parsedRequestRawURLIs asserts the raw (pre-substitution) URL of the parsed request
// at index i.
func (c *parseParts) parsedRequestRawURLIs(i int, rawURL string) *parseParts {
	c.require.Greater(len(c.parsedFile.Requests), i)
	c.assert.Equal(rawURL, c.parsedFile.Requests[i].RawURLString)

	return c
}

func (c *parseParts) parsedRequestDisabled(i int, want bool) *parseParts {
	c.require.Greater(len(c.parsedFile.Requests), i)
	c.assert.Equal(want, c.parsedFile.Requests[i].Disabled)

	return c
}

func (c *parseParts) parsedRequestSleep(i int, want time.Duration) *parseParts {
	c.require.Greater(len(c.parsedFile.Requests), i)
	c.assert.Equal(want, c.parsedFile.Requests[i].SleepDuration)

	return c
}

// parsedRequestDisabledExpr asserts the conditional @disabled expression (after `!`) at index i equals want.
func (c *parseParts) parsedRequestDisabledExpr(i int, want string) *parseParts {
	c.require.Greater(len(c.parsedFile.Requests), i)
	c.assert.Equal(want, c.parsedFile.Requests[i].DisabledExpr)

	return c
}

// parsedRequestLoopCount asserts the literal `@loop for N` count on the parsed request at index i.
func (c *parseParts) parsedRequestLoopCount(i int, want int) *parseParts {
	c.require.Greater(len(c.parsedFile.Requests), i)
	c.assert.Equal(want, c.parsedFile.Requests[i].LoopFor)

	return c
}

// parsedRequestLoopExpr asserts the raw expression on `@loop for {{expr}}` of the parsed request at index i.
func (c *parseParts) parsedRequestLoopExpr(i int, want string) *parseParts {
	c.require.Greater(len(c.parsedFile.Requests), i)
	c.assert.Equal(want, c.parsedFile.Requests[i].LoopExpr)

	return c
}

// parsedRequestLoopCollection asserts the collection name on `@loop for item of collection` at index i.
func (c *parseParts) parsedRequestLoopCollection(i int, want string) *parseParts {
	c.require.Greater(len(c.parsedFile.Requests), i)
	c.assert.Equal(want, c.parsedFile.Requests[i].LoopCollection)

	return c
}

// parsedRequestRefs asserts the parsed request at index i declares a @ref/@forceRef entry.
func (c *parseParts) parsedRequestRefs(i int, wantName string, wantForce bool) *parseParts {
	c.require.Greater(len(c.parsedFile.Requests), i)
	req := c.parsedFile.Requests[i]
	c.require.NotEmpty(req.Refs, "request %d has no refs", i)

	found := false
	for _, ref := range req.Refs {
		if ref.Name == wantName {
			c.assert.Equal(wantForce, ref.Force,
				"ref %q force flag (expected %v)", wantName, wantForce)
			found = true

			break
		}
	}
	c.require.True(found, "request %d missing ref %q", i, wantName)

	return c
}

// executingRequestAt parses the .http file and executes only the request at the given index.
func (c *parseParts) executingRequestAt(index int) *parseParts {
	c.require.NotNil(c.client)

	parsed, err := c.client.ParseFile(c.httpFilePath)
	c.require.NoError(err)

	resp, execErr := c.client.ExecuteRequest(context.Background(), parsed, index)
	c.responses = append(c.responses, resp)
	c.execErr = execErr

	return c
}
