package restclient_test

import (
	"net/http"
	"testing"
)

func TestExecuteFile_OAuth2ClientCredentialsSendsBearerToken(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anOAuth2TokenServerWithStatus(http.StatusOK).and().
		anEchoServer().and().
		aHttpFile(`### oauth2 api
GET {{server}}/api
Authorization: oauth2 p`).and().
		aClientWithOAuth2Vars()

	when.
		executeFile()

	then.
		noError().and().
		serverReceivedMethodAndPath(0, http.MethodPost, "/token").and().
		serverReceivedHeaderValue(0, "Content-Type", "application/x-www-form-urlencoded").and().
		serverReceivedBodyIs(0, "client_id=test-client&client_secret=test-secret&grant_type=client_credentials").and().
		serverReceivedMethodAndPath(1, http.MethodGet, "/api").and().
		serverReceivedHeaderValue(1, "Authorization", "Bearer test-access-token")
}

func TestExecuteFile_OAuth2ClientCredentialsCachesTokenAcrossRequests(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anOAuth2TokenServerWithStatus(http.StatusOK).and().
		anEchoServer().and().
		aHttpFile(`### first
GET {{server}}/one
Authorization: oauth2 p

### second
GET {{server}}/two
Authorization: oauth2 p`).and().
		aClientWithOAuth2Vars()

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(3).and().
		serverReceivedMethodAndPath(0, http.MethodPost, "/token").and().
		serverReceivedMethodAndPath(1, http.MethodGet, "/one").and().
		serverReceivedMethodAndPath(2, http.MethodGet, "/two").and().
		serverReceivedHeaderValue(1, "Authorization", "Bearer test-access-token").and().
		serverReceivedHeaderValue(2, "Authorization", "Bearer test-access-token")
}

func TestExecuteFile_OAuth2MissingClientSecretFailsNamingVar(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anOAuth2TokenServerWithStatus(http.StatusOK).and().
		anEchoServer().and().
		aHttpFile(`### oauth2 api
GET {{server}}/api
Authorization: oauth2 p`).and().
		aClientWithOAuth2VarsMissingSecret()

	when.
		executeFile()

	then.
		errorContains("oauth2", "p_clientSecret")
}

func TestExecuteFile_OAuth2TokenEndpointErrorFailsNamingStatus(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anOAuth2TokenServerWithStatus(http.StatusInternalServerError).and().
		anEchoServer().and().
		aHttpFile(`### oauth2 api
GET {{server}}/api
Authorization: oauth2 p`).and().
		aClientWithOAuth2Vars()

	when.
		executeFile()

	then.
		errorContains("oauth2", "p", "500")
}

func TestExecuteFile_OAuth2UnsupportedGrantFailsWithClearError(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anOAuth2TokenServerWithStatus(http.StatusOK).and().
		anEchoServer().and().
		aHttpFile(`### oauth2 api
GET {{server}}/api
Authorization: oauth2 device_code p`).and().
		aClientWithOAuth2Vars()

	when.
		executeFile()

	then.
		errorContains("unsupported", "device_code")
}
