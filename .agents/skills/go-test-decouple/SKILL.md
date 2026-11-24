---
name: go-test-decouple
description: Audit every assertion in a Go package's tests and retarget the ones that pin internal implementation onto the public contract — return values, error types, final state read through the package's own API. Expect findings in any package whose tests check err.Error() text or database server messages, query the package's own tables behind its back, read or write unexported fields, call unexported functions directly, build values no constructor produces, capture logs, or use mocks. Invoke ONLY when the user explicitly runs `/go-test-decouple` or explicitly names this skill, or when go-test-groom's pipeline explicitly directs it — do not infer it from generic phrasing like "decouple test assertions", "fix brittle tests", or "stop testing internals" alone; treat those as ordinary requests instead.
---

# go-test-decouple

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

Audit a Go package's tests and retarget assertions that pin internal implementation onto observable behavior. The test structure stays put — no consolidation, no reshaping, no new test files (that's `/go-test-reshape`). What changes is **what a test reads and how it reaches the state it reads**: the assertion target, and an arrange step that bypasses the public path to build a state (catalog §14). In a datadriven package the assertion target is chosen by the handler's renderer, so narrowing a renderer is in scope (catalog §12) while moving or splitting the test is not.

Edit only test code: `*_test.go` files, `testdata/`, and the package's **test-support files** — a `<pkg>test/` subpackage (`pgtest`, `redcachetest`) or any other non-`_test.go` file that imports `testing`. Their `RequireX` helpers and comparers are assertions too, and every test that calls them inherits their coupling. When a good assertion needs an API the package lacks, flag the gap; the user decides whether to widen the surface.

This matters most with agents: tests that pin the implementation remove exactly the flexibility that makes agents useful. Tests that pin the result let an agent rewrite the internals freely while proving the behavior is preserved. Worse, when a coupled assertion fails under a correct refactor, an agent pushing to go green tends to weaken the assertion or regenerate the expectation — every coupled assertion left in place is an open invitation to degrade the suite.

## The litmus test

An assertion is implementation-coupled exactly when a behavior-preserving refactor could break it. For each assertion, ask whether it would survive all five of these refactors, applied while every caller observes identical behavior:

1. **Rename/reorganize**: every unexported identifier renamed; private helpers merged, split, or inlined.
2. **Re-algorithm**: batching, caching, ordering of internal steps, or the whole algorithm swapped for a different one with the same outputs.
3. **Reword**: every error message and log line rephrased — including the ones a dependency or a database server writes, which change on a version bump you did not make.
4. **De-instrument**: every metric and log statement removed or renamed.
5. **Re-store**: the package's private tables, keys, files, and in-memory layout reshaped, while its exported reads return the same answers.

If any of the five breaks the assertion, it is coupled. "The current code happens to do X" is never a contract. A contract is written down — in godoc, a README or DESIGN.md, the exported type system, the openapi contract, or a persisted shape another package reads — or it isn't a contract.

The same test applies to the **arrange** half of a test. A fixture that writes an unexported field, inserts rows straight into the package's table, or builds a struct literal the constructor would never return breaks under refactors 1 and 5 just like an assertion does, and it can also put the code in a state no caller can reach — then the test proves behavior nobody gets.

## Run the audit as an inventory, not a skim

Run casually, this audit tends to return "nothing to fix" on packages that are full of coupling. Three traps cause that, and the procedure below is built to defeat them:

- **Skimming instead of inventorying.** Read top-to-bottom, most tests "look reasonable". Coupling only becomes visible when you list every assertion and run the litmus test on each one individually. So the inventory is mandatory, and the final report includes its counts.
- **Generous exceptions.** Every catalog entry has a legitimate exception ("call count is the contract for a rate limiter"), and it is comfortable to wave borderline cases through. An exception claim requires citable evidence — a doc comment, README sentence, or exported API that promises the behavior. No evidence → treat as coupled. If the behavior genuinely deserves to be a contract, the right fix is to document or expose it (flag that in the report); keeping a silent pin is not.
- **Hunting only the blatant patterns.** Mocks are rare in this repo. The common coupling is quieter: `err.Error()` string checks, a raw `NewSelect` on the package's own table where an exported read exists, tests that call unexported helpers when an exported verb reaches the same edge, a `require.Equal` whose type holds unexported fields, a fixture struct literal no constructor produces. The catalog below covers both kinds; the sweep in step 2 makes sure the quiet ones surface.

