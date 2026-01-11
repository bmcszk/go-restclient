# PR Guidelines: Comment Tracking and Resolution Protocol

## Overview

This document provides **MANDATORY** guidelines for systematically tracking and addressing **ALL** PR review comments without exception. Zero tolerance means zero missed comments.

## 🚨 Critical Rules

1. **NEVER assume you got all comments** - Always verify complete retrieval
2. **Use GitHub GraphQL API** - Most reliable method for review threads
3. **Verify BEFORE resolving** - Ensure each issue is actually fixed in code
4. **Document everything** - Track all comments in real-time
5. **Use both weapons** - `make check` AND `make test-integration` before pushing

## PR Workflow

### Phase 0: GitHub Authentication Verification

**Always verify GitHub CLI authentication before starting PR work**

```bash
# Check which GitHub user is currently authenticated
gh auth status

# Expected output shows username and authentication state
# Logged in to github.com account USERNAME (keyring)

# If logged in as wrong user, switch accounts
gh auth switch

# Follow prompts to authenticate with correct account
# - Choose: GitHub.com
# - Preferred protocol: HTTPS or SSH
# - Authenticate via: Browser or Token
```

**Verification checklist:**
- [ ] Verify you're logged in to the correct GitHub account
- [ ] Confirm account has access to the repository
- [ ] Test access: `gh repo view OWNER/REPO` (should show repo details)
- [ ] If multiple accounts configured, ensure correct one is active

**Why this matters:**
- Wrong GitHub account = "Could not resolve to Repository" errors
- Cannot create/view PRs if logged in as different user
- GraphQL API calls will fail with wrong credentials

### Phase 1: Pre-PR Preparation
1. Create feature branch following Git Flow (NEVER commit to main/master)
2. Implement with TDD - Red-Green-Refactor cycle
3. Run `make check` after EVERY code change
4. Ensure zero lint issues before creating PR

### Phase 2: PR Creation
```bash
# Create PR with comprehensive description
gh pr create --title "Brief descriptive title" \
  --body "## Summary
## Changes
## Testing
## Checklist"

# Verify PR was created
gh pr status
```

### Phase 3: Comment Retrieval (CRITICAL)

Use **GitHub GraphQL API** for complete and reliable comment retrieval:

```bash
# Get ALL review threads with their resolution status
# Note: GitHub GraphQL API has pagination limits (max 100 items per page)
# For PRs with 100+ review threads, you would need to handle pagination
gh api graphql -f query='
query($owner: String!, $repo: String!, $pr: Int!) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $pr) {
      reviewThreads(first: 100) {
        nodes {
          id
          isResolved
          comments(first: 1) {
            nodes {
              id
              body
            }
          }
        }
      }
    }
  }
}' -f owner=OWNER -f repo=REPO -F pr=PR_NUMBER
```

**Why GraphQL API?**
- Returns review thread IDs needed for resolution
- Shows resolution status (`isResolved: true/false`)
- Gets all comments in single request
- More reliable than REST API pagination

### Phase 4: Comment Verification

**BEFORE resolving any comment, verify the fix in code:**

```bash
# Example: Verify a function parameter fix
# Read the file at the line mentioned in comment
cat -n path/to/file.go | grep -A 5 "line_number"

# Verify the parameter is actually being used
grep "parameter_name" path/to/file.go
```

**Verification Checklist:**
- [ ] Read the actual code at the line referenced
- [ ] Confirm the issue exists (or existed)
- [ ] Verify your fix addresses the comment exactly
- [ ] Run `make check` to ensure no regressions
- [ ] Document the fix with file:line reference

### Phase 5: Resolution

Only resolve comments after verifying fixes:

```bash
# Resolve a review thread using its ID
gh api graphql -f query='
mutation {
  resolveReviewThread(input: {threadId: "THREAD_ID"}) {
    thread {
      id
      isResolved
    }
  }
}'

# Resolve multiple threads in batch
for thread_id in "THREAD_1" "THREAD_2" "THREAD_3"; do
  gh api graphql -f query="mutation {
    resolveReviewThread(input: {threadId: \"$thread_id\"}) {
      thread { id isResolved }
    }
  }"
done
```

### Phase 6: Quality Verification

```bash
# PRIMARY WEAPON
make check

# SECONDARY WEAPON
make test-integration

# Commit and push only if both pass
git add . && git commit -m "fix: description"
git push origin feature-branch
```

## Comment Categorization

### Priority Levels
- **HIGH (P0)**: Security vulnerabilities, architectural violations, breaking changes
- **MEDIUM (P1)**: Code quality issues, test failures, ignored parameters
- **LOW (P2)**: Code style, documentation, nitpicks

### Status Tracking
- **UNRESOLVED**: Not yet fixed
- **IN_PROGRESS**: Currently being addressed
- **VERIFIED**: Fixed and verified in code
- **RESOLVED**: GitHub thread marked as resolved

## Tracking Document Template

Create `docs/pr-comment-tracking.md` for each PR:

