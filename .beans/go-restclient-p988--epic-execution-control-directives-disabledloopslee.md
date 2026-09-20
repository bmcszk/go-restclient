---
# go-restclient-p988
title: 'Epic: execution control directives (@disabled/@loop/@sleep)'
status: in-progress
type: feature
priority: high
tags:
    - epic
    - httpyac-parity
created_at: 2026-09-19T14:16:10Z
updated_at: 2026-09-20T13:47:24Z
---

## Goal

Execution-control directives for E2E scenarios: skip in-progress requests, loop/parameterize, wait.

## Triage / Context

- httpyac (docs fetched 2026-09-19, https://httpyac.github.io/guide/metaData.html): `@disabled` (+ conditional `@disabled !expr`), `@loop for N` / `@loop for item of data` / `@loop while expr` ($index injected, responses addressable as name0..nameN), `@sleep <ms>`, `@ratelimit`. We support none (source grep: no disabled/loop/sleep in parser_state.go).
- Value: parameterized flows (seed N users), skip-wip without editing files per environment.

## Acceptance Criteria

- [ ] `# @disabled` skips request; response marked skipped (distinct from error, exit code unaffected)
- [ ] `# @disabled !expr` conditional skip: expr = variable substitution, truthy -> skip
- [ ] `# @loop for N` and `# @loop for item of collectionVar`; `$index` + `item` injected; responses addressable as `name0..nameN`
- [ ] `# @sleep <ms>` waits before the request
- [ ] CLI `--all` and library honor all three; fluent tests + docs

## Non-goals (this epic)
- `@loop while <js-expr>` (needs script engine), `@ratelimit` (follow-up bean if wanted)
