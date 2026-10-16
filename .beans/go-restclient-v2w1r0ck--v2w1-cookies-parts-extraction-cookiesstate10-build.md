---
# go-restclient-v2w1r0ck
title: 'V2W1: cookies parts extraction — cookiesState+10 builders out of client_cookies_redirects_test.go into NEW cookies_parts_test.go'
status: completed
type: task
priority: normal
created_at: 2026-10-15T15:10:54Z
updated_at: 2026-10-15T15:15:53Z
parent: go-restclient-495b
---

Triage: brainer v2 plan 2026-10-15 (fresh tree census + gates re-run). Requirement context (FULL): user demands clear three-tier separation — *_test.go = tests ONLY, *_parts_test.go = parts/builders ONLY, ≤1000 lines/file, moves-only, zero test-body edits, no //nolint, no lint-config changes.
## Scope
Move cookiesState type + var _ interface assertion + 10 cookie builder methods (client_cookies_redirects_test.go L91-230) into NEW cookies_parts_test.go (package restclient_test). Strip imports no longer used by the scenario file. 4 Test funcs stay in client_cookies_redirects_test.go (~90ln result). cookiesState = ONLY cookie-related fields+methods (REQ2).
## Acceptance
- cookies_parts_test.go NEW, zero Test funcs; client_cookies_redirects_test.go has zero parts decls
- 298 Test parity (270 root + 2 Fuzz + 26 cmd), gates: gofmt/build/vet/lint 0 issues/test -count=1
## Skills for coder
~/.agents/skills/go-dev, ~/.agents/skills/coding-rules (read fully before editing)

## Proof of Work (orchestrator-verified)
- cookiesState + 10 builders moved to NEW cookies_parts_test.go (156ln, 0 Test funcs); client_cookies_redirects_test.go now 4 tests only (0 parts decls)
- separation check: parts file Test=0 methods=10; scenario file Test=4 methods=0
- gates re-run by orchestrator: gofmt empty, build/vet 0, lint `0 issues.`, tests both pkgs ok, inventory 270+2Fuzz
- Residual: none.
