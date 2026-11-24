---
name: go-optimize-bench
description: "Optimize a Go package or file with profiles, benchmarks, and benchstat. This includes assembly kernels that Go calls. Invoke only when the user explicitly names go-optimize-bench. Do not infer it from a generic performance request. Use go-optimize-bundle for production profile archives. Use go-bench-coverage for systematic benchmark coverage. Do not change HTTP, wire, schema, or configuration contracts."
---

# go-optimize-bench

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
- Compile-check with `go vet ./...` and `go vet -tags=invariants ./...`. The
  build scripts' `-t` flag takes `invariants`.

Optimize Go code, and prove each change with an interleaved A/B of compiled test binaries and `benchstat`. Profile before guessing. Change one thing per round. Reject anything that does not clear its threshold, or that costs more readability and blast radius than the win is worth.

## Goal

Leave the named hot path faster or leaner on realistic workloads, with every accepted change proved. A change is accepted only when **all four** hold:

1. **Measured** — an interleaved A/B (*How to measure*) gives benchstat p < 0.05 on the improved metric, and the delta survives *Before you believe a delta*.
2. **Material** — the improvement on the *deciding* number (*Which number decides*) clears the metric's threshold, scaled by the change's reach (*Reach*, *Thresholds*).
3. **Correct** — the tests of every touched package pass unchanged, and the new code path is proved to be tested (step 6b). Observable behavior is identical, down to the bits of every result. A faster wrong answer is not an optimization.
4. **Worth it** — readability cost and blast radius are proportional to the win. Anything that reaches past the target package has cleared its reach's bar: announced at R1, **approved** at R2, refused at R3.

**Done when** every cost the profile named has been optimized or declined with a reason, and the final interleaved check of every touched package shows no unexplained regression.

**Deliverable**: the optimized code; any benchmarks written along the way, committed before the optimization; and the report in step 8. "The code is already well optimized for this workload" is a successful outcome. Report it and stop.

**Non-goals** — hand these off:

- Systematic benchmark coverage for a package → `/go-bench-coverage`. This skill writes benchmarks only as far as it needs them to measure its own target.
- Restructuring for clarity, layering, or API design → `/plan-ideal-refactor`.
- Changing what the code does or the contracts it exposes → `/redesign`.
- SQL, index, engine, or schema tuning → `/plan-ideal-refactor` with the `ch` or `pg` territory. Benchmark the Go-side work; a wall-clock DB benchmark measures the server.
- Speculative micro-tuning with no benchmark behind it.

## How to measure

Read this section first. Every decision in the process rests on it.

**Why not "baseline once, then compare".** Benchmark numbers on a dev box drift within minutes, and more as a long session heats it. Two sequential `-count=10` runs of the same code an hour apart have reported a 22% "win". A back-to-back pair has turned a real −3.7% into +15%. Ratios inside one run stay stable; absolutes across runs do not. So never compare a file captured earlier against a file captured now. Build both sides as binaries and run them in alternating short blocks, so drift lands on both sides equally.

Two bundled scripts do this. Run them from the repository root, and keep all files in the session's scratch directory (`$S` below), not in the tree.

```bash
SK=.agents/skills/go-optimize-bench/scripts

# Build a test binary at a revision ("." = the working tree). A revision builds
# in a detached worktree, so the working tree is never touched. GOAMD64=v3 and
# GOARM64=v8.0 match what cmd/gotest pins for every lane.
$SK/buildbin.sh [-t invariants] <base-rev> ./pkg/x $S/base.test
$SK/buildbin.sh [-t invariants] .          ./pkg/x $S/head.test

# Interleave: each round runs every binary once, in a rotating order.
# Run from the package directory: testdata paths are relative, and a binary
# run elsewhere skips its benchmarks silently (the script stops on that).
$SK/abbench.sh -d pkg/x -p '^BenchmarkFoo$/size=.*' -o $S/round3 -n 10 \
  $S/base.test $S/head.test
```

`abbench.sh` accepts more than two binaries. Use this to compare stacked variants in one run (base, A, A+B, A+C) and isolate each part. It writes `OUTPREFIX.<name>.txt` per binary and a load log. It stops when a block prints no benchmark, when two binaries are byte-identical (a stale build), or when a benchmark fails.

