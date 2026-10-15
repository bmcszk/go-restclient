---
# go-restclient-j61z
title: ""
status: ""
type: task
priority: normal
created_at: 2026-10-15T10:19:26Z
updated_at: 2026-10-15T10:22:41Z
parent: go-restclient-495b
---

# T1: parts-DSL structural split (base struct + domain structs, <domain>_parts_test.go)

Parent: go-restclient-495b (TEST-CORPUS-REORG epic). Branch: NEW off master (or off reorg/test-consolidation stacked? NO — off master after #83 merges; do NOT block on it if user merges slow: rebase).

## Research (done 2026-10-15, verified counts)
- parts struct: 25 flat fields across all domains (fluent_parts_test.go 271 ln core + 6 ext files, 167 builders total: ext 63, capture 46, core 21, validator 19, cookies 9, oauth2 8, graphql 1)
- ALL 34 scenario files mix >=2 domains inside one .and() chain (16 use 3, 4 use 4) — Go embedding promotion returns the embedded type's own receiver, so chain-across-types is impossible; builders MUST return root *parts via back-pointer to keep 270 tests byte-identical
- Suffix convention (user): `<domain>_parts_test.go` (NOT parts_<domain>_)

## Acceptance criteria (max 3)
- [ ] capture domain as pilot: cookieCheck/captured*/tracked* fields move to an embedded domain struct, builders live on it and return *parts; all 270 tests pass byte-identical chains; file = capture_parts_test.go
- [ ] remaining domains same pattern: core/file/env/server/parse/validator/oauth2/cookies; fluent_parts_ext_test.go dissolved by domain; every file < 1000 lines
- [ ] docs/fluent-testing-guideline.md DSL layout table updated in same PR

## Gates
gofmt/vet/build/lint 0 issues/test -count=1 green; Test-inventory parity 270; zero scenario-file diffs (grep `git diff master --stat -- '*client_*' '*validator_*'` empty besides none); CI 10/10.

## Skills for coder
- ~/.agents/skills/go-dev
- go-restclient-fluent-dsl skill (DSL file layout + hard rules, suffix convention)
- docs/fluent-testing-guideline.md sections 6-12 (in-repo)

## Delegation
pi contract via pi-delegation skill: GIT JAIL + gates + -t +codemode + subagent scout for pre-move symbol census; orchestrator re-gates + commits + PR.
