---
# go-restclient-v4w1643a
title: 'V4W1: state-cohesive capture_parts_test.go — captureState struct + all 55 capture methods into ONE file (receiver rule)'
status: completed
type: task
priority: normal
created_at: 2026-10-16T11:22:08Z
updated_at: 2026-10-16T11:40:36Z
parent: go-restclient-495b
---

FULL USER REQUIREMENTS (verbatim complaints, binding): struct and methods MUST be in same file; no state's methods scattered across files; withServerAddressVars (serverState) wrong in cookies pair; cookiesState methods wrong in parsefile pair; comment rules from coding-rules MUST be applied (comment audit gate). Moves-only, no test-body edits, no //nolint/lint-config, ≤1000 ln. PLACEMENT ORACLE: git show 39000b5:capture_parts_test.go — captureState struct + 55 methods (25 from parts_test.go, 3 control-pair, 2 core-pair, 2 graphql-pair, 1 refs-pair, 10 sysvars-pair, 8 vars-pair, 4 parsefile-pair). ONLY deviation: none in W1. Byte-identical method bodies, zero NEW comment lines.
## Gates
gofmt empty; build 0; vet 0; lint `0 issues.`; test -count=1 ok; 270 Test+2 Fuzz; grep '^func (\w+ \*captureState)' in exactly ONE file
## COMMENT GATE (binding): report MUST include per-touched-file comment diff (grep -c '//' before vs after); any added comment line = FAILED even if gates green. Zero narration comments (// X asserts/delegates/returns...).
## Skills for coder
~/.agents/skills/go-dev, ~/.agents/skills/coding-rules — READ §Comments AND §File-structure BEFORE first edit; binding.

## Proof of Work (orchestrator-verified)
- captureState struct + all 55 methods in capture_parts_test.go ONLY (693ln); drained from base + 7 pair files; byte-identical moves
- comment audit: pure deletions (688− lines, 0 insertions outside new file)
- gates re-run: gofmt empty, build/vet 0, lint `0 issues.`, tests ok, 270+2 parity
- Residual: W2 (parse/file/env)