Rules that make the comparison mean something:

- **Build both sides from the same test-file set.** Benchmarks are committed before the first optimization (step 2), so the base revision already holds them. When you add a benchmark mid-run, commit it, then rebuild the base from a worktree of the original revision with the new test files checked out into it (`git -C <wt> checkout <commit> -- <test files>`). Binaries from different test files have flipped the sign of a result.
- **Anchor the pattern.** `-p 'Compress/...'` also matches `BenchmarkEncoderColdCompression`. Write `^BenchmarkCompress$/...`.
- **Include a control.** Name at least one benchmark that the change cannot reach — a reference implementation, a sibling code path, an untouched arm. If the control moves by a similar amount, the delta is drift or layout, not the change.
- **Samples.** Default 10 rounds of `-c 1`, so 10 samples a side. Never fewer than 6. For a benchmark that runs for a long time, use fewer rounds with `-c 2`, not fewer samples.
- **Run nothing else heavy at the same time.** Do not run a benchmark during a `task test` lane, another benchmark, or a build. For a run longer than a few minutes, start it in the background and wait for the notification. A background run can be killed by the memory watchdog even with free memory; rerun it in the foreground if that happens.
- **Read the variance column.** If one side is at ±10% or worse while the other is at ±1%, that side ran in a noisy window. Rerun it; do not report the delta.

Use `go test -bench` directly only for a first look at a benchmark's numbers and for profiling (step 5). Never use it to accept or reject a change.

### Before you believe a delta

A delta that clears p < 0.05 in an interleaved run can still be false. Before you accept a win, or before you start investigating a regression, check:

1. **The control.** Did it stay flat? If it moved, subtract its movement, or rerun.
2. **The arithmetic.** How many times does the changed code run per iteration, and what does it cost alone? If it cannot account for the delta, the delta is not the change. A call worth tens of nanoseconds cannot explain microseconds, and it cannot explain a change of the opposite sign.
3. **The work.** Does each side do the same work? For deterministic output — compressed bytes, a ratio, a result slice — compare the outputs of both binaries. For iterative numeric code, count the iterations with a temporary counter. A win from less work (fewer iterations, a result that changed by one ULP) is a behavior change, not a speed-up.
4. **Binary layout.** A stable small delta on a benchmark whose time is dominated by code you did not touch is often layout. Removing one import changes the link set; adding a function moves the instruction stream. Test it:
   - Equalize the link set. If the change dropped an import, add `var _ = otherpkg.Func` in a test file of the head build, and rerun. If the delta disappears, it was layout.
   - Profile both binaries and compare the flat time per function. If the delta sits in a function neither side changed (often an inlined copy), it is layout.
   - Take a second layout sample. Build both sides again at `GOAMD64=v1` (`GOAMD64=v1 buildbin.sh ...`) and rerun. The instruction stream moves, so a layout artifact often shrinks or flips sign, while a real win holds. `v3` stays the build that decides, because production ships it. In a package whose hot loop is known to swing with its address, do this for every candidate.
   - Padding functions do not reproduce layout effects. Do not conclude "not layout" from a padding test.
5. **The consumer.** A kernel can win on its own benchmark and lose in its caller with identical work, because the caller's instruction stream moved. When the target is a helper that other packages call, find its hot callers (`grep` the import path) and run their benchmarks too, even at R0. *Which number decides* says what the consumer's result must show.

Treat a delta of a few percent on an end-to-end benchmark as below the noise floor of the build unless all five checks pass.

### Which number decides

A benchmark reports many rows. Decide on the ones that stand for the work the program does, and say which before you measure.

