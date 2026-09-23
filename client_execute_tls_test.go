package restclient_test

import (
	"testing"
)

// An HTTPS response records IsTLS, TLSVersion and cipher suite.
func TestExecuteFile_HTTPSResponseCapturesTLSData(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aTLSServer().and().
		aHttpFile(`### tls
GET {{server}}/secure`).and().
		aTLSInsecureClient()

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(1).and().
		responseIsTLS().and().
		responseTLSVersionMatches("TLS 1\\.3")
}
