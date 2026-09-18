---
# go-restclient-v97q
title: 'Task 3: migrate validator+hresp tests, delete Run* layer'
status: todo
type: task
priority: normal
created_at: 2026-09-18T14:59:06Z
updated_at: 2026-09-18T15:04:17Z
parent: go-restclient-t3k3
blocked_by:
    - go-restclient-s7tj
---

Migrate validator tests (validator_general, validator_status, validator_headers, validator_body, validator_placeholders, json_validator_tests) + hresp_vars to fluent DSL. Delete root wrappers (client_test.go, validator_test.go, hresp_vars_test.go) and remaining test/Run* files. AC: grep test.Run[A-Z] = 0, root *_test.go gone, make check green, count >= 231 junit testcases.

## Migration map (validator side)

New fluent test files: validator_general_test.go, validator_status_test.go, validator_headers_test.go, validator_body_test.go, validator_placeholders_test.go, json_validator_test.go, hresp_vars_test.go (RunExtractHrespDefines). DELETE root wrappers (client_test.go, validator_test.go, hresp_vars_test.go). DELETE test/RunCreateTestFileFromTemplate_DebugOutput with its helper (tests debug logging of old infra; obsolete once fixtures inline). Delete remaining unused helpers (intPtr, Ptr, assertMultierrorContents, ExtractHrespDefines helpers) if nothing references them. Final gates: grep 'test.Run[A-Z]' = 0 hits; ls *_test.go in root = empty; make check green; junit testcases >= 231.