- **Production shapes decide, not the best row.** A size or shape sweep (`n=8`, `n=100`; `32KiB`, `1MiB`) is one benchmark. Before you measure, name the rows that match what production runs, with the evidence: a constant or default at the call site, a production profile, a documented workload. Those rows must clear the bar. If you cannot tell which rows production runs, the geomean over all rows must clear it. A row that wins because it is small or large, and that production does not run, decides nothing.
- **A complexity change wins with size.** When the change lowers the order of the cost (O(n²) to O(n), a second pass removed), expect the gain to grow along the size axis and to vanish at small sizes. Check that the rows show that shape, and judge at the production size. A flat gain across sizes means the change did something other than what you think.
- **A helper is judged through its callers.** When the target matters only through code that calls it — a numeric kernel under a model fit, a codec primitive under a reader — the target's own benchmark must clear the bar, *and* each hot caller's benchmark must move the way the profile predicts: roughly the target's share of the caller times the target's gain. No caller row may regress beyond the noise floor. Where the predicted gain is above the noise, a caller that stays flat or regresses means the win did not reach the program; reject it. Where the target is too small a share of a caller to show, say so with the arithmetic, and require only that the caller does not regress.
- **Across a seam (R1/R2),** the seam benchmark decides. The callee's benchmark is supporting evidence only.

State the deciding rows and the rule next to the thresholds before each round.

## Reach — how far a change may go

The cost is often not where the slow benchmark is. An exported signature forces a copy, a caller builds a map the callee flattens, or the real kernel lives in a shared `pkg/*` helper. Refusing to cross the package boundary caps the win at the wrong place, so crossing is allowed — priced by reach.

| Reach | What changes | Threshold multiplier | Approval |
|-------|--------------|----------------------|----------|
| **R0** | Target package internals only; exported API untouched | ×1 | none |
| **R1** | Target package's exported API changes; call sites updated mechanically | ×1.5 | tell the user, then proceed |
| **R2** | Code in other packages changes — hot kernel elsewhere, or a caller's data shape is the cost | ×2 | propose and **wait** for approval |
| **R3** | A contract changes: HTTP/JSON API, Kafka or wire format, ClickHouse/Postgres schema, config keys, persisted data | — | **STOP** — not this skill |

**Precedence.** Classify by where the *substance* of the change is, not by how many files it opens. A change is **R1** when its only effect outside the target package is the call-site updates the compiler dictates. It is **R2** as soon as another package's own code changes: its logic, its data layout, a kernel it owns. A signature change that also rewrites what a caller builds is R2.

### Rules for R1 and R2

- **Evidence, not suspicion.** Cross a package boundary only when the profile or escape analysis (step 5) puts the cost on the other side. "It would be cleaner if the caller passed a slice" is a refactor; route it to `/plan-ideal-refactor`.
- **Measure across the seam.** Moving work from callee to caller improves the callee's benchmark while the program does the same work. Only a benchmark that exercises *both* sides tells a real win from relocated cost. If none exists, write one in the house style (`b.Loop()`, `b.ReportAllocs()`, realistic input — see `/go-bench-coverage`) and commit it before the change.
- **Price the blast radius before proposing.** State the packages touched, the call-site count, whether it crosses a bounded context (`CONTEXT-MAP.md`), whether both build tags are affected, and whether any touched package has its own `AGENTS.md`. A 25% win that rewrites 40 call sites across 6 packages is a *plan*; hand it to `/plan-ideal-refactor` with the measurement as evidence.
- **Keep the mechanical part separate.** One commit for the signature change and call-site updates (behavior-neutral); the next for the optimization it enables.
- **Never edit vendored or third-party code** (`pkg/tempo`, module dependencies) to win a benchmark. Report the finding instead.

### R3 — the hard stop

If the only route to the win changes something outside Go source — an `/internal/v1/...` response shape, an enum's wire value, a ClickHouse/Postgres column or ORDER BY, a Kafka message, a YAML config key, or anything already persisted — **stop**. Report it as a finding with the estimated win and the evidence, and name the skill that owns it (`/redesign`, `/plan-ideal-refactor` with the `ch` or `pg` territory).

## Thresholds

Resolve CPU time, allocated bytes, and allocation count independently.
The time metric is benchmark `sec/op`, not the package's share of a CPU profile.
Allocated bytes measure `B/op`, not retained heap size or process memory.

| Metric | Normal default | Top performance default | Numeric override |
| --- | --- | --- | --- |
| Go time (`sec/op`) | **10%** | **2%** | `--time-threshold` |
| Assembly time (`sec/op`) | **2%** | **2%** | `--time-threshold` |
| Allocated bytes (`B/op`) | **15%** | **5%** | `--mem-threshold` |
| Allocation count (`allocs/op`) | **20%** | **5%** | `--alloc-threshold` |

