---
# go-restclient-1rxc
title: 'batch 2c: final client-side fluent test migration [DONE]'
status: completed
type: task
priority: normal
created_at: 2026-09-18T19:56:18Z
updated_at: 2026-09-18T20:16:48Z
---

go-restclient-1rxc  completed   task  [normal]          
batch 2c: final client-side fluent test migration [DONE]
──────────────────────────────────────────────────      
                                                        

  Migrate the last 4 client-side Run* files to fluent DSL. Files to create at 
  repo root: client_execute_vars_test.go, client_execute_system_vars_test.go, 
  client_execute_inplace_vars_test.go, client_execute_graphql_test.go. Files  
  to delete (test/): the 4 corresponding .go files. Update client_test.go to  
  remove wrappers. Acceptance: go build, go vet, gofmt clean, golangci-lint 0 
  issues, gotestsum >= 234 testcases, remove dead helpers.


## Summary of Changes

**Files created** (all at repo root, package restclient_test):
- `client_execute_vars_test.go` (486 lines) - 9 tests: WithCustomVariables, WithProcessEnvSystemVariable, WithDotEnvSystemVariable (2 subtests), WithProgrammaticVariables, WithLocalDatetimeSystemVariable, VariableFunctionConsistency, WithHttpClientEnvJson (2 subtests), WithExtendedRandomSystemVariables, WithIndirectEnvironmentVariables
- `client_execute_system_vars_test.go` (375 lines) - 7 tests: WithGuidSystemVariable, WithIsoTimestampSystemVariable, WithDatetimeSystemVariables, WithTimestampSystemVariable, WithRandomIntSystemVariable (4 subtests), WithFakerPersonData, WithContactAndInternetFakerData
- `client_execute_inplace_vars_test.go` (438 lines) - 18 tests: all TestExecuteFile_InPlace_*
- `client_execute_graphql_test.go` (298 lines) - 7 tests: GraphQLBasicQuery, QueryWithVariables, Mutation, Fragments, Introspection, ErrorHandling, BatchQueries

**Files deleted** (test/):
- test/client_execute_vars.go
- test/client_execute_system_vars.go
- test/client_execute_inplace_vars.go
- test/client_execute_graphql.go

**client_test.go**: removed all 30 wrappers for migrated tests; only TestCreateTestFileFromTemplate_DebugOutput remains.

**client_test_helpers.go**: removed dead helpers `startMockServer` and `parseHrespBody` (no callers after migration).

**fluent_parts_ext_test.go**: added new DSL methods:
- Given: aDotEnvFile, aDotEnvFileRemoved, aFixtureCopy, anEnvJsonFile, aClientWithEnvironment, aRequestFixtureAbs
- Then: capturedRequestURLIs, capturedRequestPathIs, serverReceivedHostIs, capturedJSONStringMapIs, requestPathMatchesCapturedPath, requestHeadersMatchCaptured, requestRawBodyMatchesCapturedBody, allTrackedValuesAreValidRFC3339Timestamps, allTrackedDatetimeValuesAreWithin, allTrackedDatetimeValuesHaveUTCZone, allTrackedDatetimeValuesHaveLocalZone, capturedJSONFieldIs, capturedJSONFieldContains, capturedJSONFieldMatchesRegexp, capturedJSONFieldNotContains, serverReceivedHeaderMatchesRegexp, serverReceivedHeaderNotContains, serverReceivedHeaderFieldCountIs, serverReceivedHeaderContains, capturedRequestPathMatchesRegexp, responsesValidateAgainst, firstTrackedValueIsFloatInRange, allTrackedValuesMatchRegexp, capturedBodyContains, capturedBodyMatchesRegexp, capturedURLSegmentAt

**Verification**:
- go build: ✅
- go vet: ✅
- gofmt (touched files): ✅
- golangci-lint ./...: ✅ 0 issues
- gotestsum: ✅ 234 testcases (parity preserved), coverage 81.2%

## Proof of Work

```
$ go build ./...           # exit 0
$ go vet ./...             # exit 0
$ gofmt -l (touched files) # no output
$ golangci-lint run ./...  # 0 issues
$ gotestsum --junitfile /tmp/t2c.xml -- -cover ./...  # DONE 234 tests
$ python3 -c "import xml.etree.ElementTree as ET; print(len(ET.parse('/tmp/t2c.xml').getroot().findall('.//testcase')))" # 234

Coverage: 81.2%
```

Residual: nil
