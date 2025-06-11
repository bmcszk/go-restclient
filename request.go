package restclient

import (
	"io"
	"net/http"
	"net/url"
	"time"
)

// Script represents a JavaScript script, either inline or from an external file.
// It's used for pre-request and response handler scripts.
type Script struct {
	Path       string // Path to an external .js file, if applicable.
	Content    string // Inline script content, if applicable.
	IsExternal bool   // True if the script is from an external file.
}

// Request represents a parsed HTTP request from a .rest file.
type Request struct {
	// Name is an optional identifier for the request.
	// Parsed from "### Request Name" or "// @name Request Name" or "# @name Request Name".
	Name         string
	Method       string
	RawURLString string   // The raw URL string as read from the file, before variable substitution
	URL          *url.URL // Parsed URL, potentially after variable substitution
	HTTPVersion  string   // e.g., "HTTP/1.1"
	Headers      http.Header
	Body         io.Reader                     // For streaming body content after processing
	// Store the raw body string as read from the file, before variable substitution
	RawBody      string
	GetBody      func() (io.ReadCloser, error) // For http.Request.GetBody compatibility

	// ActiveVariables are variables resolved at the time of request execution,
	// sourced from environment, global scope (from previous scripts), and pre-request scripts.
	ActiveVariables map[string]string

	// PreRequestScript contains details of the JavaScript to be run before this request.
	PreRequestScript *Script
	// ResponseHandlerScript contains details of the JavaScript to be run after this request.
	ResponseHandlerScript *Script

	// FilePath is the absolute path to the .rest or .http file this request was parsed from.
	// Used for context, resolving relative paths for imports, script files, etc.
	FilePath string
	// LineNumber is the starting line number of this request definition in the source file.
	LineNumber int

	// Request Settings Directives (JetBrains compatibility)
	// NoRedirect indicates that this request should not follow redirects (from @no-redirect directive)
	NoRedirect bool
	// NoCookieJar indicates that this request should not use the cookie jar (from @no-cookie-jar directive)
	NoCookieJar bool
	// Timeout specifies a custom timeout for this request (from @timeout directive)
	Timeout time.Duration

	// External file body configuration
	// ExternalFilePath stores the path for external file body references (< ./path/to/file or <@ ./path/to/file)
	ExternalFilePath string
	// ExternalFileEncoding specifies the encoding for external file reading (e.g., "latin1", "utf-8")
	ExternalFileEncoding string
	// ExternalFileWithVariables indicates if the external file should have variable substitution applied (<@ syntax)
	ExternalFileWithVariables bool
}

// ParsedFile represents all content parsed from a single .rest or .http file.
// It holds multiple requests, environment context, and global variables accumulated during execution.
type ParsedFile struct {
	// FilePath is the absolute path to the parsed .rest or .http file.
	FilePath string
	// Requests is a list of all HTTP requests defined in the file.
	Requests []*Request
	// EnvironmentVariables are key-value pairs loaded from an associated environment file (e.g., http-client.env.json).
	// These are used as a base for variable substitution.
	EnvironmentVariables map[string]string
	// GlobalVariables are key-value pairs accumulated during the execution of
	// requests in this file (or imported files).
	// These are set by `client.global.set()` in response handler scripts and are available to subsequent requests.
	GlobalVariables map[string]string
	// FileVariables are key-value pairs defined directly within the .http file using the `@name = value` syntax.
	// Their scope is the current file, and they are resolved at parse time.
	FileVariables map[string]string
}
