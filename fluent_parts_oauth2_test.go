package restclient_test

import (
	"net/http"

	rc "github.com/bmcszk/go-restclient"
)

// anOAuth2TokenServerWithStatus starts a canned OAuth2 token endpoint answering with the
// given status; 200 also returns a fixed access_token payload.
func (p *parts) anOAuth2TokenServerWithStatus(status int) *parts {
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
func (p *parts) aClientWithOAuth2Vars() *parts {
	return p.aClient(rc.WithVars(map[string]any{
		"p_tokenEndpoint": p.tokenServerURL + "/token",
		"p_clientId":      "test-client",
		"p_clientSecret":  "test-secret",
	}))
}

// aClientWithOAuth2ProgrammaticVars builds a client with the given programmatic vars —
// for scenarios whose @-vars resolve through the normal lookup chain.
func (p *parts) aClientWithOAuth2ProgrammaticVars(vars map[string]any) *parts {
	return p.aClient(rc.WithVars(vars))
}

// clientWithProgrammaticVars adds programmatic vars to the existing client (SetProgrammaticVars),
// mirroring the -D/--define CLI path.
func (p *parts) clientWithProgrammaticVars(vars map[string]any) *parts {
	p.client.SetProgrammaticVars(vars)

	return p
}

// aClientWithOAuth2VarsMissingSecret omits the client secret to exercise the missing-variable error.
func (p *parts) aClientWithOAuth2VarsMissingSecret() *parts {
	return p.aClient(rc.WithVars(map[string]any{
		"p_tokenEndpoint": p.tokenServerURL + "/token",
		"p_clientId":      "test-client",
	}))
}
