---
# go-restclient-v3w1tox6
title: 'V3W1: anonymous embedding — *parts first line in all state structs, drop root field+wiring, rewrite ~135 .root refs'
status: completed
type: task
priority: normal
created_at: 2026-10-15T15:55:29Z
updated_at: 2026-10-15T16:06:50Z
parent: go-restclient-495b
---

FULL REQUIREMENTS (user verbatim, binding): 1. 3 files where possible: impl/impl_test.go/impl_parts_test.go 2. else pair X_test.go+X_parts_test.go 3. X_parts_test.go ONLY structs+methods used by X_test.go 4. embedding: `*parts` first line, NO named `root *parts` field. Mechanical receiver rewrites allowed; NO test-body semantic edits; no //nolint; no lint-config changes; ≤1000 ln/file.
## Scope (brainer v3 plan)
All 8 state structs (capture/parse/file/env/server/oauth2/cookies states in their *_parts_test.go + any in parts_test.go): replace `root *parts` field with anonymous `*parts` FIRST LINE; delete all `p.<State>.root = p` wiring in newParts; rewrite method bodies' `x.root` → promoted `x.parts` (mechanical sed-able, ~135 refs — verify count by grep before/after); builders keep returning *parts via promotion. If any same-name method collision appears, rename minimally and report.
## Gates
gofmt empty; build 0; vet 0; lint `0 issues.`; go test ./... -count=1 ok; 270 Test + 2 Fuzz parity; grep 'root \*parts' → 0; grep '\.root =' → 0
## Skills for coder
~/.agents/skills/go-dev, ~/.agents/skills/coding-rules

## Proof of Work (orchestrator-verified)
- all 7 state structs: `*parts` anonymous FIRST line (capture/cookies/env/oauth2/parse/server/file + base structs in parts_test.go); `root *parts` field deleted; all newParts wiring deleted; 0 `.root` refs remain
- net −16 lines; gates re-run: gofmt empty, build/vet 0, lint `0 issues.`, tests ok, 270+2 parity
- Residual: none.
