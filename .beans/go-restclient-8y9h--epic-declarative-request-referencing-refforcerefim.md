---
# go-restclient-8y9h
title: 'Epic: declarative request referencing (@ref/@forceRef/@import)'
status: completed
type: feature
priority: high
tags:
    - epic
    - httpyac-parity
created_at: 2026-09-19T14:14:52Z
updated_at: 2026-09-19T18:45:14Z
---

## Goal

Declarative request referencing so `.http` files express prerequisite chains (login -> use token) without CLI flags or Go code.

## Triage / Context

- Comparison vs httpyac done 2026-09-19 (docs fetched from httpyac.github.io, repo source grepped): httpyac supports `# @ref name` (run once, reuse cached response), `# @forceRef name` (always re-run), `# @import ./file.http` (cross-file refs + file-global vars). Docs: https://httpyac.github.io/guide/metaData.html
- We have only CLI `-A/--after` (one manual prerequisite) and named-response vars `{{name.response.body.x}}` (response_refs.go). No parser support for @ref/@forceRef/@import.
- Highest-value gap: auth chaining is THE core E2E scenario and is currently manual.

## Acceptance Criteria

- [x] `# @ref name` and `# @forceRef name` parsed (multiple refs per request, before or after request line)
- [x] Executor runs referenced request before referencing one: @ref reuses cached response, @forceRef always re-executes
- [x] Referenced responses addressable via existing `{{name.response.*}}` syntax
- [x] Works in library (`ExecuteFile`) and CLI (`-n` triggers its ref chain, `--all`)
- [x] Docs updated (`docs/http_syntax.md`, README)

## Progress

- [x] go-restclient-n6cb parser+executor (PR #37, commit f372bd3)


- [x] go-restclient-maez executor criteria pinned (commit 2fc0a01)


- [x] go-restclient-68op CLI ref chain (commit 644985f)


- [x] go-restclient-qe7j @import cross-file refs+vars (commit b4d8561)

## Proof of Work (epic closure)

All 4 children completed: go-restclient-n6cb (parser+executor, f372bd3), go-restclient-maez (chain semantics pinned, 2fc0a01), go-restclient-68op (CLI -n/-i ref chain, 644985f), go-restclient-qe7j (@import cross-file, b4d8561).
End-to-end verification by orchestrator on final tree:
- gotestsum -count=1 ./... : DONE 223 tests, 0 failures (full suite, both packages)
- golangci-lint run ./... : 0 issues; go build/vet clean; gofmt clean on touched files
- E2E user-flow coverage: parse -> ref chain (cache/force/transitive/cycle/unknown) -> response placeholders out-of-file-order -> @import cross-file vars+refs -> CLI -n/-i/-A composition; 12 dedicated tests in client_execute_refs_test.go + 2 CLI tests in cmd/restclient/main_test.go
- Docs: docs/http_syntax.md "Request Referencing" section + README Key Features bullet (commit 556d9df, placeholders verified byte-exact via sha256)
- PR for review: https://github.com/bmcszk/go-restclient/pull/37 — OPEN, CI green (Go 1.21-1.24) at final commit 556d9df
Residual: none for the epic scope.


PR for review: https://github.com/bmcszk/go-restclient/pull/37 (OPEN, CI green, feature/ref-force-ref -> master).
