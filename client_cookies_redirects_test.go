package restclient

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url" // Added import
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cookieJarTestCase struct {
	name                       string
	httpFilePath               string
	expectedCookieCheckValue   bool
	isNoCookieJarDirectiveTest bool // True if testing the @no-cookie-jar directive specifically
}

// runCookieJarSubtest executes a subtest for cookie jar handling scenarios.
// It uses a pointer to cookieCheck because the HTTP handler modifies this global variable.
func runCookieJarSubtest(t *testing.T, tc cookieJarTestCase, serverVars map[string]interface{}, cookieCheck *bool) {
	t.Helper()

	*cookieCheck = false // Reset for each subtest

	if tc.isNoCookieJarDirectiveTest {
		// Special handling for @no-cookie-jar directive test
		// Create a client without a cookie jar for the @no-cookie-jar test, but with server variables
		noJarClient, err := NewClient(WithVars(serverVars))
		require.NoError(t, err, "Should create client without error for @no-cookie-jar test")
		// Intentionally NOT setting a cookie jar for noJarClient

		// Parse the file. serverVars are now in noJarClient.programmaticVars.
		parsedFile, err := parseRequestFile(tc.httpFilePath, noJarClient, nil)
		require.NoError(t, err, "Should parse request file without error for @no-cookie-jar test")
		require.Len(t, parsedFile.Requests, 2, "Should have parsed two requests for @no-cookie-jar test")

		// For first request (setting cookie), use a client with jar
		firstRequest := parsedFile.Requests[0]

		// Create a new client with jar for first request, and with server variables
		jarClient, err := NewClient(WithVars(serverVars))
		require.NoError(t, err, "Should create jarClient without error for @no-cookie-jar test")
		jar, err := cookiejar.New(nil)
		require.NoError(t, err, "Should create cookie jar for jarClient for @no-cookie-jar test")
		jarClient.httpClient.Jar = jar

		// Execute first request (sets cookie)
		_, err = jarClient.executeRequest(context.Background(), firstRequest)
		require.NoError(t, err, "Should execute first request without error for @no-cookie-jar test")

		// For second request (with @no-cookie-jar directive), use client without jar
		secondRequest := parsedFile.Requests[1]
		require.True(t, secondRequest.NoCookieJar, "Second request should have NoCookieJar flag set for @no-cookie-jar test")

		// Use the client without jar for second request
		_, err = noJarClient.executeRequest(context.Background(), secondRequest)
		require.NoError(t, err, "Should execute second request without error for @no-cookie-jar test")

		assert.Equal(t, tc.expectedCookieCheckValue, *cookieCheck, "Cookie check assertion failed for @no-cookie-jar test")
	} else {
		// Default behavior test (with or without jar based on client setup)
		client, err := NewClient(WithVars(serverVars))
		require.NoError(t, err, "Should create client without error")

		// For the default 'with cookie jar' case, the client needs a jar.
		// For a hypothetical 'explicitly no jar on client' case (not currently tested this way),
		// we wouldn't add a jar here.
		// Based on tc.name, this implies the default test where client *should* have a jar.
		if tc.httpFilePath == "testdata/cookies_redirects/with_cookie_jar.http" { // A bit of a hack to infer client needs jar
			jar, err := cookiejar.New(nil)
			require.NoError(t, err, "Should create cookie jar without error")
			client.httpClient.Jar = jar
		}

		responses, err := client.ExecuteFile(context.Background(), tc.httpFilePath)
		require.NoError(t, err, "Should execute requests without error")
		require.Len(t, responses, 2, "Should have received two responses")

		assert.Equal(t, tc.expectedCookieCheckValue, *cookieCheck, "Cookie check assertion failed")
	}
}

// PRD-COMMENT: FR9.1 - Client Execution: Cookie Jar Management
// Corresponds to: Client execution behavior regarding HTTP cookies and the '@no-cookie-jar' request setting (http_syntax.md "Request Settings", "@no-cookie-jar").
// This test verifies the client's cookie jar functionality. It checks:
// 1. Default behavior: Cookies set by a server are stored in the client's cookie jar and sent with subsequent requests to the same domain.
// 2. '@no-cookie-jar' directive: When a request includes the '@no-cookie-jar' setting, the client does not use its cookie jar for that specific request (neither sending stored cookies nor saving new ones from the response).
// It uses dynamically created 'testdata/cookies_redirects/with_cookie_jar.http' and 'testdata/cookies_redirects/without_cookie_jar.http' files.
func TestCookieJarHandling(t *testing.T) {
	// Given: A test server that sets cookies
	var cookieCheck bool // This variable is modified by the HTTP handler
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/set-cookie":
			http.SetCookie(w, &http.Cookie{Name: "test-cookie", Value: "test-value"})
			w.WriteHeader(http.StatusOK)
			return
		case "/check-cookie":
			cookie, err := r.Cookie("test-cookie")
			if err == nil && cookie.Value == "test-value" {
				cookieCheck = true
			}
			w.WriteHeader(http.StatusOK)
			return
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer testServer.Close()

	// Common setup: Create test files and parse server URL
	withCookieJarFilePath := "testdata/cookies_redirects/with_cookie_jar.http"
	withoutCookieJarFilePath := "testdata/cookies_redirects/without_cookie_jar.http"
	require.NoError(t, createTestDirectories("testdata/cookies_redirects"))

	withCookieJarContent := "### Set Cookie Request\nGET {{scheme}}://{{host}}:{{port}}/set-cookie\n\n" +
		"### Check Cookie Request\nGET {{scheme}}://{{host}}:{{port}}/check-cookie\n"
	require.NoError(t, writeTestFile(withCookieJarFilePath, withCookieJarContent))

	withoutCookieJarContent := "### Set Cookie Request\nGET {{scheme}}://{{host}}:{{port}}/set-cookie\n\n" +
		"### Check Cookie Request\n// @no-cookie-jar\nGET {{scheme}}://{{host}}:{{port}}/check-cookie\n"
	require.NoError(t, writeTestFile(withoutCookieJarFilePath, withoutCookieJarContent))

	parsedURL, err := url.Parse(testServer.URL)
	require.NoError(t, err, "Failed to parse testServer.URL")
	serverVars := map[string]interface{}{
		"scheme": parsedURL.Scheme,
		"host":   parsedURL.Hostname(),
		"port":   parsedURL.Port(),
	}

	tests := []cookieJarTestCase{
		{
			name:                       "default_behavior_with_cookie_jar",
			httpFilePath:               withCookieJarFilePath,
			expectedCookieCheckValue:   true,
			isNoCookieJarDirectiveTest: false,
		},
		{
			name:                       "directive_no_cookie_jar",
			httpFilePath:               withoutCookieJarFilePath,
			expectedCookieCheckValue:   false,
			isNoCookieJarDirectiveTest: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runCookieJarSubtest(t, tc, serverVars, &cookieCheck)
		})
	}
}

