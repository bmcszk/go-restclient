package restclient

import (
	"net/http"
	"time"
)

// Response captures the details of an HTTP response received from a server.
type Response struct {
	Request        *Request // The original request that led to this response
	Status         string   // e.g., "200 OK"
	StatusCode     int      // e.g., 200
	Proto          string   // e.g., "HTTP/1.1"
	Headers        http.Header
	Body           []byte        // Raw response body
	BodyString     string        // Response body as a string (convenience)
	Duration       time.Duration // Time taken for the request-response cycle
	Size           int64         // Response size in bytes (Content-Length or actual)
	IsTLS          bool          // True if the connection was over TLS
	TLSVersion     string        // e.g., "TLS 1.3" (if IsTLS is true)
	TLSCipherSuite string        // e.g., "TLS_AES_128_GCM_SHA256" (if IsTLS is true)
	Error          error         // Error encountered during request execution or response processing

	// TODO: Add fields for cookies, redirect history if needed.
	// TODO: Add fields for validation results if comparison is done here.
}

// ExpectedResponse defines what an actual response should be compared against.
// This might be loaded from a file (e.g., request_name.expected.json or .http).
// Or it could be defined programmatically.
type ExpectedResponse struct {
	StatusCode      *int        `json:"statusCode,omitempty" yaml:"statusCode,omitempty"`
	Status          *string     `json:"status,omitempty" yaml:"status,omitempty"`
	Headers         http.Header `json:"headers,omitempty" yaml:"headers,omitempty"`           // For header presence/value checks
	Body            *string     `json:"body,omitempty" yaml:"body,omitempty"`                 // Expected body content (exact match or regex)
	BodyContains    []string    `json:"bodyContains,omitempty" yaml:"bodyContains,omitempty"` // Substrings expected in body
	BodyNotContains []string    `json:"bodyNotContains,omitempty" yaml:"bodyNotContains,omitempty"`
	// JSONPathChecks  map[string]any    `json:"jsonPathChecks,omitempty" yaml:"jsonPathChecks,omitempty"` // e.g., {"$.user.id": 123, "$.user.name": "test"}
	// HeadersContain  map[string]string `json:"headersContain,omitempty" yaml:"headersContain,omitempty"` // Check if headers contain specific key-value (substring match for value)

	// TODO: Add fields for body schema validation (JSON Schema), timing assertions, etc.
}

// ValidationResult is no longer used, ValidateResponse returns []error directly.
/*
type ValidationResult struct {
	Passed      bool
	Mismatches  []string // Descriptions of what didn't match
	RawActual   *Response
	RawExpected *ExpectedResponse
}
*/
