---
# go-restclient-p988
title: 'Epic: execution control directives (@disabled/@loop/@sleep)'
status: completed
type: feature
priority: high
tags:
    - epic
    - httpyac-parity
created_at: 2026-09-19T14:16:10Z
updated_at: 2026-09-20T18:02:25Z
---

## Goal

Execution-control directives for E2E scenarios: skip in-progress requests, loop/parameterize, wait.

## Triage / Context

- httpyac (docs fetched 2026-09-19, https://httpyac.github.io/guide/metaData.html): `@disabled` (+ conditional `@disabled !expr`), `@loop for N` / `@loop for item of data` / `@loop while expr` ($index injected, responses addressable as name0..nameN), `@sleep <ms>`, `@ratelimit`. We support none (source grep: no disabled/loop/sleep in parser_state.go).
- Value: parameterized flows (seed N users), skip-wip without editing files per environment.

## Acceptance Criteria

- [x] `# @disabled` skips request; response marked skipped (distinct from error, exit code unaffected)
- [x] `# @disabled !expr` conditional skip: expr = variable substitution, truthy -> skip
- [x] `# @loop for N` and `# @loop for item of collectionVar`; `$index` + `item` injected; responses addressable as `name0..nameN`
- [x] `# @sleep <ms>` waits before the request
- [x] CLI and library honor all directives (SKIP line, timing, iteration counts); 33 fluent tests + docs sections

## Non-goals (this epic)
- `@loop while <js-expr>` (needs script engine), `@ratelimit` (follow-up bean if wanted)

## Proof of Work (epic p988 closure)

All 4 children completed, each with its own TDD RED->GREEN round, commit-only-green:
- go-restclient-lyp1 @disabled: 7a38ab1 — Skipped response state, CLI SKIP line
- go-restclient-2qlf @disabled !expr (+ sub-bean kkp9 RED): cff8d85 — truthiness rules, undefined-var error
- go-restclient-eeft @sleep: cdcdc04 — pre-send pause, timing-assertion tests
- go-restclient-atek @loop: 4b9d069 — for N / {{var}} / item of coll; $index; name0..nameN addressing; no-op vs error semantics
- docs pass d3b8b08: docs/http_syntax.md "Execution Control Directives" (truthiness table, loop forms, nameN addressing, error messages) + README features.

End-to-end orchestrator verification on final tree (d3b8b08): 260 tests 0 failures (x3 consecutive runs), golangci-lint 0 issues, go build/vet clean, gofmt clean on all touched files.

Delegation notes: 2 pi false-success rounds caught by orchestrator verification (missing executor eval in 2qlf round 1; Skipped-response-for-zero-loop semantics bug in atek round 1) — both re-dispatched with exact fixes; git jail held (HEAD never moved again).

PR for review: https://github.com/bmcszk/go-restclient/pull/38
