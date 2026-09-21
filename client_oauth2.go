package restclient

import (
	"fmt"
	"strings"
)

// oauth2Directive captures a parsed `Authorization: oauth2 [grant] <prefix>` header.
type oauth2Directive struct {
	grant  string
	prefix string
}

const oauth2GrantClientCredentials = "client_credentials"

// applyOAuth2 replaces the `Authorization: oauth2 ...` directive header with a Bearer
// token before the request is sent. RED stub: token fetching is not implemented yet.
func (*Client) applyOAuth2(req *Request) error {
	if req == nil || req.Headers == nil {
		return nil
	}
	if !oauth2HeaderMatches(req.Headers.Get("Authorization")) {
		return nil
	}
	directive, err := parseOAuth2Header(req.Headers.Get("Authorization"))
	if err != nil {
		return err
	}
	if directive.grant != "" && directive.grant != oauth2GrantClientCredentials {
		return fmt.Errorf("oauth2 %s: unsupported grant %q, only %q is supported",
			directive.prefix, directive.grant, oauth2GrantClientCredentials)
	}

	return fmt.Errorf("oauth2 %s: token fetch not implemented", directive.prefix)
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