### Recognize the package goal

Read the target package's own `DESIGN.md` before selecting thresholds.
A top performance goal selects the lower defaults when it contains this exact marker outside a code fence:

```html
<!-- go-optimize: highly-optimized -->
```

`pkg/gzip/DESIGN.md` and `pkg/zstd/DESIGN.md` use this marker.
General performance prose does not select the lower defaults.
A parent, child, sibling, or imported package does not inherit the marker.
A missing file or marker selects the normal defaults.
Do not add or remove the marker to make an optimization pass.
The marker declares a user-approved design goal; it does not replace evidence or authorize wider changes.

A package's `DESIGN.md` can also state a goal that a speed change must not break — a compression ratio, a relative-throughput floor against a reference implementation, a detection-quality bar. Such a goal is a gate, like the tests. When the package measures itself against a reference (ours ÷ libzstd, for example), that ratio is the deciding metric, and the reference arm is also the control.

### Resolve overrides and multipliers

For each metric, use this precedence:

1. The metric's numeric flag.
2. The uniform `--threshold=N%` flag.
3. The package's top performance default, when marked.
4. The normal default.

An override for one metric leaves the other metrics unchanged.
For time, an explicit value replaces both the Go and the assembly default.

Then apply the multipliers to the resolved base:

- **Reach** always applies: R1 ×1.5, R2 ×2.
- **Readability** applies to Go time in an unmarked package, and to allocated bytes and allocation count everywhere: medium ×1.5, high ×2. It does not apply to assembly time, or to Go time in a marked package. There, a 2% base gives 2% at R0, 3% at R1, and 4% at R2.

If a round changes implementations in more than one package, resolve each package separately, and use the strictest threshold for each metric. Mechanical call-site updates do not count.

Examples at R0 with low readability cost:

| Package and flags | Time | Allocated bytes | Allocation count |
| --- | --- | --- | --- |
| Unmarked Go package | 10% | 15% | 20% |
| Marked Go package | 2% | 5% | 5% |
| Marked; `--mem-threshold=3%` | 2% | 3% | 5% |
| Marked; `--threshold=8% --time-threshold=3% --alloc-threshold=4%` | 3% | 8% | 4% |

Readability cost:

- **Low**: pre-sizing a slice, reordering struct fields, `sync.Pool` for a clearly poolable object, reusing an existing helper
- **Medium**: manual loop unrolling, replacing an interface with a concrete type, adding a fast path
- **High**: unsafe pointer tricks, hand-inlined code, bit manipulation that replaces clear arithmetic

Rate readability before you measure, and do not re-rate a candidate after it misses its bar. If the user asks whether a rejected change has a readability cost, answer honestly, and let the user decide.

Before each round, report each base threshold, its source, its multipliers, and the resulting required gain.

## Process

This skill needs `benchstat`. It is usually in `$(go env GOPATH)/bin`, which is not always on `PATH`; `abbench.sh` adds it. If it is missing, run `go install golang.org/x/perf/cmd/benchstat@latest`.

Pick a scratch directory for the session (`$S`) and keep every binary, result file, profile, and patch there.

### 1. Start from a known state

Run `git status`.

- A clean tree: go on.
- An uncommitted change that *is* the candidate the user wants measured: save it as a patch in `$S`, and measure it as round 1 against a base binary built from `HEAD`.
- Any other uncommitted change: ask the user to commit or stash it first.

Branch off `dev` if you are on it (`git switch -c perf/<topic> dev`).

### 2. Identify the target and check the premise

Parse the argument into a target package or file. If it is ambiguous, ask. The target names where the *symptom* is; the reach needed is decided in step 6a, from the profile.

- Check whether the package needs build tags (`-tags=invariants`, `-tags=invariants`), and use them for every build.
- Read the package's `README.md`, `AGENTS.md`, `DESIGN.md`, and `DESIGN-EVIDENCE.md` if present. A recorded decision may already explain the slow shape, and a record's *Alternatives* often lists ideas that were measured and lost. Do not retry one of those without a reason the record did not have.
- Resolve the thresholds (*Thresholds*).
- **If the user states a number** ("runs at 80% of libzstd", "takes 40ms"), reproduce it first. If it does not reproduce on this box, report that before you optimize anything. Numbers from another CPU often do not hold.

