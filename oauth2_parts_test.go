package restclient_test

import (
	"net/http"
	"testing"

	rc "github.com/bmcszk/go-restclient"
)

// oauth2Parts holds the oauth2 domain state of the fluent test DSL: the address of
// the canned OAuth2 token server started by anOAuth2TokenServerWithStatus.
type oauth2Parts struct {
	// tokenServerURL holds the address of the OAuth2 token server under test.
	tokenServerURL string
}

// newOauth2Parts returns the given/when/then fixture for the oauth2-domain fluent DSL.
func newOauth2Parts(t *testing.T) (given, when, then *testFixture) {
	t.Helper()

	fx := newFixture(t)
	return fx, fx, fx
}

// anOAuth2TokenServerWithStatus starts a canned OAuth2 token endpoint answering with the
// given status; 200 also returns a fixed access_token payload.
func (p *testFixture) anOAuth2TokenServerWithStatus(status int) *testFixture {
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
func (p *testFixture) aClientWithOAuth2Vars() *testFixture {
	p.aClient(rc.WithVars(map[string]any{
		"p_tokenEndpoint": p.tokenServerURL + "/token",
		"p_clientId":      "test-client",
		"p_clientSecret":  "test-secret",
	}))
	return p
}

// aClientWithOAuth2ProgrammaticVars builds a client with the given programmatic vars —
// for scenarios whose @-vars resolve through the normal lookup chain.
func (p *testFixture) aClientWithOAuth2ProgrammaticVars(vars map[string]any) *testFixture {
	p.aClient(rc.WithVars(vars))
	return p
}

// clientWithProgrammaticVars adds programmatic vars to the existing client (SetProgrammaticVars),
// mirroring the -D/--define CLI path.
func (p *testFixture) clientWithProgrammaticVars(vars map[string]any) *testFixture {
	p.client.SetProgrammaticVars(vars)

	return p
}

// aClientWithOAuth2PasswordVars builds a client whose `p`-prefixed OAuth2 password-grant
// variables point at the token server; useAuthorizationHeader=false keeps client creds in the body.
func (p *testFixture) aClientWithOAuth2PasswordVars() *testFixture {
	p.aClient(rc.WithVars(map[string]any{
		"p_tokenEndpoint":          p.tokenServerURL + "/token",
		"p_clientId":               "test-client",
		"p_clientSecret":           "test-secret",
		"p_username":               "test-user",
		"p_password":               "test-pass",
		"p_useAuthorizationHeader": "false",
	}))
	return p
}

// aClientWithOAuth2PasswordVarsAuthHeaderUnset omits useAuthorizationHeader so the
// file's @-var (or the true default) decides the Basic-header behavior.
func (p *testFixture) aClientWithOAuth2PasswordVarsAuthHeaderUnset() *testFixture {
	p.aClient(rc.WithVars(map[string]any{
		"p_tokenEndpoint": p.tokenServerURL + "/token",
		"p_clientId":      "test-client",
		"p_clientSecret":  "test-secret",
		"p_username":      "test-user",
		"p_password":      "test-pass",
	}))
	return p
}

// aClientWithOAuth2PasswordVarsMissingPassword omits the password to exercise the missing-variable error.
func (p *testFixture) aClientWithOAuth2PasswordVarsMissingPassword() *testFixture {
	p.aClient(rc.WithVars(map[string]any{
		"p_tokenEndpoint": p.tokenServerURL + "/token",
		"p_clientId":      "test-client",
		"p_clientSecret":  "test-secret",
		"p_username":      "test-user",
	}))
	return p
}

// aClientWithOAuth2VarsMissingSecret omits the client secret to exercise the missing-variable error.
func (p *testFixture) aClientWithOAuth2VarsMissingSecret() *testFixture {
	p.aClient(rc.WithVars(map[string]any{
		"p_tokenEndpoint": p.tokenServerURL + "/token",
		"p_clientId":      "test-client",
	}))
	return p
}
