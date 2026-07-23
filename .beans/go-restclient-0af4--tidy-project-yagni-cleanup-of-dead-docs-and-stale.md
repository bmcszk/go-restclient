---
# go-restclient-0af4
title: 'Tidy project: YAGNI cleanup of dead docs and stale scaffolding'
status: completed
type: task
priority: normal
created_at: 2026-07-23T12:07:19Z
updated_at: 2026-07-23T12:09:21Z
---

Remove speculative/scaffolding bloat that does nothing for a Go library shipping a CLI:

- [ ] Delete prds/ (16 PRD + task-tracking markdown files — spec-history noise, not used by build/tests)
- [ ] Delete stale planning docs: docs/test_scenarios.md, docs/testing-guidelines.md, docs/requirements.md, docs/decisions.md, docs/learnings.md, docs/tasks.md, docs/project_structure.md
- [ ] Keep docs/http_syntax.md (real user-facing reference)
- [ ] Trim .gitignore: remove ignores for .cursor/, .windsurf/, .aider*, CLAUDE.md — none exist (confirmed). Keep binary/OS/coverage ignores.
- [ ] Run go mod tidy

Scope: deletion only. No code changes here.
