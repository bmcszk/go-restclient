---
# go-restclient-l9zu
title: 'Task 1: fluent DSL core (parts, given/when/then surface)'
status: completed
type: task
priority: normal
created_at: 2026-09-18T14:58:23Z
updated_at: 2026-09-18T15:28:48Z
parent: go-restclient-t3k3
---

Build test/fluent_parts_test.go: parts struct, newParts, and(), given methods (aHttpServer, aHttpFile, aClient, withProgrammaticVars, withEnv, expectedResponseFile), when methods (executeFile, sendingRequest, validateResponses), then methods (responseAt, noError, errorContains, errorCount, responseCode, responseContains, responseNotContains, responseBodyIs, responseHeader, requestCount, validationSucceeds, validationFails). Spec: epic body (DSL surface binding). AC: full surface compiles + lint clean + covered by pilot tests.

## Summary of Changes
Created test/fluent_parts_test.go (307 lines, package test_test per testpackage linter): parts struct with full test state, newParts(t)->given/when/then, and(), given (aHttpServer with request counting+Cleanup, aHttpFile {{server}}, aClient, withProgrammaticVars, withEnv, expectedResponseFile), when (executeFile, sendingRequest by Name via ParseFile+ExecuteRequest, validateResponses), then (responseAt cursor, noError, errorContains, errorCount multierror-aware, responseCode, responseContains, responseNotContains, responseBodyIs, responseHeader, requestCount, validationSucceeds, validationFails). Field requestHits (name collision with requestCount method). dslSymbols var keeps unused linter quiet until migrations land.

## Proof of Work
go build ./... -> OK
go vet ./test -> OK
golangci-lint run ./test -> 0 issues
gotestsum --junitfile -- -cover ./... -> DONE 231 tests, PASS (baseline parity, definitions-only file)
git status -> only test/fluent_parts_test.go added, 0 existing files touched

Residual: dslSymbols placeholder removed in Task 2 batch 1; DSL extensions (regexp/JSONEq/capture) added per epic extension policy during migration.
