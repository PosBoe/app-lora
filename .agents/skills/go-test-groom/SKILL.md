---
name: go-test-groom
description: Run the full Go test grooming pipeline on a Go package in one prompt — reshape test granularity and file layout, decouple assertions from implementation, fill coverage gaps, optionally clean up readability, then always audit determinism and test wall-clock time. Invoke ONLY when the user explicitly runs `/go-test-groom` or explicitly names this skill — do not infer it from generic phrasing like "groom tests", "overhaul the tests", or "modernize the tests" alone; treat those as ordinary requests instead.
---

# go-test-groom

## Repo adaptation

This skill came from the reference repository and keeps its full method. This
repository differs in these ways; where the text below disagrees, these lines win:

- One deployment build. The second lane is `-tags=invariants`, not `-tags=cloud`.
  There is no `featflag.JSON`, `featflag.Billing` or premium tag.
- No `pkg/testbed`, `pkg/ddtest`, `dbfixture` or shared fake clock (`unixtime`,
  `schedtest`). An integration test dials the `lora_test` database
  (`docs/go-test-conventions.md`). Where the text says "bed" or "testbed", read
  "the package's own setup that dials PostgreSQL".
- No `cmd/gotest`. The lanes are the `task test:*` tasks in `Taskfile.yml`.
- No go-snaps. Where the text says "snapshot", use a `pkg/datadriven` golden.
- PostgreSQL only: no ClickHouse, Kafka or Redis.
- A file the text cites that this tree does not hold is an example from the
  reference repository, not a path to open.

Orchestrate the `go-test-*` suite over one Go package. Each phase is its own skill; this skill fixes the order, shares one baseline, runs the expensive gates once instead of per phase, and merges everything into a single report.

## Pipeline

| Phase | Skill | Fixes |
|---|---|---|
| 1 | `/go-test-reshape` | Granularity + layout: consolidate fragments, split switchboards, move misplaced tests |
| 2 | `/go-test-decouple` | Assertions retargeted from internals onto the public contract |
| 3 | `/go-test-coverage` | Cases added for uncovered public behaviors |
| 4 — only when polish was requested | `/go-test-cleanup` | Scoped types, inlined helpers, semantic assertions |
| 5 | `/go-test-determinism` | Real-clock reads, literal fake values, unscoped deletes, global-state races, order dependence, goroutine leaks |
| 6 | `/go-test-speedup` | Wall-clock time over the budget: sleeps, stress loops, `-short` gating |

The order is load-bearing: structure settles first so later phases don't edit tests that are about to move; assertions are corrected second so the coverage phase copies the right assertion style into its new cases; coverage runs before the audits so new cases land in the final structure. Phase 4 is not part of the core pipeline — include it only when the user also asked for readability/polish.

**Phases 5 and 6 always run**, with or without phase 4. They come last because they audit the suite the earlier phases produced: phase 3 adds new cases, and each new case is a new chance to read the real clock, assert on a fake value, or add seconds to the run. Running them earlier audits a suite that no longer exists.

## Procedure

Read `references/verification.md` first — it defines the scratch-path
convention, the two-tag baseline rule, the gate list, and the
proof-of-no-loss recipes this whole suite shares. Everything below assumes it.

### 0. Consider a lock-in pass first

If the package's testdata/snapshot history is heavy with regeneration
(`git log --oneline -20 -- <pkg>/testdata/ <pkg>/*.snap`), run
`/go-test-lockin` **before** grooming and surface its findings to the user.
Grooming a suite that pins bugs polishes the lock-in, and phases 1 and 3 below
each run `-rewrite`, which re-materializes whatever the code does today.

### 1. Baseline once

Follow `references/verification.md` §1–§3. Concretely, with `<pkg-slug>` the
package path with `/` → `-`:

