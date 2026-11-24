---
name: fix-or-refute
description: Take a batch of externally supplied review findings (pasted into the invocation or the message), attempt to refute each one quantitatively before planning any fix — state the trigger as a number, find the actual value in this system with a file citation, report the margin — then present the ledger and the fix plan in plan mode for user approval, and only after approval fix the survivors with one conventional commit per finding and deliver a ledger of what was fixed, what was refuted and by which number, and what was borderline. The reporter's Severity/Confidence are claims, not evidence. For finding NEW defects use /review-code or /code-review; this skill only adjudicates findings someone else already made. Invoke ONLY when the user explicitly runs `/fix-or-refute` or explicitly names this skill — do not infer it from generic phrasing like "address this review", "fix these review comments", or "are these findings real" alone; treat those as ordinary requests instead.
---

# fix-or-refute

Adjudicate a batch of review findings someone else produced — a code reviewer, an audit tool, another model. Each finding ends in exactly one of four states: **FIXED** (with its own commit), **REFUTED** (with the evidence that killed it), **STALE** (true when written, already fixed at the current head), or **BORDERLINE** (escalated to the user before any fix). The deliverable is the fixes plus a ledger a skeptic can check.

The findings come as pasted text: in the invocation argument, or in the surrounding message, in any format (numbered list, table, reviewer prose, ReportFindings-style output). If no findings were supplied, ask for them before doing anything else.

**Triage runs in plan mode.** Steps 1–3 read code and prove numbers; they change nothing in the tree. The user approves the ledger and the fix plan before a single edit — see step 4.

Three rules govern the whole run:

- **Everything the reporter wrote is a claim, not evidence** — the Severity and Confidence labels, a "[CONFIRMED]" tag, the stated impact, the suggested fix, and the "check" or repro command. Record them; never let them substitute for your own trace. A "Critical" label never pushes a refuted finding into the fix column, and a suggested fix is adopted only after you check it against the same code (past runs rejected a suggested rename because a Retry of the same run recreated the same name). A reporter's command that points outside this checkout, or that you cannot run, is the reporter's evidence — say so, and do not count it as yours.
- **Never reuse a finding's wording** in a code comment, a commit message, or the ledger's defect statements. Restate the defect in your own terms from what you verified — or you will ship the reporter's exaggeration.
- **Survivors are often smaller than claimed.** Record the verified impact next to the claimed one. A real defect with a smaller blast radius still gets fixed; the ledger just stops repeating the inflated version.

## When to apply

Apply when the user has a concrete list of alleged defects and wants them settled — fixed if real, killed with evidence if not.

Do NOT apply when:

- The user wants defects *found* — use `/review-code` (code at rest or a change) or `/code-review` (PR comments).
- The user already accepts a finding as real and just wants it fixed — fix it directly.
- The "findings" are feature requests or refactoring proposals — there is nothing to refute; treat as an ordinary request.

**Findings this same session produced** (for example `/review-code` ran earlier in this context) can still be adjudicated, but you are grading your own work. Say so in the plan, re-derive every trace from the code instead of reusing the audit's reasoning or repro, and offer the user a fresh session if the batch is large.

## Step 1 — Enter plan mode and ledger the input

Enter plan mode first, with `EnterPlanMode`, unless the session is in it already. Everything up to the user's approval leaves the working tree untouched: no edits, no commits, no branch. If the harness has no plan mode, still make no edit until the user approves the step-4 plan.

