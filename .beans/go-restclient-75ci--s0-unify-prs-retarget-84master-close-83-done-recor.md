---
# go-restclient-75ci
title: 'S0: unify PRs — retarget #84→master, close #83 (DONE, recorded)'
status: completed
type: task
priority: normal
created_at: 2026-10-15T11:20:03Z
updated_at: 2026-10-15T11:20:41Z
parent: go-restclient-495b
---

# S0: unify PRs
Parent: go-restclient-j61z (T1) under epic go-restclient-495b. Wave 0.
## Triage
Research: brainer completion-plan (2026-10-15) recommended option (b): retarget #84→master, close #83. Verified gh pr edit allows base change; lineage branch kept.
## Acceptance criteria
- [x] #84 base = master
- [x] #83 closed superseded, evidence moved to #84 body
- [x] single open reorg PR remains
## Skills for coder
- n/a (GitHub PR ops, no code)
## Summary of Changes
gh pr edit 84 --base master; gh pr close 83 --comment supersede; #84 body rewritten carrying #83 PoW + wave plan.
## Proof of Work
gh pr view 84 → baseRefName=master, state OPEN; gh pr list --state open → only #84 (+#17 unrelated).

Residual: none.
