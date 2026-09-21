package restclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// oauth2Directive captures a parsed `Authorization: oauth2 [grant] <prefix>` header.
type oauth2Directive struct {
	grant  string
	prefix string
}

const (
	oauth2GrantClientCredentials = "client_credentials"
	oauth2GrantPassword          = "password"
)

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
	grant, err := c.oauth2GrantFor(directive, parsedFile, restClientReq, osEnvGetter)
	if err != nil {
		return err
	}

	token, err := c.oauth2Token(grant, directive.prefix, parsedFile, restClientReq, osEnvGetter)
	if err != nil {
		return err
	}
	restClientReq.Headers.Set("Authorization", "Bearer "+token.accessToken)

	return nil
}

// oauth2GrantFor validates the directive grant, inferring it from the presence
// of `<prefix>_username` when the directive omits it.
func (c *Client) oauth2GrantFor(
	directive oauth2Directive,
	parsedFile *ParsedFile,
	restClientReq *Request,
	osEnvGetter func(string) (string, bool),
) (string, error) {
	switch directive.grant {
	case "":
		return c.oauth2DetectGrant(directive.prefix, parsedFile, restClientReq, osEnvGetter), nil
	case oauth2GrantClientCredentials, oauth2GrantPassword:
		return directive.grant, nil
	default:
		return "", fmt.Errorf("oauth2 %s: unsupported grant %q, only %q and %q are supported",
			directive.prefix, directive.grant, oauth2GrantClientCredentials, oauth2GrantPassword)
	}
}

// oauth2DetectGrant infers the grant when the directive omits it: password grant
// when `<prefix>_username` is present, client_credentials otherwise.
func (c *Client) oauth2DetectGrant(
	prefix string,
	parsedFile *ParsedFile,
	restClientReq *Request,
	osEnvGetter func(string) (string, bool),
) string {
	rctx := c.directiveResolveContext(parsedFile, restClientReq, osEnvGetter)
	if _, ok := lookupVar(prefix+"_username", rctx); ok {
		return oauth2GrantPassword
	}

	return oauth2GrantClientCredentials
}

// oauth2Token returns the cached token for prefix, fetching a new one on miss.
func (c *Client) oauth2Token(
	grant, prefix string,
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

	var token oauth2Token
	switch grant {
	case oauth2GrantPassword:
		token, err = c.fetchOAuth2PasswordToken(
			prefix, endpoint, clientID, clientSecret, parsedFile, restClientReq, osEnvGetter)
	default:
		token, err = c.fetchOAuth2Token(
			prefix, endpoint, oauth2Form(oauth2GrantClientCredentials, clientID, clientSecret), "", "")
	}
	if err != nil {
		return oauth2Token{}, err
	}
	c.oauth2Tokens[prefix] = token

	return token, nil
}

// fetchOAuth2PasswordToken resolves the user credentials for the password grant
// and POSTs them to the token endpoint.
func (c *Client) fetchOAuth2PasswordToken(
	prefix, endpoint, clientID, clientSecret string,
	parsedFile *ParsedFile,
	restClientReq *Request,
	osEnvGetter func(string) (string, bool),
) (oauth2Token, error) {
	username, err := c.oauth2Var(prefix, "username", parsedFile, restClientReq, osEnvGetter)
	if err != nil {
		return oauth2Token{}, err
	}
	password, err := c.oauth2Var(prefix, "password", parsedFile, restClientReq, osEnvGetter)
	if err != nil {
		return oauth2Token{}, err
	}
	useAuthHeader, err := c.oauth2UseAuthorizationHeader(prefix, parsedFile, restClientReq, osEnvGetter)
	if err != nil {
		return oauth2Token{}, err
	}

	var basicUser, basicPass string
	if useAuthHeader {
		basicUser, basicPass = clientID, clientSecret
	}

	form := url.Values{
		"grant_type": {oauth2GrantPassword},
		"username":   {username},
		"password":   {password},
	}
	if !useAuthHeader {
		form.Set("client_id", clientID)
		form.Set("client_secret", clientSecret)
	}

	return c.fetchOAuth2Token(prefix, endpoint, form.Encode(), basicUser, basicPass)
}

// oauth2UseAuthorizationHeader resolves `<prefix>_useAuthorizationHeader` via the
// raw lookup chain: "false" (case-insensitive) disables the Basic auth header;
// missing or "true" keeps it; anything else is an error.
func (c *Client) oauth2UseAuthorizationHeader(
	prefix string,
	parsedFile *ParsedFile,
	restClientReq *Request,
	osEnvGetter func(string) (string, bool),
) (bool, error) {
	rctx := c.directiveResolveContext(parsedFile, restClientReq, osEnvGetter)
	val, ok := lookupVar(prefix+"_useAuthorizationHeader", rctx)
	if !ok {
		return true, nil
	}
	s, ok := val.(string)
	if !ok {
		s = fmt.Sprintf("%v", val)
	}
	switch {
	case strings.EqualFold(s, "true"):
		return true, nil
	case strings.EqualFold(s, "false"):
		return false, nil
	default:
		return false, fmt.Errorf("oauth2 %s: invalid useAuthorizationHeader %q, must be \"true\" or \"false\"",
			prefix, s)
	}
}

