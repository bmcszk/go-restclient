package restclient_test

import (
	"net/url"
	"testing"

	rc "github.com/bmcszk/go-restclient"
)

// fuzzBaseURLInvariants asserts parseAndSanitizeBaseURL output invariants.
func fuzzBaseURLInvariants(t *testing.T, base string) {
	t.Helper()

	got, err := rc.FuzzParseAndSanitizeBaseURLFn(base)
	if err != nil || got == nil {
		return
	}
	re, err := url.Parse(got.String())
	if err != nil {
		t.Fatalf("output not re-parseable: %v (input %q)", err, base)
	}
	if got.String() != re.String() {
		t.Fatalf("round-trip changed URL: %q -> %q", got.String(), re.String())
	}
}

// FuzzParseAndSanitizeBaseURL checks the base-URL parser never panics and its
// output is always re-parseable and round-trip stable.
func FuzzParseAndSanitizeBaseURL(f *testing.F) {
	f.Add("https://example.com")
	f.Add("https://example.com/v1/api")
	f.Add("ht tp://bad url")
	f.Add("://missing-scheme")
	f.Add("ftp://files.example.com/base")
	f.Add("/relative/only")
	f.Add("")

	f.Fuzz(func(t *testing.T, base string) {
		fuzzBaseURLInvariants(t, base)
	})
}

// fuzzResolveInvariants asserts resolveWithBaseURL output invariants.
func fuzzResolveInvariants(t *testing.T, fresh *url.URL, baseURL string) {
	t.Helper()

	got, err := rc.FuzzResolveWithBaseURLFn(fresh, baseURL)
	if err != nil || got == nil {
		return
	}
	if fresh.IsAbs() && !got.IsAbs() {
		t.Fatalf("absolute request made relative: %q -> %q", fresh.String(), got.String())
	}
}

// FuzzResolveWithBaseURL checks request/base URL resolution never panics and
// absolute request URLs are never made relative.
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
		fuzzResolveInvariants(t, fresh, baseURL)
	})
}
