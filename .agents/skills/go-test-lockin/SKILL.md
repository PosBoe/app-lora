---
name: go-test-lockin
description: Audit a Go package's tests for locked-in wrong behavior — expectations that pass but encode a bug, a spec violation, or a bad contract, sometimes even with a comment rationalizing them. Independently derives what each suspicious case SHOULD produce (from docs, spec, math, callers) before trusting the asserted value, then reports each confirmed lock-in with why it is wrong and what the correct behavior is. Reports only; it never fixes — fixing requires changing product code, which the user decides. Invoke ONLY when the user explicitly runs `/go-test-lockin` or explicitly names this skill — do not infer it from generic phrasing like "check the tests", "are these tests right", or "audit the tests" alone; treat those as ordinary requests instead.
---

# go-test-lockin

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

Audit a Go package's tests for **locked-in wrong behavior**: cases where the test passes, but the expectation itself encodes a bug, a spec violation, or a contract nobody would defend if asked. The deliverable is a short, ranked list of such lock-ins — each with the pinned behavior, why it is wrong, and what the correct behavior (and test) should be. Report and stop: every real finding implies a *product code* change followed by an expectation change, and that is the user's call.

The target package comes from the invocation argument (e.g. `/go-test-lockin ./system`). If none was given, ask which package before reading anything.

This is the inverse of the usual test audit. `/go-test-decouple` asks *how* tests assert (internals vs. contract); `/go-test-coverage` asks *what's missing*. This skill asks whether **what the tests assert is even true**. A green suite proves the code does what the tests say — it proves nothing about whether what the tests say is right. Lock-ins are born three ways:

- A golden/snapshot was **regenerated without review** (`-rewrite`, `UPDATE_SNAPS=true`) and the expectation became "whatever the code did that day", bug included.
- A test was **written against the implementation** — the author ran the code, observed the output, and asserted it, never independently checking it was correct.
- A failing test was **"fixed" by updating the expectation** instead of the code, sometimes with a comment explaining the surprise away.

## When to apply

Apply when the user wants the *truth* of a package's test expectations audited: do the pinned outputs match the documented contract, the spec or standard being implemented, the domain math, and what real callers need?

Do NOT apply when:

- The concern is brittle assertion *targets* (mocks, internals, error strings) — `/go-test-decouple`.
- The concern is missing cases — `/go-test-coverage`; readability — `/go-test-cleanup`; structure — `/go-test-reshape`.
- The concern is expectations that are *unstable* rather than untrue (flakes, order dependence, literal fake values, cross-test interference) — `/go-test-determinism`. The two audits meet on regenerated goldens: a golden that churns because the output is nondeterministic is that skill's finding, while a golden that is stable and wrong is this one's.
- The subject is the production source itself — `/review-code` (though the two overlap: a lock-in found here usually points at a source bug there).
- The user disagrees with a *design* the tests correctly pin ("I'd have made this API different") — that is `/plan-ideal-refactor` territory, not a wrong expectation.

## The bar for "wrong"

"I would have designed this differently" is not a finding. A locked-in behavior is wrong only if at least one of these holds:

1. **Contradicts the written contract** — godoc, README, CONTEXT.md, an RFC/standard the package implements, or a repo-wide rule in CLAUDE.md.
2. **Contradicts the math/domain** — a value you can independently derive (a percentile, a rate, a boundary, a unit conversion) and the derivation disagrees with the golden.
3. **Contradicts the code's own evident intent** — the function is named `Validate` and the test asserts invalid input passes; the case is named `rejects empty input` and asserts no error.
4. **Damages a real caller** — a consumer (another package, the frontend, a third-party API contract being emulated) demonstrably relies on the behavior being otherwise.
5. **Internally inconsistent** — two cases in the same suite assert contradictory principles for symmetric inputs, so at least one of them is wrong.

Deliberate bug-compatibility is the standing exception: matching a third-party API's quirks (e.g. Grafana/Tempo semantics), preserving wire compatibility, or a documented deprecation path can make "wrong" behavior the correct contract. Evidence of deliberateness (a comment citing the constraint, a doc, a linked issue) downgrades a finding to "confirm the constraint still holds" — or kills it.

