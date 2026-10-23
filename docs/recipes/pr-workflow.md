# PR workflow, review rounds & close-out

Binding repo workflow rules. Canonical copy — keep in sync with AGENTS.md.

## Review-comment classes (binding: one comment = one CLASS — sweep WHOLE PR diff, never fix only the flagged line)
- "excessive comments" / "tldr" → collapse EVERY multi-line comment run in the PR diff to <=1 line, not just the flagged one.
- "should be a part of DSL" → receiver methods in scenario/test files; move to the owning `<domain>_parts_test.go` DSL file. Audit: receiver methods in non-parts `*_test.go` files = 0 (test files hold zero receiver methods).
- "dont you ever use func with underscore name!" → zero `func _*` anywhere, any file. Grep `grep -rn 'func _' --include='*.go'` must be 0 — including pre-existing code in files the PR touches.
- "wrongly placed" → symbol location, not wording (see layout table below).
- "pointers to pointers? overpointering?" → the class is ALL out-param pointer plumbing: `**T` AND `*[]T` params. Fix by returning values (`([]R, error)`); sweep `grep -rn '\*\[\]\|\*\*' --include='*.go'` non-test = 0. When a class applies to a function's params, fix EVERY param of that function in the same commit — the flagged param is never the only instance.
- "use simple test funcs not t.Run to reduce complexity!" → NO t.Run for simple cases; flatten subtests into plain test funcs (1:1, verbatim bodies, own given/when/then, func-name godocs). Sweep `grep -rn 't.Run(' --include='*_test.go'` = 0.
- "check github issues, own them in one feature branch, address one by one, PR" → full loop in "Issue-closure sweep" below; docs per user-facing behavior land in the SAME PR (see Definition of DONE).
- NEVER pass the body as `-f body=@-` + herestring: under subprocess/sh the stdin never reaches gh and it posts the LITERAL `@-` as the comment body. Pass argv-form: `['-f', f'body={text}']` as separate subprocess args (no shell), then GET-read back and assert body != '@-'.
- Expect placement+hygiene classes to recur in every round; sweep for them even when this round's reviewer didn't flag one.

## Post-feature simplification pass (standing policy: after each epic ships, BEFORE reviewer asks)
Recipe:
- READ each file END-TO-END before proposing changes — proposals drafted from grep hits alone duplicate existing structure and miss the file's own idioms.
- Ponytail ladder when choosing an implementation: reuse an existing primitive first → stdlib second → minimal new code last; never introduce an abstraction before the ladder is exhausted.
- Prefer DELETING lines over adding files; a simplification that only nets new files/symbols is scaffolding, not simplification.
- ONE lookup primitive per precedence chain (`lookupVar` replaced 4 parallel chains).
- ONE builder per repeated construction (`directiveResolveContext` replaced 3 inline literals).
- ONE runner per duplicated loop (kill `*T`/`**T` out-params; return `([]R, error)`).
- ONE loop-iteration runner shared by both executors; transient Request state kept as ONE named family of fields, not ad-hoc scattered members.
- No wrapper over stdlib (decodeJSONInto deleted).
- No interface/registry for a single implementation; keep a dispatch table only when the upstream reference demonstrably uses the same shape.
- File ceiling discipline: files approaching ~1000 lines get COHESIVE pure-move splits (parser_state.go 990→840 via parser_directives.go) — move a domain out whole, never sprinkle methods across files.
- Do NOT abstract: single-impl dispatch stays concrete (no interface/registry); bail-out handler tables match httpyac architecture.
- Research anchors: Go Wiki CodeReviewComments, Cheney Practical Go (handle errors once), httpyac metaDataHandler pattern.

## Definition of DONE for any feature PR (binding)
A feature task is NOT done when tests pass — it is done when the PR is complete for a REVIEWER:
1. Code + fluent tests (RED→GREEN discipline).
2. **docs/http_syntax.md section for every user-facing behavior added** — same PR, same round as the feature. "Fluent tests + docs" in a bean = docs is not an optional extra, it's an acceptance criterion equal to tests.
3. README only if it adds a top-level capability bullet (feature-level: yes; task-level within an epic: check whether the bullet already covers it).
4. PR body lists the docs commit alongside code commits.
Pre-completion self-check before reporting done: `git diff master...HEAD --stat` must show docs touched for any feature that changes user-facing syntax/behavior. If zero docs files → task is NOT done, do not post PoW, do not close the bean.

