---
# go-restclient-uldf
title: 'go-restclient CLI: require --all when -f given without -n/-i/-l (#33 Option A)'
status: completed
type: task
priority: normal
created_at: 2026-09-22T09:51:10Z
updated_at: 2026-09-22T10:01:51Z
---

## Goal
CLI safety: when -f given without -n/-i/-l (single-request selectors), refuse to run all named requests. Require explicit --all.

## Scope (Option A from issue)
- `restclient -f file.http --all` — run entire file
- `restclient -f file.http -n foo` — single request (unchanged)
- `restclient -f file.http -i 0` — single request (unchanged)
- `restclient -f file.http -l` — list only (unchanged)
- `restclient -f file.http` (no selector, no --all) → print usage hint, exit 2

## Acceptance
- [x] New test RED: `restclient -f file.http` (multiple requests, no selector) → exit 2 + hint
- [x] GREEN: same argv with --all → executes all in order
- [x] Existing -n/-i/-l/-A paths unchanged (regression-free)
- [x] kong groups: --all in "selection" group
- [x] README updated: safer default note in CLI section
- [x] docs/http_syntax.md untouched (this is CLI, not .http syntax)

## Implementation summary
- `cmd/restclient/main.go`: after `kong.Parse`, early-exit when `-f` is set and none of
  `--all` / `-n` / `-i` / `-l` are set — prints hint to stderr, calls `os.Exit(2)`.
  Dead branch in `dispatchExecution` removed (now unreachable under the new check).
  `--all` flag already present in kong "selection" group.
- `cmd/restclient/main_test.go`: 3 new tests
  (`TestExecuteFile_RequiresAllOrSelector`, `TestExecuteFile_AllFlagRunsAllRequests`,
  `TestExecuteFile_NameSelectorUnchanged`) + 1-line fix to `TestCLI_MissingFile`
  (now passes `--all` so it still tests file-missing under an explicit selector).
- `README.md`: replaced the legacy one-liner with a 3-line "safer default" note that
  mentions `--all`, the hint-to-stderr behavior, and exit code 2.

## Branch / PR
- Branch: `feature/issue-33-require-all-without-selector` (to be created by orchestrator)
- Base: `master` @ `27b0def`
- PR body: what it does (Option A enforced) + PoW below + `Fixes #33` footer
- Working tree left UNCOMMITTED for orchestrator to branch+commit+PR.

## Proof of Work

### RED (before fix)
`go test -count=1 -run TestExecuteFile_RequiresAllOrSelector ./cmd/restclient/...`

```
--- FAIL: TestExecuteFile_RequiresAllOrSelector (0.33s)
    main_test.go:486:
        Error: Not equal:
            expected: 2
            actual  : 1
        Test: TestExecuteFile_RequiresAllOrSelector
        Messages: missing selector with -f must exit 2; stdout=error: specify -n NAME, -i INDEX, or --all to run requests
    main_test.go:487:
        Error: "error: specify -n NAME, -i INDEX, or --all to run requests\n" does not contain "hint:"
    main_test.go:490:
        Error: "error: specify -n NAME, -i INDEX, or --all to run requests\n" does not contain "-i IDX"
    main_test.go:491:
        Error: "error: specify -n NAME, -i INDEX, or --all to run requests\n" does not contain "-l (list)"
    main_test.go:492:
        Error: "error: specify -n NAME, -i INDEX, or --all to run requests\n" does not contain "Refusing to execute the whole file by default"
FAIL    github.com/bmcszk/go-restclient/cmd/restclient    1.100s
```

(The two sibling tests — `TestExecuteFile_AllFlagRunsAllRequests` and
`TestExecuteFile_NameSelectorUnchanged` — already passed before the fix; they are
regression-style guards.)

### GREEN (after fix)
`go test -count=1 -v -run TestExecuteFile_ ./cmd/restclient/...`

```
=== RUN   TestExecuteFile_RequiresAllOrSelector
--- PASS: TestExecuteFile_RequiresAllOrSelector (0.39s)
=== RUN   TestExecuteFile_AllFlagRunsAllRequests
--- PASS: TestExecuteFile_AllFlagRunsAllRequests (0.34s)
=== RUN   TestExecuteFile_NameSelectorUnchanged
--- PASS: TestExecuteFile_NameSelectorUnchanged (0.37s)
PASS
ok      github.com/bmcszk/go-restclient/cmd/restclient    1.093s
```

Full suite (`go test -count=1 ./cmd/restclient/...`):

```
ok      github.com/bmcszk/go-restclient/cmd/restclient    10.090s
```

### Lint / fmt / vet / build
- `golangci-lint run ./cmd/...` -> `0 issues.`
- `gofmt -l cmd/` -> empty
- `go vet ./cmd/...` -> clean
- `go build ./...` -> ok

### Quality gates
- `cmd/restclient/main.go` -> 416 lines (under 1000-line ceiling)
- `cmd/restclient/main_test.go` -> 533 lines (under 1000-line ceiling)
- `grep -rn 'func _' cmd/` -> no matches
- `grep 't.Run' cmd/restclient/main_test.go` -> no matches
- No out-params (`**T`, `*[]T`) introduced.

### Commit / SHA
- Working tree UNCOMMITTED (per orchestrator contract).
- Base: `27b0def26ea96c2afb86c31fa097b40b7fa3b40b` (master HEAD).
- Branch+commit to be created by orchestrator: `feature/issue-33-require-all-without-selector`.
