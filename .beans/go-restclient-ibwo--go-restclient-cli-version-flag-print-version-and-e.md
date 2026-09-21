---
# go-restclient-ibwo
title: 'go-restclient CLI --version flag: print version and exit'
status: completed
type: task
priority: normal
created_at: 2026-09-21T19:22:09Z
updated_at: 2026-09-21T20:24:10Z
---

## Requirement
Standard `--version` flag on the `restclient` CLI (PR #50 branch). kong supports it natively via a `Version string kong:"-"` field + `kong.Vars{"version": ...}` — use ldflags-injectable `var version = "dev"` so `go install` builds report `dev` and release builds can stamp real versions.

## Acceptance Criteria
- [x] `restclient --version` prints non-empty version string, exit 0
- [x] `restclient -V` (short) works — kong VersionFlag + short:\"V\" tag
- [x] Works without `-f` (kong VersionFlag.BeforeReset runs before required validation)
- [x] Unit tests asserting version output + exit path (TestCLI_VersionFlag, TestCLI_VersionShortFlag — real binary via buildBinary/runBinary)
- [x] `make check` green (golangci-lint 0, 281 tests 0 failures)

## Proof of work

Dispatch: `pi -p -ne -nc -a --no-session --model _minimax_first` (mandated flag set), completed ~4 min, 13 omniroute turns.
GREEN verified independently: 281 tests 0 failures, lint 0, build/vet clean.
Real-binary smoke: `--version`→"dev" exit 0; `-V`→"dev" exit 0; ldflags-stamped build prints `1.2.3-test`.
Residual: none.
