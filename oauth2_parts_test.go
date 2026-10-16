package restclient_test

import (
	"net/http"
	"testing"

	rc "github.com/bmcszk/go-restclient"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type oauth2Parts struct {
	parts
}

func newOauth2Parts(t *testing.T) (given, when, then *oauth2Parts) {
	t.Helper()

	o := &oauth2Parts{parts: parts{
		trackedValues: make(map[string][]string),
		T:             t,
		require:       require.New(t),
		assert:        assert.New(t),
		baseDir:       t.TempDir(),
	}}

	return o, o, o
}

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

func (p *parts) aClientWithOAuth2Vars() *parts {
	p.aClient(rc.WithVars(map[string]any{
		"p_tokenEndpoint": p.tokenServerURL + "/token",
		"p_clientId":      "test-client",
		"p_clientSecret":  "test-secret",
	}))
	return p
}

func (p *parts) aClientWithOAuth2ProgrammaticVars(vars map[string]any) *parts {
	p.aClient(rc.WithVars(vars))
	return p
}

func (p *parts) clientWithProgrammaticVars(vars map[string]any) *parts {
	p.client.SetProgrammaticVars(vars)

	return p
}

// aClientWithOAuth2PasswordVars builds a client whose `p`-prefixed OAuth2 password-grant
// variables point at the token server; useAuthorizationHeader=false keeps client creds in the body.
func (p *parts) aClientWithOAuth2PasswordVars() *parts {
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
func (p *parts) aClientWithOAuth2PasswordVarsAuthHeaderUnset() *parts {
	p.aClient(rc.WithVars(map[string]any{
		"p_tokenEndpoint": p.tokenServerURL + "/token",
		"p_clientId":      "test-client",
		"p_clientSecret":  "test-secret",
		"p_username":      "test-user",
		"p_password":      "test-pass",
	}))
	return p
}

func (p *parts) aClientWithOAuth2PasswordVarsMissingPassword() *parts {
	p.aClient(rc.WithVars(map[string]any{
		"p_tokenEndpoint": p.tokenServerURL + "/token",
		"p_clientId":      "test-client",
		"p_clientSecret":  "test-secret",
		"p_username":      "test-user",
	}))
	return p
}

func (p *parts) aClientWithOAuth2VarsMissingSecret() *parts {
	p.aClient(rc.WithVars(map[string]any{
		"p_tokenEndpoint": p.tokenServerURL + "/token",
		"p_clientId":      "test-client",
	}))
	return p
}
