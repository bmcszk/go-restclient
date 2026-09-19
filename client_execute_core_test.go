package restclient_test

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// from a .http file (http_syntax.md).
// 'test/data/http_request_files/single_request.http' and retrieve the response.

func TestExecuteFile_SingleRequest(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/users", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "user data")
		}).and().
		aHttpFileFromTemplate("single_request.http").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseCode(http.StatusOK).and().
		responseBodyIs("user data")
}

func TestExecuteFile_MultipleRequests(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/req1":
				assert.Equal(t, http.MethodGet, r.Method)
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, "response1")
			case "/req2":
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				bodyBytes, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				assert.JSONEq(t, `{"key": "value"}`, string(bodyBytes))
				w.WriteHeader(http.StatusCreated)
				_, _ = fmt.Fprint(w, "response2")
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}).and().
		aHttpFileFromTemplate("multiple_requests.http").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(2).and().
		capturedRequestCount(2).and().
		noError().and().
		responseAt(0).and().
		responseCode(http.StatusOK).and().
		responseBodyIs("response1").and().
		responseAt(1).and().
		responseCode(http.StatusCreated).and().
		responseBodyIs("response2").and().
		responsesValidateAgainstFixture("client_multiple_requests_expected.hresp")
}

func TestExecuteFile_RequestWithError(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "good response")
		}).and().
		aHttpFileFromTemplate("request_with_error.http").and().
		aClient()

	when.
		executeFile()

	then.
		errorContains(
			"1 error occurred:",
			"request 1 (GET http://localhost:12346/bad) processing resulted in error",
		).and().
		responseCount(2).and().
		responseAt(0).and().
		responseHasError("failed to execute HTTP request:").and().
		responseAt(1).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		responseBodyIs("good response")
}

func TestExecuteFile_ParseError(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aRequestFixture("parse_error.http").and().
		aClient()

	when.
		executeFile()

	then.
		errorContains("no requests found in file test/data/http_request_files/parse_error.http")
}
func TestExecuteFile_NoRequestsInFile(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aRequestFixture("comment_only_file.http").and().
		aClient()

	when.
		executeFile()

	then.
		errorContains("no requests found in file test/data/http_request_files/comment_only_file.http")
}

// followed by content that causes a parsing error (http_syntax.md).
// the client executes valid requests up to the point of the parse error and then reports
// the parsing error, potentially halting further execution from that file.
func TestExecuteFile_ValidThenInvalidSyntax(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet && r.URL.Path == "/first" {
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, "response from /first")
			} else if r.Method == "INVALID_METHOD" && r.URL.Path == "/second" {
				w.WriteHeader(http.StatusNotImplemented)
				_, _ = fmt.Fprint(w, "method not implemented")
			} else {
				t.Logf("Mock server received UNEXPECTED request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusTeapot)
			}
		}).and().
		aHttpFileFromTemplate("valid_then_invalid_syntax.http").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(2).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		responseBodyIs("response from /first").and().
		responseAt(1).and().
		responseHasNoError().and().
		responseCode(http.StatusNotImplemented).and().
		responseContains("method not implemented")
}

// designed to fail) to verify that each failing request's error is captured in its
// respective response object and that an aggregated error is returned by ExecuteFile.
func TestExecuteFile_MultipleErrors(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aRequestFixture("multiple_errors.http").and().
		aClient()

	when.
		executeFile()

	then.
		errorContains(
			"request 1 (GET http://localhost:12347/badreq1) processing resulted in error",
			":12347: connect: connection refused",
			"request 2 (POST http://localhost:12348/badreq2) processing resulted in error",
			":12348: connect: connection refused",
		).and().
		responseCount(2).and().
		responseAt(0).and().
		responseHasError(":12347: connect: connection refused").and().
		responseAt(1).and().
		responseHasError(":12348: connect: connection refused")
}

// (http_syntax.md "Response Object").
// the client correctly stores received headers, including multi-value headers, in the Response
// object.
func TestExecuteFile_CapturesResponseHeaders(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.Header().Add("X-Custom-Header", "value1")
			w.Header().Add("X-Custom-Header", "value2")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "{\"data\": \"headers test\"}")
		}).and().
		aHttpFileFromTemplate("captures_response_headers.http").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseCode(http.StatusOK).and().
		responseHeader("Content-Type", "application/vnd.api+json").and().
		responseHeaderValues("X-Custom-Header", "value1", "value2").and().
		responseHeaderEmpty("Non-Existent-Header")
}

// mock transport (http_syntax.md).
// verify the fundamental request execution flow, ensuring the correct method, URL, and headers
// are prepared and sent.
func TestExecuteFile_SimpleGetHTTP(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aRequestFixture("simple_get.http").and().
		aMockTransportClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseCode(http.StatusOK).and().
		clientRequestInterceptorCaptures("GET", "https://jsonplaceholder.typicode.com/todos/1").and().
		clientSentNoHeaders()
}

