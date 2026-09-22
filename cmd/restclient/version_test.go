package main

import (
	"runtime/debug"
	"strings"
	"testing"
)

func withVersion(t *testing.T, v string) {
	t.Helper()
	orig := version
	version = v
	t.Cleanup(func() { version = orig })
}

func TestResolveVersion_LdflagsWins(t *testing.T) {
	withVersion(t, "v9.9.9")
	if got := resolveVersion(); got != "v9.9.9" {
		t.Fatalf("resolveVersion() = %q, want ldflags value v9.9.9", got)
	}
}

func TestResolveVersion_BuildInfoRevision(t *testing.T) {
	withVersion(t, "dev")
	info, ok := debug.ReadBuildInfo()
	if !ok {
		t.Skip("no build info available")
	}
	rev := vcsInfo(info)
	if rev.revision == "" || len(rev.revision) < 7 {
		t.Skip("no usable vcs.revision in build info")
	}
	got := resolveVersion()
	want := "devel-" + rev.revision[:7] + rev.dirty
	if got != want {
		t.Fatalf("resolveVersion() = %q, want %q", got, want)
	}
}

func TestResolveVersion_NeverEmpty(t *testing.T) {
	if got := resolveVersion(); got == "" {
		t.Fatal("resolveVersion() returned empty string")
	}
}

func TestResolveVersion_DevelContainsDev(t *testing.T) {
	withVersion(t, "dev")
	got := resolveVersion()
	if !strings.Contains(got, "dev") {
		t.Fatalf("resolveVersion() = %q, want prefix containing dev", got)
	}
}
