package restclient_test

import (
	"net/http"
	"testing"
)

func TestCookieJarHandling_DefaultBehavior(t *testing.T) {
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
}
func TestCookieJarHandling_NoCookieJarDirective(t *testing.T) {
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
}
func TestRedirectHandling_FollowsByDefault(t *testing.T) {
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
}
func TestRedirectHandling_NoRedirectDirective(t *testing.T) {
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
}