## Epic close-out (binding)
- After epic finish + PR merged to master: IMMEDIATELY switch to latest master (`git checkout master && git pull`) BEFORE anything else. Never keep working off a merged feature branch. Then: bean completed, feature branch deleted local+remote, stale worktrees removed.
- Squash merges break ancestry checks: `git merge-base --is-ancestor <branch> master` reports NOT-MERGED for squash-merged branches. Verify by PR association (`gh pr list --state merged`, match headRefName), then `git branch -D` local and `git push origin --delete` remote; never delete the branch of an OPEN PR.

## Review-round workflow (each round)

1. Read the round contract (per-comment actions + generalization rules + gates). Confirm every open decision is settled IN the contract: target dir for every new file/move, comment style for moved code (one-line, no "moved from X" banners), scope of file moves.
2. Run the generalization greps YOURSELF first; put concrete hit lists into the contract as numbered items. Reviewer-style workers miss pattern matches; the orchestrator sweep is the reliable one.
3. Delegate via pi (see pi-delegation skill: one contract file, `_minimax_first`, headless, in-repo). Runs read the prompt once — no mid-run addenda. Feature beans: two runs — RED contract pins the new API symbols, GREEN contract diffs Test-func inventory against a /tmp baseline.
4. Verify gates locally, in order: `go build ./... && go vet ./... && golangci-lint run ./...` (expect `0 issues`), `gotestsum --junitfile /tmp/<round>.xml -- -count=1 -cover ./...` (expect 0 failures), `gofmt -l` on touched files only (see pi-delegation: exclude .kilo worktrees), Test-func inventory parity (`grep -h '^func Test' *_test.go | sort` before vs after; expect drift as rounds add tests).
5. Resolve every "wrongly placed" / "excessive" review comment against the DSL file layout table BEFORE drafting the fix — those phrases flag symbol location, not comment wording; editing prose leaves the violation in place. New `parts` methods land in their canonical file (parse-when/fixture helpers → ext), never in scenario files. Audit the whole feature diff: `git diff master -- '*_test.go' | grep -E '^\+func '` must show only `func Test` additions outside canonical DSL files — a stray `func (p *parts)` in a scenario file is a move-back, not a comment fix.
6. Check the diff for contract violations the gates cannot see: banner comments, invented directories, methods moved against the layout above. Fix locally, re-run gates, report PLAIN TEXT (per-comment changes + file-split inventory + generalization hits + gate results), leave UNCOMMITTED unless told otherwise.
7. Before any local fix after a delegation, diff `git status` against your last snapshot — unexpected changes mean another actor is mutating the tree; stop and flag (see pi-delegation concurrency guard).
8. Close the round on GitHub: reply to EACH inline review comment with the fix SHA (one reply per comment thread, never a single summary comment), then verify every reply POST with a follow-up GET — reply POSTs 404 mid-loop; re-POST until the GET returns the reply.

## Issue-closure sweep ("address all gh issues" / "find and address another gh issue")
- Enumerate with `gh issue list --state open` AT dispatch time — never reuse a count from earlier in the session; issues land while you work. Cross-check CLOSED issues too: a prior merged PR body may claim "Closes #N" while only a subset landed — grep master for the feature before trusting the closed state.
- Per-issue closure loop: bean (task) → branch off master → contract-file dispatch → OWN green run of the NEW tests before committing (child's self-report is a hypothesis, not a result) → commit `feat: ... (#N)` with `Fixes #N` footer → push → PR (body = what it does + PoW; footer auto-closes issue) → poll `gh pr checks` → assert local HEAD == PR head → fix-SHA comment on the issue → bean completed.
- Stacked fixes ride one branch/PR: commit on top, update the PR body, re-watch CI — don't multiply PRs; the reviewer gets one review.

## Repo workflow (binding)
- Epic bean stays `in-progress` while its PR is open even when every child is done — it closes only after the PR merges. Don't "fix" it to completed early.
- Master is PR-only (protected, GH006 on direct push). Even beans/docs chores ship as a tiny PR: `git checkout -b chore/...`, push, `gh pr create`, `gh pr merge --squash --auto` once checks pass.
- A commit failing with `Couldn't get agent socket?` means that shell lacks SSH_AUTH_SOCK: re-run with `export SSH_AUTH_SOCK=~/.ssh/agent-env/agent.sock &&` prefixed.
