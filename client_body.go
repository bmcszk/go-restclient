package restclient

// Variable-substitution chain driving single-request execution (URL, headers, body).

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// executeRequestWithVariables handles variable substitution and execution for a single request
func (c *Client) executeRequestWithVariables(
	ctx context.Context,
	restClientReq *Request,
	parsedFile *ParsedFile,
	osEnvGetter func(string) (string, bool),
	index int,
) (*Response, error) {
	requestScopedSystemVars := c.generateRequestScopedSystemVariables()

	// Substitute variables for URL and Headers
	err := c.substituteRequestURLAndHeaders(restClientReq, parsedFile, requestScopedSystemVars, osEnvGetter)
	if err != nil {
		return &Response{Request: restClientReq, Error: err}, fmt.Errorf(
			"variable substitution failed for request %s (index %d): %w",
			restClientReq.Name, index, err)
	}

	// Substitute variables for Body
	err = c.substituteRequestBody(restClientReq, parsedFile, requestScopedSystemVars, osEnvGetter)
	if err != nil {
		return &Response{Request: restClientReq, Error: err}, fmt.Errorf(
			"error processing body for request %s (index %d): %w",
			restClientReq.Name, index, err)
	}

	// Substituted request headers carry the oauth2 directive; it is replaced
	// here before the request is sent, leaving the resolved Bearer token as
	// the actual Authorization header.
	if err := c.applyOAuth2(restClientReq, parsedFile, osEnvGetter); err != nil {
		return &Response{Request: restClientReq, Error: err}, fmt.Errorf(
			"oauth2 token fetch failed for request %s (index %d): %w",
			restClientReq.Name, index, err)
	}

	// Execute the HTTP request
	resp, execErr := c.doHTTPRequest(ctx, restClientReq)
	if execErr != nil {
		return &Response{Request: restClientReq, Error: execErr}, nil
	}
	return resp, nil
}

// substituteRequestURLAndHeaders handles URL and header variable substitution
func (c *Client) substituteRequestURLAndHeaders(
	restClientReq *Request,
	parsedFile *ParsedFile,
	requestScopedSystemVars map[string]string,
	osEnvGetter func(string) (string, bool),
) error {
	finalParsedURL, subsErr := substituteRequestVariables(
		restClientReq,
		parsedFile,
		c.BaseURL,
		c.directiveResolveContext(parsedFile, restClientReq, osEnvGetter, requestScopedSystemVars, nil, -1),
	)
	if subsErr != nil {
		return subsErr
	}
	restClientReq.URL = finalParsedURL
	return nil
}

// substituteRequestBody handles body variable substitution including external files
func (c *Client) substituteRequestBody(
	restClientReq *Request,
	parsedFile *ParsedFile,
	requestScopedSystemVars map[string]string,
	osEnvGetter func(string) (string, bool),
) error {
	finalSubstitutedBody, err := c.resolveRequestBody(restClientReq, parsedFile, requestScopedSystemVars, osEnvGetter)
	if err != nil {
		return err
	}

	c.setRequestBody(restClientReq, finalSubstitutedBody)
	return nil
}

// resolveRequestBody handles the core body resolution logic
func (c *Client) resolveRequestBody(
	restClientReq *Request,
	parsedFile *ParsedFile,
	requestScopedSystemVars map[string]string,
	osEnvGetter func(string) (string, bool),
) (string, error) {
	if restClientReq.ExternalFilePath != "" {
		return c.processExternalFile(restClientReq, parsedFile, requestScopedSystemVars, osEnvGetter)
	}

	if restClientReq.RawBody == "" {
		return "", nil
	}

	if c.isMultipartFormWithFileReferences(restClientReq) {
		return c.processMultipartFormWithFiles(restClientReq, parsedFile, requestScopedSystemVars, osEnvGetter)
	}

	return c.processRegularBody(restClientReq, parsedFile, requestScopedSystemVars, osEnvGetter), nil
}

// processRegularBody handles regular body processing (non-multipart, non-external)
func (c *Client) processRegularBody(
	restClientReq *Request,
	parsedFile *ParsedFile,
	requestScopedSystemVars map[string]string,
	osEnvGetter func(string) (string, bool),
) string {
	loopAliases, loopIndex := currentLoopBindings(restClientReq)
	rctx := c.directiveResolveContext(parsedFile, restClientReq, osEnvGetter,
		requestScopedSystemVars, loopAliases, loopIndex)
	resolvedBody := resolveVariablesInText(restClientReq.RawBody, rctx)
	return substituteDynamicSystemVariables(resolvedBody, c.currentDotEnvVars, c.programmaticVars)
}

// setRequestBody sets the final body content on the request
func (*Client) setRequestBody(restClientReq *Request, finalSubstitutedBody string) {
	if finalSubstitutedBody != "" {
		restClientReq.RawBody = finalSubstitutedBody
		restClientReq.Body = strings.NewReader(finalSubstitutedBody)
		restClientReq.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(finalSubstitutedBody)), nil
		}
	} else {
		restClientReq.Body = nil
		restClientReq.GetBody = nil
	}
}
