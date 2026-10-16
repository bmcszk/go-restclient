---
# go-restclient-98p9
title: 'task V10W4 ponytail: parts only, one ctor'
status: completed
type: task
priority: normal
created_at: 2026-10-16T21:10:02Z
updated_at: 2026-10-16T21:10:07Z
parent: go-restclient-j3z4
---

## Proof of Work: ponytail simplification (V10W4, user: 'just parts')
- 9 DParts wrapper structs deleted; 9 duplicate ctors → ONE newParts (parts_test.go)
- 270 ctor sites retargeted to newParts; empty client_parts_test.go deleted; imports pruned
- net −~150 lines; every parts file = pure method group on *parts
- gates: gofmt EMPTY, vet 0, lint `0 issues.`, junit 315, parity EXACT
Residual: none.