## Lock-in catalog

What to hunt for, and why each pattern is a red flag rather than proof:

### 1. Comment-excused surprise

A comment on an expectation that explains *away* a surprise: "this is weird but", "for some reason", "surprisingly", "note that X returns Y here", "for now", "quirk of the parser". A justified expectation cites a constraint ("Tempo encodes durations as strings — match it"); a rationalized one just narrates the code's behavior ("the parser eats the sign, so we get 0"). The second kind is a bug wearing a comment. Every such comment gets verified against the constraint it claims — or, if it claims none, against the contract.

### 2. Degenerate output for rich input

Expected empty slice / zero / nil / no-op for an input that obviously should produce something: a query test whose golden is zero rows, a parser case whose extracted-fields golden is empty, an alert evaluation returning no events for input engineered to fire. The classic cause: the code path silently failed (bad fixture, wrong time window, misconfigured matcher), produced nothing, and `-rewrite` locked the nothing in. The inverse also counts — asserting an error for input that looks valid, or success for input that looks like it must be rejected.

### 3. Contract contradiction

The expectation disagrees with a written promise: godoc says "returns ErrNoRows when absent" but the test asserts `(nil, nil)`; the README documents inclusive bounds but the golden shows exclusive; a repo rule is violated and the test blesses it (this repo's known traps: `(T, error)` returning `(nil, nil)`; enum values not `snake_case`). Also: behavior diverging between the default and `-tags=invariants` builds where a test pins one side and nothing covers the other.

### 4. Internal inconsistency

Two cases in one table that can't both be right: `a<b` handled one way, `b<a` the opposite way with no documented asymmetry; the same input shape accepted in one case and rejected in another; a case name or comment that contradicts its own expectation. At least one of the pair is a lock-in; figuring out which one requires the contract.

### 5. Regenerated-golden drift

Git shows an expectation *flip* — a numeric value changed, an error became a success (or vice versa), output rows appeared/disappeared — in a commit whose message claims no behavior change ("refactor:", "cleanup:", "fix tests"). A deliberate behavior change updates expectations *and says so*; an expectation that drifted under a refactor commit is a regression the suite absorbed instead of caught.

### 6. Silenced truth

`t.Skip("broken since ...")`, commented-out assertions, or an assertion weakened over time (git shows `Equal` → `Contains` → `NotNil`). The strong assertion was usually *right* and the code drifted; the weakening is the lock-in mechanism. A skip whose reason is a bug reference is a known-wrong behavior with the alarm unplugged.

### 7. Hand-checkable math violations

Values you can derive independently: percentile of a small fixed dataset, a rate over a known window, boundary inclusivity (`>=` vs `>` at exactly the threshold), timezone/DST handling at a chosen timestamp, rounding direction, unit scaling. Where the derivation and the golden disagree and the code agrees with the golden, the code and test are wrong together — the definition of lock-in.

## Procedure

### Step 1 — Establish ground truth before reading expectations

The central failure mode of this audit is **anchoring**: read the golden first and your brain will rationalize it the same way the original author did. So build the reference model first:

1. Read the package's README/CONTEXT.md and the godoc of the exported surface — the *promises*.
2. Identify any external spec the package implements (an RFC, a query-language grammar, a third-party API being emulated, PromQL/SQL semantics) and pull the relevant rules.
3. Skim the main callers (`grep -rl` the import path; check the frontend if the output crosses the API) — what do consumers actually rely on?
4. Note the repo-wide contracts that apply (CLAUDE.md rules: no `(nil, nil)`, snake_case enums, `invariants` build-tag differences).

Write the model down in a scratch note. It is the yardstick every expectation gets measured against.

### Step 2 — Mechanical sweep

Cheap red-flag harvest over `*_test.go` and `testdata/`:

```
grep -rnEi 'for now|weird|strange|surpris|somehow|quirk|for some reason|not sure|shouldn.t|should be|actually|known issue|works around' <pkg>/*_test.go <pkg>/testdata/
grep -rnE 't\.Skip|TODO|FIXME|XXX|HACK|BUG'                                <pkg>/*_test.go
grep -rnE '^\s*//\s*(require|assert|Expect|want|got)'                      <pkg>/*_test.go   # commented-out assertions
git log --oneline -15 -- <pkg>/*_test.go <pkg>/testdata/                                     # expectation churn
```

Every hit becomes a candidate to verify in step 3 — a hit is a flag, not a finding. Then read the test files once end to end anyway: the worst lock-ins are uncommented and grep-invisible.

### Step 3 — Predict, then compare

For each candidate (and for a sample of the "boring" cases in every table or golden file — lock-ins hide in unflagged cases too):

1. Read the **input** and the ground-truth model. Cover the expectation.
2. **Predict** the output: derive it by hand, from the spec, or from a trusted reference (a reference implementation, the third-party API's docs, a quick scratch program computing the math independently of the package).
3. Only then read the asserted value. Agreement → move on. Disagreement → trace the code to see which side is wrong. If the code produces the golden but your derivation says otherwise, you have a candidate lock-in; if your derivation was wrong, record why (it sharpens the model).

For golden files, pay extra attention to expectations that are *large*: nobody reviews a 200-line regenerated golden, which is exactly where drift lives. Diff the golden's git history (`git log -p -- <testdata file>`) for unexplained flips (catalog #5).

### Step 4 — Refute before reporting

For every candidate, actively try to kill it:

- Search for evidence the behavior is **deliberate**: a comment citing a constraint, a linked issue/PR, a changelog entry, the commit that introduced the expectation (`git log -S` on the asserted value), bug-compat with an emulated API.
- Check whether a *caller* depends on the current behavior — if the frontend or another package compensates for it, the finding is still real but the blast radius belongs in the report.
- Re-verify your own derivation once more against the spec. A wrong audit finding is worse than none: it teaches the user to ignore the report.

Label every survivor **CONFIRMED** (derivation checked against spec/math/contract and the disagreement traced into the code) or **PLAUSIBLE** (expectation looks indefensible but you could not fully establish the intended contract).

Do not report: style, brittle-but-true assertions (that's `/go-test-decouple`), missing tests, or design preferences with no contract violation behind them.

## Output to the user

Rank findings most-severe first: user-visible wrong output > silent data corruption (unit scaling, boundary) > internal inconsistency > silenced tests.

```
## Locked-in behavior findings (most severe first)

1. <test file:line (and testdata file if applicable)> [CONFIRMED|PLAUSIBLE] (<catalog category>)
   Pinned behavior: <what the test currently asserts, in one sentence>
   Why it is wrong: <the contract/spec/derivation it contradicts, with the evidence — doc quote, hand computation, caller>
   Correct behavior: <what it should be>
   Path out: <one or two sentences: the product-code fix, then the expectation update — in that order. Never "just fix the test": updating the expectation without the code change re-locks the bug.>

2. …
```

Close with one or two sentences on what was audited and found sound ("all 34 percentile goldens re-derived by hand and match; skip-list empty") so a short list reads as evidence, not omission. If nothing is locked in, say so plainly — do not manufacture findings.

## The test-* suite

This skill is orthogonal to the grooming pipeline (`/go-test-reshape` → `/go-test-decouple` → `/go-test-coverage` → `/go-test-cleanup`): those improve *how* tests are written; this audits *whether their expectations are true*. Run it **before** a grooming pass when the package's history is heavy with golden regeneration — grooming a suite that pins bugs polishes the lock-in — and again **after**, since `/go-test-reshape` and `/go-test-coverage` both run `-rewrite` and every regenerated golden is a fresh candidate. `/go-test-groom` brackets its pipeline with exactly those two invitations.

`/go-test-determinism` is the other audit-only sibling: it asks whether expectations are *stable*, this one whether they are *true*. `/go-test-speedup` is orthogonal to both.
