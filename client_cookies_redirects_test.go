// Code in this file is migrated from test/client_cookies_redirects.go (RunCookieJarHandling, RunRedirectHandling).
package restclient_test

import (
	"net/http"
	"testing"
)

// PRD-COMMENT: FR9.1 - Client Execution: Cookie Jar Management
// Corresponds to: Client execution behavior regarding HTTP cookies and the '@no-cookie-jar'
// request setting (http_syntax.md "Request Settings", "@no-cookie-jar").
// This test verifies the client's cookie jar functionality. It checks:
//  1. Default behavior: Cookies set by a server are stored in the client's cookie jar and sent
//     with subsequent requests to the same domain.
//  2. '@no-cookie-jar' directive: When a request includes the '@no-cookie-jar' setting, the
//     client does not use its cookie jar for that specific request (neither sending stored
//     cookies nor saving new ones from the response).
func TestCookieJarHandling(t *testing.T) {
	t.Run("default_behavior_with_cookie_jar", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aFreshCookieCheck().and().
			aCookieTestServer().and().
			aCookieRedirectFixture("with_cookie_jar.http").and().
			withServerAddressVars().and().
			aCookieJarClient()

		when.
			executeFile()

		then.
			responseCount(2).and().
			noError().and().
			cookieWasStored()
	})

	t.Run("directive_no_cookie_jar", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aFreshCookieCheck().and().
			aCookieTestServer().and().
			aCookieRedirectFixture("without_cookie_jar.http").and().
			withServerAddressVars().and().
			aCookieJarClient()

		when.
			executeFile()

		then.
			responseCount(2).and().
			noError().and().
			cookieWasNotStored()
	})
}

// PRD-COMMENT: FR9.2 - Client Execution: Redirect Handling
// Corresponds to: Client execution behavior regarding HTTP redirects and the '@no-redirect'
// request setting (http_syntax.md "Request Settings", "@no-redirect").
// This test verifies the client's redirect handling. It checks:
//  1. Default behavior: The client automatically follows HTTP redirects (e.g., 302 Found).
//  2. '@no-redirect' directive: When a request includes the '@no-redirect' setting, the client
//     does not automatically follow redirects and instead returns the redirect response itself.
func TestRedirectHandling(t *testing.T) {
	t.Run("follows_redirect_by_default", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aRedirectTestServer().and().
			aCookieRedirectFixture("with_redirect.http").and().
			withServerAddressVars()

		when.
			executeFile()

		then.
			responseCount(1).and().
			noError().and().
			responseAt(0).and().
			responseCode(http.StatusOK).and().
			responseBodyIs("Target page")
	})

	t.Run("no_redirect_directive_returns_redirect_response", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aRedirectTestServer().and().
			aCookieRedirectFixture("without_redirect.http").and().
			withServerAddressVars().and().
			aNoRedirectClient()

		when.
			executeFile()

		then.
			responseCount(1).and().
			noError().and().
			responseAt(0).and().
			responseCode(http.StatusFound)
	})
}
