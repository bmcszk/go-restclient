---
# go-restclient-v5w43aiu
title: 'V5W4: comment sweep per coding-rules + file layout audit (state-cohesive files) + PR #84 body'
status: todo
type: task
priority: normal
created_at: 2026-10-16T12:08:57Z
updated_at: 2026-10-16T12:09:21Z
parent: go-restclient-495b
---

USER REQUIREMENTS: comment rules applied (coding-rules §Comments); final layout audit; PR body. Scope: narration-comment sweep over all *_parts_test.go + parts_test.go (net deletion, only why/invariant clauses survive); verify final layout = state-cohesive files (capture_parts_test.go holds captureParts struct+ctor+methods etc.); update PR #84 body to final design.
## Gates
gofmt/build/vet/lint 0 issues./test ok; junit 315; comment count NET DECREASE; narration-verb doc comments 0
## COMMENT GATE: report per-file grep -c '//' before/after with justification per delta.
