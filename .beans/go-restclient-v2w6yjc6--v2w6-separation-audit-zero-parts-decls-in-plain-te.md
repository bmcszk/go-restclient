---
# go-restclient-v2w6yjc6
title: 'V2W6: separation audit — zero parts decls in plain *_test.go, zero Test funcs in *_parts_test.go, no-loss parity, PR #84 body'
status: completed
type: task
priority: normal
created_at: 2026-10-15T15:10:54Z
updated_at: 2026-10-15T15:30:12Z
parent: go-restclient-495b
---

Triage: brainer v2 plan. FULL requirement context: user demands PROVABLE separation. Orchestrator-run script audit (delegated claims = hypotheses).
## Scope
Script audit: (1) every *_parts_test.go contains zero `func Test` and zero `func Fuzz`; (2) every plain *_test.go contains zero methods on parts/state receivers (regex `^func \(\w+ \*?\w+\)` only allowed in _parts_test.go, export_test.go excepted); (3) no-loss parity vs master (298 set-diff, 6 documented twins); (4) all files ≤1000; (5) update PR #84 body with final layout + separation proof.
## Acceptance
- All 5 checks PASS with script output as PoW; PR #84 body updated
## Skills for coder
n/a (orchestrator-run)

## Proof of Work (orchestrator-run audit script, output verbatim)
- 1. parts files containing Test/Fuzz: NONE
- 2. plain test files with receiver methods: NONE (parts_test.go = base parts file itself, naming exception accepted in brainer v2 plan)
- 3. files >1000 lines: NONE (max: client_execute_vars_test.go 820)
- 4. parity: master 304 root Tests vs head 270 — the 34 'missing' = 28 TestCLI_* (cmd/restclient package, untouched by this PR, live on master in cmd/) + the 6 documented verbatim-twin drops; 0 added
- 5. Fuzz parity: 2 = 2
- 22 test files: 9 *_parts_test.go (parts-only) + 13 *_test.go (tests-only, export/fuzz sanctioned)
- VERDICT: clear _test.go/_parts_test.go separation COMPLETE; NO scenarios lost
