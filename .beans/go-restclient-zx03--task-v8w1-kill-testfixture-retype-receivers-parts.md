---
# go-restclient-zx03
title: 'task V8W1 kill testFixture: retype receivers *parts, fold state fields, ctors return *parts'
status: completed
type: task
priority: normal
created_at: 2026-10-16T18:57:10Z
updated_at: 2026-10-16T19:11:47Z
parent: go-restclient-pxon
---

## Summary of Changes

- parts_test.go: testFixture composite, 8 domain state structs and newFixture deleted; parts now holds all 18 state fields (field names preserved); newParts(t) constructs parts directly (given, when, then *parts); core init (T, require, assert, baseDir, trackedValues) lives in newParts.
- All 8 domain ctors (newCaptureParts/newServerParts/newFileParts/newEnvParts/newValidatorParts/newOauth2Parts/newCookiesParts/newParseParts) construct the parts core inline, no newFixture/newParts delegation; capture ctor owns trackedValues init.
- All DSL receivers/returns retyped *testFixture -> *parts; .and() intact; test bodies untouched.
- Comment purge (V8W2 folded): narration doc comments deleted in all 9 touched files; contract clauses kept (see per-file counts in report).
- Imports fixed per file after field folds (httptest/atomic moved to parts_test.go; testify added where ctors now assert).

Gates: gofmt EMPTY, go build 0, go vet 0, golangci-lint 0 issues., 315 test passes, Test-func diff vs 494ef81 EMPTY, grep testFixture EMPTY, grep 'return newParts(t)' in *_parts_test.go EMPTY.
