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
}

// ExpectedResponse defines what an actual response should be compared against.
// This might be loaded from a file (e.g., request_name.expected.json or .http).
// Or it could be defined programmatically.
type ExpectedResponse struct {
	StatusCode *int
	Status     *string
	Headers    http.Header // For header presence/value checks
	Body       *string     // Expected body content (exact match or regex)
}