// PRD-COMMENT: FR9.2 - Client Execution: Redirect Handling
// Corresponds to: Client execution behavior regarding HTTP redirects and the '@no-redirect' request setting (http_syntax.md "Request Settings", "@no-redirect").
// This test verifies the client's redirect handling. It checks:
// 1. Default behavior: The client automatically follows HTTP redirects (e.g., 302 Found).
// 2. '@no-redirect' directive: When a request includes the '@no-redirect' setting, the client does not automatically follow redirects and instead returns the redirect response itself.
// It uses dynamically created 'testdata/cookies_redirects/with_redirect.http' and 'testdata/cookies_redirects/without_redirect.http' files.
func TestRedirectHandling(t *testing.T) {
	// Given: A test server that performs redirects
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			// Send a redirect
			http.Redirect(w, r, "/target", http.StatusFound)
			return
		case "/target":
			// Target of redirect
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Target page"))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer testServer.Close()

	// Create test files for redirect testing
	withRedirectFilePath := "testdata/cookies_redirects/with_redirect.http"
	withoutRedirectFilePath := "testdata/cookies_redirects/without_redirect.http"

	// Create required test directories if not already done
	require.NoError(t, createTestDirectories("testdata/cookies_redirects"))

	// Create HTTP request files with variables
	withRedirectContent := "### Follow Redirect Request\nGET {{scheme}}://{{host}}:{{port}}/redirect\n"
	require.NoError(t, writeTestFile(withRedirectFilePath, withRedirectContent))

	withoutRedirectContent := "### No Redirect Request\n// @no-redirect\nGET {{scheme}}://{{host}}:{{port}}/redirect\n"
	require.NoError(t, writeTestFile(withoutRedirectFilePath, withoutRedirectContent))

	// Parse testServer.URL to get scheme, host, and port
	parsedURL, err := url.Parse(testServer.URL)
	require.NoError(t, err, "Failed to parse testServer.URL")
	serverVars := map[string]interface{}{
		"scheme": parsedURL.Scheme,
		"host":   parsedURL.Hostname(),
		"port":   parsedURL.Port(),
	}

	// When/Then: Test with redirect following (default)
	// Create a fresh client for redirect tests, with server variables
	client, err := NewClient(WithVars(serverVars))
	require.NoError(t, err, "Should create client without error")

	// Execute file with default redirect behavior
	responses, err := client.ExecuteFile(context.Background(), withRedirectFilePath)
	require.NoError(t, err, "Should execute request without error")
	require.Len(t, responses, 1, "Should receive one response")

	// Check that redirect was followed and we got the target page
	assert.Equal(t, http.StatusOK, responses[0].StatusCode,
		"Response should have 200 OK status after following redirect")
	assert.Equal(t, "Target page", string(responses[0].Body), "Response body should be from target page")

	// When/Then: Test without redirect following (@no-redirect directive)

	// For the @no-redirect test, create a client with a custom CheckRedirect function
	// that prevents following redirects, and with server variables
	client, err = NewClient(WithVars(serverVars))
	require.NoError(t, err, "Should create client without error")

	// Override the client's CheckRedirect function to capture the redirect status
	client.httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse // Don't follow redirects
	}

	// Execute file with @no-redirect directive
	responses, err = client.ExecuteFile(context.Background(), withoutRedirectFilePath)
	require.NoError(t, err, "Should execute request without error")
	require.Len(t, responses, 1, "Should receive one response")

	// Check that redirect was not followed
	assert.Equal(t, http.StatusFound, responses[0].StatusCode,
		"Response should have 302 Found status when not following redirect")
}

// Helper function to create test directories
func createTestDirectories(dirs ...string) error {
	for _, dir := range dirs {
		// Ensure directory exists, creating it if necessary
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

// Helper function to create test files
func writeTestFile(filePath, content string) error {
	// Create parent directories if needed
	dir := dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filePath, []byte(content), 0644)
}

// Helper function to get directory portion of a file path
func dir(filePath string) string {
	for i := len(filePath) - 1; i >= 0; i-- {
		if filePath[i] == '/' {
			return filePath[:i]
		}
	}
	return ""
}
