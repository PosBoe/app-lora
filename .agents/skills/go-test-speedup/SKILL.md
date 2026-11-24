---
name: go-test-speedup
description: Cut the wall-clock time of a Go package's tests when they exceed a time budget (default 15s) — profile per-test timings, replace sleep-based synchronization with real synchronization or a fake clock, shrink stress loops, gate genuinely long tests behind -short, and delete tests only when the time saved is significant and the coverage loss negligible. Invoke ONLY when the user explicitly runs `/go-test-speedup` or explicitly names this skill — do not infer it from generic phrasing like "tests are slow", "speed up the tests", or "optimize test time" alone; treat those as ordinary requests instead.
---

# go-test-speedup

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

Make a Go package's tests fast without losing what they verify. The trigger is a time budget — **15 seconds per package** by default, or whatever the user specifies — and the fixes come in strict preference order, cheapest-in-coverage first:

1. **Remove artificial waits** — a sleep that exists to synchronize is a bug in the test, not a cost of the behavior. Replacing it with real synchronization keeps identical coverage and is a pure win.
2. **Rework the test** — same or similar coverage through a faster mechanism: fake clock instead of real time, injected short intervals instead of production ones, fewer iterations where the extras add no new behavior.
3. **Gate behind `-short`** — for tests that are genuinely long by nature (stress, soak, large property sweeps). Coverage is preserved in full runs; the developer inner loop gets fast.
4. **Delete** — last resort, only when the coverage decrease is zero or trivial and the time reduction is significant. Requires coverage proof, not judgment alone.

Speed never comes at the price of flakiness: shrinking a sleep that was masking a race produces a test that fails one run in fifty, which costs more than the seconds saved. Every timing-related rework must survive a stability run before it counts.

This skill does NOT restructure test granularity (`/go-test-reshape`), fix assertion targets (`/go-test-decouple`), or add cases (`/go-test-coverage`). It changes how long tests take, not what they verify.

## When to apply

Apply when a package's tests exceed the time budget, or when the user names a slow package or test. Typical symptoms in this repo:

- `time.Sleep` used to wait for a goroutine, a channel consumer, a background flush, or a watcher to "catch up".
- `require.Eventually` with a coarse tick (≥100ms) or a fixed sleep before the first check.
- Tests exercising retry/backoff/TTL/expiry logic against the real clock instead of `unixtime.FakeClock`.
- Production-scale intervals (flush every 5s, poll every 1s) reaching the test because the test didn't inject a shorter one through an existing config/option.
- Stress or property tests running large iteration counts unconditionally.
- "Nothing happens" tests that wait out a full window to assert an event did *not* fire.
- Unit tests that could run `t.Parallel()` but don't (unit lane only — see repo constraints).

Do NOT apply when:

- The package is already inside the budget — don't shave milliseconds for sport.
- The time is inherent to the contract under test and already minimal (e.g. a debounce test that must observe one real 50ms window).
- The tests are integration tests and the proposed remedy is concurrency — the lane is DB-bound and serialized by design, so `t.Parallel()` and concurrent tag lanes are off the table. The lane is *not* out of scope as a whole: catalog item 7 has the remedies that do work there (seed less, fewer duplicate query paths, narrower queries).

## Slow-test catalog

### 1. Sleep-as-synchronization → real synchronization

**Problem**: the test sleeps N ms hoping concurrent work has finished by then. The sleep is tuned to the slowest CI machine, so every run pays the worst case.

**Fix**, in order of preference:

1. **Synchronize on the event itself**: a channel the worker closes, a `sync.WaitGroup`, a done callback, a hook the production code already exposes. Zero wait when the work is fast, no flake when it's slow.
2. **`require.Eventually(t, cond, timeout, tick)`** with a *generous* timeout and a *tight* tick (5–10ms). The timeout is a ceiling, not a duration — the test pays only until `cond` turns true. The anti-pattern is the reverse: tight timeout (flaky) with coarse tick (slow).

```go
// Before — pays 500ms every run, still flaky on slow CI
go worker.Run(ctx)
time.Sleep(500 * time.Millisecond)
require.Equal(t, 3, store.Count())

// After — pays ~5ms typically, never flakes under 5s
go worker.Run(ctx)
require.Eventually(t, func() bool { return store.Count() == 3 },
    5*time.Second, 5*time.Millisecond)
```

