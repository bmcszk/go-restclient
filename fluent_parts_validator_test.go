package restclient_test

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"os"

	rc "github.com/bmcszk/go-restclient"
)

// validator tests (batch 3). The validator tests never execute HTTP requests:
// they hand-construct rc.Response values and call p.client.ValidateResponses
// against a fixture file, so the given-methods below populate p.responses
// directly. validateResponses (from fluent_parts_test.go) then operates on
// those responses unchanged.
//
// Same conventions: state lives in parts, assertions use parts.require /
// parts.assert, .and() stays at end of line.

// aResponseWith appends a hand-constructed rc.Response to p.responses. Body and
// The statusText is used verbatim as rc.Response.Status.
func (p *parts) aResponseWith(statusCode int, statusText, body string, headers http.Header) *parts {
	resp := &rc.Response{
		StatusCode: statusCode,
		Status:     statusText,
		Headers:    headers,
	}
	if body != "" {
		resp.BodyString = body
		resp.Body = []byte(body)
	}
	p.responses = append(p.responses, resp)
	return p
}

// aResponseWithStatus is a shorthand for aResponseWith with no body and no headers.
func (p *parts) aResponseWithStatus(statusCode int, statusText string) *parts {
	return p.aResponseWith(statusCode, statusText, "", nil)
}

// aResponseFromRawHTTPFile reads a raw HTTP response from a file at the given
// repo-root-relative path, parses it with http.ReadResponse, and appends a
// fully populated rc.Response (StatusCode, Status, Proto, Headers clone, Body,
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

	return p
}

// withStatusCode rewrites the StatusCode of the LAST response in p.responses.
// Requires len(p.responses) > 0.
func (p *parts) withStatusCode(code int) *parts {
	p.require.NotEmpty(p.responses)
	p.responses[len(p.responses)-1].StatusCode = code

	return p
}

// withStatusText rewrites the Status string of the LAST response in p.responses.
// Requires len(p.responses) > 0.
func (p *parts) withStatusText(s string) *parts {
	p.require.NotEmpty(p.responses)
	p.responses[len(p.responses)-1].Status = s

	return p
}

// withHeader Sets the given key/value pair on the Headers of the LAST response
// in p.responses. Requires len(p.responses) > 0.
func (p *parts) withHeader(key, value string) *parts {
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
func (p *parts) withoutHeader(key string) *parts {
	p.require.NotEmpty(p.responses)
	resp := p.responses[len(p.responses)-1]
	if resp.Headers != nil {
		resp.Headers.Del(key)
	}

	return p
}

// withBody replaces the Body and BodyString of the LAST response in p.responses.
// Requires len(p.responses) > 0.
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
// always prepends the committed-fixture directory).
func (p *parts) anExpectedResponseFileAt(path string) *parts {
	p.expectedFilePath = path

	return p
}

// validateResponsesWithIndex calls client.ValidateResponsesWithOptions with the
// given expected index (selecting a single expected response). The result is
// stored in p.validationErr for assertions.
func (p *parts) validateResponsesWithIndex(index int) *parts {
	p.validationErr = p.client.ValidateResponsesWithOptions(
		p.expectedFilePath,
		rc.ValidateOptions{ExpectedIndex: index},
		p.responses...,
	)
	return p
}
