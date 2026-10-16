package restclient_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newParseParts(t *testing.T) (given, when, then *parts) {
	t.Helper()

	fx := &parts{
		T:       t,
		require: require.New(t),
		assert:  assert.New(t),
		baseDir: t.TempDir(),
	}

	return fx, fx, fx
}

// parsingFile parses the .http file without executing any request.
func (p *parts) parsingFile() *parts {
	p.require.NotNil(p.client)
	p.parsedFile, p.parseErr = p.client.ParseFile(p.httpFilePath)

	return p
}

func (p *parts) parseSucceeded() *parts {
	p.require.NoError(p.parseErr)

	return p
}

func (p *parts) parseErrorContains(texts ...string) *parts {
	p.require.Error(p.parseErr)

	for _, text := range texts {
		p.require.Contains(p.parseErr.Error(), text)
	}

	return p
}

func (p *parts) parsedRequestCount(n int) *parts {
	p.require.NotNil(p.parsedFile)
	p.require.Len(p.parsedFile.Requests, n)

	return p
}

func (p *parts) parsedRequestBodyIs(i int, want string) *parts {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].RawBody)

	return p
}

func (p *parts) parsedRequestName(i int, name string) *parts {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(name, p.parsedFile.Requests[i].Name)

	return p
}

func (p *parts) parsedRequestMethod(i int, method string) *parts {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(method, p.parsedFile.Requests[i].Method)

	return p
}

func (p *parts) parsedRequestRawURLIs(i int, rawURL string) *parts {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(rawURL, p.parsedFile.Requests[i].RawURLString)

	return p
}

func (p *parts) parsedRequestDisabled(i int, want bool) *parts {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].Disabled)

	return p
}

func (p *parts) parsedRequestSleep(i int, want time.Duration) *parts {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].SleepDuration)

	return p
}

func (p *parts) parsedRequestDisabledExpr(i int, want string) *parts {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].DisabledExpr)

	return p
}

func (p *parts) parsedRequestLoopCount(i int, want int) *parts {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].LoopFor)

	return p
}

func (p *parts) parsedRequestLoopExpr(i int, want string) *parts {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].LoopExpr)

	return p
}

func (p *parts) parsedRequestLoopCollection(i int, want string) *parts {
	p.require.Greater(len(p.parsedFile.Requests), i)
	p.assert.Equal(want, p.parsedFile.Requests[i].LoopCollection)

	return p
}

func (p *parts) parsedRequestRefs(i int, wantName string, wantForce bool) *parts {
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
func (p *parts) executingRequestAt(index int) *parts {
	p.require.NotNil(p.client)

	parsed, err := p.client.ParseFile(p.httpFilePath)
	p.require.NoError(err)

	resp, execErr := p.client.ExecuteRequest(context.Background(), parsed, index)
	p.responses = append(p.responses, resp)
	p.execErr = execErr

	return p
}
