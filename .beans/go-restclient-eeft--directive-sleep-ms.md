---
# go-restclient-eeft
title: 'Directive: @sleep <ms>'
status: completed
type: task
priority: normal
created_at: 2026-09-19T14:16:10Z
updated_at: 2026-09-20T17:10:24Z
parent: go-restclient-p988
---

Parent epic: execution control directives (@disabled/@loop/@sleep).

## Requirement
`# @sleep <ms>`: executor waits given milliseconds before sending the request (after refs resolve, before loop iteration send). Non-negative integer; invalid value -> parse error.

## Acceptance Criteria
- [x] Sleep honored in library (both exec paths); 0 valid no-op; CLI unaffected by design
- [x] Fluent tests with timing assertions (capturedRequestTimes gap DSL, loose lower bounds)
- [x] Docs (epic docs pass)
- [x] `make check` passes (golangci-lint 0 issues, 248 tests 0 failures)

## Proof of Work (eeft)

- TDD RED verified: 6 tests failed only on undefined Request.SleepDuration; capture DSL extended with capturedRequestTimes + gapBetweenCapturedRequestTimesAtLeast (timing hook).
- GREEN commit cdcdc04: SleepDuration on Request; handleSleepDirective (parser_sleep.go, new file — parser_state.go back under 1000-line limit); executor sleeps after refs resolve on both paths; 0 = no-op.
- Orchestrator lint fix round: fmt.Errorf->errors.New + handler moved (1001->988 lines).
- Orchestrator-verified: 248 tests 0 failures, golangci-lint 0 issues, build/vet clean, gofmt clean.
