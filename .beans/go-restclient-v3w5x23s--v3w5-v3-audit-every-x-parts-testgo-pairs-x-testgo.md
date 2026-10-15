---
# go-restclient-v3w5x23s
title: 'V3W5: v3 audit — every X_parts_test.go pairs X_test.go + contains only its pair-used decls; parity; PR body'
status: todo
type: task
priority: normal
created_at: 2026-10-15T15:55:30Z
updated_at: 2026-10-15T15:55:58Z
parent: go-restclient-495b
---

FULL REQUIREMENTS: same as epic v3. Orchestrator-run script audit.
## Scope
Script: (1) every *_parts_test.go basename X_parts_test.go has existing X_test.go; (2) each parts file's decls used ONLY by its pair (+base); (3) zero `root *parts`; (4) `*parts` first-line embedding in every state struct; (5) 270+2 parity, ≤1000 ln; (6) PR #84 body v3 update.
## Gates
all PASS with verbatim script output; PR body updated
