---
# go-restclient-v97q
title: 'Task 3: migrate validator+hresp tests, delete Run* layer'
status: completed
type: task
priority: normal
created_at: 2026-09-18T14:59:06Z
updated_at: 2026-09-18T21:37:57Z
parent: go-restclient-t3k3
blocked_by:
    - go-restclient-s7tj
---

Migrate validator tests (validator_general, validator_status, validator_headers, validator_body, validator_placeholders, json_validator_tests) + hresp_vars to fluent DSL. Delete root wrappers (client_test.go, validator_test.go, hresp_vars_test.go) and remaining test/Run* files. AC: grep test.Run[A-Z] = 0, root *_test.go gone, make check green, count >= 231 junit testcases.

## Migration map (validator side)

New fluent test files: validator_general_test.go, validator_status_test.go, validator_headers_test.go, validator_body_test.go, validator_placeholders_test.go, json_validator_test.go, hresp_vars_test.go (RunExtractHrespDefines). ALL migrated tests land in repo ROOT (package restclient_test) per user decision recorded in epic; DELETE root wrappers (client_test.go, validator_test.go, hresp_vars_test.go). DELETE test/RunCreateTestFileFromTemplate_DebugOutput with its helper (tests debug logging of old infra; obsolete once fixtures inline). Delete remaining unused helpers (intPtr, Ptr, assertMultierrorContents, ExtractHrespDefines helpers) if nothing references them. Final gates: grep 'test.Run[A-Z]' = 0 hits; ls *_test.go in root = empty; make check green; junit testcases >= 231.

## Summary of Changes
Migrated validator+hresp tests to fluent DSL (validator_general/status/headers/body/placeholders/json/hresp_vars root files + fluent_parts_validator_test.go with aResponseWith* given family, validator_options_test.go). Deleted: root client_test.go, validator_test.go, old hresp_vars_test.go wrapper; ALL test/*.go legacy helpers (test/ now has zero go files, only test/data fixtures).

## Proof of Work
gotestsum --junitfile -- -count=1 -cover ./... -> 234 junit testcases, 0 failures (>= baseline 231)
golangci-lint run ./... -> 0 issues; go build/vet OK; gofmt clean
grep -rE 'test.Run[A-Z]' --include='*.go' . -> 0 hits
make check -> Checks completed.
Commit: 383bb77

Residual: none.
