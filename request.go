package restclient

import (
	"io"
	"net/http"
	"net/url"
)

// Request represents a parsed HTTP request from a .rest file.
type Request struct {
	Name        string // Optional name for the request (from ### Name comment)
	Method      string
	URL         *url.URL
	HTTPVersion string // e.g., "HTTP/1.1"
	Headers     http.Header
	Body        io.Reader // For streaming body content
	RawBody     string    // Store the raw body string for potential reuse/inspection

	// FilePath is the path to the .rest file this request was parsed from.
	// Useful for context or finding associated expected response files.
	FilePath string
	// LineNumber is the starting line number of this request in the file.
	LineNumber int

	// TODO: Add fields for variables, pre-request scripts/commands if supported later.
}

// ParsedFile represents all requests parsed from a single .rest file.
// This might be useful if a single file can contain multiple request blocks.
type ParsedFile struct {
	FilePath string
	Requests []*Request
	// TODO: Add global variables or settings defined at the file level if supported.
}
