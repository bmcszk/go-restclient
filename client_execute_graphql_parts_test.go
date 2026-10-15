package restclient_test

import (
	"encoding/json"
	"net/http"
)

// graphqlResponse is the mocked server response body for GraphQL requests.
type graphqlResponse struct {
	Data   any            `json:"data,omitempty"`
	Errors []graphqlError `json:"errors,omitempty"`
}

// graphqlError mirrors the GraphQL error entry shape.
type graphqlError struct {
	Message   string `json:"message"`
	Locations []any  `json:"locations,omitempty"`
	Path      []any  `json:"path,omitempty"`
}

// aGraphQLServer serves the given responses in order, one per request, as HTTP 200 JSON.
func (s *serverState) aGraphQLServer(responses ...graphqlResponse) *parts {
	p := s.parts
	idx := 0
	return p.aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		i := idx
		if i >= len(responses) {
			i = len(responses) - 1
		}
		_ = json.NewEncoder(w).Encode(responses[i])
		idx++
	})
}

func (c *captureState) capturedBodyMatchesRegexp(index int, pattern string) *parts {
	p := c.parts
	p.require.Greater(len(p.capturedBodies), index)
	p.assert.Regexp(pattern, p.capturedBodies[index])

	return p
}

func (c *captureState) capturedJSONFieldContains(index int, fragment string, path ...string) *parts {
	p := c.parts
	leaf, ok := p.capturedJSONLeaf(index, path).(string)
	p.require.True(ok, "field %v is not a string", path)
	p.assert.Contains(leaf, fragment)

	return p
}
