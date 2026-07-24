---
# go-restclient-76vr
title: Add CLI app installable via go install for running .http requests
status: completed
type: task
priority: normal
created_at: 2026-07-23T12:07:34Z
updated_at: 2026-07-23T12:32:05Z
---

Goal: go install github.com/bmcszk/go-restclient/cmd/restclient@latest produces a restclient binary that runs a .http file and prints responses.

Minimal design (ponytail: one file, stdlib flag):
- [ ] Create cmd/restclient/main.go with package main
- [ ] Flags: -f <file> (required). Ship only -f first.
- [ ] Use existing restclient.NewClient() + ExecuteFile(ctx, path)
- [ ] Print each response: METHOD URL -> STATUS (DURATION), then headers + body. stdout for bodies, stderr for errors.
- [ ] Exit code 1 if any response has an error; 0 otherwise.
- [ ] Add make build-cli target and CLI Usage section to README with the go install line.
- [ ] Smoke test: go vet/build ./cmd/restclient

Constraints: no new deps. No interactive REPL/TUI — YAGNI.

## Lint fixes (round 2)

- Removed .golangci.yml.backup (config left untouched).
- Rewrote cmd/restclient/main.go to satisfy golangci-lint v2 (enable-all-rules):
  - flag.Parse() back in main(); run() takes the path string (deep-exit rule).
  - Split run/emit/formatResponse to keep cognitive complexity <= 7.
  - Pure string concat + slices.Sort, no strings.Builder (revive unhandled-error flags Builder methods).
  - Single checked os.Stdout.WriteString via flushOutput; stderr writes use _, _ = (flushError).
- Renamed beforeTime/afterTime -> beforeTimeSec/afterTimeSec in test helpers (epoch-naming).
- Shortened 2 over-long (>120) assert messages (line-length-limit).
- Result: make check -> 0 issues, 192 tests pass.
