---
# go-restclient-vidx
title: 'gh-34: Meta: Form3 event-replay-api manual testing gaps'
status: todo
type: epic
priority: normal
created_at: 2026-07-24T11:33:57Z
updated_at: 2026-07-24T11:33:57Z
---

## Issue #34

End-to-end manual testing findings from form3tech/api-event-replay.

### Pain points to address:
1. Dotenv-shaped .http files don't compose with --after (#29)
2. Empty access_token in .env → API 400 (#28)
3. Token persistence needs custom bash (#30)
4. Async submission status needs poll/wait (#31)
5. Accidental full-file run without -n (#32)
6. Output mode documentation (#27)

### Sub-tasks:
- [ ] Fix dotenv composition with --after
- [ ] Handle empty dotenv values
- [ ] Add dotenv write-back
- [ ] Implement poll/wait
- [ ] Add safer defaults
- [ ] Improve documentation
