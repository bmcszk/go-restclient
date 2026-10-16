---
# go-restclient-v5w21yzv
title: 'V5W2: domain builder return types *parts → *DParts, return p → return receiver (compiler-proven)'
status: todo
type: task
priority: normal
created_at: 2026-10-16T12:08:57Z
updated_at: 2026-10-16T12:09:21Z
parent: go-restclient-495b
---

USER MODEL: captureParts extends parts; capture methods return *captureParts (fluent type-safe). Mechanical: for every domain builder (receivers from V5W1): return type *parts → *DParts; `return p` → `return c` (the receiver var). Chain results never assigned/passed (brainer-verified) → compiler-proven safe. and() stays on base *parts.
## Gates
gofmt/build/vet/lint `0 issues.`/test ok; junit 315; no //nolint; ≤1000 ln
## COMMENT GATE: same as V5W1 (added comment = FAILED).
## Skills for coder
~/.agents/skills/go-dev, ~/.agents/skills/coding-rules — read §Comments/§File-structure first; binding.
