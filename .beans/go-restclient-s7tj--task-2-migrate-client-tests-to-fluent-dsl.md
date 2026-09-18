---
# go-restclient-s7tj
title: 'Task 2: migrate client tests to fluent DSL'
status: in-progress
type: task
priority: normal
created_at: 2026-09-18T14:58:36Z
updated_at: 2026-09-18T16:06:28Z
parent: go-restclient-t3k3
blocked_by:
    - go-restclient-l9zu
---

Migrate all client execution tests (client_init, cookies/redirects, execute_core, execute_edgecases, execute_external_file, execute_graphql, execute_inplace_vars, execute_vars, execute_system_vars, execute_uuid_consistency, multiple_system_vars_consistency, client_parsefile) to fluent DSL. Test names preserved. Inline aHttpFile where feasible; keep fixtures only for big/binary. Delete migrated Run* helper files. Parity per current assertions. AC: client-side Run* files gone, tests green.

## Migration map (client side)

USER DECISION: tests live NEXT TO CODE. New fluent test files in REPO ROOT, package restclient_test (DSL core moves: git mv test/fluent_parts_test.go ./fluent_parts_test.go + package restclient_test). Fixture paths stay root-relative (test/data/...) — root tests run with CWD=repo root: client_init_test.go (RunNewClient, RunNewClient_WithOptions), client_cookies_redirects_test.go, client_execute_core_test.go (12 Run funcs), client_execute_edgecases_test.go, client_execute_external_file_test.go, client_execute_graphql_test.go (7), client_execute_inplace_vars_test.go (18), client_execute_vars_test.go, client_execute_system_vars_test.go, client_execute_uuid_consistency_test.go, client_execute_multiple_system_vars_consistency_test.go, client_parsefile_test.go (RunParseFile_* + RunExecuteRequest_*), response_refs_test.go, system_vars_test.go. Delete each Run* source file after its tests migrate. Extensions allowed per epic DSL extension policy.
