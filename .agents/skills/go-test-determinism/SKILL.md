---
name: go-test-determinism
description: Audit and fix the ways a Go package's tests can pass today and fail tomorrow — real-clock reads and clock moves behind the scheduler, assertions on random fake values, reads and writes wider than the fixture on the shared test databases, shared locks and keys the harness does not scope, tests that spend the shared server's connection budget, waits on the wrong signal or on a timing margin, races on process-global state, order dependence, expectations that hold in only one build, and goroutine leaks. Targets defects that a green run cannot reveal, and that surface as flakes or as damage to sibling tests sharing the database. Invoke ONLY when the user explicitly runs `/go-test-determinism` or explicitly names this skill — do not infer it from generic phrasing like "the tests are flaky", "fix the flake", "make tests deterministic", or "why did CI fail" alone; treat those as ordinary requests instead.
---

# go-test-determinism

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
- Inject a clock into the code under test; there is no process-global fake
  clock. Seed randomness explicitly; there is no `UPTRACE_TEST_SEED`.

Audit a Go package's tests for **nondeterminism and cross-test interference**, and fix what you find. The other skills in this suite ask whether the tests are well-shaped (`/go-test-reshape`), well-targeted (`/go-test-decouple`), complete (`/go-test-coverage`), readable (`/go-test-cleanup`) or true (`/go-test-lockin`). This one asks whether they will still pass on the next run, on a loaded runner, in a different order, in a different build, next to a sibling test — and whether they damage tests they don't own.

That last part is what makes this more than a flake hunt. Integration tests here share one Postgres/ClickHouse/Redis across every package in the lane, and the beds never delete a row, so every table holds tens of thousands of rows other tests left. A test that reads more than its fixture owns, or spends more of the server than its share, does not fail — it makes *some other package* fail, intermittently, with a stack trace pointing nowhere near the cause.

The house rules this audit enforces live in `docs/go-test-conventions.md` (§ "The shared databases never shrink", § "Determinism", § "Running a lane") and `pkg/testbed/AGENTS.md`. Read both before the sweep; where this skill and they disagree, they win — say so in the report.

The target package comes from the invocation argument (e.g. `/go-test-determinism ./alerting/monitors`). If none was given, ask which package before reading anything.

## When to apply

Apply when a package's tests are suspected of flaking, when a package was just grown or reshaped (new cases are new chances to introduce these), or as a standing audit on an integration package that writes to the shared databases.

Do NOT apply when:

- The tests are slow but stable — `/go-test-speedup`. (The two meet at synchronization; see the boundary note in catalog §7.)
- The expectations are stable but *wrong* — `/go-test-lockin`.
- A specific test is failing reproducibly right now. That is a bug to diagnose, not an audit; fix it directly.
- The nondeterminism is in the production code rather than the test. Fix it there and say so — an unstable API is not a test defect, and papering over it with `require.Eventually` hides a real contract problem.

## A green run proves nothing here

Every defect in this catalog passes when you run it on an idle laptop. The seed that collides is one of 2⁶⁴; the stray row that breaks a count arrives only when another package writes in the same second; the timer that splits a batch fires early only on a starved CI runner. So the audit is a **read** of every test against the catalog, followed by targeted stress (procedure §4) that raises the odds enough to be worth something.

For each assertion, ask three questions. Most findings fall out of them:

1. **What else could put a value where this assertion looks?** Another package's rows, messages or mail; a previous run's rows or cache entries; this test's own earlier subtest.
2. **What does this wait actually prove happened?** The event the assertion reads, or only something that usually precedes it?
3. **Under which build, load and machine does the expected value hold?** No tags, `-tags=invariants`, the premium tag; a runner with one free CPU; arm64.

Three traps make this audit come back empty on packages that are full of defects:

