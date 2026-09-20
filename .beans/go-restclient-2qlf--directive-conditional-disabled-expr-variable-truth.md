---
# go-restclient-2qlf
title: 'Directive: conditional @disabled !expr (variable truthiness)'
status: completed
type: task
priority: normal
created_at: 2026-09-19T14:16:10Z
updated_at: 2026-09-20T16:50:30Z
parent: go-restclient-p988
blocked_by:
    - go-restclient-lyp1
---

Parent epic: execution control directives (@disabled/@loop/@sleep). Blocked by @disabled static skip.

## Requirement
`# @disabled !expr`: evaluate expr after variable substitution; if result is truthy ("true", non-zero number, non-empty - define exactly in docs), skip. Evaluation must NOT require a JS engine: variable substitution + literal truthiness only.

## Acceptance Criteria
- [x] Truthiness rules tested (true/false case-insensitive, 0/1, empty/non-empty); docs in epic docs pass
- [x] Unknown variable -> runtime error naming the variable (picked: error)
- [x] Fluent tests + docs (docs in epic docs pass)
- [x] `make check` passes (golangci-lint 0 issues, 239 tests 0 failures)

## Proof of Work (2qlf)

- TDD RED verified (working tree): 10 tests in client_execute_disabled_cond_test.go failed ONLY on undefined Request.DisabledExpr; RED sub-bean go-restclient-kkp9 closed.
- GREEN took 4 pi rounds (executor evaluation missing in round 1, cognitive-complexity 3x): final = skipRequest() helper used by runOneRequest + ExecuteRequest; evaluateDisabledExpr + disabledExprVariableExists (complexity 10->6) in client_disabled_expr.go; URL helpers moved client.go -> client_url_utils.go (pure move, verified against master).
- Truthiness: case-insensitive true/false; numeric 0 falsy non-zero truthy; empty falsy; other non-empty truthy. Unknown {{var}} -> runtime error naming variable.
- GREEN commit cff8d85 on feature/execution-control (PR #38), commit-only-green per repo rule.
- Orchestrator-verified on final tree: gotestsum -count=1 ./... = 239 tests 0 failures; golangci-lint 0 issues; build+vet clean.
