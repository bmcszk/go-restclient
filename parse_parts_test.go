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

var _ = []any{
	parseParts{},
}

// parsingFile parses the .http file without executing any request.
func (c *parseParts) parsingFile() *parseParts {
	p := c.parts

	p.require.NotNil(p.client)
	p.parsedFile, p.parseErr = p.client.ParseFile(p.httpFilePath)

	return c
}

func (c *parseParts) parseSucceeded() *parseParts {
	p := c.parts

	p.require.NoError(p.parseErr)

	return c
}

// parseErrorContains asserts parsing failed and the error mentions every given text.
// Called without texts it only asserts the error itself.
func (c *parseParts) parseErrorContains(texts ...string) *parseParts {
	p := c.parts

	p.require.Error(p.parseErr)

	for _, text := range texts {
		p.require.Contains(p.parseErr.Error(), text)
	}

	return c
}

func (c *parseParts) parsedRequestCount(n int) *parseParts {
	p := c.parts

	p.require.NotNil(p.parsedFile)
	p.require.Len(p.parsedFile.Requests, n)

	return c
}

func (c *parseParts) parsedRequestBodyIs(i int, want string) *parseParts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].RawBody)

	return c
}

func (c *parseParts) parsedRequestName(i int, name string) *parseParts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(name, p.parsedFile.Requests[i].Name)

	return c
}

func (c *parseParts) parsedRequestMethod(i int, method string) *parseParts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(method, p.parsedFile.Requests[i].Method)

	return c
}

// parsedRequestRawURLIs asserts the raw (pre-substitution) URL of the parsed request
// at index i.
func (c *parseParts) parsedRequestRawURLIs(i int, rawURL string) *parseParts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(rawURL, p.parsedFile.Requests[i].RawURLString)

	return c
}

func (c *parseParts) parsedRequestDisabled(i int, want bool) *parseParts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].Disabled)

	return c
}

func (c *parseParts) parsedRequestSleep(i int, want time.Duration) *parseParts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].SleepDuration)

	return c
}

// parsedRequestDisabledExpr asserts the conditional @disabled expression (after `!`) at index i equals want.
func (c *parseParts) parsedRequestDisabledExpr(i int, want string) *parseParts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].DisabledExpr)

	return c
}

// parsedRequestLoopCount asserts the literal `@loop for N` count on the parsed request at index i.
func (c *parseParts) parsedRequestLoopCount(i int, want int) *parseParts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].LoopFor)

	return c
}

// parsedRequestLoopExpr asserts the raw expression on `@loop for {{expr}}` of the parsed request at index i.
func (c *parseParts) parsedRequestLoopExpr(i int, want string) *parseParts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].LoopExpr)

	return c
}

// parsedRequestLoopCollection asserts the collection name on `@loop for item of collection` at index i.
func (c *parseParts) parsedRequestLoopCollection(i int, want string) *parseParts {
	p := c.parts

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].LoopCollection)

	return c
}

// parsedRequestRefs asserts the parsed request at index i declares a @ref/@forceRef entry.
func (c *parseParts) parsedRequestRefs(i int, wantName string, wantForce bool) *parseParts {
	p := c.parts

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

	return c
}

// executingRequestAt parses the .http file and executes only the request at the given index.
func (c *parseParts) executingRequestAt(index int) *parseParts {
	p := c.parts

	p.require.NotNil(p.client)

	parsed, err := p.client.ParseFile(p.httpFilePath)
	p.require.NoError(err)

	resp, execErr := p.client.ExecuteRequest(context.Background(), parsed, index)
	p.responses = append(p.responses, resp)
	p.execErr = execErr

	return c
}
