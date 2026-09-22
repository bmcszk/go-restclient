---
# go-restclient-ak18
title: 'go-restclient CLI --env flag: named dotenv via WithEnvName (#52)'
status: in-progress
type: task
priority: normal
created_at: 2026-09-22T10:04:26Z
updated_at: 2026-09-22T10:04:26Z
---


## Goal
CLI --env <name> flag wiring the existing library option WithEnvName (issue #52). Docs already describe `restclient -f items.http --env staging`; CLI lacks the flag.

## Acceptance
- [x] --env flag in "variables" kong group
- [x] setupClient passes restclient.WithEnvName(c.EnvName) when set
- [x] RED: --env unknown → parse error; GREEN: staging overlay wins over .env
- [x] Regression: without --env, default .env only
- [x] make check green (incl. #33 --all tests), lint 0
- [x] No parser/options changes (CLI wiring only)

## Proof of Work

Base SHA: a759277 (branch feature/issue-33-require-all-without-selector)

### RED (before wiring)
```
--- FAIL: TestCLI_EnvFlagRunsNamedDotenv
    restclient: error: unknown flag --env
FAIL	github.com/bmcszk/go-restclient/cmd/restclient
```

### GREEN (after wiring)
```
--- PASS: TestCLI_EnvFlagRunsNamedDotenv (0.33s)
--- PASS: TestCLI_EnvFlagAbsentKeepsDefaultDotenv (0.33s)
ok  	github.com/bmcszk/go-restclient/cmd/restclient
```

### make check tail
```
Linting...
0 issues.
Running unit tests...
✓  . (cached) (coverage: 83.2% of statements)
✓  cmd/restclient (10.551s) (coverage: 0.0% of statements)

DONE 286 tests in 10.860s
Checks completed.
```

### Quality gates
- go test -count=1 ./cmd/restclient/... — pass (incl. #33 --all tests)
- make check — lint 0 issues, 286 tests pass
- gofmt -l cmd/ — empty; go vet clean
- Files ≤1000 lines (main.go 416, main_test.go 577); no func _, no t.Run, no out-params; comments ≤1 line
- parser.go / options.go / client.go / docs untouched

### Changes
- cmd/restclient/main.go: EnvName field with `name:"env"` tag in variables group; setupClient builds restclient.WithEnvName(c.EnvName) option when non-empty; removed newClient() wrapper (smallest change: restclient.NewClient(opts...) inline).
  Kong v1.16.0 quirk: long:"env" on a field named EnvName is overridden — kong auto-expands to --env-name, so name:"env" is required for the --env spelling.
- cmd/restclient/main_test.go: TestCLI_EnvFlagRunsNamedDotenv + TestCLI_EnvFlagAbsentKeepsDefaultDotenv (tmp-fixture .env/.env.staging, echo-server asserts X-Key via {{$dotenv KEY}}).
