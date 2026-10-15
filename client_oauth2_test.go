package restclient_test

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"

	rc "github.com/bmcszk/go-restclient"
)

// oauth2State holds the oauth2 domain state of the fluent test DSL: the address of
// the canned OAuth2 token server started by anOAuth2TokenServerWithStatus.
type oauth2State struct {
	// root points back at the enclosing parts so the oauth2 builders can return
	// the fluent chain; parts reads it by field promotion as p.root as well.
	root *parts
	// tokenServerURL holds the address of the OAuth2 token server under test.
	tokenServerURL string
}

var _ = []any{
	oauth2State{},
}

// anOAuth2TokenServerWithStatus starts a canned OAuth2 token endpoint answering with the
// given status; 200 also returns a fixed access_token payload.
func (s *oauth2State) anOAuth2TokenServerWithStatus(status int) *parts {
	p := s.root
	p.aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if status == http.StatusOK {
			_, _ = w.Write([]byte(`{"access_token":"test-access-token","token_type":"Bearer"}`))
		}
	})
	p.tokenServerURL = p.serverURL

	return p
}

// aClientWithOAuth2Vars builds a client whose `p`-prefixed OAuth2 variables point at the token server.
func (s *oauth2State) aClientWithOAuth2Vars() *parts {
	p := s.root
	return p.aClient(rc.WithVars(map[string]any{
		"p_tokenEndpoint": p.tokenServerURL + "/token",
		"p_clientId":      "test-client",
		"p_clientSecret":  "test-secret",
	}))
}

// aClientWithOAuth2ProgrammaticVars builds a client with the given programmatic vars —
// for scenarios whose @-vars resolve through the normal lookup chain.
func (s *oauth2State) aClientWithOAuth2ProgrammaticVars(vars map[string]any) *parts {
	p := s.root
	return p.aClient(rc.WithVars(vars))
}

// clientWithProgrammaticVars adds programmatic vars to the existing client (SetProgrammaticVars),
// mirroring the -D/--define CLI path.
func (s *oauth2State) clientWithProgrammaticVars(vars map[string]any) *parts {
	p := s.root
	p.client.SetProgrammaticVars(vars)

	return p
}

// aClientWithOAuth2PasswordVars builds a client whose `p`-prefixed OAuth2 password-grant
// variables point at the token server; useAuthorizationHeader=false keeps client creds in the body.
func (s *oauth2State) aClientWithOAuth2PasswordVars() *parts {
	p := s.root
	return p.aClient(rc.WithVars(map[string]any{
		"p_tokenEndpoint":          p.tokenServerURL + "/token",
		"p_clientId":               "test-client",
		"p_clientSecret":           "test-secret",
		"p_username":               "test-user",
		"p_password":               "test-pass",
		"p_useAuthorizationHeader": "false",
	}))
}

// aClientWithOAuth2PasswordVarsAuthHeaderUnset omits useAuthorizationHeader so the
// file's @-var (or the true default) decides the Basic-header behavior.
func (s *oauth2State) aClientWithOAuth2PasswordVarsAuthHeaderUnset() *parts {
	p := s.root
	return p.aClient(rc.WithVars(map[string]any{
		"p_tokenEndpoint": p.tokenServerURL + "/token",
		"p_clientId":      "test-client",
		"p_clientSecret":  "test-secret",
		"p_username":      "test-user",
		"p_password":      "test-pass",
	}))
}

// aClientWithOAuth2PasswordVarsMissingPassword omits the password to exercise the missing-variable error.
func (s *oauth2State) aClientWithOAuth2PasswordVarsMissingPassword() *parts {
	p := s.root
	return p.aClient(rc.WithVars(map[string]any{
		"p_tokenEndpoint": p.tokenServerURL + "/token",
		"p_clientId":      "test-client",
		"p_clientSecret":  "test-secret",
		"p_username":      "test-user",
	}))
}

// aClientWithOAuth2VarsMissingSecret omits the client secret to exercise the missing-variable error.
func (s *oauth2State) aClientWithOAuth2VarsMissingSecret() *parts {
	p := s.root
	return p.aClient(rc.WithVars(map[string]any{
		"p_tokenEndpoint": p.tokenServerURL + "/token",
		"p_clientId":      "test-client",
	}))
}

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
