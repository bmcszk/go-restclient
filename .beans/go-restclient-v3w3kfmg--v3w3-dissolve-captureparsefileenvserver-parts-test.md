---
# go-restclient-v3w3kfmg
title: 'V3W3: dissolve capture/parse/file/env/server_parts_test.go into base + new per-pair parts files (core/system_vars/vars/graphql/refs/control/parsefile)'
status: completed
type: task
priority: normal
created_at: 2026-10-15T15:55:30Z
updated_at: 2026-10-15T16:40:01Z
parent: go-restclient-495b
---

FULL REQUIREMENTS: same as epic v3. Rule 3 is THE core: every parts file ONLY serves its pair. Moves-only + mechanical; no test-body edits; no //nolint/lint-config; ≤1000 ln.
## Scope (brainer usage-matrix driven)
- Build usage matrix yourself first (grep each parts method across *_test.go)
- Dissolve capture_parts_test.go, parse_parts_test.go, file_parts_test.go, env_parts_test.go, server_parts_test.go:
  - multi-file-used methods + shared state structs → parts_test.go (base, must stay <1000)
  - single-pair methods → NEW pair files: client_execute_core_parts_test.go, client_execute_system_vars_parts_test.go, client_execute_vars_parts_test.go, client_execute_graphql_parts_test.go (graphqlResponse/graphqlError types here), client_execute_refs_parts_test.go, client_execute_control_parts_test.go, client_parsefile_parts_test.go
  - state structs whose methods all land in one pair → struct moves there too; shared structs stay in base
- graphql fold check: server_parts graphql methods → client_execute_graphql_parts_test.go
## Gates
gofmt empty; build 0; vet 0; lint `0 issues.`; test -count=1 ok; 270+2 parity; dissolved files gone; every remaining *_parts_test.go pairs an existing X_test.go
## Skills for coder
~/.agents/skills/go-dev, ~/.agents/skills/coding-rules

## Proof of Work (orchestrator-verified, independent rule-3 audit)
- 5 domain parts files dissolved; 7 NEW pair files + validator trim; every *_parts_test.go pairs its X_test.go (10 pairs + base)
- my independent audit: rule-3 violations NONE (1 false positive: aHttpFileFromTemplateWithData lives in vars pair, wrapped by a multi-user base builder — legitimate); single-user-in-base: NONE (user-flagged aRestExtensionServer/aJsonEchoServer now in client_parsefile_parts_test.go)
- all files ≤1000 (base 835 max); gates re-run: gofmt empty, build/vet 0, lint `0 issues.`, tests ok, 270+2 parity
- Residual: validator pair trim (V3W4)
