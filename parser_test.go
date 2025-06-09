package restclient_test

import (
	"log/slog"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slog.SetDefault(slog.New(h))
	os.Exit(m.Run())
}

// TODO: Refactor tests for request naming (@name directive), expected response parsing
// (separator comments, simple cases), and variable scoping.
// These tests previously used unexported parser functions (parseRequestFile, parseExpectedResponses).
// They need to be rewritten to use the public Client.ExecuteFile API,
// mock HTTP servers, and assertions on the returned Response or errors.
// Ensure coverage for:
// - FR1.3: Request Naming (# @name directive, ### Name, precedence, whitespace handling, mixed usage)
// - Expected response parsing:
//   - Separator comments affecting response block association.
//   - Basic structure: status line, headers, body.
//   - Different body types (JSON, text).
//   - Header parsing (single, multiple values).
// - FR2.4: Variable Scoping and Templating:
//   - Nested variable references.
//   - File-level vs. request-specific variable overrides.
//   - Restoration of file-level variables.
//   - Variable expansion in request bodies (JSON).