- **"It's always passed."** Age is not evidence. The shared tables grow every run, CI load varies every push, and the random base seed is minted fresh per run (`pkg/testbed`'s `runEnv`).
- **Judging scope by the file.** The lane runs whole packages; the neighbor that breaks you may be three directories away. Scope every judgment to the shared database and the shared process.
- **Reading only the test's own lines.** A test that calls a product function which sweeps, claims, lists or counts across the whole database has an unscoped read even though no `SELECT` appears in the test. Follow each call the assertion depends on until you know what set it reads.

## Nondeterminism catalog

### 1. Real clock, or a clock moved behind the scheduler

**Pattern**: `time.Now()`/`time.Since` in a test or in the code path under test; a fake clock seeded from `time.Now()`; in a testbed test, a `*unixtime.FakeClock` that the test `Set`s or `Adjust`s itself; production code that waits with `time.Sleep`/`time.After`/`time.NewTimer` where the test needs to drive it.

**Why it breaks**: a real-clock value differs every run, so the assertion is either worthless or fails at a boundary, on a slow machine, or under the race detector. A fake clock moved by a second writer moves time under the waiters the scheduler armed, which then fire early, late or never — and the failure lands on the next `adjust_clock`, often in another corpus file.

**Fix**: read time through `unixtime.Now()` and move it through the one writer:

- **In a testbed test**, move time with `App.Sched()` (`Advance`, `AdvanceTo`, `AdvanceToNext`, `Rewind`). `App.Clock()` is a read-only `dbfixture.ClockReader`. The testbed installs the fake clock at the epoch `2025-01-01`, and the datadriven runner rewinds it before every corpus file.
- **Without a testbed**, build the pair yourself from a **literal instant**: `schedtest.New(unixtime.NewFakeClock(epoch))`. Pass the test's `t` to each call that moves time, so a subtest fails itself. A fake wound to the wall clock still moves every run. A bare `unixtime.SetClock` is process-global: the test must not be parallel and must restore it in `t.Cleanup`.
- **Production code that sleeps on package `time`** cannot be driven by any fake. Flag it as a product-code change to take a `sched.Clock` (see `pkg/sched/schedtest/README.md`, "Driving a loop"); do not paper over it with a longer wait.

**Exception**: a test of time handling itself, and a pure unit test with no testbed that stamps a value it never asserts on.

### 2. Assertions on literal fake values

**Pattern**: `require.Equal(t, "brianna_hodkiewicz@example.com", user.Email)`, a hardcoded generated name, token or id in a Go assertion or a datadriven golden.

**Why it breaks**: a test mints a **random base seed per run**, mixed with the test's name, and fake data derives from it — so a literal is right for exactly one run. Values a database-wide unique index would reject twice (`users.email`, the token columns) come from a run **nonce** that is random even when the seed is pinned. Fixture ids come from the Postgres sequence, which `task test:setup` and `task test:reset` restart — never assert on a literal id either.

**Fix**: name fixture identities symbolically — the loaded model in Go, or `{user:alice.email}` / `{project:default.id}` interpolation in a corpus. Assert the *property*, not the value.

**Reproducing a real failure**: `UPTRACE_TEST_SEED` pins the base seed a failed run logged. The nonce stays fresh, so the replay seeds cleanly against the rows the failed run left behind. It is a debugging tool — no lane sets it.

### 3. Reads and writes wider than the fixture

The fixture's org and project are the only isolation a test has. Any read or write whose set is wider than what the fixture owns is a defect in an integration test, whatever the file looks like.

**Patterns**:

- A delete, update or count with no project filter: `Where("1=1")`, `DELETE FROM <table>`, `TRUNCATE`, `SELECT count(*) FROM <table>`.
- **A table with no project column** — mail, users, jobs, queue entries, audit rows — read by a range (`id > watermark`, `created_at > t0`) or in full, then counted or indexed (`rows[0]`). A watermark bounds *when* rows arrived, not *who* wrote them.
- **A product call whose own scope is the whole deployment** — a sweep, a claim, a list, a stats read — followed by an assertion on the size or the order of what it returned or produced.
- A globally scoped, page-limited read (`ORDER BY id LIMIT n`) that must return the test's own row. It passes until the shared backlog crosses the page, then fails for good.

**Why it breaks**: other packages write to the same tables in the same second, and earlier runs left their rows behind. A count or a position over a shared set is a claim about the whole database.

**Fix**: scope every write to the fixture's project. Make every read name something the fixture owns — its project id, its users' addresses, its monitors' ids — and assert on that subset. Where the product call is global by design, keep the call and filter what it produced by the fixture's own entities; position a paged read at the fixture's row (`sweep(t, inc)` in `alerting/incidents/sweep_test.go` is the model). A delete is justified only for subtest isolation or a globally unique value the test spells itself — say which in a comment.

**Exception**: none for integration tests. A pure unit test with an in-memory store may do as it likes.

### 4. The next message from a process-global channel

**Pattern**: reading one value from a `LISTEN`/`NOTIFY` channel, a package-level event bus, a shared Redis queue or a shared Kafka topic, and asserting on it directly.

**Why it breaks**: other tests sharing the servers emit on the same channels. The message you get is not necessarily the message you caused.

**Fix**: drain until you match on something you own — your own entity's id — with a single deadline across all skipped messages, then assert the rest. `alerting/monitors/pgnotif_test.go`'s `receivePGNotif` is the model. "Nothing was emitted" cannot be proven by reading one message: drain for a bounded window and check nothing matching *your* id arrived.

### 5. Shared state outside the harness's scope

**What the harness already scopes**: `pkg/testbed/diverge.go` gives every graph a hermetic in-memory `redcache`, and namespaces the rate limiter and the task-queue keys by run. A read through those is not shared — do not report it. Check that file before you call any cache or Redis read a defect.

**Pattern**: state the harness does not namespace, with a fixed name that every run and every package spells the same way — a distributed lock key (`distlock`), an advisory lock id, a Redis key a test or a product path builds by hand, a ClickHouse or Postgres table with a fixed name, a file outside `t.TempDir()`; a test that dials Redis or builds its own cache instead of taking the graph's.

**Why it breaks**: a concurrent run, the other build's lane, or a run killed a few seconds ago can hold the lock or the key. The code under test then skips its work — and the test either fails, or passes without testing anything.

**Fix**: take the graph's scoped dependency, or install a test double for the lock or key (a locker the test holds), or make the name unique per run. Register cleanup of what remains with `testbed.Cleanup`. Ask of every fixed name: "what if another process holds this right now?"

### 6. Spending the shared server

**Pattern**: a test that fans out goroutines, each holding or opening a connection; a loop that opens a new session per iteration (dial, cancel request, `LISTEN` connection); a stress test that ignores `testing.Short()`.

**Why it breaks**: the unit lane runs its packages in parallel against one Postgres with a fixed `max_connections`, and the integration lane shares one server between both builds. A test that demands backends faster than a loaded server can fork them fails with `too many clients` — and so do the packages beside it, whose logs then blame something else.

**Fix**: size the fan-out as a small share of the server's budget, not as "enough to hit the race". Count the connections one iteration costs, times the goroutines, times the rate. The fix lowers the peak: fewer goroutines, fewer sessions per second. A larger pool ("one connection per worker") never lowers it — it only moves the queue from the client to the server. Keep the dwell and the number of races if they matter; cut the concurrency. Gate the long form with `testing.Short()`.

### 7. Timing assumptions

This section covers every wait whose correctness depends on how fast something ran.

**Patterns**:

- `time.Sleep` before an assertion; `require.Eventually` with a tight timeout; a shrunk sleep constant.
- **A wait on a proxy signal.** The test releases on a value the handler sets, then asserts on a value that a hook or a deferred call sets *after* the handler returns. Nothing orders the two.
- **A timing margin as the proof.** The test proves a timeout, an expiry or a "still running" state by a small constant (a 50 ms client timeout, a 30 s flat budget) that a slow runner or a slow record write exceeds.
- **A budget in wall time for work counted in rounds.** A flat deadline for a burst that arrives only after N retries, rejoins or stall rounds.
- `require`/`t.Fatal` inside a goroutine the test started. `FailNow` off the test goroutine does not stop the test; it hangs or reports on the wrong test.

**Why it breaks**: each is tuned to one machine's timing. Under the race detector, or on a runner that shares its CPUs, it is tuned to nothing.

**Fix**:

- Wait on **the exact signal the assertion reads** — a channel the worker closes, a counter the assertion checks, a done callback — and assert the proxy as a consequence.
- Replace a margin with an **observed state**: hold the thing under test (an advisory lock, a stopped merge scheduler, a paused consumer) and release it only once the state is observed from outside (`pg_stat_activity`, `system.mutations`, a queue length). Then the timeout can be generous.
- Count a budget in the unit of work (`stalls × perStall`), not in seconds.
- Use `assert` off the test goroutine, or send the error back on a channel.
- `require.Eventually` takes a **generous timeout and a tight tick** (5–10 ms). Never shrink a sleep to make a test faster.

**Boundary**: `/go-test-speedup` §1 owns sleeps as a *cost* problem. Fixing it once satisfies both skills. If a rework here keeps flaking, it was masking a real race — do not re-widen the wait.

### 8. Unrestored global state

**Pattern**: `unixtime.SetClock` without `t.Cleanup`; a `featflag` override left installed; `os.Setenv` instead of `t.Setenv`; a mutated package-level var; a registry a test appends to; `t.Parallel()` in a test that touches any of these.

**Why it breaks**: it leaks into every test that runs after, in a way that depends on order — so it flakes somewhere else.

**Fix**: restore in `t.Cleanup`, always. Prefer `t.Setenv` (it restores and refuses a parallel test). A test that installs a global must not be parallel.

### 9. Order dependence and iteration order

**Pattern**: a test that passes only after another test ran (a cache warmed by a neighbor, a row a sibling seeded, a subtest that relies on the previous subtest's writes); a slice built by ranging a map compared against a fixed order; a golden whose rows come out of a map.

**Fix**: sort before comparing, or `require.ElementsMatch` when order is not part of the contract; render sorted in datadriven handlers. Make each test seed what it needs — `-shuffle=on` in procedure §4 is what catches it.

**Boundary**: `/go-test-decouple` §9 owns the same pattern as a *coupling* problem. Same fix; whichever skill is running does it.

### 10. Expectations that hold in only one build or machine

**Pattern**:

- An assertion whose expected value depends on a `featflag` — `Billing`, `Premium()`, a feature that `-tags=invariants` turns on — written as a constant. The comment often claims "holds under every build".
- A test that passes in the lanes but was never run under the **premium** tag, which is hosted-only and which no `cmd/gotest` lane runs.
- A float pinned by bits, or rendered into a golden at more than `%.12g` (see § "Determinism" in the conventions guide).
- Local timezone, DST at a chosen instant, locale, path separators, or a third-party/OS error message.

**Why it breaks**: CI is not your laptop, and the lanes are not every build the package compiles under. An expectation hard-coded for one build is a failure waiting for the build nobody runs.

**Fix**: take the expectation from the build under test (derive it from the same `featflag` the product reads), or exclude the test from the build where it cannot hold (`//go:build !cloud`) and say why. Run the package under every build it compiles in (procedure §4). Pin the timezone; choose instants away from DST unless the boundary is the behavior; compare floats with a tolerance; for third-party wording see `/go-test-decouple` §4.

### 11. Goroutine leaks

**Pattern**: a package with tests and no `main_test.go`; a `goleak.VerifyTestMain` with an ignore entry that has no comment, or one copied from a sibling.

**Why it breaks**: a leaked goroutine holds a connection or a lock past the test that created it, and the failure lands on whoever runs next.

**Fix**: follow § "goleak" in `docs/go-test-conventions.md`: start bare, add an ignore only when a failure names the goroutine, comment why it is benign, and never copy a sibling's list. `core/tenancy/main_test.go` is the house shape.

### Enforced elsewhere — confirm, do not hunt

A subtest that reports on its parent's `t` (an HTTP client or closure built on the outer `t`) is refused by `TestSubtestsReportOnTheirOwnT` in the repository root. A fixture that declares a globally unique literal is refused by `TestFixturesDeclareNoGloballyUniqueLiterals` in `dbfixture`. If the package predates either, run the policy test instead of sweeping by hand.

## Procedure

### 1. Baseline

Establish that the package is green *before* you touch it, so a failure during verification is attributable:

```
go test -race -count=1 ./<pkg>/...            # unit scope
go test -race -count=1 -p 1 ./<pkg>/...       # integration scope
```

Integration packages need the databases up (`task up`, then `task test:setup`) — see `.agents/skills/go-test-groom/references/verification.md` §4. If this run fails, stop: you have a reproducible bug, not an audit.

**`-p 1` on every integration run in this skill.** A recursive `./<pkg>/...` can expand to several packages, and Go runs distinct packages concurrently by default — on one shared server, which is the interference this audit exists to find.

### 2. Sweep mechanically, then read

Sweep the **tree**, not the directory: `<pkg>/*_test.go` expands only to the files directly under `<pkg>`.

```
grep -rnE --include='*_test.go' 'time\.Now\(\)|time\.Since|time\.Sleep|time\.After'   <pkg>/   # (1, 7)
grep -rnE --include='*_test.go' 'SetClock|FakeClock|\.Adjust\(|\.Set\(|Sched\(\)'      <pkg>/   # (1, 8)
grep -rnE --include='*_test.go' 'os\.Setenv|featflag\.|t\.Parallel|go:build'           <pkg>/   # (8, 10)
grep -rnE --include='*_test.go' 'Where\("1=1"\)|DELETE FROM|TRUNCATE|count\(|Count\('  <pkg>/   # (3)
grep -rnE --include='*_test.go' '"id > |id > \?|created_at > |Limit\(|\[0\]'          <pkg>/   # (3) watermark / first-row reads
grep -rnE --include='*_test.go' 'Listen|Notify|<-ch\b|range ch\b|Reserve|Consume'      <pkg>/   # (4)
grep -rnE --include='*_test.go' 'distlock|Locker|advisory|redis\.|NewClient|os\.(Create|WriteFile)' <pkg>/   # (5)
grep -rnE --include='*_test.go' 'for [a-z]+ := (0|range [0-9]+)|go func'               <pkg>/   # (6, 7) fan-out, goroutines
grep -rnE --include='*_test.go' 'Eventually|WaitGroup|Interval|Timeout'                <pkg>/   # (7)
grep -rnE --include='*_test.go' 'for [a-z]+ := range .*[Mm]ap|maps\.Keys'              <pkg>/   # (9)
grep -rnE --include='*_test.go' 'rand\.|uuid\.|time\.Now\(\)\.Unix'                    <pkg>/   # (2)
ls <pkg>/main_test.go                                                                           # (11)
grep -rnE '@example\.(com|org)|[0-9a-f]{8}-[0-9a-f]{4}-' <pkg>/testdata/                        # (2) corpora
```

The grep finds candidates; it does not find §3's product calls, §5's lock and key names inside product code, or §7's proxy waits. For those, **read every test that asserts a count, a position, a first element, or a value read after a wait**, and answer the three questions above for it. Follow each product call the assertion depends on far enough to state the set it reads.

### 3. Classify and fix

One row per candidate: `file:line | category | verdict | fix`. Verdicts are `defect`, `safe` (with the reason), or `borderline`. For borderline, ask what a sibling test in another package, or last week's run, would put there; if the answer is "it depends", it is a defect.

**Before you call anything a defect, write its failure path**: the concrete trigger (which row, which process, how long a stall, at which line), and what the test then does — fails, hangs, or passes without testing its claim. Then check the path against the code, not against the pattern:

- For a stray row or a stray match, read the whole query, including its joins and `NOT EXISTS` clauses, and the fixture preset. A filter that looks too wide may already exclude the row you imagine.
- For a sleep, name the event it waits for and ask whether that event can still be in flight when the sleep ends. A sleep that waits for something already done (a loopback close, a synchronous call) is redundant, not a margin. Remove it only if it costs time; it is not a flake.
- For a deadline or budget, compute the expected duration from the code (a retry interval, a round count). Raise a budget only when the expected duration comes near it; padding a budget with slack is not a fix.

A candidate whose path you cannot write is `borderline`, not `defect`, and it gets no edit. Mark in the report how each defect is proven: planted, reproduced, or read-only.

Fix in place with the smallest edit that removes the nondeterminism. Do not reshape, rename, or add cases while you are in there.

When the nondeterminism is in the **production code** (it reads `time.Now()` directly, sleeps on package `time`, returns a map where callers need order), do not add an interface just to make the test deterministic — CLAUDE.md forbids that. Switch to `unixtime.Now()` or `sched.Clock` where that is a natural fit, or flag it for the user.

### 4. Verify — repetition, shuffle, race, builds, load

A single green run means nothing here. Each check targets a different class:

| Check | Unit package | Integration package (imports `pkg/testbed`) | Catches |
|---|---|---|---|
| Repetition | `go test -count=20 ./<pkg>/...` | `go test -p 1 -count=5 ./<pkg>/...` | seed and timing variance |
| Shuffle | `go test -shuffle=on -count=10 ./<pkg>/...` | `go test -p 1 -shuffle=on -count=3 ./<pkg>/...` | order, leaked globals |
| Race | `go test -race -count=10 ./<pkg>/...` | `go test -p 1 -race -count=2 ./<pkg>/...` | data races, tight waits |
| One CPU | `go test -cpu=1 -count=10 ./<pkg>/...` | `go test -p 1 -cpu=1 -count=2 ./<pkg>/...` | timer and margin assumptions (§7) |
| Builds | add `-tags=invariants` and `-tags=premium` runs where the package compiles under them | same | §10 |

```
grep -rl --include='*_test.go' 'pkg/testbed' <pkg>/   # which column?
```

The counts are a floor, not a target: raise them when a specific defect needs the odds, and say in the report what you ran. Check `N × single-run time` first. Never run these concurrently with each other or with another lane — they share one server, and parallel runs destroy failure attribution.

For an integration package, also run it next to the siblings that share its fixtures: `go test -p 1 -count=2 ./<parent>/...`.

For a §3 or §5 finding, a green stress run is weak evidence, because the stray row or stale key must exist first. **Prove the finding by planting it**: insert one row that another project would write, or take the lock or key another process would hold, and show the old assertion fails and the new one passes. Remove the planted state afterwards. Do the same for a §6 or §7 finding where you can starve the resource (fewer CPUs, a lower connection limit on a throwaway server).

Verify the counts took: the pass total should be exactly `tests × N`. A short total means tests were skipped or the flag did not apply.

One failure in any of these is a finding, not noise — re-run to capture the logged seed, then pin it with `UPTRACE_TEST_SEED`. A failure that survives the pin is reproducible, not necessarily timing: drop the pin (a failure only that seed produces is data-dependent), then `-shuffle=on` at a fixed seed (a failure only some orders produce is order dependence). Only what survives both is timing.

If a failure is a connection or constraint error unrelated to your edits, confirm no other process is using those databases before diagnosing (`references/verification.md` §4 in `go-test-groom`).

### 5. Finalize

```
task test:pkg -- ./<pkg>/...              # both build tags
```

Escalate to `task test` if the fixes touched shared fixtures, `main_test.go` ignores, or helpers outside the package.

## Output to the user

- **Sweep summary**: candidates per category, verdict counts, and whether the corpora and the product calls behind count/position assertions were read. A zero-defect report must show the per-category evidence.
- **Defects fixed**: file, line, category, what it was → what it became, and how it was proven (planted state, starved run, or stress).
- **Judged safe**: each with its reason — especially `time.Now()` calls, `t.Parallel()` uses, and counts that were kept.
- **Verification**: the runs performed, the lane column, the builds, and the pass totals. If a run flaked, say so and what you concluded.
- **Product-code flags**: nondeterminism that lives in the source, with the change you would recommend.
- **Cross-test risk left in place**: anything that damages siblings but could not be fixed within this package's scope.

Stay terse. The diff is the artifact; the summary routes attention.

## The test-* suite

This skill is orthogonal to the grooming pipeline (`/go-test-reshape` → `/go-test-decouple` → `/go-test-coverage` → `/go-test-cleanup`); run it standalone, or after grooming, since new cases are new chances to introduce these defects. `/go-test-groom` runs the pipeline end-to-end and always ends with this audit.

## Additional resources

- `docs/go-test-conventions.md` and `pkg/testbed/AGENTS.md` — the rules this audit enforces.
- `pkg/testbed/README.md` — "The run environment": the seed, the clock epoch, `UPTRACE_TEST_SEED`, and `testbed.Cleanup`.
- `pkg/testbed/diverge.go` — what the harness substitutes: the clock, the seeded faker, the id and token generators, the hermetic cache and limiter. It answers "is this value the harness's or the code's?"
- `pkg/sched/schedtest/README.md` — driving a loop from the fake clock, and why the scheduler is the clock's only writer.
- `alerting/monitors/pgnotif_test.go` — `receivePGNotif`, the drain-until-match model for §4.
- `alerting/incidents/sweep_test.go` — `sweep(t, inc)`, positioning a paged read at the fixture's own row for §3.
- `core/tenancy/main_test.go` — the `goleak.VerifyTestMain` house shape.
