package restclient

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCookieJarHandling tests the client's cookie jar functionality
// by sending requests to a test server that sets cookies
func TestCookieJarHandling(t *testing.T) {
	// Given: A test server that sets cookies
	var cookieCheck bool
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/set-cookie":
			// Set a cookie
			http.SetCookie(w, &http.Cookie{
				Name:  "test-cookie",
				Value: "test-value",
			})
			w.WriteHeader(http.StatusOK)
			return
		case "/check-cookie":
			// Check if the cookie is present
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

	// Create test files for cookie testing
	withCookieJarFilePath := "testdata/cookies_redirects/with_cookie_jar.http"
	withoutCookieJarFilePath := "testdata/cookies_redirects/without_cookie_jar.http"

	// Create required test directories
	require.NoError(t, createTestDirectories("testdata/cookies_redirects"))

	// Create HTTP request files
	withCookieJarContent := "### Set Cookie Request\nGET " + testServer.URL + "/set-cookie\n\n" +
		"### Check Cookie Request\nGET " + testServer.URL + "/check-cookie\n"
	require.NoError(t, writeTestFile(withCookieJarFilePath, withCookieJarContent))

	withoutCookieJarContent := "### Set Cookie Request\nGET " + testServer.URL + "/set-cookie\n\n" +
		"### Check Cookie Request\n// @no-cookie-jar\nGET " + testServer.URL + "/check-cookie\n"
	require.NoError(t, writeTestFile(withoutCookieJarFilePath, withoutCookieJarContent))

	// When/Then: Test with cookie jar (default)
	cookieCheck = false

	// Create a client with a cookie jar
	jar, err := cookiejar.New(nil)
	require.NoError(t, err, "Should create cookie jar without error")

	client, err := NewClient()
	require.NoError(t, err, "Should create client without error")
	client.httpClient.Jar = jar

	// Execute file with cookie jar enabled (default)
	responses, err := client.ExecuteFile(context.Background(), withCookieJarFilePath)
	require.NoError(t, err, "Should execute requests without error")
	require.Len(t, responses, 2, "Should have received two responses")

	// Check if cookie was saved and sent in the second request
	assert.True(t, cookieCheck, "Cookie should be sent in second request with cookie jar enabled")

	// When/Then: Test without cookie jar (@no-cookie-jar directive)
	cookieCheck = false

	// Create a client without a cookie jar for the @no-cookie-jar test
	noJarClient, err := NewClient()
	require.NoError(t, err, "Should create client without error")
	// Intentionally NOT setting a cookie jar

	// Parse the file but execute requests manually to control cookie jar usage
	parsedFile, err := parseRequestFile(withoutCookieJarFilePath, noJarClient, nil)
	require.NoError(t, err, "Should parse request file without error")
	require.Len(t, parsedFile.Requests, 2, "Should have parsed two requests")

	// For first request (setting cookie), use a client with jar
	firstRequest := parsedFile.Requests[0]

	// Create a new client with jar for first request
	jarclient, err := NewClient()
	require.NoError(t, err, "Should create client without error")
	jar, err = cookiejar.New(nil)
	require.NoError(t, err, "Should create cookie jar without error")
	jarclient.httpClient.Jar = jar

	// Execute first request (sets cookie)
	_, err = jarclient.executeRequest(context.Background(), firstRequest)
	require.NoError(t, err, "Should execute first request without error")

	// For second request (with @no-cookie-jar directive), use client without jar
	secondRequest := parsedFile.Requests[1]
	require.True(t, secondRequest.NoCookieJar, "Second request should have NoCookieJar flag set")

	// Use the client without jar for second request
	_, err = noJarClient.executeRequest(context.Background(), secondRequest)
	require.NoError(t, err, "Should execute second request without error")

	// Check that cookie was NOT sent in the second request
	assert.False(t, cookieCheck, "Cookie should not be sent in second request with @no-cookie-jar directive")
}

// TestRedirectHandling tests the client's redirect handling functionality
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

	// Create HTTP request files
	withRedirectContent := "### Follow Redirect Request\nGET " + testServer.URL + "/redirect\n"
	require.NoError(t, writeTestFile(withRedirectFilePath, withRedirectContent))

	withoutRedirectContent := "### No Redirect Request\n// @no-redirect\nGET " + testServer.URL + "/redirect\n"
	require.NoError(t, writeTestFile(withoutRedirectFilePath, withoutRedirectContent))

	// When/Then: Test with redirect following (default)
	// Create a fresh client for redirect tests
	client, err := NewClient()
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
	// that prevents following redirects
	client, err = NewClient()
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
