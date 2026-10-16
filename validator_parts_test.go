package restclient_test

import (
	"bufio"
	"bytes"
	rc "github.com/bmcszk/go-restclient"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func (p *parts) aResponseWithStatus(statusCode int, statusText string) *parts {
	return p.aResponseWith(statusCode, statusText, "", nil)
}

// aResponseFromRawHTTPFile parses a raw HTTP response file into a rc.Response.
func (p *parts) aResponseFromRawHTTPFile(path string) *parts {
	content, err := os.ReadFile(path)
	p.require.NoError(err, "read raw http file %s", path)

	reader := bufio.NewReader(bytes.NewReader(content))
	httpResp, err := http.ReadResponse(reader, nil)
	p.require.NoError(err, "parse raw http response %s", path)
	defer func() { _ = httpResp.Body.Close() }()

	bodyBytes, err := io.ReadAll(httpResp.Body)
	p.require.NoError(err, "read body from %s", path)

	p.responses = append(p.responses, &rc.Response{
		StatusCode: httpResp.StatusCode,
		Status:     httpResp.Status,
		Proto:      httpResp.Proto,
		Headers:    httpResp.Header.Clone(),
		Body:       bodyBytes,
		BodyString: string(bodyBytes),
	})
	p.builtResponses = append(p.builtResponses, p.responses[len(p.responses)-1])

	return p
}

func (p *parts) withStatusCode(code int) *parts {
	p.require.NotEmpty(p.responses)
	p.responses[len(p.responses)-1].StatusCode = code

	return p
}

func (p *parts) withStatusText(s string) *parts {
	p.require.NotEmpty(p.responses)
	p.responses[len(p.responses)-1].Status = s

	return p
}

func (p *parts) withHeader(key, value string) *parts {
	p.require.NotEmpty(p.responses)
	resp := p.responses[len(p.responses)-1]
	if resp.Headers == nil {
		resp.Headers = http.Header{}
	}
	resp.Headers.Set(key, value)

	return p
}

func (p *parts) withoutHeader(key string) *parts {
	p.require.NotEmpty(p.responses)
	resp := p.responses[len(p.responses)-1]
	if resp.Headers != nil {
		resp.Headers.Del(key)
	}

	return p
}

func (p *parts) withBody(body string) *parts {
	p.require.NotEmpty(p.responses)
	resp := p.responses[len(p.responses)-1]
	resp.BodyString = body
	resp.Body = []byte(body)

	return p
}

// noActualResponses leaves p.responses nil (a nil slice, distinct from an empty slice).
func (p *parts) noActualResponses() *parts {
	p.responses = nil

	return p
}

// anEmptyResponseSlice sets p.responses to an empty (non-nil) slice.
func (p *parts) anEmptyResponseSlice() *parts {
	p.responses = []*rc.Response{}

	return p
}

// aNilResponse sets p.responses to []*rc.Response{nil}.
func (p *parts) aNilResponse() *parts {
	p.responses = []*rc.Response{nil}

	return p
}

// anExpectedResponseFileAt sets p.expectedFilePath verbatim. Used to point at a
// committed fixture (validateResponses always prepends the committed-fixture
// directory).
func (p *parts) anExpectedResponseFileAt(path string) *parts {
	p.expectedFilePath = path

	return p
}

// validateResponsesWithIndex calls ValidateResponsesWithOptions with one expected index.
func (p *parts) validateResponsesWithIndex(index int) *parts {
	p.validationErr = p.client.ValidateResponsesWithOptions(
		p.expectedFilePath,
		rc.ValidateOptions{ExpectedIndex: index},
		p.responses...,
	)
	return p
}

// responsesValidateAgainst validates responses against an explicit .hresp path.
func (p *parts) responsesValidateAgainst(expectedPath string) *parts {
	p.require.NotNil(p.client)
	p.assert.NoError(p.client.ValidateResponses(expectedPath, p.responses...))

	return p
}

func (p *parts) responseHeaderValues(key string, values ...string) *parts {
	p.require.Equal(values, p.current().Headers[key])

	return p
}

func (p *parts) responseHeaderEmpty(key string) *parts {
	p.require.Empty(p.current().Headers.Get(key))

	return p
}

func (p *parts) responseIsTLS() *parts {
	p.require.True(p.current().IsTLS, "expected TLS response, got plain")

	return p
}

func (p *parts) responseTLSVersionMatches(pattern string) *parts {
	p.assert.Regexp(pattern, p.current().TLSVersion)

	return p
}

// responsesValidateAgainstFixture validates responses against a committed response-files fixture.
func (p *parts) responsesValidateAgainstFixture(name string) *parts {
	path := filepath.Join(responseFilesDir, name)
	p.assert.NoError(p.client.ValidateResponses(path, p.responses...))

	return p
}
