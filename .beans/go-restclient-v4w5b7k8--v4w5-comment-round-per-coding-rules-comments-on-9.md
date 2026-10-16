---
# go-restclient-v4w5b7k8
title: 'V4W5: comment round per coding-rules §Comments on 9 parts files + comment-audit gate + final audit + PR body'
status: todo
type: task
priority: normal
created_at: 2026-10-16T11:22:08Z
updated_at: 2026-10-16T11:22:08Z
parent: go-restclient-495b
---

FULL USER REQUIREMENTS: comment rules applied everywhere (user complaint: coding-rules not applied). Scope: coding-rules §Comments round over the 9 final parts files: delete narration doc comments (// X asserts..., // RoundTrip delegates...), keep why/invariant/contract clauses only; re-point parts-struct embedding-field cross-reference comments to state files. Then orchestrator final audit + PR #84 body update.
## Gates
gofmt/build/vet/lint 0 issues./test -count=1; 270+2; comment count NET DECREASE vs round start; zero narration-verb doc comments (asserts|delegates|writes|returns|builds|resets|hosts|adapts|copies|renders|starts) on moved blocks
## COMMENT GATE: report includes per-file grep -c '//' before/after with per-delta coding-rules justification.
## Skills for coder
~/.agents/skills/go-dev, ~/.agents/skills/coding-rules — §Comments IS this wave's scope.
