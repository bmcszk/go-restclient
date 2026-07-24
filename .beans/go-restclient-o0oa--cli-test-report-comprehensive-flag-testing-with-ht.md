---
# go-restclient-o0oa
title: 'CLI test report: comprehensive flag testing with .http files'
status: completed
type: task
priority: normal
created_at: 2026-07-24T09:00:44Z
updated_at: 2026-07-24T09:11:24Z
---

The docs/cli_test_report.md is incomplete. Every flag needs to be tested with real examples.

Tasks:
- [ ] Create real .http test files for each flag
- [ ] Test -f/--file flag
- [ ] Test -n/--name flag
- [ ] Test -i/--index flag
- [ ] Test -e/--expected flag
- [ ] Test --e-name flag
- [ ] Test --e-index flag
- [ ] Test -l/--list flag
- [ ] Test -E/--fail-on-error flag
- [ ] Test -o/--output flag (body, jsonpath, env)
- [ ] Test -A/--after flag
- [ ] Test -D/--define flag (single and multiple)
- [ ] Test -h/--help flag
- [ ] Test error cases (missing file, invalid flags, etc.)
- [ ] Update docs/cli_test_report.md with commands and results
