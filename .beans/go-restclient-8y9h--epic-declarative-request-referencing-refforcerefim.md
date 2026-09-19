---
# go-restclient-8y9h
title: 'Epic: declarative request referencing (@ref/@forceRef/@import)'
status: todo
type: feature
priority: high
tags:
    - epic
    - httpyac-parity
created_at: 2026-09-19T14:14:52Z
updated_at: 2026-09-19T17:45:40Z
---

## Goal

Declarative request referencing so `.http` files express prerequisite chains (login -> use token) without CLI flags or Go code.

## Triage / Context

- Comparison vs httpyac done 2026-09-19 (docs fetched from httpyac.github.io, repo source grepped): httpyac supports `# @ref name` (run once, reuse cached response), `# @forceRef name` (always re-run), `# @import ./file.http` (cross-file refs + file-global vars). Docs: https://httpyac.github.io/guide/metaData.html
- We have only CLI `-A/--after` (one manual prerequisite) and named-response vars `{{name.response.body.x}}` (response_refs.go). No parser support for @ref/@forceRef/@import.
- Highest-value gap: auth chaining is THE core E2E scenario and is currently manual.

## Acceptance Criteria

- [ ] `# @ref name` and `# @forceRef name` parsed (multiple refs per request, before or after request line)
- [ ] Executor runs referenced request before referencing one: @ref reuses cached response, @forceRef always re-executes
- [ ] Referenced responses addressable via existing `{{name.response.*}}` syntax
- [ ] Works in library (`ExecuteFile`) and CLI (`-n` triggers its ref chain, `--all`)
- [ ] Docs updated (`docs/http_syntax.md`, README)

## Progress

- [x] go-restclient-n6cb parser+executor (PR #37, commit f372bd3)


- [x] go-restclient-maez executor criteria pinned (commit 2fc0a01)
