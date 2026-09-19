package restclient_test

import (
	"fmt"
	"net/http"
	"testing"
)

func TestSystemVar_Base64Encode(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		}).and().
		aHttpFile("### auth\n" +
			"GET {{server}}/api\n" +
			"Authorization: Basic {{$base64encode user:pass}}\n").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		serverReceivedHeaderValue(0, "Authorization", "Basic dXNlcjpwYXNz")
}

func TestSystemVar_Base64EncodeInBody(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}).and().
		aHttpFile("### send\n" +
			"POST {{server}}/data\n" +
			"Content-Type: text/plain\n\n" +
			"{{$base64encode hello world}}\n").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		serverReceivedBodyIs(0, "aGVsbG8gd29ybGQ=")
}
