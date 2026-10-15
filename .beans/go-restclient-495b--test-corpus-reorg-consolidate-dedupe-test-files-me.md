---
# go-restclient-495b
title: ""
status: in-progress
type: epic
priority: normal
created_at: 2026-10-15T10:18:58Z
updated_at: 2026-10-15T14:51:27Z
---

# TEST-CORPUS-REORG: consolidate + dedupe test files, meaningful parts DSL split

## Goal
User-directed test reorg (2026-10-15): fewer files, all < 1000 lines, no duplicate scenarios, and a meaningful parts-DSL split with a base struct + per-domain extensions (`<domain>_parts_test.go` suffix convention).

## Acceptance criteria
- [x] duplicate Test funcs removed (hash-verified verbatim twins) — 6 dropped, 276 → 270
- [x] single-func domain files merged (inplace_vars, consistency pairs, validator options+status) — 45 → 40 files, net −147 lines
- [x] every test file < 1000 lines (max 807)
- [x] gates green: gofmt/vet/build/lint 0/test -count=1 — orchestrator-verified + CI 10/10
- [ ] parts-DSL structural split: base parts struct (lifecycle + response asserts only), domain structs embedded anonymously, builders return root *parts so `.and()` chains stay byte-identical
- [ ] parts files renamed to `<domain>_parts_test.go` suffix (core/file/env/server/parse/capture/validator/oauth2/cookies); fluent_parts_ext_test.go (751 ln) decomposed by domain
- [ ] docs/fluent-testing-guideline.md + repo skill DSL layout table updated in the same PR as the split

## Status
- PR #83 (dedup + consolidation): CI 10/10 green, MERGEABLE, awaiting user merge — https://github.com/bmcszk/go-restclient/pull/83
- parts-DSL structural split: NEXT (pi contract, pilot = capture domain proof-of-shape)

## Proof of Work
- dedup verified by normalized-body hash across all *_test.go: 5 groups / 6 drops (listed in PR #83 body)
- orchestrator re-ran all gates post-pi: gofmt empty, build OK, vet OK, golangci-lint `0 issues.`, full test suite green, Test count 270, max file 807 ln
- CI on PR #83: all 9 checks pass + Scorecard skip-as-designed; mergeable CLEAN
- Residual: parts-DSL structural split open (criteria 5-7)


## Proof of Work (final audit 2026-10-15, orchestrator-run)
- 45 → 21 test files; ZERO fluent_* files; max file 839ln (< 1000); 270 Test funcs (6 documented dedup drops only)
- 15 waves executed (S0-S8 + S6a/S6b split + C1-C7), every wave: pi contract (-t +codemode + scout + go-dev/coding-rules skills) → orchestrator re-gates (gofmt/build/vet/lint 0/test -count=1/inventory) → commit Refs: bean-id → push → CI watched terminal
- CI final: 9 pass + Scorecard skip; MERGEABLE CLEAN on PR #84
- DONE criteria met; epic closes on user merge of https://github.com/bmcszk/go-restclient/pull/84


## No-Loss Audit (2026-10-15, brainer + orchestrator script verification)
- Test-name set diff master vs HEAD: 304 → 298; missing = EXACTLY the 6 documented dedup drops; 0 new, 0 changed bodies (270/270 survivors byte-identical after whitespace-normalize, verified programmatically)
- Each of the 6 dropped tests proven verbatim twin: body-sans-name identical to its surviving pair (listed in wave beans)
- Subtests t.Run: 0 on master, 0 on HEAD (no subtest layer exists) — no scenario granularity lost
- Helpers/types: 0 lost; +7 = domainState structs (capture/cookies/env/file/oauth2/parse/server) = the split itself
- cmd/restclient: untouched. Non-test diff: .beans ledger + AGENTS.md + fluent-testing-guideline.md only
- Net test-file delta +4341/−4314 lines (boilerplate/import-merge arithmetic only)
- VERDICT: NO TEST SCENARIOS LOST
