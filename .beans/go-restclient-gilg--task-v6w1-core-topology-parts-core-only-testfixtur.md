---
# go-restclient-gilg
title: 'task V6W1 core topology: parts core-only + testFixture + state-only domains + retype receivers *testFixture; executeFile->file_parts; requestCount->server_parts'
status: completed
type: task
priority: normal
created_at: 2026-10-16T16:41:53Z
updated_at: 2026-10-16T16:56:18Z
parent: go-restclient-zvmy
---

## Proof of Work V6W1 (orchestrator-verified)
- parts = core only (zero domain embeds); testFixture{ parts + 6 state-only domain structs } (envParts dropped as empty per scratch)
- ALL fluent receivers/returns *testFixture; 270 ctor calls -> newParts(t) (given/when/then); W2 merged into W1 commit (W1 alone uncompilable — 193 removed ctor refs)
- executeFile -> file_parts_test.go, requestCount -> server_parts_test.go
- MY gates: gofmt EMPTY, build OK, vet OK, lint `0 issues.`, junit 315, root Test-name parity vs 494ef81 PARITY-EXACT
- comment audit: +5/-19 (3 doc comments: parts, testFixture, executeFile); no //nolint; only *_test.go + bean ledger touched
- file layout: struct -> ctor -> methods order holds in every _parts_test.go