```markdown
# PR Comment Tracking Document

**Pull Request:** #NUMBER - Title
**Branch:** feature-branch-name
**Date:** YYYY-MM-DD
**Total Comments:** COUNT

## Summary
Brief overview of what comments address.

## High Priority Issues (P0)

### 1. Issue Title
- **Thread ID:** PRRT_xxxxx
- **Comment ID:** PRRC_xxxxx
- **File:** path/to/file.go:123
- **Issue:** Detailed description
- **Fix:** What was changed
- **Commit:** abc1234
- **Status:** ✅ VERIFIED & RESOLVED

## Medium Priority Issues (P1)
[Same format]

## Low Priority Issues (P2)
[Same format]

## Resolution Summary

**Phase 1: High Priority** - ✅ COMPLETED
- Fixed X issues in commits abc1234, def5678

**Phase 2: Medium Priority** - ✅ COMPLETED
- Fixed Y issues in commits ghi9012, jkl3456

## Quality Gates
- [x] All comments verified in code
- [x] `make check` passes - All tests, 0 lint issues
- [x] `make test-integration` passes - All integration tests
- [x] All GitHub threads resolved
- [x] Changes committed and pushed

---
**Zero tolerance means zero missed comments. No exceptions.**
```

## Common Mistakes to Avoid

### ❌ NEVER:
1. Start PR work without verifying GitHub CLI authentication
2. Resolve threads without verifying fixes in code
3. Use REST API without pagination handling
4. Assume comment IDs from one source include all comments
5. Mark resolved before running both weapons
6. Skip documentation because "it's obvious"
7. Ignore automated tool comments (Copilot, etc.)

### ✅ ALWAYS:
1. Verify GitHub CLI authentication before starting PR work
2. Use GraphQL API for complete thread retrieval
3. Read the actual code before claiming a fix
4. Document thread IDs, comment IDs, files, and lines
5. Run `make check` after each fix
6. Run `make test-integration` before pushing
7. Keep tracking document updated in real-time

## GitHub CLI Reference

### Essential GraphQL Queries

**Get all review threads:**
```bash
gh api graphql -f query='
query($owner: String!, $repo: String!, $pr: Int!) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $pr) {
      reviewThreads(first: 100) {
        nodes {
          id
          isResolved
          comments(first: 10) {
            nodes {
              id
              body
              path
              line
            }
          }
        }
      }
    }
  }
}' -f owner=OWNER -f repo=REPO -F pr=NUMBER
```

**Resolve thread:**
```bash
gh api graphql -f query='
mutation {
  resolveReviewThread(input: {threadId: "PRRT_xxxxx"}) {
    thread {
      id
      isResolved
    }
  }
}'
```

**Unresolve thread (if needed):**
```bash
gh api graphql -f query='
mutation {
  unresolveReviewThread(input: {threadId: "PRRT_xxxxx"}) {
    thread {
      id
      isResolved
    }
  }
}'
```

### Verification Queries

**Count unresolved threads:**
```bash
gh api graphql -f query='
query($owner: String!, $repo: String!, $pr: Int!) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $pr) {
      reviewThreads(first: 100) {
        nodes {
          isResolved
        }
      }
    }
  }
}' -f owner=OWNER -f repo=REPO -F pr=NUMBER | \
jq '[.data.repository.pullRequest.reviewThreads.nodes[] | select(.isResolved == false)] | length'
```

## Real-World Example

From PR #6 (11 comments addressed):

```bash
# 1. Retrieved all threads
gh api graphql -f query='...' -f owner=bmcszk -f repo=blackswan -F pr=6

# Output showed 11 unresolved threads with IDs like:
# - PRRT_kwDOPnswh85dvM3R (expectResponseCode)
# - PRRT_kwDOPnswh85dvM3T (withUserID)
# etc.

# 2. Verified each fix in code
cat services/mock-ib/simulator_test.go | grep -A 2 "expectResponseCode"
# Confirmed: Now uses 'code' parameter ✅

# 3. Resolved all threads in batch
for thread_id in "PRRT_kwDOPnswh85dvM3R" "PRRT_kwDOPnswh85dvM3T" ...; do
  gh api graphql -f query="mutation { resolveReviewThread(...) }"
done

# 4. Verified all resolved
# Query returned: 0 unresolved threads ✅
```

## Quality Gates

### Before Resolving ANY Comment:
- [ ] Read the actual code at line referenced
- [ ] Confirm issue exists (or existed)
- [ ] Verify fix addresses comment exactly
- [ ] Run `make check` - must pass
- [ ] Document fix in tracking document

### Before Marking PR Ready:
- [ ] All comments verified in code
- [ ] All GitHub threads resolved
- [ ] `make check` passes (0 lint issues, all unit tests)
- [ ] `make test-integration` passes (integration tests)
- [ ] Tracking document complete
- [ ] All changes committed with clear messages
- [ ] Changes pushed to remote

### Post-Merge:
- [ ] Archive tracking document for reference
- [ ] Update guidelines with lessons learned
- [ ] Share learnings with team

## Process Improvements

### Metrics to Track
- **Comment retrieval accuracy**: Target 100%
- **False resolutions**: Target 0
- **Time to resolution**: Target <24 hours
- **Quality gate failures**: Target 0

### Root Cause Analysis
When issues occur:
1. Document what went wrong
2. Update this guideline to prevent recurrence
3. Add verification step if needed
4. Review with team

## Final Checklist

Before claiming "all comments addressed":

- [ ] Used GraphQL API to get ALL review threads
- [ ] Verified EVERY fix by reading actual code
- [ ] Ran `make check` after each fix
- [ ] Ran `make test-all` before pushing
- [ ] Documented all thread IDs and resolutions
- [ ] Resolved all threads in GitHub
- [ ] Verified 0 unresolved threads remain
- [ ] Committed and pushed all changes

---

## 🚨 FINAL REMINDER

**Zero tolerance means:**
- Zero missed comments
- Zero unverified fixes
- Zero skipped quality gates
- Zero exceptions

**The GraphQL API method is MANDATORY** - it's the only reliable way to get complete thread information including IDs needed for resolution.
