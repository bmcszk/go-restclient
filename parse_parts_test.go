package restclient_test

import (
	"context"
	"time"

	rc "github.com/bmcszk/go-restclient"
)

// parseState holds the parse-domain state of the fluent test DSL: the result of
// parsing the .http file under test plus the parse error asserted by the
// parsed* builders.
type parseState struct {
	// root points back at the enclosing parts so the parse builders can return
	// the fluent chain; parts reads it by field promotion as p.root as well.
	root *parts
	// parsedFile holds the result of parsingFile for parsed-file assertions.
	parsedFile *rc.ParsedFile
	// parseErr holds the error returned by parsingFile.
	parseErr error
}

var _ = []any{
	parseState{},
}

// parsingFile parses the .http file without executing any request.
func (c *parseState) parsingFile() *parts {
	p := c.root

	p.require.NotNil(p.client)
	p.parsedFile, p.parseErr = p.client.ParseFile(p.httpFilePath)

	return p
}

// executingRequestAt parses the .http file and executes only the request at the given index.
func (c *parseState) executingRequestAt(index int) *parts {
	p := c.root

	p.require.NotNil(p.client)

	parsed, err := p.client.ParseFile(p.httpFilePath)
	p.require.NoError(err)

	resp, execErr := p.client.ExecuteRequest(context.Background(), parsed, index)
	p.responses = append(p.responses, resp)
	p.execErr = execErr

	return p
}

// parseSucceeded asserts the file was parsed without error.
func (c *parseState) parseSucceeded() *parts {
	p := c.root

	p.require.NoError(p.parseErr)

	return p
}

// parseErrorContains asserts parsing failed and the error mentions every given text.
// Called without texts it only asserts the error itself.
func (c *parseState) parseErrorContains(texts ...string) *parts {
	p := c.root

	p.require.Error(p.parseErr)

	for _, text := range texts {
		p.require.Contains(p.parseErr.Error(), text)
	}

	return p
}

// parsedRequestCount asserts how many requests the parse produced.
func (c *parseState) parsedRequestCount(n int) *parts {
	p := c.root

	p.require.NotNil(p.parsedFile)
	p.require.Len(p.parsedFile.Requests, n)

	return p
}

// parsedRequestMethod asserts the HTTP method of the parsed request at index i.
func (c *parseState) parsedRequestMethod(i int, method string) *parts {
	p := c.root

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(method, p.parsedFile.Requests[i].Method)

	return p
}

// parsedRequestBodyIs asserts the raw body of the parsed request at index i.
func (c *parseState) parsedRequestBodyIs(i int, want string) *parts {
	p := c.root

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].RawBody)

	return p
}

// parsedRequestName asserts the name of the parsed request at index i.
func (c *parseState) parsedRequestName(i int, name string) *parts {
	p := c.root

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(name, p.parsedFile.Requests[i].Name)

	return p
}

// parsedRequestRawURLIs asserts the raw (pre-substitution) URL of the parsed request
// at index i.
func (c *parseState) parsedRequestRawURLIs(i int, rawURL string) *parts {
	p := c.root

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(rawURL, p.parsedFile.Requests[i].RawURLString)

	return p
}

// parsedRequestDisabled asserts the parsed request at index i has the disabled flag set to want.
func (c *parseState) parsedRequestDisabled(i int, want bool) *parts {
	p := c.root

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].Disabled)

	return p
}

// parsedRequestSleep asserts the parsed request at index i has SleepDuration set to want.
func (c *parseState) parsedRequestSleep(i int, want time.Duration) *parts {
	p := c.root

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].SleepDuration)

	return p
}

// parsedRequestDisabledExpr asserts the conditional @disabled expression (after `!`) at index i equals want.
func (c *parseState) parsedRequestDisabledExpr(i int, want string) *parts {
	p := c.root

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].DisabledExpr)

	return p
}

// parsedRequestLoopCount asserts the literal `@loop for N` count on the parsed request at index i.
func (c *parseState) parsedRequestLoopCount(i int, want int) *parts {
	p := c.root

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].LoopFor)

	return p
}

// parsedRequestLoopExpr asserts the raw expression on `@loop for {{expr}}` of the parsed request at index i.
func (c *parseState) parsedRequestLoopExpr(i int, want string) *parts {
	p := c.root

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].LoopExpr)

	return p
}

// parsedRequestLoopCollection asserts the collection name on `@loop for item of collection` at index i.
func (c *parseState) parsedRequestLoopCollection(i int, want string) *parts {
	p := c.root

	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].LoopCollection)

	return p
}

// parsedRequestRefs asserts the parsed request at index i declares a @ref/@forceRef entry.
func (c *parseState) parsedRequestRefs(i int, wantName string, wantForce bool) *parts {
	p := c.root

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

	return p
}
