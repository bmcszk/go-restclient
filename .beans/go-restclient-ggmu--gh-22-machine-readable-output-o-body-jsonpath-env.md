---
# go-restclient-ggmu
title: 'gh-22: Machine-readable output (-o body, jsonpath, env)'
status: completed
type: feature
priority: normal
created_at: 2026-07-23T17:21:22Z
updated_at: 2026-07-23T17:37:10Z
---

go-restclient-ggmu  todo   feature  [normal]           
gh-22: Machine-readable output (-o body, jsonpath, env)
──────────────────────────────────────────────────     
                                                       

  Implements #22: Add -o flag for machine-readable output modes.

## Done\n- \`-o body\` - raw response body\n- \`-o jsonpath expr\` - JSON path extraction (nested paths, array indices)\n- \`-o env key\` - prints key=value for shell sourcing\n- 227 tests pass
