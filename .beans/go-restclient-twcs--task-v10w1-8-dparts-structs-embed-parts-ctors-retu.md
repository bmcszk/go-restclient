---
# go-restclient-twcs
title: 'task V10W1: 8 DParts structs (embed parts) + ctors return *DParts; chains intact'
status: completed
type: task
priority: normal
created_at: 2026-10-16T19:48:18Z
updated_at: 2026-10-16T19:53:17Z
parent: go-restclient-j3z4
---

## Proof of Work V10W1 (pi-executed, orchestrator-gated)
- 8/8 domain files: `type <D>Parts struct { parts }` + ctor `(given, when, then *<D>Parts)` — struct extends base, ctor returns domain pointer ×3 (user law)
- builders remain on *parts returning *parts → mixed chains + .and() untouched, zero test-body edits (scratch dsl-v11 pattern)
- gates: gofmt EMPTY, vet 0, lint `0 issues.`, junit 315, parity EXACT, testFixture grep EMPTY
Residual: all state still lives on base parts (V8 flatten); DParts are stateless extensions by design.
