---
# go-restclient-qe7j
title: 'Parser+executor: @import ./file.http (cross-file refs + file-global vars)'
status: completed
type: task
priority: normal
created_at: 2026-09-19T14:16:10Z
updated_at: 2026-09-19T18:35:15Z
parent: go-restclient-8y9h
blocked_by:
    - go-restclient-maez
---

Parent epic: declarative request referencing (@ref/@forceRef/@import).

## Requirement
`# @import ./other.http` at file top: parse imported file first; its file-global variables and named requests become referenceable (@ref/@forceRef + `{{name.response.*}}`) from the importing file. httpyac semantics: only file-global-scope variables are imported.

## Acceptance Criteria
- [x] Imported file parsed once; its named requests resolvable via @ref/@forceRef
- [x] File-global vars from imported file visible in importing file (existing var precedence preserved: programmatic > env > file)
- [x] Missing import file -> clear error
- [x] Fluent tests + docs section in http_syntax.md
- [ ] - [x] `make check` passes (golangci-lint 0 issues, full test suite 0 failures)

## Proof of Work

TDD RED (pi): 3 tests failed with `unknown referenced request "token"` / missing-file not raising error (orchestrator re-verified).
TDD GREEN (pi): +60 lines across parser_state.go (handleImportDirective: relative resolve, import-stack cycle guard, var merge with local-wins, append-after-local), request.go (Imported flag), client.go (executor skips imported in main loop).
Orchestrator re-ran all gates: 3/3 import tests PASS, all 9 refs tests PASS, golangci-lint 0 issues, gotestsum DONE 223 tests 0 failures, gofmt clean (parser.go/parser_helpers.go pre-existing drift also fixed).
Committed b4d8561 (PR #37).
Residual: docs section for @import done in epic docs pass (commit follows).


Docs delivered in epic docs pass (docs/http_syntax.md Request Referencing + README Key Features).
