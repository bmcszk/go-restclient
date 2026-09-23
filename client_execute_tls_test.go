package restclient_test

import (
	"crypto/tls"
	"net/http"
	"testing"

	rc "github.com/bmcszk/go-restclient"
)

// An HTTPS response records IsTLS, TLSVersion and cipher suite.
func TestExecuteFile_HTTPSResponseCapturesTLSData(t *testing.T) {
	given, when, then := newParts(t)

	httpClient := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	given.
		aTLSServer().and().
		aHttpFile(`### tls
GET {{server}}/secure`).and().
		aClient(rc.WithHTTPClient(httpClient))

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(1).and().
		responseIsTLS().and().
		responseTLSVersionMatches("TLS 1\\.3")
}

// responseIsTLS asserts the current response used TLS.
func (p *parts) responseIsTLS() *parts {
	p.require.True(p.current().IsTLS, "expected TLS response, got plain")

	return p
}

// responseTLSVersionMatches asserts the current response TLS version against a regexp.
func (p *parts) responseTLSVersionMatches(pattern string) *parts {
	p.assert.Regexp(pattern, p.current().TLSVersion)

	return p
}
