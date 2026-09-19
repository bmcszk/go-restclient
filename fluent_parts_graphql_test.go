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
func (p *parts) aGraphQLServer(responses ...graphqlResponse) *parts {
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
