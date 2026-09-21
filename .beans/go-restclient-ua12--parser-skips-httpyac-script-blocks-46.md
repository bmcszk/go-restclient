---
# go-restclient-ua12
title: parser skips httpyac script blocks (#46)
status: completed
type: task
priority: normal
created_at: 2026-09-21T16:15:37Z
updated_at: 2026-09-21T16:22:19Z
---

## Goal
Parser skips httpyac `{{ ... }}` inline script blocks (issue #46) — RED tests + skip in the line loop.

## Contract
- RED tests (parser_script_test.go, NO t.Run):
  1. `TestParseFile_ScriptBlockBetweenRequestsIsIgnored` — request A ### script block ### request B → exactly 2 requests, no error.
  2. `TestExecuteFile_ScriptBlockInRequestKeepsVarsWorking` — @name A with script block inside request section, then request B → 2 executed requests, noError.
- Parser: when a line trims to exactly `{{`, consume until a line trims to exactly `}}`, discard. One-line comment: httpyac script blocks are ignored.

## Acceptance
- [x] New tests RED for the right reason (with fix disabled), rest green
- [x] `go build ./... && go vet ./...`
- [x] `gotestsum --junitfile /tmp/iss46_red.xml -- -count=1 ./...`
- [x] `golangci-lint run ./...`
- [x] `gofmt -l *.go` (excluding known-excluded files) empty

## Proof of Work

RED (fix toggled off via temporary local edit — restored after):
```
$ go test -run 'TestParseFile_ScriptBlockBetweenRequestsIsIgnored|TestExecuteFile_ScriptBlockInRequestKeepsVarsWorking' -v .
--- FAIL: TestParseFile_ScriptBlockBetweenRequestsIsIgnored  (expected body "", actual "{{\n  const { randomUUID } = require('crypto');\n  exports.id = randomUUID();\n}}")
--- FAIL: TestExecuteFile_ScriptBlockInRequestKeepsVarsWorking (serverReceivedBodyIs(0,"") got the script block as body)
$ gotestsum --junitfile /tmp/iss46_red.xml -- -count=1 ./...
DONE 275 tests, 2 failures  (only the 2 new tests)
```

Note: contract's predicted RED modes (orphan warn / 3 requests / parse error) did not occur —
bare `{{` lines after a complete request line are treated as request body (no warn, count stays 2).
Discriminating assertions: parsedRequestBodyIs(0, "") + serverReceivedBodyIs(0, "").
Added DSL method parsedRequestBodyIs (fluent_parts_ext_test.go).

GREEN (fix restored):
```
$ go build ./... && go vet ./...        # ok
$ gotestsum --junitfile /tmp/iss46_final.xml -- -count=1 ./...
DONE 275 tests in 8.896s   (0 failures)
$ golangci-lint run ./...   → 0 issues.
$ gofmt -l *.go | grep -vE 'faker.go|hresp_vars.go|multipart.go|options.go'  → empty
```

Residual: commit 90a6c2d (parser fix + docs + superseded client_script_test.go) was made by a
concurrent writer during this task — left untouched; contract deliverable (parser_script_test.go,
parsedRequestBodyIs) stays UNCOMMITTED. Old client_script_test.go names (Execute variant) kept.


## Proof of work

Implemented by orchestrator after pi run was killed (wedged, no output; its late
flush delivered these 2 parse-level tests + parsedRequestBodyIs helper, audited,
kept). Skip logic + first 2 tests + docs: commit 90a6c2d on
feature/issue-fixes-45-49, PR #50, CI all-pass (275 tests, 0 failures, lint 0).
