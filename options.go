package restclient

import "net/http"

// ResolveOptions controls the behavior of variable substitution.
// If both FallbackToOriginal and FallbackToEmpty are false, and a variable is not found,
// an error or specific handling might occur (though current implementation defaults to empty string if not original).
type ResolveOptions struct {
	FallbackToOriginal bool // If true, an unresolved placeholder {{var}} becomes "{{var}}"
	FallbackToEmpty    bool // If true, an unresolved placeholder {{var}} becomes "" (empty string)
}

// ClientOption is a functional option for configuring the Client.
type ClientOption func(*Client) error

// WithHTTPClient allows providing a custom http.Client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) error {
		if hc == nil {
			c.httpClient = &http.Client{}
		} else {
			c.httpClient = hc
		}
		return nil
	}
}

// WithBaseURL sets a base URL for the client.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) error {
		c.BaseURL = baseURL
		return nil
	}
}

// WithDefaultHeader adds a default header to be sent with every request.
func WithDefaultHeader(key, value string) ClientOption {
	return func(c *Client) error {
		c.DefaultHeaders.Add(key, value)
		return nil
	}
}

// WithDefaultHeaders adds multiple default headers.
func WithDefaultHeaders(headers http.Header) ClientOption {
	return func(c *Client) error {
		for key, values := range headers {
			for _, value := range values {
				c.DefaultHeaders.Add(key, value)
			}
		}
		return nil
	}
}

// WithVars sets programmatic variables for the client instance.
// These variables can be used in .http and .hresp files.
// Programmatic variables have the highest precedence during substitution,
// overriding file-defined variables, environment variables, and .env variables.
// If called multiple times, the provided vars are merged with existing ones,
// with new values for existing keys overwriting old ones.
func WithVars(vars map[string]any) ClientOption {
	return func(c *Client) error {
		if c.programmaticVars == nil {
			c.programmaticVars = make(map[string]any)
		}
		for k, v := range vars {
			c.programmaticVars[k] = v
		}
		return nil
	}
}

// WithEnvironment sets the name of the environment to be used from http-client.env.json.
func WithEnvironment(name string) ClientOption {
	return func(c *Client) error {
		c.selectedEnvironmentName = name
		return nil
	}
}