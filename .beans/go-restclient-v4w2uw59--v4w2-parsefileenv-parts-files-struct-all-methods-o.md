---
# go-restclient-v4w2uw59
title: 'V4W2: parse/file/env parts files — struct + all methods of each state, one file per state'
status: completed
type: task
priority: normal
created_at: 2026-10-16T11:22:08Z
updated_at: 2026-10-17T19:28:49Z
parent: go-restclient-495b
---

FULL USER REQUIREMENTS: struct+methods same file per state; receiver rule = placement oracle git show 39000b5:{parse,file,env}_parts_test.go. Moves-only, byte-identical, no new comments, no //nolint/lint-config, ≤1000 ln.
## Scope
parse_parts_test.go (parseState+16), file_parts_test.go (fileState+17+externalFilesFixtureDir const), env_parts_test.go (envState+7). Drain those methods from wherever they sit now (base + control/refs/parsefile/vars/core pairs).
## Gates
gofmt/build/vet/lint 0 issues./test -count=1; 270+2; each state's methods in exactly ONE file
## COMMENT GATE: comment diff per touched file; added narration comment = FAILED.
## Skills for coder
~/.agents/skills/go-dev, ~/.agents/skills/coding-rules — read §Comments/§File-structure first; binding.