// fetchOAuth2Token POSTs the given token request body; when basicUser/basicPass
// are set they are sent as a Basic authorization header instead of body fields.
func (c *Client) fetchOAuth2Token(prefix, endpoint, body, basicUser, basicPass string) (oauth2Token, error) {
	httpReq, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(body))
	if err != nil {
		return oauth2Token{}, fmt.Errorf("oauth2 %s: building token request: %w", prefix, err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if basicUser != "" || basicPass != "" {
		httpReq.SetBasicAuth(basicUser, basicPass)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return oauth2Token{}, fmt.Errorf("oauth2 %s: token request failed: %w", prefix, err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	return c.parseOAuth2TokenResponse(prefix, httpResp)
}

// parseOAuth2TokenResponse validates the token endpoint response and extracts the access token.
func (*Client) parseOAuth2TokenResponse(prefix string, httpResp *http.Response) (oauth2Token, error) {
	bodyBytes, err := io.ReadAll(io.LimitReader(httpResp.Body, 4096))
	if err != nil {
		return oauth2Token{}, fmt.Errorf("oauth2 %s: reading token response: %w", prefix, err)
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode > 299 {
		return oauth2Token{}, fmt.Errorf("oauth2 %s: token endpoint returned %d: %s",
			prefix, httpResp.StatusCode, string(bodyBytes))
	}

	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
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

// oauth2Var resolves `<prefix>_<suffix>` via the standard variable lookup chain,
// then expands any nested `{{...}}` placeholders inside the value.
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

	expanded, err := expandPlaceholders(s, rctx)
	if err != nil {
		return "", fmt.Errorf("oauth2 %s: variable %s: %w", prefix, name, err)
	}

	return expanded, nil
}

// oauth2PlaceholderRe matches a single `{{...}}` placeholder token,
// using the same brace style as the variable substitution in variables.go.
var oauth2PlaceholderRe = regexp.MustCompile(`{{\s*(.*?)\s*}}`)

// oauth2MaxExpansionDepth bounds recursive placeholder expansion hops.
const oauth2MaxExpansionDepth = 5

// expandPlaceholders expands `{{...}}` placeholders in value using the variable
// lookup chain. Unresolvable placeholders are left as-is.
func expandPlaceholders(value string, rctx resolveContext) (string, error) {
	pieces := make([]string, 0, 4)
	last := 0
	for _, loc := range oauth2PlaceholderRe.FindAllStringIndex(value, -1) {
		pieces = append(pieces, value[last:loc[0]])
		expanded, err := expandPlaceholderToken(value[loc[0]+2:loc[1]-2], rctx, 0)
		if err != nil {
			return "", err
		}
		pieces = append(pieces, expanded)
		last = loc[1]
	}
	pieces = append(pieces, value[last:])

	return strings.Join(pieces, ""), nil
}

// expandPlaceholderToken resolves one `{{...}}` token body, recursing when the
// resolved variable value itself contains placeholders (depth-bounded).
func expandPlaceholderToken(directive string, rctx resolveContext, depth int) (string, error) {
	directive = strings.TrimSpace(directive)
	if depth >= oauth2MaxExpansionDepth {
		return "", errors.New("oauth2: placeholder expansion too deep")
	}
	if v, ok := expandNamedSource(directive, rctx); ok {
		return v, nil
	}
	return expandVarToken(directive, rctx)
}

// expandNamedSource resolves `$processEnv X` / `$dotenv X` tokens; ok=false when
// the token is not a named-source directive.
func expandNamedSource(directive string, rctx resolveContext) (string, bool) {
	if name, ok := strings.CutPrefix(directive, "$processEnv "); ok {
		if v, ok := rctx.osEnvGetter(strings.TrimSpace(name)); ok {
			return v, true
		}

		return "{{" + directive + "}}", true
	}
	if name, ok := strings.CutPrefix(directive, "$dotenv "); ok {
		if v, ok := rctx.dotEnvVars[strings.TrimSpace(name)]; ok {
			return v, true
		}

		return "{{" + directive + "}}", true
	}

	return "", false
}

// expandVarToken resolves a plain variable token via the lookup chain.
func expandVarToken(directive string, rctx resolveContext) (string, error) {
	resolved, ok := lookupVar(directive, rctx)
	if !ok {
		return "{{" + directive + "}}", nil
	}
	s, ok := resolved.(string)
	if !ok {
		s = fmt.Sprintf("%v", resolved)
	}
	out, err := expandPlaceholders(s, rctx)
	if err != nil {
		return "", err
	}

	return out, nil
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