Find the benchmarks: `grep -n "func Benchmark" <package_dir>/*_test.go`.

**A benchmark must exercise what you claim to optimize.** Profile it (step 5) and confirm the target function is in the profile. Existing benchmarks have used a mode that never reaches the code in question, or a size too small to show a quadratic cost. If the benchmark is missing or wrong, write or fix it before anything else. Benchmark the public API with realistic input. Keep setup out of the loop with `for b.Loop()`:

```go
func BenchmarkProcess(b *testing.B) {
    data := buildLargeTestInput() // setup, excluded from timing
    b.ReportAllocs()
    for b.Loop() {
        Process(data)
    }
}
```

Include sizes that expose growth: if a cost can be superlinear, benchmark at least two sizes an order of magnitude apart. If the package's benchmark surface is broadly missing or untrustworthy, say so and suggest `/go-bench-coverage` first.

Commit new or fixed benchmarks before any optimization (`test: add benchmarks for <target>`). That commit is the **base revision** for the session.

### 3. Validate and look

```bash
go test <tags> -run='^$' -bench='<pattern>' -benchmem -count=1 <package>
```

Fix compile and runtime errors. Report these first numbers to the user as orientation, not as a baseline.

Build the base binary now: `$SK/buildbin.sh <base-rev> <package> $S/base.test`. It stays the reference for the whole session.

### 4. Establish the noise floor

When the required gain is small (5% or less), run an A/A first: the base binary against a copy of itself, over the target pattern and the control.

```bash
cp $S/base.test $S/aa.test
$SK/abbench.sh -a -d <package_dir> -p '<pattern>|<control>' -o $S/aa $S/base.test $S/aa.test
```

The spread you get is this box's noise for these benchmarks today, and a win must be well clear of it. If the noise is as large as the required gain, say so: raise the rounds, or tell the user that this box cannot prove a change that small. Run it again if the box's load changes (another job starts, a long run heats it).

### 5. Profile before guessing

Profile the benchmark, not the code you suspect. Profile the current state (`$S/cur.test`): the base binary in the first round, the last accepted round's binary later. Run CPU and memory profiles separately, with a `-benchtime` long enough for a few thousand samples (the CPU profiler samples at about 100 Hz):

```bash
cd <package_dir>
$S/cur.test -test.run='^$' -test.bench='<pattern>' -test.benchtime=3s -test.cpuprofile=$S/cpu.prof
$S/cur.test -test.run='^$' -test.bench='<pattern>' -test.benchtime=3s -test.memprofile=$S/mem.prof
go tool pprof -top $S/cur.test $S/cpu.prof
go tool pprof -top -cum $S/cur.test $S/cpu.prof           # which callers own the cost
go tool pprof -peek='<hot func>' $S/cur.test $S/cpu.prof  # its callers and callees
go tool pprof -list='<hot func>' $S/cur.test $S/cpu.prof  # line-level cost
go tool pprof -sample_index=alloc_space -top $S/cur.test $S/mem.prof
```

**Profile the corners of a sweep, not one row.** The cost mix changes along a benchmark's axes: a cost that grows with one axis (horizon, batch size, depth) can be 4% of the largest-history row and 20% of the longest-horizon row. Profile the smallest and largest row of each axis, and each mode or arm, separately — `-test.bench` with one row per run. A profile over the whole sweep is dominated by its slowest rows and hides the rest. Before you rank or decline a cost, state its share on each deciding row.

Also read the hot path for costs that grow faster than the input: a loop that restarts from the beginning at each step, a search inside a loop, a copy per append. A profile at one size cannot show a quadratic cost; the rows at two sizes can.

**Attribute the cost to a package, not only a function.** If the top cumulative entries are in another package — a shared helper, a conversion the caller forces, an interface dispatch across the seam — that is the evidence an R2 change needs. If they are all inside the target package, stay at R0.

When allocations are the cost, find out why they escape:

```bash
go test <tags> -gcflags='-m' -run='^$' -bench='^$' <package> 2>&1 | grep 'escapes to heap'
```

