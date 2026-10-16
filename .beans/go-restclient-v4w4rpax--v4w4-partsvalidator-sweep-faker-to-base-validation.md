---
# go-restclient-v4w4rpax
title: 'V4W4: parts/validator sweep — faker to base, validation surface to validator_parts, drain-delete 7 client_execute_*/parsefile pair files'
status: todo
type: task
priority: normal
created_at: 2026-10-16T11:22:08Z
updated_at: 2026-10-16T11:22:08Z
parent: go-restclient-495b
---

FULL USER REQUIREMENTS: receiver rule — *parts methods split by domain: client-core in parts_test.go, validation surface in validator_parts_test.go; fakerHeaderRule+serverReceivedFakerHeaders (*parts) → parts_test.go. Moves-only, byte-identical, no new comments, no //nolint/lint-config, ≤1000 ln.
## Scope
11 client-core *parts methods → parts_test.go (+clientConfigSnapshot); expectedResponseFile/responsesValidateAgainst/responsesValidateAgainstFixture/anExpectedResponseFixture → validator_parts_test.go; drain-delete the 7 empty pair files (client_execute_{control,core,graphql,refs,system_vars,vars}_parts_test.go, client_parsefile_parts_test.go).
## Gates
gofmt/build/vet/lint 0 issues./test -count=1; 270+2; deleted files gone; ls *_parts_test.go = 9 files
## COMMENT GATE: comment diff per touched file; added narration = FAILED.
## Skills for coder
~/.agents/skills/go-dev, ~/.agents/skills/coding-rules — read §Comments/§File-structure first; binding.
