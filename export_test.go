package restclient

// Exposes internal URL-resolution functions for external-package fuzz tests.
var (
	FuzzParseAndSanitizeBaseURLFn = parseAndSanitizeBaseURL
	FuzzResolveWithBaseURLFn      = resolveWithBaseURL
)
