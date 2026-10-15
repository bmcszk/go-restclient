---
# go-restclient-v2w2x3bd
title: 'V2W2: oauth2 parts extraction — oauth2State+8 builders out of client_oauth2_test.go into NEW oauth2_parts_test.go'
status: todo
type: task
priority: normal
created_at: 2026-10-15T15:10:54Z
updated_at: 2026-10-15T15:11:28Z
parent: go-restclient-495b
---

Triage: brainer v2 plan 2026-10-15. FULL requirement context: clear *_test.go (tests only) vs *_parts_test.go (parts only) separation; ≤1000 ln; moves-only; no test-body edits; no //nolint; no lint-config changes.
## Scope
Move oauth2State + 8 oauth2 builder methods (client_oauth2_test.go L14-114) into NEW oauth2_parts_test.go. 12 Test funcs stay (~293ln). Imports recomputed. oauth2State = ONLY oauth2 fields+methods (REQ2).
## Acceptance
- oauth2_parts_test.go NEW zero Test funcs; client_oauth2_test.go zero parts decls; 298 parity; gates green
## Skills for coder
~/.agents/skills/go-dev, ~/.agents/skills/coding-rules
