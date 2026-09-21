package restclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// oauth2Directive captures a parsed `Authorization: oauth2 [grant] <prefix>` header.
type oauth2Directive struct {
	grant  string
	prefix string
}

const oauth2GrantClientCredentials = "client_credentials"

// applyOAuth2 replaces the `Authorization: oauth2 ...` directive header with a Bearer
// token before the request is sent, fetching (and caching) the token if needed.
func (c *Client) applyOAuth2(
	restClientReq *Request,
	parsedFile *ParsedFile,
	osEnvGetter func(string) (string, bool),
) error {
	if restClientReq == nil || restClientReq.Headers == nil {
		return nil
	}
	if !oauth2HeaderMatches(restClientReq.Headers.Get("Authorization")) {
		return nil
	}
	directive, err := parseOAuth2Header(restClientReq.Headers.Get("Authorization"))
	if err != nil {
		return err
	}
	if directive.grant != "" && directive.grant != oauth2GrantClientCredentials {
		return fmt.Errorf("oauth2 %s: unsupported grant %q, only %q is supported",
			directive.prefix, directive.grant, oauth2GrantClientCredentials)
	}

	token, err := c.oauth2Token(directive.prefix, parsedFile, restClientReq, osEnvGetter)
	if err != nil {
		return err
	}
	restClientReq.Headers.Set("Authorization", "Bearer "+token.accessToken)

	return nil
}

// oauth2Token returns the cached token for prefix, fetching a new one on miss.
func (c *Client) oauth2Token(
	prefix string,
	parsedFile *ParsedFile,
	restClientReq *Request,
	osEnvGetter func(string) (string, bool),
) (oauth2Token, error) {
	c.oauth2Mu.Lock()
	defer c.oauth2Mu.Unlock()

	if c.oauth2Tokens == nil {
		c.oauth2Tokens = make(map[string]oauth2Token)
	}
	if t, ok := c.oauth2Tokens[prefix]; ok {
		return t, nil
	}

	endpoint, err := c.oauth2Var(prefix, "tokenEndpoint", parsedFile, restClientReq, osEnvGetter)
	if err != nil {
		return oauth2Token{}, err
	}
	clientID, clientSecret, err := c.oauth2Credentials(prefix, parsedFile, restClientReq, osEnvGetter)
	if err != nil {
		return oauth2Token{}, err
	}

	token, err := c.fetchOAuth2Token(prefix, endpoint, clientID, clientSecret)
	if err != nil {
		return oauth2Token{}, err
	}
	c.oauth2Tokens[prefix] = token

	return token, nil
}

// fetchOAuth2Token POSTs the client_credentials grant to the token endpoint.
func (c *Client) fetchOAuth2Token(prefix, endpoint, clientID, clientSecret string) (oauth2Token, error) {
	httpReq, err := http.NewRequest(
		http.MethodPost, endpoint,
		strings.NewReader(oauth2Form(oauth2GrantClientCredentials, clientID, clientSecret)))
	if err != nil {
		return oauth2Token{}, fmt.Errorf("oauth2 %s: building token request: %w", prefix, err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return oauth2Token{}, fmt.Errorf("oauth2 %s: token request failed: %w", prefix, err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(httpResp.Body, 4096))
	if err != nil {
		return oauth2Token{}, fmt.Errorf("oauth2 %s: reading token response: %w", prefix, err)
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode > 299 {
		return oauth2Token{}, fmt.Errorf("oauth2 %s: token endpoint returned %d: %s",
			prefix, httpResp.StatusCode, string(body))
	}

	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return oauth2Token{}, fmt.Errorf("oauth2 %s: parsing token response: %w", prefix, err)
	}
	if payload.AccessToken == "" {
		return oauth2Token{}, fmt.Errorf("oauth2 %s: token response has no access_token", prefix)
	}

	return oauth2Token{accessToken: payload.AccessToken}, nil
}

// oauth2Credentials resolves clientId/clientSecret for prefix via the lookup chain.
func (c *Client) oauth2Credentials(
	prefix string,
	parsedFile *ParsedFile,
	restClientReq *Request,
	osEnvGetter func(string) (string, bool),
) (clientID, clientSecret string, err error) {
	clientID, err = c.oauth2Var(prefix, "clientId", parsedFile, restClientReq, osEnvGetter)
	if err != nil {
		return "", "", err
	}
	clientSecret, err = c.oauth2Var(prefix, "clientSecret", parsedFile, restClientReq, osEnvGetter)
	if err != nil {
		return "", "", err
	}

	return clientID, clientSecret, nil
}

// oauth2Form encodes the client_credentials token request body.
func oauth2Form(grant, clientID, clientSecret string) string {
	return url.Values{
		"grant_type":    {grant},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	}.Encode()
}

// oauth2Var resolves `<prefix>_<suffix>` via the standard variable lookup chain.
func (c *Client) oauth2Var(
	prefix string,
	suffix string,
	parsedFile *ParsedFile,
	restClientReq *Request,
	osEnvGetter func(string) (string, bool),
) (string, error) {
	rctx := c.directiveResolveContext(parsedFile, restClientReq, osEnvGetter)
	name := prefix + "_" + suffix
	val, ok := lookupVar(name, rctx)
	if !ok {
		return "", fmt.Errorf("oauth2 %s: missing variable %s", prefix, name)
	}
	s, ok := val.(string)
	if !ok || s == "" {
		return "", fmt.Errorf("oauth2 %s: variable %s is not a non-empty string", prefix, name)
	}

	return s, nil
}

// oauth2Token is a cached OAuth2 access token, keyed by directive prefix.
type oauth2Token struct {
	accessToken string
}

// oauth2HeaderMatches reports whether the Authorization header value is an oauth2 directive.
func oauth2HeaderMatches(value string) bool {
	parts := strings.Fields(value)

	return len(parts) >= 2 && parts[0] == "oauth2"
}

// parseOAuth2Header parses `oauth2 [grant] <prefix>` into grant (optional) and prefix.
func parseOAuth2Header(value string) (oauth2Directive, error) {
	parts := strings.Fields(value)
	if len(parts) < 2 || parts[0] != "oauth2" {
		return oauth2Directive{}, fmt.Errorf("invalid oauth2 authorization header %q", value)
	}
	if len(parts) == 2 {
		return oauth2Directive{prefix: parts[1]}, nil
	}

	return oauth2Directive{grant: parts[1], prefix: parts[2]}, nil
}