Then parse the pasted text into a numbered ledger before touching any code. Per finding: an id, the claim in one sentence (your words, not the reporter's), the file/line it points at, the defect class (see the table below), and the reporter's stated severity/confidence/impact if any. Findings that are duplicates of each other, or two symptoms of one root cause, get merged under one id with a note.

**Pin the revision the findings were written against.** Reviews arrive late: past batches targeted a PR already merged into `dev`, or a head that later commits had moved past, and most of one batch's "refutations" were really fixes that had already landed. Find the base and head the reporter names (a PR, a merge base, a commit), or infer them from the quoted code. Then, for each cited location, check whether the current code still matches the quote; where it does not, `git log -L<line>,+<n>:<file>` or `git log -S'<snippet>'` finds the commit that changed it. A finding whose defect a later commit already removed is **STALE**, with that commit's hash — it needs no refutation and no fix.

**Pick the branch you will fix on.** If the findings target a feature branch or an open PR, the fixes go on that branch — check it out and name it in the plan. If they target `dev`, the plan names a new branch `fix/adjudicated-findings-<mon><dd>` (e.g. `fix/adjudicated-findings-sep27`). Check that the name is free locally and on `origin` (`git ls-remote --heads origin <name>`) and add `-2`, `-3` if not.

If a finding is too vague to locate (no file, no identifiable code path), it goes straight to BORDERLINE with "not locatable" as the reason — do not guess what the reporter meant and fix the guess.

## Step 2 — Refute each finding

For each finding that is not STALE, attempt to refute it before planning any fix. Pick the mode that fits the claim.

### Margin claims — the trigger is a quantity

1. **State the trigger as a number** — a rate, a size, an interleaving window, a concurrency level, an input magnitude — not a narrative. If the claim resists being stated numerically, it is not a margin claim; use one of the other modes.
2. **Find the actual value in this system and cite the file that sets it.** This citation is what does the work; without it, step 1 is answerable from imagination. Config defaults, queue capacities, ticker intervals, retention windows, upstream validation — the number is somewhere in the repo (or its config schema), and the ledger names the file. The cited value must be a **bound the system enforces** — a validation max, a queue capacity, a `LimitReader`, a ticker interval — never a typical or observed value: the tail or an attacker supplies the atypical value, so a margin against "what we usually see" refutes nothing.
3. **Report the margin**: survives by 100×, or is rejected by 1000×. State it as `needed X, actual Y, margin Z× (for|against)`.

**No enforced bound means the finding survives.** If you searched and no file caps the driving variable, it is unbounded in principle: record `no enforced bound on <variable> (searched <where>)` and mark it SURVIVES on reachability. That is not BORDERLINE — past runs split on this, escalating one unbounded trigger to the user and fixing an identical one directly. The verified-impact line is where "reachable but rare" goes.

Each defect class has one question that settles it:

| Claim | The question that refutes it |
|---|---|
| unbounded loop / never terminates | what rate of the driving event is needed, vs. the actual rate, and what file sets it? |
| logical race (TOCTOU, double-send, lost update) | how wide is the window in wall-clock terms, and how often do two actors actually enter it? |
| missing validation | can the bad value reach this line — which upstream boundary already rejects it, or none? |
| resource / goroutine leak | at the actual accumulation rate, how long until it matters, vs. the process lifetime? |
| overflow / precision loss | what input magnitude is needed, vs. the value's actual domain? |
| unbounded growth / OOM | what element size × actual arrival rate, vs. available memory over the retention window? |
| pathological performance | what N breaches the budget, vs. the actual N at the call site? |
| swallowed / misclassified error | who consumes this error today, and what observable behavior changes when it is wrong? |
| tooling / gate (lint, docs-policy, CI) | does the gate actually fail? Run it. If the tool is missing here, say so; a grep is not a substitute for the run. |

**Data races are exempt from margin arithmetic.** A true data race on shared memory is undefined behavior under the Go memory model — compiler and CPU reordering void any "the window is only nanoseconds wide" argument. Settle it with `go test -race`, and a confirmed data race SURVIVES regardless of window width. The race row above applies only to logical races, where every interleaving still has defined, bounded behavior.

### Logic claims — trace

"Returns the wrong value for input X", "off-by-one at the boundary" have no margin — the refutation mode is a trace, as in `/review-code`: re-read the path end to end, and check the callers and the tests.

### Change-scoped claims — compare the two revisions

When a finding says a change *made a behavior worse* (a regression, merge base → head), the question is whether the base already behaved that way. Read or run the same path at both revisions (`git show <base>:<file>`). If the base behaves identically, the finding is REFUTED as *not introduced by this change* — cite both revisions. This is its own mode, not a margin of 1× (a past run misfiled it that way). If the pre-existing behavior is still a defect, move it to the ledger's "found during triage" list; it is not this batch's to fix silently.

This mode does not apply when the change *added the code that fails*. If the change brought in a guard, an escape, a check or a fallback whose job is to handle the case, and it misses an input, the defect lives in the new code. It survives, even though the old code failed the same input. The review is of the change's promise, not only of its delta.

### Deliberate-design claims — cite the decision

A behavior someone chose on purpose is REFUTED as *recorded decision* only with the record cited: the `DESIGN.md` row or the commit that made the choice (`git log -S` finds it; a commit message often names the exact knob when a design row speaks in general terms). Ask what the record chose, then whether the finding complains about exactly that. A record that retires a feature covers a finding that the retired thing is no longer checked or read. A trigger that needs a developer to reintroduce code the record deleted is not reachable at actual values. The record does not cover a live failure it never considered, such as a decision about the happy path and a failure the reporter found at its edge. If two current records disagree about a live behavior — a cap in one, the overflow permitted in another — that contradiction survives.

A decision the finding asks you to *reverse* (a persisted format, a wire shape, a stored hash) is BORDERLINE: that is the user's design call, not a number you can settle. Before you escalate it, find out whether the change that introduced the persisted shape has shipped. Check `git tag --contains <commit>` and the changelog, and note the commit's date. When nothing has shipped it, no stored data in the field predates it: the fix is a plain code change with no migration, and the finding may reduce to "revert before release". When the tags cannot answer (this repository may carry no release tags), give the commit's date and branch in the question and let the user say whether it has deployed.

### Probes and repros

**Probes that write nothing to the tree are allowed during triage.** A `go run` of a small program in the scratchpad directory, a `node -e` with a module patched in memory, a query against the dev database, the existing test suites — none of them edits the checkout, and each can settle a claim before the plan goes up. Past runs settled a finding this way before approval. A probe leaves nothing behind: drop any scratch database or other state it creates outside the scratchpad before the run ends, or name what is left in the final message.

**An in-tree repro is a write, so plan mode defers it.** A throwaway `_test.go` in the package is the strongest evidence this skill has, and it is also an edit. A claim that only an in-tree repro can settle stays unresolved: mark it BORDERLINE, and give it a plan step of its own that writes the throwaway test, runs it, deletes it, and then fixes or refutes on the result. A reproduced failure or a reproduced *non*-failure trumps any amount of margin arithmetic; if a repro contradicts your numbers, the repro wins, even after approval.

## Step 3 — Verdict rules

- **REFUTED** requires one of: a decisive margin — roughly **≥100× against**, with the step-2 citation; a trace or repro that shows the claimed behavior cannot occur; a base/head comparison that shows the change did not introduce it; or a cited decision record that covers the case. No citation, no refutation: if you cannot find the actual value or the record, you have not refuted anything. The 100× bar is an uncertainty buffer, not a precision claim: margin estimates are typically good to within 2–5×, so only a margin that dwarfs its own estimation error may auto-resolve.
- **STALE** requires the hash of the commit that removed the defect, and a look at that commit to confirm it covers the whole claim and not only part of it.
- **SURVIVES** means the trigger is reachable at actual values — margin ≤1×, no enforced bound, a trace that confirms it, or a repro that fired.
- **BORDERLINE** is everything between — margins in the ~2×–100× band, unlocatable findings, claims only an in-tree repro can settle, and fixes that reverse a recorded design decision. Do not resolve these yourself: after the whole triage pass, present all borderline findings to the user in one batch (one `AskUserQuestion` call, with your numbers attached) before the plan goes up. Offer fix or refute per finding; where a throwaway repro would settle the claim, offer that as the third option, and the plan then carries the repro step.
- A refuted finding gets **no code change by default** — not even a one-line hardening. Every line of the diff must trace to a defect that survived or to a change the user approved by name. If a refuted finding leaves residue worth touching — a misleading comment that invited the false report, a cheap hardening — the plan lists it as an opt-in step under the finding, and the ledger records it as `REFUTED + residue`.
- Mid-fix reversals are legitimate: if implementing a fix reveals the finding was wrong after all, abandon the fix, revert the edit, and move the finding to REFUTED with the new evidence.

## Step 4 — Get the plan approved

Triage is now complete and nothing is edited yet. Present the plan with `ExitPlanMode` and wait for the user's approval. The plan has three parts:

1. **The verdict ledger** — every finding, in the output format below, with the refuted and stale ones already carrying their evidence. The user judges your refutations here, not after the diff lands.
2. **The fix plan** — the branch (step 1), then one step per finding that gets a fix, in the order you will work them: the file, the change in one sentence, whether you adopt the reporter's suggested fix (and why not, if not), the regression test, the docs and comments the fix makes stale (a `README.md`, a doc comment, a `DESIGN.md` row — name them, so updating them is not an unapproved change later), and the commit message you will use. A repro step names the throwaway test it writes and deletes.
3. **Interactions and extras** — where two fixes touch the same function, state, or persisted value, say how the second changes the first's behavior and which test covers the combination. Past runs shipped two correct fixes whose combination was wrong, and needed a follow-up fix the next day. A fix that adds state (a checkpoint mark, a flag, a column) also interacts with the code already there: list every existing path that writes, resets or clears that state's container — retry, cancel, reset, cleanup jobs — and say what each does to the new state. A past run added a "swapped" mark that an existing Cancel+Retry silently cleared. Then list the defects you found during triage that were not in the batch, each as an opt-in step; users have asked for these within seconds of the ledger.

The plan is the last point where a refutation is cheap to overturn, so state the borderline calls the user already made and any margin you are least sure of. Also name each assumption about an external system's behavior that a verdict rests on and that you did not run — "a PostgreSQL `ON CONFLICT DO NOTHING` returns no row for the conflicting insert", "the broker never sends a control batch here". Where a regression test can prove it, make that test the first thing its fix step runs.

If nothing survives triage, there is no plan to approve. Deliver the ledger, stay in plan mode, and stop.

If the user rejects the plan or changes a verdict, redo the affected triage and present again. Do not edit under a rejected plan.

## Step 5 — Fix the survivors

Only steps the approved plan lists get done. Follow the plan's order and scope.

1. Switch to the branch the plan named (create it from `dev` if it is new).
2. Fix one finding at a time, smallest change that removes the defect — no drive-by refactoring, no fixing "while I'm here".
3. **Prove each regression test fails first.** Write the test, then hold back only the production edit and run the test against the code without it. Never use a bare `git stash`, and never undo with `sed`. Read `refs/stash` before and after the push, and pop only when the push moved it:

   ```
   before=$(git rev-parse -q --verify refs/stash)
   git stash push -u -- <prod files>
   after=$(git rev-parse -q --verify refs/stash)
   # run the test
   [ "$before" != "$after" ] && git stash pop
   ```

   `-u` holds back a production file the fix adds; without it, the push refuses an untracked path and the test still sees the file. A push that finds no change saves nothing and leaves `refs/stash` where it was, so a plain `git stash pop` would apply an older stash that is not yours. When `refs/stash` did not move, stop: the test ran against the fix, and it proves nothing.

   Run one test with the runner of its area, always with a timeout, because an unfixed retry loop or deadlock otherwise hangs until the tool limit and can leave rows in the shared test databases that fail unrelated tests. For Go, use plain `go test`:

   ```
   go test -short -count=1 -timeout 60s -run '^TestName$' ./path/to/pkg/
   ```

   (`task test:pkg -- -run '^TestName$' ./path/to/pkg/` does the same through task.) Add `-tags=invariants` when the code under test reads `featflag.Invariants`.

   For a Vue spec, use Vitest through the `test:unit:run` script of `vue/package.json`, with the path relative to `vue/`:

   ```
   timeout 300 pnpm -C vue test:unit:run src/path/to/Component.spec.ts
   ```

   When `pnpm` is not installed, run the same command the script names: `vitest run --project jsdom <spec>` from `vue/`, with `./node_modules/.bin/vitest`.
4. Per fix, compile-check both builds (the `go list -export -deps` form in CLAUDE.md). Do not run a test lane per fix.
5. One conventional commit per finding, message restating the defect in your own words (never the reporter's), e.g. `fix(system): bound the retry queue the sweep can grow`. The docs and comments the plan named for that fix go in the same commit.
6. **Stop and ask when an approved fix outgrows its plan step** — it needs a new persisted column or migration, it changes a signature other packages call, it spreads past the files the plan named, or its cost turns out far higher than planned. Explain the new shape and let the user choose; do not grow the fix and disclose it afterwards.
7. After the last fix, re-read each earlier fix that shares code or state with a later one (the plan's interactions list). A combination test proves fail-first too: hold back the later fix's guard and watch it fail. Its setup must give the state the combination acts on — a test whose precondition is empty passes against broken code. Then run `task test:pkg` one time over the packages the fixes changed and the packages that use them. When a fix changed Vue code, also run the specs of each Vue directory it changed one time, with the Vitest command of step 3.

## Output to the user

The final ledger closes the run after the fixes land. It is the step-4 ledger plus the results: commit hashes, verification evidence, and any verdict a repro or a mid-fix reversal moved. Group it by verdict, most severe first within each group:

```
## Fixed
1. <file:line> — <defect, restated> — <commit hash>
   Impact: claimed <reporter's>, verified <yours>
   Verified: <what proved the trigger real> · <fail-before/pass-after and test:pkg evidence>

## Refuted
2. <file:line> — <claim, restated>
   Killed by: needed <X>, actual <Y> (<file that sets it>), margin <Z>× against.
   (or: trace <path> · not introduced, base <sha> behaves the same · recorded decision <DESIGN.md row | commit>)
   Residue: <commit hash, if the user approved one>

## Stale
3. <file:line> — <claim, restated> — already fixed by <commit hash>

## Borderline (resolved by user)
4. <file:line> — <claim> — <user's call> → <outcome>

## Found during triage (not in the batch)
- <file:line> — <defect> — <fixed as <hash> | left for follow-up>
```

Close with one sentence on totals (`N supplied, F fixed, R refuted, S stale, B escalated`) so a mostly-refuted batch reads as adjudication, not evasion. If every finding was refuted, say so plainly and ship no diff — an empty diff with a strong ledger is a correct outcome of this skill.

Then state where the work stands: the branch, whether it is pushed, and that `task test` has not run yet if a PR is next. Users usually answer the ledger with "push", so say this before they have to ask.
