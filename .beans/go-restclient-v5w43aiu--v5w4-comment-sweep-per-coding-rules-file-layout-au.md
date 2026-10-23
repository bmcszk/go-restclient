---
# go-restclient-v5w43aiu
title: 'V5W4: comment sweep per coding-rules + file layout audit (state-cohesive files) + PR #84 body'
status: completed
type: task
priority: normal
created_at: 2026-10-16T12:08:57Z
updated_at: 2026-10-17T19:28:49Z
parent: go-restclient-495b
---

USER REQUIREMENTS: comment rules applied (coding-rules §Comments); final layout audit; PR body. Scope: narration-comment sweep over all *_parts_test.go + parts_test.go (net deletion, only why/invariant clauses survive); verify final layout = state-cohesive files (capture_parts_test.go holds captureParts struct+ctor+methods etc.); update PR #84 body to final design.
## Gates
gofmt/build/vet/lint 0 issues./test ok; junit 315; comment count NET DECREASE; narration-verb doc comments 0
## COMMENT GATE: report per-file grep -c '//' before/after with justification per delta.

## Proof of Work (orchestrator fix-up, user lesson applied)
- removed ALL 119 `p := c.parts` hop lines (user: embedding means NO re-binding); rewrote 426 remaining `p.` refs to receiver-promoted (`c.require`, `c.assert`, `c.parsedFile`, `c.aHttpServer`...); 20 stray blank lines cleaned (revive empty-lines)
- zero value-usages of p (verified before rewrite) → pure promoted-member access, compiles
- gates: gofmt empty, build/vet 0, lint `0 issues.`, junit 315, Test-name parity vs pre-V5 EXACT
- LESSON STORED: Go embedding = promoted members; never re-bind host

## Orchestrator fix-up (user review round 2)
- all 7 `var _ = []any{ DParts{} }` compile-assert blocks removed (28 lines)
- gates: gofmt empty, lint `0 issues.`, junit 315, Test-name parity exact; CI 9 pass + Scorecard skip