An argument that escapes because of an *exported signature* (an `any` parameter, a returned pointer) is R1 evidence.

For assembly and for code below about 100ns per call, also read *Low-level and assembly work*.

### 6. Optimization loop

Repeat for each optimization. Apply one logically grouped change per round.

#### 6a. Analyze, classify reach, optimize

Prioritize by expected impact, from the profile:

1. **Algorithmic** — better data structures, fewer passes, lower complexity
2. **Allocation reduction** — pool, reuse, stack-allocate, pre-size slices and maps
3. **Boundary shape** — an exported signature or a caller's data layout that forces copying, boxing, or a conversion (R1/R2)
4. **Layout** — struct field order, cache-line alignment
5. **Hot-path shortcuts** — fast paths for common cases, early returns
6. **Syscall and I/O reduction** — batching, buffering

For each candidate, state before touching code: **metric moved, expected win, reach, readability cost, required gain.** Also state the mechanism: why this change should make the CPU do less, wait less, or allocate less. A candidate without a mechanism is a guess; profile more.

For **R1**, tell the user the signature change and how many call sites it moves, then proceed. For **R2**, present the cost attribution, the shape of the change, every package touched, the call-site count, and the expected win, and **wait for approval**. For **R3**, stop and report.

#### 6b. Verify correctness

Before you measure a candidate, prove it is correct:

- **Tests.** Run `go test <tags>` over the touched packages. At R1 also run every package with a rewritten call site, and compile-check both builds:
  ```bash
  go vet ./...
  go vet -tags=invariants ./...
  ```
  Do not run a `task test:*` lane per round. Run `task test:pkg` once when the code is complete (step 7).
- **The new path is tested.** A new fast path, special case, or kernel must be reached by a test that fails when it is wrong. Break it on purpose — return early, flip a condition, skip a byte — and run the tests. If they still pass, restore the code and note the gap. Write the missing test case after the round is measured (6c), and commit it on its own. A test added before the measurement changes the head binary but not the base, and that alone can move unrelated benchmarks by 10% through layout. Undo each mutation with `git checkout HEAD -- <file>` and confirm with `git diff`.
- **Output identity.** Compare the outputs of the base and head binaries, built the same way (`buildbin.sh` pins `GOAMD64=v3`, as production does): encoded bytes, a ratio, a sorted result, every float bit. Equal outputs are the contract.
  - If any bit differs, the change alters behavior, even when every test passes and the difference is one ULP. A rounding change can send an iterative caller down a different path, so it can change the *work* the benchmark measures. Do not accept it as an optimization. Report the largest difference, what it does to each caller's work (iteration counts, allocations), and any tolerance the package's `DESIGN.md` states, and put it to the user as a trade. Without the user's decision, reject it.
  - Do not confuse this with a difference between two *builds*: a bare `go test` without `GOAMD64=v3` can round differently from the lanes. Compare binaries built the same way (see `anomaly/detectors/detseries/DESIGN.md`).
- **Fuzz.** If the package has a fuzz target for the changed path, run it for 30 seconds (`-fuzz=<Target> -fuzztime=30s`).
- **A quality gate.** If the package's `DESIGN.md` states a goal the change can move (ratio, detection quality), measure it now. A speed win that fails that goal is rejected, or taken to the user as a trade.

If anything fails, fix the candidate or revert it.

#### 6c. Measure the round

Build the candidate and run it interleaved against the **previous accepted state** — the binary of the last accepted round, or the base binary in round 1. This isolates the round's own effect. Include the control in the pattern:

```bash
$SK/buildbin.sh . <package> $S/r3.test
$SK/abbench.sh -d <package_dir> -p '<pattern>|<control>' -o $S/round3 $S/r2.test $S/r3.test
```

Then apply *Before you believe a delta*.

For an R1/R2 change, the deciding number is the **seam benchmark**, not the callee's benchmark. Report both. A callee that got 40% faster while the seam benchmark moved `~` means the cost moved, and was not removed.

#### 6d. Accept or reject

**Accept** when the deciding number (*Which number decides*) is significant (p < 0.05) and clears the required gain, the delta passes *Before you believe a delta*, and the outputs are identical (6b).