Never just shrink the sleep constant — that trades time for flakiness at a different point on the same bad curve.

### 2. Real time → fake clock

**Problem**: the test waits for real seconds to elapse so TTLs, retention windows, backoff, or rate limits kick in.

**Fix**: install the repo's fake clock and advance it instantly. Code that reads time via `unixtime.Now()`/`unixtime.NowCached()` follows the installed clock:

```go
clock := unixtime.NewFakeClock(unixtime.ToNano(time.Now()))
unixtime.SetClock(clock)
t.Cleanup(func() { unixtime.SetClock(nil) })

// ... arrange ...
clock.Adjust(2 * time.Hour) // instead of waiting
```

Caveats:

- `SetClock` is **process-global**: a test that installs it must not run under `t.Parallel()`, and must restore with `t.Cleanup(func() { unixtime.SetClock(nil) })`.
- The fake clock only moves *reads* of time. Code blocked in `time.Sleep`/`time.After`/`unixtime.Sleep` does not wake when the fake advances — for those, inject a shorter interval (next item) or synchronize on the event instead.
- If the code under test reads `time.Now()` directly, don't add an interface just for the test (see CLAUDE.md); prefer switching the production code to `unixtime.Now()` if that's a natural fit, or fall back to interval injection.

### 3. Production intervals reaching tests → inject short ones

**Problem**: the component polls/flushes/rotates on a production-scale interval, and the test waits out one or more full periods.

**Fix**: pass a millisecond-scale interval through the existing config field or option the component already takes. Most timed components here take their interval from config; the test should never inherit the production default. Only when no knob exists, consider adding one — as a normal option with a sane default, not a test-only backdoor.

### 4. Oversized loops → sized to the behavior

**Problem**: a stress/property/fuzz-style test runs 100k iterations when the failure mode it hunts appears within hundreds, or table tests replicate near-identical heavy cases.

**Fix**: size the loop to the behavior — enough iterations to make the targeted race/leak/overflow likely, not a round impressive number. If large N genuinely matters (leak detection needs sustained load), keep N and gate the test behind `-short` instead (next item).

### 5. Genuinely long tests → gate behind `-short`

**Problem**: the test is long because the behavior is long — stress, soak, leak detection, large corpus sweeps. It shouldn't shrink, but it also shouldn't tax every inner-loop run.

**Fix**: the house idiom, first thing in the test (see `pkg/kafka/kproducer/stress_test.go`, `pkg/inmem/stress_test.go`):

```go
func TestProducerStressMetadataChurn(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping stress test in short mode")
    }
    // ...
}
```

Know what this buys in this repo: **every default lane passes `-short`** — per-push CI, `task test`, `task test:pkg`, and the scoped inner loop — so a gated test skips in all of them. Coverage is preserved by the nightly full suite (`.forgejo/workflows/nightly.yml`, and `task test:full` locally), which sets `GOTEST_FULL=1` to drop `-short`. So `-short` gating genuinely cuts both per-push CI time and inner-loop time here; report it as a real win on both. The one lane that never passes `-short` is race (stress tests matter most under the detector).

Reserve the gate for tests that are long *by nature*. Gating a test that's slow only because it sleeps is hiding a fixable defect.

### 6. "Nothing happens" tests → shrink the window or synchronize on the alternative

**Problem**: the test asserts an event did NOT occur by waiting out a full window (`time.Sleep(5s); require.Zero(t, calls)`). Negative waits can't use `Eventually` — there is no condition to turn true.

**Fix**: synchronize on something that *does* happen and implies the negative — e.g. wait for the second tick of a related action, then assert the forbidden one still hasn't fired; or drive the component with a fake clock/short interval so the window is milliseconds. If neither is possible, shrink the window to the minimum that still fails reliably when the bug is reintroduced (verify by temporarily breaking the code), and note the residual cost. These are also prime deletion candidates when the positive path already pins the same logic.

### 7. Integration tests → fewer and cheaper queries, not concurrency

**Problem**: the package imports `pkg/testbed` and its time is real database work.

**First, the levers that do NOT apply here.** The lane runs `-p 1` on a shared ClickHouse/Postgres/Redis, so `t.Parallel()` and running the two tag lanes concurrently are not available — the shared server saturates and concurrency makes it slower. And the lane is **ClickHouse-bound, not boot-bound**: shaving fx wiring or reusing beds buys nothing measurable, so don't restructure the harness chasing it.