Finding nothing is a suspicious outcome, not a success. A zero-fix report must show, per category, the evidence that the sweep ran and came up empty.

## Implementation-coupled assertion catalog

Each entry: the pattern, why it's brittle, the exception (which requires evidence), and the replacement. `references/examples.md` has a before/after for each.

### 1. Mock call counts and call verification

**Pattern**: `mock.AssertNumberOfCalls(t, "Method", 3)`, `mock.AssertCalled(t, "Method", args...)`, `AssertExpectations` pinning exact call sequences, asserting on captured mock arguments that aren't part of the contract. The same applies to a hand-written fake that records calls (`fake.calls`, `len(fake.requests)`).

**Why brittle**: how many times an internal dependency is called is an implementation choice. Batching, caching, or reordering changes the count without changing the result.

**Exception**: call count IS the contract — rate limiters ("must not exceed N calls"), idempotency ("exactly-once delivery"), circuit breakers ("stops calling after N failures"), or the call *is* the output (a request sent to an external API is what the package exists to produce).

**Replace with**: the return value or side effect the calls produce. If the mock's data flows to the output, the output assertion already covers it; if it writes to a store, use a fake that keeps state and read the store's final state.

### 2. Unexported field inspection

**Pattern**: `require.Equal(t, 42, obj.internalCount)`, reading lowercase struct fields, `reflect`/`unsafe` peeking at internal state, and go-cmp options that open unexported fields to the diff (`cmp.AllowUnexported`, `cmp.Exporter`).

**Quiet form**: `require.Equal`/`reflect.DeepEqual` on a value whose type has unexported fields compares those fields too — a borrowed flag, a cached hash, a pointer into a pool. The test reads internals without naming one. Check the type of every whole-value comparison.

**Why brittle**: unexported fields are free to change name, type, or semantics in any refactor; in-package tests won't even fail to compile in a way that signals the problem.

**Replace with**: the public method or return value that exposes the same information. For a whole-value comparison, compare what the type exports (its accessors, its text), or `cmpopts.IgnoreUnexported` when the exported fields are the contract. If the type needs an `Equal` method to compare correctly, that is an API gap — flag it. If the field is internal bookkeeping (cache size, buffer position), drop the assertion.

### 3. Internal event ordering

**Pattern**: ordered mock expectations (`NotBefore`, `mock.InOrder`), channel-receive order, sequence counters — when the public contract doesn't specify order.

**Why brittle**: concurrency, batching, and reordering optimizations change internal order without changing observable behavior.

**Replace with**: the final result. If ordering matters to the caller, assert on the order of the *output*, not of internal steps.

### 4. Error substring matching

**Pattern**: `require.Contains(t, err.Error(), "unexpected")`, `require.ErrorContains(t, err, "failed to parse")`, comparing `err.Error()` to literals.

**Default stance**: most error-substring assertions are fine — leave them. A message literal you own survives renames and re-algorithms; a test that breaks only when you *deliberately* reword is cheap to fix and arguably doing its job. Act only in the two situations below. Outside them, a plain `ErrorContains` on a stable, user-facing message you own is a defensible choice, not a sin.

**Act when the wording is not yours** — a database server, a dependency, the OS, the standard library's regexp/glob/syscall errors. It changes across versions and platforms, so the test goes red in CI on a bump you didn't make. Find the wording's owner before you judge a substring: a message like `"Code: 62. Syntax error"` or `"violates foreign key constraint"` comes from ClickHouse or PostgreSQL, not from the package. Replace it with the owner's structured code — `errors.AsType[*chproto.Exception]` and its `Code`, `errors.AsType[pgconn.Error]` and its code or `IntegrityViolation()`, `errors.Is` against `fs.ErrNotExist` or a context error. With no code to read, use a bare `require.Error(t, err)`; the contract is "it was rejected", and the case's input already says why.