**Also accept** a result that is unchanged (`~`) or below the threshold when the new code is **simpler** than the old code: fewer lines, fewer concepts, removed complexity. This applies at R0 only. A change that adds a fast path, a branch, a kernel, assembly, or unsafe code is never "simpler", and the clause cannot keep it. Say which rule accepted each round.

**Reject** when:

- p ≥ 0.05 on the claimed improvement: not significant.
- the gain is below the required gain, and the code is not simpler.
- the win appears only in the callee's benchmark, not across the seam: cost relocated.
- the win appears in one row of a sweep, or in the helper but not in its callers.
- any output bit differs, and the user has not accepted that trade.
- the delta fails *Before you believe a delta*.
- any metric regresses without a compensating gain.

When you reject, save the candidate's diff as `$S/rejected-<name>.patch`, revert only this round's edits (`git checkout HEAD -- <files>`), and record the result for the report: the change, the numbers, and the reason. A measured loss is a finding. It stops the next session from trying the same idea.

**Commit each accepted round before the next** (`perf(<scope>): <what it did>`), and keep its binary as the next round's reference. Then a revert can never discard an earlier win.

#### 6e. Loop or stop

If the profile still names costs you have not addressed, re-profile the new head and go back to 6a. Otherwise go to step 7.

### 7. Final check

1. **Cumulative win.** Run `abbench.sh` with the base binary against the head binary of the last `perf` commit, over the target pattern and the control. Build that head before any test-only commit, so both binaries share the same test files. This is the net result you report.
2. **No regression.** For every touched package, build base and head binaries of *that* package and run its full suite (`-p .`) interleaved. Choose the rounds so the run finishes in reasonable time; for a slow suite, use `-t` with a shorter benchtime rather than fewer than 6 samples. A package edited at R1/R2 that has no benchmarks of its own cannot be checked this way; say so in the report.
3. **Explain every regression** with *Before you believe a delta*. A regression on a benchmark that runs none of the changed code is usually layout. To find which commit moved it, build one binary per commit and interleave them.
4. **Tests.** Run `task test:pkg -- <touched packages> <their callers>` once. For an R2 change, say that a scoped run is not the integration lane, and run `task test` before a PR.

### 8. Report

```
## Optimization results

Target: <package>          Benchmark: <pattern>          Control: <benchmark>
Deciding number: <rows or geomean, and callers — see *Which number decides*>
Method: interleaved <rounds>x<count> per side, GOAMD64=v3, noise floor <±N%>
Reach: R0 / R1 / R2 — packages touched: <list>
Thresholds: <each metric's base, source, multipliers, and required gain>

### Accepted
| Commit | Change | Reach | sec/op | B/op | allocs/op | p | Accepted by |
|--------|--------|-------|--------|------|-----------|---|-------------|
| abc123 | ...    | R0    | -25%   | -30% | -40%      | 0.000 | threshold |
| def456 | ...    | R0    | ~      | ~    | ~         | —     | simpler code |

### Rejected
| Change | Reach | Result | Reason |
|--------|-------|--------|--------|
| ...    | R0    | +1.3%  | adds a multiply and two spills to the loop |

### Blast radius (R1/R2 only)
Packages changed, call sites rewritten, contexts crossed, build tags affected, test lane run.

### Not acted on
R3 items and other out-of-scope findings, with the estimated win and the owning skill.

### Cumulative benchstat
<base vs final head, target pattern and control>

### Regression check
<per touched package: clean, or each regression and what explained it>
```

After the report, offer to record the non-obvious rejected ideas where the next session will read them: the *Alternatives* of the matching decision in the package's `DESIGN.md` (use the `design-doc` skill's format), or `DESIGN-EVIDENCE.md` for the numbers. Record the mechanism, not only the number: "+1.3%: two extra spills in the loop" stops a retry; "+1.3%" does not.

### 9. Self-review

- Could the same gain come from a simpler change, or at a smaller reach?
- Did any accepted change make the code harder to maintain, or push cost onto callers?
- Do the benchmarks reflect real use: realistic sizes, real data shapes?
- Does any accepted change need a `COMPROMISE:` comment (contained in one file or function) or a `DESIGN.md` entry (a decision two or more files must uphold together — common after R1/R2)?

