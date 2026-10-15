---
# go-restclient-v2w5gx68
title: 'V2W5: merge client_script_test.go (4 tests) into client_parsefile_test.go; delete script file'
status: todo
type: task
priority: normal
created_at: 2026-10-15T15:10:54Z
updated_at: 2026-10-15T15:11:28Z
parent: go-restclient-495b
---

Triage: brainer v2 plan 2026-10-15. FULL requirement context: fewer files (REQ4), clear separation, moves-only, no test-body edits, no //nolint/lint-config, ≤1000 ln.
## Scope
Append client_script_test.go's 4 Test funcs verbatim into client_parsefile_test.go (same parse/execute-file domain), dedupe imports, DELETE client_script_test.go. Result ~606ln/27 tests.
## Acceptance
- client_script_test.go gone; parsefile 27 tests <1000; 298 parity; gates green
## Skills for coder
~/.agents/skills/go-dev, ~/.agents/skills/coding-rules