// ensuring all are processed sequentially (http_syntax.md "Request Separation").
// ensure the client can handle a larger number of requests in a file.
func TestExecuteFile_MultipleRequests_GreaterThanTwo(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/req1":
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, "response1")
			case "/req2":
				w.WriteHeader(http.StatusCreated)
				_, _ = fmt.Fprint(w, "response2")
			case "/req3":
				w.WriteHeader(http.StatusAccepted)
				_, _ = fmt.Fprint(w, "response3")
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}).and().
		aHttpFileFromTemplate("multiple_requests_gt2.http").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(3).and().
		noError().and().
		responsesValidateAgainstFixture("multiple_responses_gt2_expected.http")
}

// ? and & line continuations (http_syntax.md "Query Parameters on Multiple Lines").
// spread across multiple lines using the VS Code REST Client syntax.
func TestExecuteFile_MultilineQueryParameters(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/comments":
				query := r.URL.Query()
				assert.Equal(t, "2", query.Get("page"))
				assert.Equal(t, "10", query.Get("pageSize"))
				assert.Equal(t, "active", query.Get("filter"))
				assert.Equal(t, "created_at", query.Get("sort"))
				assert.Equal(t, "desc", query.Get("order"))
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `{"comments": [], "page": 2}`)
			case "/api/search":
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `{"results": [], "total": 0}`)
			default:
				t.Logf("Server received unexpected request to: %s", r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		}).and().
		aHttpFileFromTemplate("multiline_query_parameters.http").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(2).and().
		noError().and().
		responseAt(0).and().
		responseCode(http.StatusOK).and().
		requestRawURLContains("page=2", "pageSize=10", "filter=active", "sort=created_at", "order=desc").and().
		responseAt(1).and().
		requestRawURLContains("q=test query", "limit=50", "offset=0")
}

// & line continuations (http_syntax.md "Form Data on Multiple Lines").
// spread across multiple lines using the VS Code REST Client syntax.
func TestExecuteFile_MultilineFormData(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, r *http.Request) {
			require.NoError(t, r.ParseForm())

			switch r.URL.Path {
			case "/api/login":
				assert.Equal(t, "testuser", r.FormValue("username"))
				assert.Equal(t, "testpass123", r.FormValue("password"))
				assert.Equal(t, "true", r.FormValue("remember_me"))
				assert.Equal(t, "read,write", r.FormValue("scope"))
				assert.Equal(t, "https://example.com/callback", r.FormValue("redirect_uri"))
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `{"token": "abc123"}`)
			case "/api/submit":
				assert.Equal(t, "value1", r.FormValue("field1"))
				assert.Equal(t, "value with spaces", r.FormValue("field2"))
				assert.Equal(t, "special chars", r.FormValue("field3"))
				assert.Equal(t, "multi", r.FormValue("field4"))
				assert.Equal(t, "data", r.FormValue("line"))
				w.WriteHeader(http.StatusCreated)
				_, _ = fmt.Fprint(w, `{"success": true}`)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}).and().
		aHttpFileFromTemplate("multiline_form_data.http").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(2).and().
		capturedRequestCount(2).and().
		noError().and().
		responseAt(0).and().
		responseCode(http.StatusOK).and().
		responseContains(`"token"`).and().
		responseAt(1).and().
		responseCode(http.StatusCreated).and().
		responseContains(`"success"`)
}

// < file references (http_syntax.md "File Upload").
// that include file uploads using the < file reference syntax.
func TestExecuteFile_MultipartFileUploads(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseMultipartForm(32 << 20) // 32MB max
			require.NoError(t, err)

			switch r.URL.Path {
			case "/api/upload":
				assert.Equal(t, "File upload test", r.FormValue("description"))

				file1, file1Header, err := r.FormFile("file1")
				require.NoError(t, err)
				assert.Equal(t, "sample_text.txt", file1Header.Filename)
				assert.Equal(t, "text/plain", file1Header.Header.Get("Content-Type"))
				_ = file1.Close()

				file2, file2Header, err := r.FormFile("file2")
				require.NoError(t, err)
				assert.Equal(t, "sample_image.jpg", file2Header.Filename)
				assert.Equal(t, "image/jpeg", file2Header.Header.Get("Content-Type"))
				_ = file2.Close()

				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `{"uploaded": ["file1", "file2"]}`)
			case "/api/upload-json":
				metadata, metadataHeader, err := r.FormFile("metadata")
				require.NoError(t, err)
				assert.Equal(t, "application/json", metadataHeader.Header.Get("Content-Type"))
				_ = metadata.Close()

				document, documentHeader, err := r.FormFile("document")
				require.NoError(t, err)
				assert.Equal(t, "sample_document.pdf", documentHeader.Filename)
				assert.Equal(t, "application/pdf", documentHeader.Header.Get("Content-Type"))
				_ = document.Close()

				w.WriteHeader(http.StatusCreated)
				_, _ = fmt.Fprint(w, `{"processed": true}`)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}).and().
		anUploadsFixtureCopy().and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(2).and().
		capturedRequestCount(2).and().
		noError().and().
		responseAt(0).and().
		responseCode(http.StatusOK).and().
		responseContains(`"uploaded"`).and().
		responseAt(1).and().
		responseCode(http.StatusCreated).and().
		responseContains(`"processed"`)
}
