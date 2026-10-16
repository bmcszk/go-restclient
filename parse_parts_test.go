package restclient_test

import (
	"context"
	"testing"
	"time"

	rc "github.com/bmcszk/go-restclient"
)

// parseParts holds the parse-domain state of the fluent test DSL: the result of
// parsing the .http file under test plus the parse error asserted by the
// parsed* builders.
type parseParts struct {
	parsedFile *rc.ParsedFile
	parseErr   error
}

// newParseParts returns the given/when/then fixture for the parse-domain fluent DSL.
func newParseParts(t *testing.T) (given, when, then *testFixture) {
	t.Helper()

	return newParts(t)
}

// parsingFile parses the .http file without executing any request.
func (p *testFixture) parsingFile() *testFixture {
	p.require.NotNil(p.client)
	p.parsedFile, p.parseErr = p.client.ParseFile(p.httpFilePath)

	return p
}

func (p *testFixture) parseSucceeded() *testFixture {
	p.require.NoError(p.parseErr)

	return p
}

// parseErrorContains asserts parsing failed and the error mentions every given text.
// Called without texts it only asserts the error itself.
func (p *testFixture) parseErrorContains(texts ...string) *testFixture {
	p.require.Error(p.parseErr)

	for _, text := range texts {
		p.require.Contains(p.parseErr.Error(), text)
	}

	return p
}

func (p *testFixture) parsedRequestCount(n int) *testFixture {
	p.require.NotNil(p.parsedFile)
	p.require.Len(p.parsedFile.Requests, n)

	return p
}

func (p *testFixture) parsedRequestBodyIs(i int, want string) *testFixture {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].RawBody)

	return p
}

func (p *testFixture) parsedRequestName(i int, name string) *testFixture {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(name, p.parsedFile.Requests[i].Name)

	return p
}

func (p *testFixture) parsedRequestMethod(i int, method string) *testFixture {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(method, p.parsedFile.Requests[i].Method)

	return p
}

// parsedRequestRawURLIs asserts the raw (pre-substitution) URL of the parsed request
// at index i.
func (p *testFixture) parsedRequestRawURLIs(i int, rawURL string) *testFixture {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(rawURL, p.parsedFile.Requests[i].RawURLString)

	return p
}

func (p *testFixture) parsedRequestDisabled(i int, want bool) *testFixture {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].Disabled)

	return p
}

func (p *testFixture) parsedRequestSleep(i int, want time.Duration) *testFixture {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].SleepDuration)

	return p
}

// parsedRequestDisabledExpr asserts the conditional @disabled expression (after `!`) at index i equals want.
func (p *testFixture) parsedRequestDisabledExpr(i int, want string) *testFixture {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].DisabledExpr)

	return p
}

// parsedRequestLoopCount asserts the literal `@loop for N` count on the parsed request at index i.
func (p *testFixture) parsedRequestLoopCount(i int, want int) *testFixture {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].LoopFor)

	return p
}

// parsedRequestLoopExpr asserts the raw expression on `@loop for {{expr}}` of the parsed request at index i.
func (p *testFixture) parsedRequestLoopExpr(i int, want string) *testFixture {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].LoopExpr)

	return p
}

// parsedRequestLoopCollection asserts the collection name on `@loop for item of collection` at index i.
func (p *testFixture) parsedRequestLoopCollection(i int, want string) *testFixture {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].LoopCollection)

	return p
}

// parsedRequestRefs asserts the parsed request at index i declares a @ref/@forceRef entry.
func (p *testFixture) parsedRequestRefs(i int, wantName string, wantForce bool) *testFixture {
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

// executingRequestAt parses the .http file and executes only the request at the given index.
func (p *testFixture) executingRequestAt(index int) *testFixture {
	p.require.NotNil(p.client)

	parsed, err := p.client.ParseFile(p.httpFilePath)
	p.require.NoError(err)

	resp, execErr := p.client.ExecuteRequest(context.Background(), parsed, index)
	p.responses = append(p.responses, resp)
	p.execErr = execErr

	return p
}
