package restclient

import (
	"net/url"
	"testing"
)

// FuzzParseAndSanitizeBaseURL checks parseAndSanitizeBaseURL never panics and
// always returns a re-parseable absolute-or-opaque URL when it succeeds.
func FuzzParseAndSanitizeBaseURL(f *testing.F) {
	f.Add("https://example.com")
	f.Add("https://example.com/v1/api")
	f.Add("ht tp://bad url")
	f.Add("://missing-scheme")
	f.Add("ftp://files.example.com/base")
	f.Add("/relative/only")
	f.Add("")

	f.Fuzz(func(t *testing.T, base string) {
		got, err := parseAndSanitizeBaseURL(base)
		if err != nil {
			return
		}
		if got == nil {
			t.Fatalf("parseAndSanitizeBaseURL(%q) returned nil URL with nil error", base)
		}
		re, err := url.Parse(got.String())
		if err != nil {
			t.Fatalf("output not re-parseable: %v (input %q)", err, base)
		}
		if got.String() != re.String() {
			t.Fatalf("round-trip changed URL: %q -> %q", got.String(), re.String())
		}
	})
}

// FuzzResolveWithBaseURL checks resolveWithBaseURL never panics for any
// request-URL/base combination and always returns an absolute URL on success.
func FuzzResolveWithBaseURL(f *testing.F) {
	f.Add("https://example.com/api/users", "https://base.example.com/v1")
	f.Add("/api/users", "https://base.example.com/v1")
	f.Add("api/users", "https://base.example.com/v1")
	f.Add("https://other.example.com/x", "")
	f.Add("mailto:user@example.com", "https://base.example.com")
	f.Add("../up", "https://base.example.com/a/b")

	f.Fuzz(func(t *testing.T, requestURL, baseURL string) {
		fresh, err := url.Parse(requestURL)
		if err != nil {
			return
		}
		got, err := resolveWithBaseURL(fresh, baseURL)
		if err != nil {
			return
		}
		if got == nil {
			t.Fatalf("resolveWithBaseURL(%q, %q) returned nil URL with nil error", requestURL, baseURL)
		}
		if !got.IsAbs() && fresh.IsAbs() {
			t.Fatalf("absolute request made relative: %q", got.String())
		}
	})
}