**Act when a typed error exists — or should** — and assert the structured discriminator, not the sentence:

- `errors.AsType[*SpecificError](err)` plus its fields, or `errors.Is` against a sentinel for wrapped errors. This repo's usual discriminators: `*validerr.Error` and its field, `*httperr.Error` and its `Code`, a package's own typed error (`*ast.Error` with its position, a `*StepError`).
- If several rules share an input shape and only the message tells them apart, the substring is the *last* discriminator — don't drop it to a bare `require.Error`. Give the error a typed field/code and assert that; treat "type this error" as a **recommended follow-up in the report, not a footnote**, especially for user-facing validation a caller or frontend must distinguish.
- A substring so loose it matches unrelated errors (`Contains(err.Error(), "unexpected")` passes for any error with that word) is a false-negative bug regardless of brittleness — tighten to a typed check.

**Exception**: the message content itself is the contract under test (a CLI's user-facing output, a rendered error golden in a datadriven test, a refusal text the openapi contract or a README promises).

### 5. Log output assertions

**Pattern**: capturing `slog`/`zap` output and asserting on lines, levels, or fields.

**Why brittle**: logs are a debugging aid, not a behavioral contract; wording and levels change constantly.

**Exception**: logging IS the unit under test — a logging library, formatter, or documented audit trail.

**Replace with**: the error return (if the log signals an error), the resulting public state (if it signals a transition), or the outcome the log describes.

### 6. Metric counter assertions

**Pattern**: `testutil.ToFloat64(counter)`, Prometheus/OTel readings compared to exact values.

**Why brittle**: instrumentation is an observability concern; adding, removing, or renaming metrics shouldn't break functional tests.

**Exception**: the metric is the documented product ("emit metric X when Y") or the package is itself a metrics library.

**Replace with**: the behavior the metric tracks — processing results, queue's public depth, etc.

### 7. SQL / generated-query string assertions

**Pattern**: `require.Contains(t, query, "WHERE id = ?")`, substring or exact matching on generated SQL in ordinary unit tests.

**Why brittle**: the same result can come from different SQL (join vs. subquery, formatting churn).

**Replace with**: the query *result* in integration tests. For query-builder units, a datadriven golden of the full rendered query (`chschema.QueryAppender` output) — the whole query is the reviewable artifact, not a substring.

### 8. Goroutine and scheduler assertions

**Pattern**: `runtime.NumGoroutine()` before/after checks, `time.Sleep` then assert.

**Why brittle**: goroutine lifecycle and timing belong to the runtime; these are flaky by construction.

**Replace with**: the observable outcome — function returned, channel closed, context cancelled — with `require.Eventually` for async results. Leak checks belong to goleak (`docs/go-test-conventions.md`), not to a goroutine count.

### 9. Map iteration order

**Pattern**: comparing a slice built by iterating a map against a fixed order.

**Why brittle**: Go randomizes map iteration; passes locally, fails in CI.

**Replace with**: sort before comparing, or `require.ElementsMatch` when order isn't part of the contract. If callers need order, the API should return a slice — flag it.

### 10. White-box tests of unexported functions

**Pattern**: a test invokes an unexported function directly (`got := normalizeAttrs(in)`) when the same behavior is observable through an exported API — or, in a `cmd/` or application package, through the command, verb, route, or job that calls it.

**Why brittle**: it pins the internal decomposition — inlining, renaming, or re-splitting the helper breaks the test while the package behaves identically.

**Exception**: a genuinely complex internal algorithm (parser core, rebalancing routine) whose edge cases would be indirect and noisy to reach through the public API. A ten-line helper does not qualify, and neither does a helper whose one caller is a verb the test can run with a temp directory or the package's test database.

**Replace with**: retarget the case at the exported caller, choosing inputs that hit the same edge, and assert what that caller shows — its return, its error, the text it prints. Keep the case table; change only what each row feeds in and reads back. If no exported path reaches the helper, it may be dead code — flag it.

### 11. Overspecified equality (Go assertions)

**Pattern**: `require.Equal(t, wantStruct, got)` on a large struct or map where only a few fields are the contract; snapshots/goldens dumping whole internal objects.

**Why brittle**: pins incidental fields — internal defaults, bookkeeping counters, estimator outputs — so unrelated changes break the test.

**Exception**: whole-value fidelity is the point (round-trip/serialization tests), or a datadriven golden rendered from a purpose-built struct that already selects only contract fields (see `/go-test-datadriven`).

**Replace with**: assertions on the contract fields only.

### 12. Coupled golden output

**Pattern**: a datadriven golden or snapshot that pins internal state rather than the contract — a renderer dumping the whole domain model instead of a purpose-built view struct; a golden containing unexported bookkeeping (cache sizes, buffer positions, estimator intermediates); generated SQL goldened where the *result* is the contract; a golden whose churn under `-rewrite` tracks refactors rather than behavior changes.

**Why brittle**: a golden is an assertion whose target is chosen by the renderer. Categories 2, 7 and 11 above all have a golden form, and the golden form is *worse*: `require.Equal` on an internal field fails loudly and gets read, while a 200-line golden gets regenerated. Reshaping and conversion migrate coupling into goldens without changing it — a `require.Equal(t, wantStruct, got)` that becomes `yaml.Marshal(got)` is the same overspecification in a format nobody reviews line-by-line.

**Exception**: whole-value fidelity is the point (round-trip/serialization corpora), or the rendered artifact *is* the product (a formatted query the user sees, a CLI's output).

**Replace with**: narrow the **renderer**, not the golden. Marshal a purpose-built struct with explicit `yaml` tags holding exactly the contract fields (`/go-test-datadriven`'s rendering rules), then `-rewrite` and review the shrinkage. Where the output is large but genuinely all load-bearing, that is not this category — that is a layout problem, and the fix is pretty-printing with a round-tripped formatter, also per `/go-test-datadriven`.

This is the one catalog entry whose fix edits Go code that is not an assertion. That is still in scope: the renderer *is* the assertion.

### 13. Reading the package's own store behind its back

**Pattern**: the test queries the table, key, or file that the package under test owns — `db.NewSelect().Model((*jobRow)(nil)).Where("status = ?", done).Count(ctx)`, a raw `SELECT` on the package's table, a Redis `GET` on its key format, a read of its on-disk layout — when the package exports a read that answers the same question (`ListJobs`, `Status`, a `Get` route).

**Why brittle**: the row layout, the table name, the key format, and the unexported model type are the package's private choice. Refactor 5 breaks the test while every caller still gets the same answers. The raw query also skips the filter the exported read applies, so it can count rows the package itself would never return.

**Exception**: the persisted shape is a contract in its own right — another package or another release reads it, `pkg/wirepin` pins it, or a DESIGN.md freezes it. Then read it raw on purpose, and say so in a comment. Also exempt: the package exports no read, and the stored state is the outcome under test — then keep the query and flag the missing read.

**Replace with**: the package's exported read. Filter its result for the fact the old query counted. When a helper wraps the raw query (`jobDone(t, db, id)`), retarget the helper and every call site follows.

### 14. Arranging state through internals

**Pattern**: the arrange step reaches a state by a path no caller has — it writes an unexported field (`q.window = ...`), builds a struct literal of a type whose constructor keeps an invariant (`Summary{Count: 3}` where production values only come from an aggregator that also fills `Sum`), or inserts rows straight into the package's table when an exported write (`Enqueue`, `Create`, a route) makes them.

**Why brittle**: it breaks under refactors 1 and 5 like any internal read. Worse, it can build a state the constructor would refuse — a window the planner did not choose, a summary with a count and no sum — so the test proves behavior of an impossible value and hides the path the real value takes.

**Exception**: no exported path can produce the state in a reasonable test — a corrupted or legacy row the package must survive, a state only a crash leaves behind. Then build it raw, and say in a comment which production event makes it.

**Replace with**: the exported constructor, the exported write, or the verb that makes the state. When a case exists only because the literal made an impossible state reachable, it is a `/go-test-lockin` question, not a retarget — report it.

### 15. Tests that test their own copy

**Pattern**: a `_test.go` file holds a function that mirrors production logic — a local reimplementation of an encoder, a sort the production code does inline, a helper that production code used to export — and the test asserts on that copy. A variant: the test's "expected" value is computed by calling the same production function it is checking.

**Why brittle**: it pins nothing the package ships. The production code can break and the test stays green; the copy can drift and the test goes red with no product change.

**Replace with**: rewrite the test against the production entry point that the copy stood for, asserting the property the copy's test held. This is a retarget of the same test, not a new case, so the "don't add cases" rule does not hold it back. Handle every test that calls the copy: a test is not safe to keep because it holds a property no other test holds — that is the reason to rewrite it against production. Delete a test outright only when another test already holds its property, and name that test in the report.

## Procedure

`.agents/skills/go-test-groom/references/verification.md` holds the suite's baseline and gate protocol. This procedure cites it and does not restate it. Drop `-race` from its commands: the root `CLAUDE.md` runs the race lane only on request.

### 1. Baseline

Retargets change what a case reads, and step 4 may delete a test, so take a baseline to prove nothing else was lost. Follow verification.md §2: decide the lanes (take the invariants lane too when the package has build-tagged files or reads `featflag`), write the coverage profile and the case inventory for each lane. For an integration package (its tests dial PostgreSQL), check §4 first — the databases must be up, and every command takes `-p 1`.

Under `/go-test-groom`, the reshape phase's `after` artifacts are this phase's `before`; do not re-run them.

### 2. Sweep mechanically, then read

First list the test-support files, because the greps below read only `*_test.go` and would pass over them:

```
grep -rlE --include=*.go '^\s*"testing"$' <pkg>/ | grep -v '_test\.go$'
```

Read each hit in full: a comparer, a `RequireEqual`, or a fake there is an assertion every caller shares. Then run these greps over the package's test files and record every hit (category in parentheses). Pass the **directory** with `-r --include`, not a `<pkg>/*_test.go` glob: the glob misses every subpackage that `./<pkg>/...` covers. Re-run the ones for categories 1, 2 and 4 with `--include=*.go` over the test-support files.

```
grep -rnE --include=*_test.go 'AssertNumberOfCalls|AssertCalled|AssertNotCalled|AssertExpectations|NotBefore|\.Calls\b|\.calls\b' <pkg>/   # (1) mock interactions
grep -rnE --include=*_test.go 'ErrorContains|err\.Error\(\)|EqualError'                          <pkg>/   # (4) error text
grep -rnE --include=*_test.go 'NewTextHandler|NewJSONHandler|zaptest|observer\.New'              <pkg>/   # (5) log capture
grep -rnE --include=*_test.go 'ToFloat64|CollectAndCompare|metricdata|ManualReader'              <pkg>/   # (6) metrics
grep -rnE --include=*_test.go 'NumGoroutine|time\.Sleep'                                         <pkg>/   # (8) scheduler/time
grep -rnE --include=*_test.go 'reflect\.(ValueOf|DeepEqual)|unsafe\.|cmp\.Exporter|AllowUnexported' <pkg>/ # (2) reflection
grep -rniE --include=*_test.go 'contains\(t?, ?(query|sql|q,)|"(SELECT|INSERT INTO|WHERE) ' <pkg>/  # (7) SQL fragments
grep -rnE --include=*_test.go 'NewSelect\(|NewInsert\(|NewUpdate\(|NewDelete\(|NewRaw\(|QueryRow|\.Exec\(ctx' <pkg>/     # (13/14) direct store access
```

For every error-text hit, find who wrote the message — `grep -rn '<the substring>' <pkg>/ --include=*.go` over production files. No hit in the package means the wording is a dependency's or a server's: category 4's first case.

For every direct-store hit, list the package's exported reads and writes (`go doc -all ./<pkg> | grep -E '^func|^    func'`) and check whether one answers the same question (13) or makes the same state (14).

Greps can't see everything. Then read every test file once, looking specifically for the quiet categories: lowercase field reads **and writes** on the type under test (2, 14), direct calls to unexported functions (10), fixed-order comparisons of map-derived slices (9), `require.Equal` against large structs or types with unexported fields (2, 11), struct literals of types the package builds through a constructor (14), functions defined in a test file that do what a production function does (15), ordered channel receives (3). To list unexported calls quickly: `grep -noE '\b[a-z][A-Za-z0-9]*\(' <file>` and keep the names the package defines in non-test files.

**Then sweep the golden corpora.** `*_test.go` is only half the assertion surface: a package with `testdata/` or `.snap` files keeps assertions there, and they are the *more* likely home for coupling because nobody reviews a regenerated golden line-by-line. This matters most right after a reshape or conversion, which migrate coupled assertions into goldens verbatim.

```
ls -d <pkg>/testdata/ <pkg>/**/__snapshots__/ 2>/dev/null                # is there a corpus at all?
grep -rnE '(SELECT|INSERT INTO|CREATE TABLE) ' <pkg>/testdata/           # (7/12) goldened SQL
```

For each corpus, read the **renderer** in the Go handler, not just the goldens: what does it choose to marshal? A handler that marshals the domain model, `%+v`-dumps a struct, or renders fields the package's godoc never promises is category 12 regardless of how tidy the goldens look. Then spot-check the widest goldens for internal bookkeeping the contract does not mention.

Do not read the package's git history for this audit. Past commits of the same kind prime you to copy them instead of judging the tests as they are.

### 3. Classify with the litmus test

Build the inventory — one row per hit:

| file:line | assertion target | category | verdict | evidence |
|---|---|---|---|---|

Verdicts: `coupled` (fails the litmus test), `contract` (exception applies — cite the godoc/README/exported API in the evidence column), or `borderline`. For borderlines, check the documentation; if the behavior is undocumented, the verdict is `coupled` — retarget it and, when the behavior looks like it *should* be a contract, additionally flag the missing documentation.

### 4. Retarget

For each `coupled` row, make the minimal edit:

- Replace the assertion with one on the public contract, per the catalog's replacement strategy. Don't restructure, rename, or move the test, and don't add cases.
- Run the litmus test on the replacement too, arrange included. A retarget that reads an unexported field or builds a value no constructor makes, to reach the state it needs, moves the coupling instead of removing it. Reach the state through exported calls — an older registry plus an exported write is better than a hand-built one — or keep the old form and flag the gap.
- Before you settle on a replacement, ask what the old assertion **caught**. A retarget that no longer fails when that behavior breaks is a weakening, not a decoupling. `require.Error` in place of a server-text substring is correct only when no code discriminates the error; a `len(records)` in place of a filtered count is wrong when the count read one state.
- Remove setup that only served the removed assertion (a mock expectation nothing else uses, a log buffer, an import).
- If you remove the last test call of an unexported production function, the `unused` linter fails on it. Keep a call and flag the function as dead code; deleting it is a production change the user decides.
- If removing assertions leaves a test asserting nothing observable, the test was testing nothing — delete it and record that in the report (the behavior becomes a `/go-test-coverage` candidate).
- If the replacement needs a return value or method that doesn't exist, flag the API gap rather than adding the API yourself — the user decides whether to expand the surface.

Compile-check with `go vet ./<pkg>/...` (add `-tags=invariants` when the invariants lane applies).

### 5. Prove each retarget still catches

A retargeted assertion that passes proves only that it holds. For each retarget, break the behavior the **old** assertion guarded for a moment — skip the write, return a different error, flip the state — and run the case. The new assertion must fail. Then restore the code, and confirm with `git diff -- ':!*_test.go' ':!**/testdata/**'` that no production change is left (a test-support file you edited on purpose is the one expected hit).

If the new assertion still passes with the behavior broken, it is weaker than the one it replaced. Fix it, or keep the old one and record why no decoupled form exists.

### 6. Verify

1. Run each lane with `-count=1`. All tests must pass. If a replacement assertion fails, the old test was pinning behavior that differs from the public contract — investigate: if the public API is right and the pinned value was an internal quirk, fix the expectation; if the public API exposes wrong behavior, flag it, don't paper over it.
2. If a renderer was narrowed (category 12), regenerate and read the shrinkage as a diff — every line that disappeared is an assertion you deliberately dropped, and any line that *changed* rather than vanished means the narrowing altered behavior, not just scope. The flag goes after the package list: `go test ./<pkg>/... -rewrite`.
3. Write the `after` coverage and case inventory for each lane, and diff each against its own baseline (verification.md §6). A deleted test that asserted nothing observable is a sanctioned coverage drop; name it. Any other drop means a retarget lost a case.
4. `git status --short` shows only `*_test.go`, `testdata/`, and test-support changes.
5. Gate 2: `task test:pkg -- ./<pkg>/...`. Use gate 3 (`task test`) only if the retargeting reached shared test helpers outside the package — a test-support package that other packages import counts, so check with `grep -rl '<its import path>' --include=*.go .` before you edit one.

## Output to the user

- **Inventory summary**: assertions audited, hits per category, verdict counts. State the corpus sweep separately — how many testdata/snapshot files and renderers were read — since a Go-only inventory is the classic false-negative.
- **Renderers narrowed** (category 12): which handler, what fields the golden stopped pinning, and the line-count shrinkage.
- **Assertions retargeted**: file, line, category, what it was → what it became; grouped by test function. Mark each one proven by step 5.
- **Assertions kept as contract**: each with its cited evidence (godoc/README/API), not just a hunch.
- **Tests deleted**: tests with nothing observable left, and what they nominally covered.
- **Flags**: API gaps (behavior only visible via internals), errors that need a code, undocumented behaviors that look like contracts, and fixtures that built an impossible state.
- **Proof**: lanes run, coverage and case diffs, gate 2 result.
- **Self-assessment**: borderline calls you'd revisit, replacements that feel weaker than the original.

If the result is zero retargets, the report must show the per-category sweep evidence that justifies it.

Stay terse. The diff is the artifact; the summary just routes attention.

## The go-test-* suite

This skill is step 2 of the pipeline that `/go-test-groom` runs on one package: `/go-test-reshape` → `/go-test-decouple` → `/go-test-coverage` → `/go-test-cleanup`. `/go-test-datadriven` owns the datadriven authoring idioms the others lean on. Siblings sit outside the pipeline: `/go-test-speedup` (run it when the package's tests blow the time budget), `/go-test-determinism` (flake and cross-test-interference sources), and `/go-test-lockin` (whether the expectations are true at all).

## Additional resources

- `references/examples.md` — before/after rewrites for the catalog categories, including the quiet ones.
- `.agents/skills/go-test-groom/references/verification.md` — the suite-wide baseline, gate-list and proof-of-no-loss protocol.
- `.agents/skills/go-test-datadriven/SKILL.md` — the rendering rules a category-12 fix has to land on (purpose-built view structs, round-tripped formatters).
- `docs/writing-good-tests-ai-era.md` — principles for what to assert on, and why coupled assertions degrade under agents.
- `docs/go-test-conventions.md` — the testbed entry point, regenerating corpora, goleak, the shared databases.
- CLAUDE.md test rules: assert inputs/outputs, not steps; `errors.AsType`, not `errors.As`.
