package restclient_test

import (
	"net/http"

	rc "github.com/bmcszk/go-restclient"
)

// oauth2Parts holds the oauth2 domain state of the fluent test DSL: the address of
// the canned OAuth2 token server started by anOAuth2TokenServerWithStatus.
type oauth2Parts struct {
	*parts
	// tokenServerURL holds the address of the OAuth2 token server under test.
	tokenServerURL string
}

// anOAuth2TokenServerWithStatus starts a canned OAuth2 token endpoint answering with the
// given status; 200 also returns a fixed access_token payload.
func (s *oauth2Parts) anOAuth2TokenServerWithStatus(status int) *oauth2Parts {
	s.aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if status == http.StatusOK {
			_, _ = w.Write([]byte(`{"access_token":"test-access-token","token_type":"Bearer"}`))
		}
	})
	s.tokenServerURL = s.serverURL

	return s
}

// aClientWithOAuth2Vars builds a client whose `p`-prefixed OAuth2 variables point at the token server.
func (s *oauth2Parts) aClientWithOAuth2Vars() *oauth2Parts {
	s.aClient(rc.WithVars(map[string]any{
		"p_tokenEndpoint": s.tokenServerURL + "/token",
		"p_clientId":      "test-client",
		"p_clientSecret":  "test-secret",
	}))
	return s
}

// aClientWithOAuth2ProgrammaticVars builds a client with the given programmatic vars —
// for scenarios whose @-vars resolve through the normal lookup chain.
func (s *oauth2Parts) aClientWithOAuth2ProgrammaticVars(vars map[string]any) *oauth2Parts {
	s.aClient(rc.WithVars(vars))
	return s
}

// clientWithProgrammaticVars adds programmatic vars to the existing client (SetProgrammaticVars),
// mirroring the -D/--define CLI path.
func (s *oauth2Parts) clientWithProgrammaticVars(vars map[string]any) *oauth2Parts {
	s.client.SetProgrammaticVars(vars)

	return s
}

// aClientWithOAuth2PasswordVars builds a client whose `p`-prefixed OAuth2 password-grant
// variables point at the token server; useAuthorizationHeader=false keeps client creds in the body.
func (s *oauth2Parts) aClientWithOAuth2PasswordVars() *oauth2Parts {
	s.aClient(rc.WithVars(map[string]any{
		"p_tokenEndpoint":          s.tokenServerURL + "/token",
		"p_clientId":               "test-client",
		"p_clientSecret":           "test-secret",
		"p_username":               "test-user",
		"p_password":               "test-pass",
		"p_useAuthorizationHeader": "false",
	}))
	return s
}

// aClientWithOAuth2PasswordVarsAuthHeaderUnset omits useAuthorizationHeader so the
// file's @-var (or the true default) decides the Basic-header behavior.
func (s *oauth2Parts) aClientWithOAuth2PasswordVarsAuthHeaderUnset() *oauth2Parts {
	s.aClient(rc.WithVars(map[string]any{
		"p_tokenEndpoint": s.tokenServerURL + "/token",
		"p_clientId":      "test-client",
		"p_clientSecret":  "test-secret",
		"p_username":      "test-user",
		"p_password":      "test-pass",
	}))
	return s
}

// aClientWithOAuth2PasswordVarsMissingPassword omits the password to exercise the missing-variable error.
func (s *oauth2Parts) aClientWithOAuth2PasswordVarsMissingPassword() *oauth2Parts {
	s.aClient(rc.WithVars(map[string]any{
		"p_tokenEndpoint": s.tokenServerURL + "/token",
		"p_clientId":      "test-client",
		"p_clientSecret":  "test-secret",
		"p_username":      "test-user",
	}))
	return s
}

// aClientWithOAuth2VarsMissingSecret omits the client secret to exercise the missing-variable error.
func (s *oauth2Parts) aClientWithOAuth2VarsMissingSecret() *oauth2Parts {
	s.aClient(rc.WithVars(map[string]any{
		"p_tokenEndpoint": s.tokenServerURL + "/token",
		"p_clientId":      "test-client",
	}))
	return s
}
