---
# go-restclient-v5w21yzv
title: 'V5W2: domain builder return types *parts → *DParts, return p → return receiver (compiler-proven)'
status: completed
type: task
priority: normal
created_at: 2026-10-16T12:08:57Z
updated_at: 2026-10-16T12:40:08Z
parent: go-restclient-495b
---

USER MODEL: captureParts extends parts; capture methods return *captureParts (fluent type-safe). Mechanical: for every domain builder (receivers from V5W1): return type *parts → *DParts; `return p` → `return c` (the receiver var). Chain results never assigned/passed (brainer-verified) → compiler-proven safe. and() stays on base *parts.
## Gates
gofmt/build/vet/lint `0 issues.`/test ok; junit 315; no //nolint; ≤1000 ln
## COMMENT GATE: same as V5W1 (added comment = FAILED).
## Skills for coder
~/.agents/skills/go-dev, ~/.agents/skills/coding-rules — read §Comments/§File-structure first; binding.

## Proof of Work (orchestrator-verified)
- all 111 domain builders now return their DParts type (0 *parts returns on domain receivers remain); requestRawBodyIs → *captureParts (user's example fixed)
- pi process hit wall-clock timeout post-edit (EXIT:124) but work complete: gates re-run by orchestrator — gofmt empty, build/vet 0, lint `0 issues.`, junit parity 315
- Residual: V5W3 constructors
