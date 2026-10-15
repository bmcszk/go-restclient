---
# go-restclient-v2w6yjc6
title: 'V2W6: separation audit — zero parts decls in plain *_test.go, zero Test funcs in *_parts_test.go, no-loss parity, PR #84 body'
status: todo
type: task
priority: normal
created_at: 2026-10-15T15:10:54Z
updated_at: 2026-10-15T15:11:28Z
parent: go-restclient-495b
---

Triage: brainer v2 plan. FULL requirement context: user demands PROVABLE separation. Orchestrator-run script audit (delegated claims = hypotheses).
## Scope
Script audit: (1) every *_parts_test.go contains zero `func Test` and zero `func Fuzz`; (2) every plain *_test.go contains zero methods on parts/state receivers (regex `^func \(\w+ \*?\w+\)` only allowed in _parts_test.go, export_test.go excepted); (3) no-loss parity vs master (298 set-diff, 6 documented twins); (4) all files ≤1000; (5) update PR #84 body with final layout + separation proof.
## Acceptance
- All 5 checks PASS with script output as PoW; PR #84 body updated
## Skills for coder
n/a (orchestrator-run)
