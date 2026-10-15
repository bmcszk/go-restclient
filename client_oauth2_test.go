package restclient_test

import (
	"encoding/base64"
	"fmt"
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

func TestExecuteFile_OAuth2TokenEndpointNestedVarExpands(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anOAuth2TokenServerWithStatus(http.StatusOK).and().
		anEchoServer().and().
		aHttpFile(fmt.Sprintf("@host = %[1]s\n@acme_tokenEndpoint = {{host}}/oauth2/token\n\n"+
			"### oauth2 api\nGET {{server}}/api\nAuthorization: oauth2 acme", given.tokenServerURL)).and().
		aClientWithOAuth2ProgrammaticVars(map[string]any{
			"acme_clientId":     "nested-client",
			"acme_clientSecret": "nested-secret",
		})

	when.
		executeFile()

	then.
		noError().and().
		serverReceivedMethodAndPath(0, http.MethodPost, "/oauth2/token").and().
		serverReceivedMethodAndPath(1, http.MethodGet, "/api").and().
		serverReceivedHeaderValue(1, "Authorization", "Bearer test-access-token")
}

func TestExecuteFile_OAuth2ClientCredentialsDefineOverridesNestedVar(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anOAuth2TokenServerWithStatus(http.StatusOK).and().
		anEchoServer().and().
		aHttpFile(`@acme_clientId = {{cid}}

### oauth2 api
GET {{server}}/api
Authorization: oauth2 acme`).and().
		aClientWithOAuth2ProgrammaticVars(map[string]any{
			"acme_tokenEndpoint": given.tokenServerURL + "/token",
			"acme_clientSecret":  "acme-secret",
		}).and().
		clientWithProgrammaticVars(map[string]any{"cid": "real-client-id"})

	when.
		executeFile()

	then.
		noError().and().
		serverReceivedMethodAndPath(0, http.MethodPost, "/token").and().
		serverReceivedBodyIs(0, "client_id=real-client-id&client_secret=acme-secret"+
			"&grant_type=client_credentials").and().
		serverReceivedMethodAndPath(1, http.MethodGet, "/api").and().
		serverReceivedHeaderValue(1, "Authorization", "Bearer test-access-token")
}

func TestExecuteFile_OAuth2ProcessEnvPlaceholderInAtVarExpands(t *testing.T) {
	given, when, then := newParts(t)

	given.
		withEnv("ISS45_CID", "env-client-id").and().
		anOAuth2TokenServerWithStatus(http.StatusOK).and().
		anEchoServer().and().
		aHttpFile(`@cid = {{$processEnv ISS45_CID}}
@acme_clientId = {{cid}}

### oauth2 api
GET {{server}}/api
Authorization: oauth2 acme`).and().
		aClientWithOAuth2ProgrammaticVars(map[string]any{
			"acme_tokenEndpoint": given.tokenServerURL + "/token",
			"acme_clientSecret":  "acme-secret",
		})

	when.
		executeFile()

	then.
		noError().and().
		serverReceivedMethodAndPath(0, http.MethodPost, "/token").and().
		serverReceivedBodyIs(0, "client_id=env-client-id&client_secret=acme-secret&grant_type=client_credentials")
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

func TestExecuteFile_OAuth2PasswordGrantPostsUserCredentials(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anOAuth2TokenServerWithStatus(http.StatusOK).and().
		anEchoServer().and().
		aHttpFile(`### oauth2 api
GET {{server}}/api
Authorization: oauth2 password p`).and().
		aClientWithOAuth2PasswordVars()

	when.
		executeFile()

	then.
		noError().and().
		serverReceivedMethodAndPath(0, http.MethodPost, "/token").and().
		serverReceivedHeaderValue(0, "Content-Type", "application/x-www-form-urlencoded").and().
		serverReceivedHeaderValue(0, "Authorization", "").and().
		serverReceivedBodyIs(0, "client_id=test-client&client_secret=test-secret&grant_type=password"+
			"&password=test-pass&username=test-user").and().
		serverReceivedMethodAndPath(1, http.MethodGet, "/api").and().
		serverReceivedHeaderValue(1, "Authorization", "Bearer test-access-token")
}

func TestExecuteFile_OAuth2UseAuthorizationHeaderFalseSendsCredsInBody(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anOAuth2TokenServerWithStatus(http.StatusOK).and().
		anEchoServer().and().
		aHttpFile(`@p_useAuthorizationHeader = false

### oauth2 api
GET {{server}}/api
Authorization: oauth2 password p`).and().
		aClientWithOAuth2PasswordVars()

	when.
		executeFile()

	then.
		noError().and().
		serverReceivedMethodAndPath(0, http.MethodPost, "/token").and().
		serverReceivedHeaderValue(0, "Authorization", "").and().
		serverReceivedBodyIs(0, "client_id=test-client&client_secret=test-secret&grant_type=password"+
			"&password=test-pass&username=test-user").and().
		serverReceivedMethodAndPath(1, http.MethodGet, "/api").and().
		serverReceivedHeaderValue(1, "Authorization", "Bearer test-access-token")
}

func TestExecuteFile_OAuth2UseAuthorizationHeaderTrueSendsBasicHeader(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anOAuth2TokenServerWithStatus(http.StatusOK).and().
		anEchoServer().and().
		aHttpFile(`@p_useAuthorizationHeader = true

### oauth2 api
GET {{server}}/api
Authorization: oauth2 password p`).and().
		aClientWithOAuth2PasswordVarsAuthHeaderUnset()

	when.
		executeFile()

	then.
		noError().and().
		serverReceivedMethodAndPath(0, http.MethodPost, "/token").and().
		serverReceivedHeaderValue(0, "Authorization",
			"Basic "+base64.StdEncoding.EncodeToString([]byte("test-client:test-secret"))).and().
		serverReceivedBodyIs(0, "grant_type=password&password=test-pass&username=test-user").and().
		serverReceivedMethodAndPath(1, http.MethodGet, "/api").and().
		serverReceivedHeaderValue(1, "Authorization", "Bearer test-access-token")
}

func TestExecuteFile_OAuth2PasswordGrantMissingPasswordFails(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anOAuth2TokenServerWithStatus(http.StatusOK).and().
		anEchoServer().and().
		aHttpFile(`### oauth2 api
GET {{server}}/api
Authorization: oauth2 password p`).and().
		aClientWithOAuth2PasswordVarsMissingPassword()

	when.
		executeFile()

	then.
		errorContains("oauth2", "p_password").and().
		capturedRequestCount(0)
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