**Fix**, in the order they pay:

1. **Seed less.** A fixture or `> seed` directive that loads entities the case never touches costs a write on every file reseed. Seed exactly what the case reads.
2. **Drop duplicate query paths.** Several corpus files driving the same query with cosmetically different inputs each pay a full CH round trip. Merge them into one file's case list where the assertion shape is identical — that is `/go-test-reshape`'s consolidation, invoked here for time rather than shape; hand it over rather than reshaping in place.
3. **Narrow the query under test.** A case that selects a wide time range or a whole table to assert on three rows pays for the rest. Shrink the range, not the assertion.
4. **Check the loaders.** `load_metrics` / `load_traces` with large row counts where a handful demonstrates the behavior is item 4's problem wearing an integration costume.

Everything else in this catalog still applies — an integration test can sleep for synchronization (item 1) just as easily as a unit test, and that is still the cheapest win.

### 8. Redundant tests → delete with coverage proof

**Problem**: several expensive tests exercise the same paths; dropping the slowest loses little or nothing.

**Fix**: delete only when both hold, and show both in the report:

- **Coverage**: the package coverage diff (procedure step 4) shows no drop, or a trivial one (≲0.5 percentage points) confined to lines other tests still reach in spirit — never a uniquely-covered behavior.
- **Time**: the deletion saves a significant share of the package's wall-clock (as a rule of thumb, ≥20% or ≥5s).

Coverage is a floor, not the whole story: a slow test may be the only one *asserting* an output that a fast test merely executes. Read what the candidate asserts before deleting; if its assertion is unique, rework it (items 1–3) rather than delete. When a deleted test nominally covered a behavior worth keeping, hand it to `/go-test-coverage` to re-cover cheaply.

## Repo constraints

- **Lanes** (`cmd/gotest`): unit runs in parallel, integration (packages importing `pkg/testbed`) runs serially with `-p 1` on a shared ClickHouse/Postgres/Redis. Do not parallelize integration tests or try to speed the lane by running hosted/cloud concurrently — the shared server saturates. Catalog item 7 has the wins that do apply there.
- **The DBs must be up** for any integration measurement or verification run (`task up`, then `task test:setup`) — see `.agents/skills/go-test-groom/references/verification.md` §4. A timing run against a cold or contended database measures the database, not your change; and a failure mid-run may be another process on the shared test DBs rather than your edit.
- **`t.Parallel()`** is a legitimate speedup only in the unit lane, only for tests sharing no mutable state, and never for datadriven cases or tests that install a global (fake clock, `featflag` overrides).
- Tests asserting on real elapsed time may need `pkg/racedetect` awareness — race instrumentation dilates time; check how neighbors handle it before tightening any timing assertion.

## Procedure

Five steps, in order.

### 1. Measure

Time the package with a fresh run (`-count=1` — cached results lie about time), then rank individual tests:

```
mkdir -p /tmp/gotest-skills/<pkg-slug>
go test -count=1 ./<pkg>/... 2>&1 | grep -E '^(ok|FAIL)'
go test -v -count=1 ./<pkg>/... 2>&1 | grep -E '^--- (PASS|FAIL)' | sort -t'(' -k2 -gr | head -20 \
  | tee /tmp/gotest-skills/<pkg-slug>/slow.before.txt
```

**Measure both modes.** A bare `go test` does not pass `-short`, but every default lane does — per-push CI, `task test`, `test:pkg`. So the number above is what the *nightly* suite pays, while the number below is what every developer and every push pays:

```
go test -short -count=1 ./<pkg>/... 2>&1 | grep -E '^(ok|FAIL)'
```

Judge the budget against the mode the user cares about — default to the `-short` number, since that is the inner loop — and report both. A package already inside budget under `-short` whose unqualified time is large has a *stress-test* profile, not a speed problem: say so and stop.

If the package is within budget, report the numbers and stop — no speculative edits. Otherwise also capture the coverage baseline (the reference for reworks and deletions), per `.agents/skills/go-test-groom/references/verification.md` §1–§4 — including the `-tags=invariants` lane when the package has build-tagged files or reads `featflag`, since a deletion justified by a hosted-only profile can drop the only coverage of a cloud-gated path:

```
go test -coverprofile=/tmp/gotest-skills/<pkg-slug>/cover.before.default.out ./<pkg>/...
go tool cover -func=/tmp/gotest-skills/<pkg-slug>/cover.before.default.out | tail -n 1
```

A wall-clock total far above CPU time (`time go test ...`: `real` ≫ `user`+`sys`) means the tests are *waiting*, not computing — expect catalog items 1–3, 6.

### 2. Diagnose each offender

Read every test above ~1s (and any cluster of smaller ones that adds up). Classify against the catalog: what is the time actually spent on — a sleep, a real-time wait, a production interval, honest iteration count, honest DB work? Note the intended remedy per test. If the cause is genuinely inherent and already minimal, say so and leave it alone.

### 3. Fix in preference order

Apply remedies per the catalog: remove waits first, rework second, `-short`-gate third, delete last. Keep each fix minimal — this skill does not restructure tests while it's in there.

### 4. Verify: correctness, stability, coverage, time

```
go test -race -count=1 ./<pkg>/...
```

Then three checks specific to this skill:

- **Stability** — every test whose synchronization or timing changed must pass repeatedly, including under race (race dilates time and exposes tightened waits):

  ```
  go test -run 'TestA|TestB' -count=10 ./<pkg>/...
  go test -race -run 'TestA|TestB' -count=5 ./<pkg>/...
  ```

  One flake means the rework is wrong — fix the synchronization, never re-widen a sleep to paper over it.

- **Coverage** — must hold within the deletion tolerance:

  ```
  go test -coverprofile=/tmp/gotest-skills/<pkg-slug>/cover.after.default.out ./<pkg>/...
  diff <(go tool cover -func=/tmp/gotest-skills/<pkg-slug>/cover.before.default.out) \
       <(go tool cover -func=/tmp/gotest-skills/<pkg-slug>/cover.after.default.out)
  ```

  Diff each tag lane against its own baseline; profiles from different tag sets are not comparable.

  Any drop must be attributable to an intentional deletion and within its stated tolerance; a drop from a *rework* means the faster test no longer reaches a path the slow one did — fix the rework.

- **Time and `-short`** — re-measure, and confirm gated tests skip and everything else still runs:

  ```
  go test -count=1 ./<pkg>/... 2>&1 | grep -E '^(ok|FAIL)'
  go test -short -count=1 ./<pkg>/... 2>&1 | grep -E '^(ok|FAIL)'
  ```

### 5. Finalize

```
task test:pkg -- ./<pkg>/...
```

(or `task test` if the change touched shared helpers beyond the package).

## Output to the user

When done, report:

- **Time delta**: package wall-clock `before → after`, plus `-short` time if any test was gated.
- **Per fix**: test name, cause (catalog item), remedy, seconds saved — one line each.
- **Gated tests**: which tests now skip under `-short` — cutting both per-push CI and the inner loop (every default lane passes `-short`), with the reminder that the nightly full suite (`task test:full` / `nightly.yml`) still runs them.
- **Deletions**: each deleted test with its coverage evidence and time saved.
- **Stability evidence**: the `-count` runs performed on reworked tests.
- **Left alone**: slow tests whose cost is inherent, with one line why.
- **Self-assessment**: any rework you'd flag for review — a timeout you chose, a window you shrank, a deletion that was borderline.

Stay terse. The diff is the artifact; the summary just routes attention.

## The test-* suite

This skill is orthogonal to the grooming pipeline (`/go-test-reshape` → `/go-test-decouple` → `/go-test-coverage` → `/go-test-cleanup`); run it standalone whenever a package's tests blow the time budget. If a deletion drops a behavior worth keeping, `/go-test-coverage` re-covers it cheaply.

`/go-test-determinism` is this skill's closest neighbor and they meet at synchronization: a `time.Sleep` before an assertion is both a cost (item 1 here) and a flake (there). Fixing it once satisfies both. When a rework here keeps failing intermittently, the underlying defect is a determinism problem — hand it over rather than widening a timeout.

`/go-test-datadriven` owns the datadriven authoring idioms the others lean on. `/go-test-groom` runs the pipeline end-to-end in one prompt; `references/verification.md` under it holds the suite-wide baseline and gate protocol.