```
mkdir -p /tmp/gotest-skills/<pkg-slug>
go test -p 1 -coverprofile=/tmp/gotest-skills/<pkg-slug>/cover.baseline.default.out ./<pkg>/...
go tool cover -func=/tmp/gotest-skills/<pkg-slug>/cover.baseline.default.out | tail -n 1
go test -p 1 -v ./<pkg>/... 2>&1 | grep -E '^--- (PASS|FAIL)' > /tmp/gotest-skills/<pkg-slug>/tests.baseline.default.txt
```

Plus the `-tags=invariants` lane when §2 of the protocol says the package needs it
(build-tagged files or `featflag` reads) — writing `…cover.baseline.invariants.out`,
`…tests.baseline.invariants.txt` and `…time.baseline.invariants.txt`. The rule covers the
timing artifact below like every other one.

Every sub-skill opens with its own baseline step. **Skip those and reuse these
files**, passing the phase name so the artifacts stay distinct: each phase reads
the previous phase's `after` as its `before`, and writes
`cover.<phase>.<tag>.out` / `tests.<phase>.<tag>.txt` on the way out. The
sub-skills' own examples say `before`/`after`; substitute the phase names so
nothing is overwritten and the final report can attribute movement per phase.
The names are `references/verification.md` §1's, and every phase has one:

```
baseline → reshape → decouple → coverage → cleanup → determinism → speedup
```

A phase left out of that substitution keeps the sub-skill's `before`/`after`,
which the phase before it already claimed — so the earlier phase's evidence is
overwritten and its movement can no longer be attributed.

Phase 6 is the exception: it needs per-test *timings*, not coverage, and it must
measure the suite the earlier phases produced. Let it take its own timing
profile when it starts. Capture the wall-clock total here as well, so the report
can state the before → after time.

Run the timing recipe in `references/verification.md` §3, which writes
`time.baseline.<tag>.txt`. It measures both modes — the unqualified number is
what the nightly suite pays, the `-short` number is what every push pays — and
it keeps the run's exit status, which a pipe into `tail` would throw away. Do
not re-spell it here.

### 2. Run the phases

Invoke each phase with the Skill tool — `go-test-reshape`, `go-test-decouple`, `go-test-coverage`, then `go-test-cleanup` if polish was requested, then `go-test-determinism`, then `go-test-speedup` — passing the package path and any scoping the user gave. Finish a phase completely — including its package-level verification — before starting the next. If the Skill tool is unavailable, read the phase's instructions from `.agents/skills/<name>/SKILL.md` and follow them directly.

Adjustments to the sub-skills (each is written to also run standalone):

- **Baselines**: reuse per step 1 instead of re-capturing from scratch.
- **Gates**: after each phase run only gate 1 (`go test -race -count=1 ./<pkg>/...`, both tags when the package needs it). The sub-skills finalize at gate 2 (`task test:pkg`) — skip that per phase and run it once after the last phase, then gate 3 (`task test`) once at the very end. See the gate list in `references/verification.md` §5.
- **Reporting**: don't emit each phase's report as you go; collect the material and merge it into the single final report.

### 3. Skip honestly

