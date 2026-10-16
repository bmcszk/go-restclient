package restclient_test

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"os"

	rc "github.com/bmcszk/go-restclient"
)

// Validator DSL: builds rc.Response values directly, never executes HTTP requests.

// aResponseWithStatus is a shorthand for aResponseWith with no body and no headers.
func (p *testFixture) aResponseWithStatus(statusCode int, statusText string) *testFixture {
	return p.aResponseWith(statusCode, statusText, "", nil)
}

// aResponseFromRawHTTPFile parses a raw HTTP response file into a rc.Response.
func (p *testFixture) aResponseFromRawHTTPFile(path string) *testFixture {
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

	return p
}

// withStatusCode rewrites the StatusCode of the LAST response in p.responses.
// Requires len(p.responses) > 0.
func (p *testFixture) withStatusCode(code int) *testFixture {
	p.require.NotEmpty(p.responses)
	p.responses[len(p.responses)-1].StatusCode = code

	return p
}

// withStatusText rewrites the Status string of the LAST response in p.responses.
// Requires len(p.responses) > 0.
func (p *testFixture) withStatusText(s string) *testFixture {
	p.require.NotEmpty(p.responses)
	p.responses[len(p.responses)-1].Status = s

	return p
}

// withHeader Sets the given key/value pair on the Headers of the LAST response
// in p.responses. Requires len(p.responses) > 0.
func (p *testFixture) withHeader(key, value string) *testFixture {
	p.require.NotEmpty(p.responses)
	resp := p.responses[len(p.responses)-1]
	if resp.Headers == nil {
		resp.Headers = http.Header{}
	}
	resp.Headers.Set(key, value)

	return p
}

// withoutHeader removes the given header key from the LAST response in p.responses.
// Requires len(p.responses) > 0.
func (p *testFixture) withoutHeader(key string) *testFixture {
	p.require.NotEmpty(p.responses)
	resp := p.responses[len(p.responses)-1]
	if resp.Headers != nil {
		resp.Headers.Del(key)
	}

	return p
}

// withBody replaces the Body and BodyString of the LAST response in p.responses.
// Requires len(p.responses) > 0.
func (p *testFixture) withBody(body string) *testFixture {
	p.require.NotEmpty(p.responses)
	resp := p.responses[len(p.responses)-1]
	resp.BodyString = body
	resp.Body = []byte(body)

	return p
}

// noActualResponses leaves p.responses nil (a nil slice, distinct from an empty slice).
func (p *testFixture) noActualResponses() *testFixture {
	p.responses = nil

	return p
}

// anEmptyResponseSlice sets p.responses to an empty (non-nil) slice.
func (p *testFixture) anEmptyResponseSlice() *testFixture {
	p.responses = []*rc.Response{}

	return p
}

// aNilResponse sets p.responses to []*rc.Response{nil}.
func (p *testFixture) aNilResponse() *testFixture {
	p.responses = []*rc.Response{nil}

	return p
}

// anExpectedResponseFileAt sets p.expectedFilePath verbatim. Used to point at a
// always prepends the committed-fixture directory).
func (p *testFixture) anExpectedResponseFileAt(path string) *testFixture {
	p.expectedFilePath = path

	return p
}

// validateResponsesWithIndex calls ValidateResponsesWithOptions with one expected index.
func (p *testFixture) validateResponsesWithIndex(index int) *testFixture {
	p.validationErr = p.client.ValidateResponsesWithOptions(
		p.expectedFilePath,
		rc.ValidateOptions{ExpectedIndex: index},
		p.responses...,
	)
	return p
}