## Low-level and assembly work

Below about 100ns per operation, and in any assembly kernel, the usual intuition fails. Instructions that look free cost time, and removing instructions can make code slower.

- **Build as the lanes build.** `buildbin.sh` sets `GOAMD64=v3`. Compile-check arm64 too: `GOARCH=arm64 go vet <package>`.
- **Find what bounds the loop before changing it.** A loop is either latency-bound (one long dependency chain per item: load → hash → table → compare) or throughput-bound (instruction and load count). In a latency-bound loop, a change that adds a link to the chain or an unpredictable branch loses even when it removes instructions; replacing a well-predicted branch with CMOVs can make it slower. In a throughput-bound loop, instruction count is the currency: two fewer instructions per item can win 2%, and a prefetch that adds four can lose. Test the hypothesis with a probe, such as a smaller table or a skipped step, before you commit to a design.
- **Hardware counters.** Try `sudo -n perf stat -e instructions,cycles,branches,branch-misses,L1-dcache-load-misses <binary> -test.run='^$' -test.bench='<pattern>' -test.benchtime=<N>x`. `-test.benchtime=Nx` runs more than N iterations in total (the benchmark framework warms up), so difference two iteration counts to get per-iteration numbers. If `perf` is refused (`/proc/sys/kernel/perf_event_paranoid` above 2 and no sudo), use a per-function profile of each binary instead.
- **pprof skid.** A sample lands on the instruction *after* the one that stalled. After a contended atomic or a cache miss, the next line gets the blame. Read `go tool pprof -disasm='<func>'` and find the stalling instruction. Check that the assembler emitted what you meant with `go tool objdump -s '<func>' <binary>`.
- **Allocation alignment.** A struct whose size falls in a larger Go size class lands on more than one cache-line offset. If the base is bimodal (±15%) and the variant is not, with the fast samples grouped per process, a hot field is sharing a line in half of the allocations.
- **Temporary instrumentation.** To count work or histogram inputs, use a throwaway `zz_*_test.go` file, or `go test -overlay` so the tree stays clean. Delete it before you commit.

## Traps

- `git stash push -- <file>` does nothing after the change is committed. Both sides then measure the same code, and the result looks like a clean "no change". Build the base from a revision with `buildbin.sh`.
- `git checkout -- <file>` does not restore a staged file. Use `git checkout HEAD -- <file>`.
- A benchstat input filtered with `grep` must keep its `goos:`, `goarch:`, `pkg:`, and `cpu:` lines, or benchstat splits it into separate tables. Filter with `benchstat -filter` instead, for example `-filter '/enc:ours'`, or `-filter '.unit:sec/op'`.
- `-test.bench` splits its pattern on `/`, one part per sub-benchmark level. An alternation that spans a `/` breaks: `'^BenchmarkA$|^BenchmarkB$/x=1'` means `(^BenchmarkA$|^BenchmarkB$)` at the top and `x=1` below, for both. Keep `/` out of alternations, or run separate patterns.
- Two sub-benchmarks of one run can be compared by renaming one with `sed` and passing the halves as two files.
- Do not pipe a benchmark or a test lane through `tail`: the exit status becomes `tail`'s.
- A benchmark under `cmd/benchmark` without `-short` can time out and rewrite a committed report.

## What not to do

- Don't accept or reject a change on sequential runs; interleave compiled binaries.
- Don't optimize without measuring, and don't optimize a benchmark that does not reach the target code.
- Don't cross a package boundary on suspicion — only on profile evidence, with the reach's approval and threshold met.
- Don't accept a cross-package win measured only on the callee's benchmark; that is relocated cost.
- Don't keep a complex change under the "simpler code" clause; the clause is for code that got simpler.
- Don't rewrite other packages for a neutral-performance readability win; that is a refactor.
- Don't break an API, wire, schema, or config contract to win a benchmark — report it (R3) and stop.
- Don't use `unsafe` unless the user asks for it and the gain is at least three times the threshold.
- Don't write benchmarks on unrealistic inputs (empty slices, single elements).
- Don't use fewer than 6 samples per side.
- Don't skip the tests after an optimization — a faster wrong answer is not an optimization.