A phase whose "when to apply" criteria find nothing (already right-shaped; no coupled assertions; nothing uncovered that's observable) records "no changes" with one line of evidence and moves on. Don't force edits just to have something to show.

Two phases are held to a stricter standard, because a green run cannot show what they hunt:

- `/go-test-decouple` — its own instructions explain why "nothing to fix" is usually a scan failure. Hold it to its full inventory requirement, and carry the inventory evidence into the combined report.
- `/go-test-determinism` — the defects it finds do not fail today, so "the tests pass" is not evidence. It reports clean only after it runs its own audit over the whole package. Carry that evidence into the report the same way.

`/go-test-speedup` is the one phase that legitimately stops early: run it always, let it measure, and if the suite is under its time budget it records the measured time and does nothing. A skipped measurement is not the same as a measurement under budget — state which one happened.

### 4. Mind the handoffs

The phases interact; pass information forward instead of rediscovering it:

- Reshape translates assertions as-is — coupled assertions survive translation, so decouple still audits everything, including freshly consolidated tests.
- If decouple deletes a test that asserted nothing observable, hand the behavior it nominally covered to the coverage phase as a candidate to re-cover through the public API.
- If decouple's removals dropped incidental coverage, the coverage phase restores it only where the behavior is observable; otherwise the final report explains the drop.
- Give the determinism phase the list of cases phase 3 added and phase 1 moved. A new case is where a real-clock read or a fake literal enters, and a moved case can land in a file whose fixture scoping differs from the one it left.
- Give the speedup phase the same list, plus the baseline time from step 1. A suite that went over budget during the run tells you which new case did it; a suite that was over budget before the run is pre-existing, and the report says so.
- If speedup gates a test behind `testing.Short()`, that test leaves every default lane — per-push CI, `task test`, `test:pkg`. Say so in the report. It does **not** leave the coverage baseline: `references/verification.md` §3 takes both profiles without `-short`, so the gated test still runs for coverage and the two numbers still compare. A coverage drop there is a real one, not an artifact of the gate. Report a separate `-short` coverage figure only where the gate's cost to the default lanes is the point, and label it as its own measurement.

### 5. Final gate and report

```
go test -race -count=1 ./<pkg>/...
task test:pkg -- ./<pkg>/...     # both build tags
task test                        # full pre-PR gate, once, here only
```

The race gate is load-bearing after phase 5, for one half of what phase 5
audits. `-race` proves a fix to a data race on shared state. It proves nothing
about a goroutine leak: it does not fail because a goroutine outlived the test
that started it. `goleak.VerifyTestMain` in the package's `main_test.go` is the
gate for that half, and it runs in every lane, `-race` included — so read the
leak fix against a green run of that, and where the package has no
`VerifyTestMain`, against an assertion on the goroutine's own lifecycle.

Then close the loop on what the pipeline itself manufactured: phases 1 and 3 run
`-rewrite`, so every regenerated golden is a fresh lock-in candidate. Name them
in the report, and recommend `/go-test-lockin` when more than a couple changed.

## Output to the user

One combined report:

- **Header**: package; phases run/skipped (with skip evidence); tests + cases before → after; coverage baseline → final, attributing movement per phase; suite wall-clock time before → after.
- **Per phase**: that phase skill's own report format, condensed to what changed.
- **Handoffs**: anything one phase left for another — deleted tests re-covered, API gaps flagged, coverage drops explained.
- **Determinism**: what the audit covered, and each defect fixed with the failure it would have caused. Clean means audited-and-clean; say that, not "passes".
- **Time**: measured time against the budget, and what phase 6 changed to get under it. Name every test newly gated behind `testing.Short()` — those leave the default lanes.
- **Review pointers**: rewritten goldens/snapshots worth a human check.

Stay terse. The diff is the artifact; the summary just routes attention.

## The test-* suite

This skill orchestrates the pipeline:

1. `/go-test-reshape` — granularity and file layout.
2. `/go-test-decouple` — retarget assertions from internals onto the public contract.
3. `/go-test-coverage` — add cases for uncovered public behaviors.
4. `/go-test-cleanup` — readability polish (optional phase).
5. `/go-test-determinism` — audit flake and cross-test-interference sources (real clock, literal fake values, unscoped deletes, goroutine leaks).
6. `/go-test-speedup` — measure the suite against its time budget and cut what puts it over.

`/go-test-datadriven` owns the datadriven authoring idioms the others lean on, and `/go-test-to-datadriven` executes a user-directed conversion of named tests.

One sibling sits outside the pipeline; this skill does not run it, but it brackets the pipeline with it:

- `/go-test-lockin` — audits whether the expectations are *true*. Run it **before** grooming when the package's golden history is churny (step 0), and **after** when the pipeline regenerated goldens (step 5).

## Additional resources

- `references/verification.md` — the suite-wide verification protocol: scratch paths, two-tag baselining, `-short` semantics, integration-DB prerequisites, the gate list, and the proof-of-no-loss recipes. Every sibling skill cites it.
