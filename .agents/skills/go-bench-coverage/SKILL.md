---
name: go-bench-coverage
description: Ensure a Go package has good and sufficient benchmarks for its hot paths — identify hot code from evidence (named production callers, pipeline position, existing perf artifacts), audit existing benchmarks for realism and measurement correctness, and add missing ones in the house style. Owns the package's benchmarks — may consolidate or remove existing ones to keep the count and suite runtime lean, not only add. Measures only — it never optimizes; hand off to /go-optimize-bench for that. Invoke ONLY when the user explicitly runs `/go-bench-coverage` or explicitly names this skill — do not infer it from generic phrasing like "add benchmarks", "benchmark this package", or "check the benchmarks" alone; treat those as ordinary requests instead.
---

# go-bench-coverage

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

Ensure every hot path in a Go package has a benchmark that would actually catch a performance regression. The driving question is not "does a `Benchmark*` function exist" but "if this code got 3x slower or started allocating per item, would a benchmark notice."

A benchmark answers that question only when it runs **the work production runs**: the entry point a real caller calls, with the configuration that caller builds, on input shaped like what that caller passes. Most defects this skill has missed in practice were a benchmark that ran cleanly and measured something else:

- A forecaster benchmark built the model with `SeasonalOn`; production uses `SeasonalAuto`, so `Fit` ran no regression and every fit row measured 240 ns of nothing.
- An autoregressive benchmark fitted three lags and no harmonics, a recipe nothing ships. The harmonic columns production computes for every row never ran.
- A `256KiB` decode case sliced a 254,142-byte fixture past its length into spare capacity, so it read 8 KB of garbage and did not fail.
- A Huffman-table benchmark decoded one 108-symbol table in a loop. Production decodes hundreds of 256-symbol tables, and the branch predictor learned the single one.
- A JSON suite kept Go's upstream `code.json` benchmark (a 1.9 MB document no caller sends) and had no benchmark for `Parse` with `ZeroCopy`, which the CloudWatch ingest path calls per record.

Keep these in mind through every step: the question for each benchmark is **"which production call does this stand for, and would it notice that call regressing?"**

This skill only builds and repairs the measurement surface. It never optimizes code, and it does not edit production code at all — that is `/go-optimize-bench`, which consumes the benchmarks this skill guarantees.

This skill **owns the package's benchmarks**. It may change, consolidate, or remove existing benchmarks — not only add them. More benchmarks is not the goal; one benchmark per hot path that would catch a regression is. Redundant, orphaned, or low-value benchmarks cost suite runtime, maintenance, and noise, so prune them. Say what you removed and why; prefer repair or consolidation, but delete a benchmark that earns its keep for nothing.

## How many benchmarks, and how long

- **Count**: roughly **one benchmark (or one sweep) per (hot path, cost axis)**. A package with 3–4 genuine hot paths wants ~3–8 benchmark functions, not dozens. If the count climbs past that, look for redundancy, orphans, or paths that are not hot before adding more.
- **Sweep width**: 2–4 points on a cost axis, spanning realistic magnitudes (e.g. `10 / 100 / 1000`). Sweep only axes that change the code path or its cost per item.
- **Per-benchmark duration**: keep each sub-benchmark in the microsecond-to-low-millisecond range per iteration. The largest sweep point should finish an iteration well under ~10 ms unless the hot path is inherently coarse-grained (per-batch, per-flush). A benchmark compares relative cost; it does not need production volume.
- **Whole-suite runtime**: `go test -bench=. -run=^$` on the package should finish in seconds to low tens of seconds, not minutes. Treat that as a budget you act on, not one you only report: when the suite is over ~20 s after your changes, list its costliest sub-benchmarks and cut a sweep point, shrink an input, or gate a genuinely expensive case behind `testing.Short()`. Report whatever stays over budget with the reason it stays.

## When to apply

Apply when a package sits on a hot path (ingestion, query compilation, parsing, serialization, compression, per-point scoring) and its benchmarks are missing, stale, or of unknown quality.

Do NOT apply when:

- The package is cold end-to-end (fx wiring, config loading, CLI commands, migrations, admin handlers). Report "no hot paths" and stop.
- The user wants code made faster. That is `/go-optimize-bench`; run this skill first only if its benchmarks are missing.
- The hot work is a database query's *execution*. Benchmark the Go-side work (query building, row decoding), not ClickHouse/Postgres round-trips.

## Procedure

### 1. Map the hot paths from their callers

A path is hot when it runs **per item** (span, log record, datapoint, row, token, block) or **per request** on a high-frequency endpoint. Build the map from callers, because the callers are also where the realistic input and configuration come from.

1. **Enumerate every production call site** of the package's exported API — completely, not a `head`-truncated sample. A missed caller is a missed hot path.

   ```bash
   go list -f '{{.ImportPath}} {{join .Imports " "}}' ./... | grep ' github.com/waneock/app-lora/<pkg>\b' | cut -d' ' -f1
   grep -rn '<pkgname>\.' --include=*.go <each importer dir> | grep -v _test.go
   ```

   Group the call sites by entry point (`Parse` with which flags, `Marshal` vs `Encoder`, which constructor with which params). Different flags or options that select different code are different hot paths.

2. **Classify each call site** by pipeline position: called from a consumer loop, an ingestion stage, query compilation, OTLP conversion, or an API response writer → hot. Called once from `module.go`, config load, or a CLI command → cold. Check `CONTEXT-MAP.md` and the package README when the position is unclear.
3. **Use perf artifacts as supporting evidence** — `sync.Pool`, scratch buffers, computed preallocation, existing profiles or `profile_test.go` harnesses. Code shape (loops over unbounded input, recursive walks, byte-level parsing) points where to look inside a hot entry point.

For each hot path, record **the call site that makes it hot** (`cloudwatch/metrics.go: Parse(..., ZeroCopy)` per record), **the configuration that caller uses** (flags, params, options — cite the constant or literal, e.g. `forecastHarmonics` in `alerting/detection/lineup_forecast.go`), and **the cost axes** (the input dimensions cost scales with: attr count, record count, message size, table count), with the largest magnitude production reaches on each — super-linear cost shows only near the top. A benchmark in the package, or a fixture copied from an upstream project, is not evidence that a path is hot — only a caller in this repository is.

Two cases need care:

- **Offline consumers.** When the package's only callers are benchmark or evaluation lanes (`cmd/benchmark`), the lane is the production caller: take the configuration from the candidate or recipe the lane runs, and cite it.
- **Build variants.** When the input shape or configuration differs by build (a cloud row model is wider than the hosted one), benchmark the costlier variant, or both when the difference changes the code path. Skip a variant only after you list what it adds (columns, codecs, value kinds) and name the benchmark that already covers each one.

A shared kernel (integer formatting, string escaping) may get its own row; name the entry-point rows above it that make it hot instead of a separate call site.

Note the build tags the package needs (`-tags=invariants`, `invariants`); they apply to every `go test` below.

### 2. Inventory existing benchmarks

```bash
grep -rn "func Benchmark" <pkg>/*_test.go
```

Map each benchmark to the hot path it stands for. Three buckets result:

- **Covered** — a benchmark exercises the path. It still goes through the audit in step 3; "covered" is provisional until the benchmark is shown to run the production work.
- **Uncovered** — no benchmark reaches the path. Goes to step 4.
- **Orphaned** — no production caller behind it: a cold path, a function only tests call, a standard-library function timed on its own, or an upstream workload no caller here sends. Remove it unless it guards a real, if secondary, cost.
- **Evidence** — a benchmark that a `DESIGN.md` goal, a README claim, or a code comment cites as its measurement. A comparison against an alternative library (pdata, `encoding/json`) belongs here when a document states the claim it backs. Keep it: removing it silently voids that claim. When its input is unrealistic, add the realistic case next to the cited one rather than rewriting the cited fixture, so the documented numbers can still be re-measured. If you must change what a cited benchmark measures, name the document and the figures that are now stale in the report. When you rename it, update the citation.

Before you remove or rename any benchmark, search the repository for its name:

```bash
grep -rn '<BenchmarkName>' --include=*.md --include=*.go . | grep -v '_test.go:.*func Benchmark'
```

Flag **redundant** benchmarks too — two that exercise the same hot path at the same point on the same axis. Fold them into one sweep.

### 3. Audit each benchmark against production

A bad benchmark is worse than none: it reports green while the hot path regresses. For each covered path, compare the benchmark with the call site from step 1 and fix in place what differs.

**Does it run the production work?**

- **Entry point.** It calls what the caller calls, with the same flags. `Unmarshal` does not stand for `Parse(..., ZeroCopy)`; an `Encoder` configured like `pkg/httpkit` is not the same path as `Marshal`. Benchmark an unexported kernel directly only when it is the identified hot spot and the public entry buries it in noise (precedent: `BenchmarkScanSegmentEnd` in `pkg/logparser/loglexer`).
- **Configuration.** Params, options, and recipes match what production constructs — reference the production constant where the package boundary allows it, or copy it with a comment that names the source. A configuration that skips the expensive branch makes every row measure the cheap one.
- **Input shape.** The fixture looks like what the caller passes: a real CloudWatch record, an OTLP request with a representative attr count, an API row with the fields responses carry. Match the production type's *structure*, not only its size: embedded structs, pointer fields, optional and variant fields, and every kind of value a field can hold in production select different code (a reflect walk through an embedded struct is not the walk over a flat one). Use the production type itself when the import graph allows; otherwise copy it with a comment that names the source type. Reuse realistic unit-test fixtures. Empty slices, single elements, and `"test"` strings are edge cases, not the representative case.
- **Add, don't rewrite.** When the input is incomplete rather than wrong — it lacks a value kind, a row shape, a record type production also sends — add it as its own arm and leave the existing arms' input alone. Each arm then varies one thing, so a cost axis stays readable and earlier numbers stay comparable. Rewrite an existing arm's input only when it measures the wrong thing (a path production never takes, a size its name does not match), and say in the report that its old numbers no longer compare.
- **Input diversity.** When the work branches on data (per-symbol weights, per-token kind, escape density), one small input repeated every iteration lets the branch predictor and caches memorize it. Rotate through a set of inputs harvested from a realistic source, as production sees many.
- **It reaches the hot code.** Prove it instead of assuming it. Plausibility first — a "fit" at 240 ns or a "parse a full document" at 2 ns is not doing its work. When in doubt, run the benchmark once under a coverage profile and confirm the hot function's lines are hit:

  ```bash
  go test -run='^$' -bench='^BenchmarkX$' -benchtime=1x -coverprofile=/tmp/x.cov <tags> <pkg>
  go tool cover -func=/tmp/x.cov | grep -w <hotFunc>   # expect well above 0.0%
  ```

  Add `-coverpkg=<dep>` when the hot code lives in a package the benchmark reaches through a dependency.

**Is the measurement correct?**

- **The name matches the input.** A case named `256KiB` or `records=100` processes exactly that. Build sized input with a helper that returns exactly `n` bytes (tiling a fixture if needed) or fails; never slice a fixture past its length, since spare capacity hides the overrun. Set `b.SetBytes` from the actual input length.
- **Errors are checked.** Check every error inside the loop with `b.Fatal`; `_ =` hides a benchmark that fails on the first call and then times the error path. Before the loop, check once that the call produces the output the benchmark claims (a decoded record, a non-empty buffer, no unread tail, a season the fit kept). A fixture tweak can silently move the call onto a cheaper path; this check fails the benchmark instead. Make the check through the public result where possible — do not move a benchmark into the internal test package only to read unexported state.
- **Setup stays outside the loop.** Fixture building goes before `for b.Loop()`. `b.Loop()` excludes pre-loop setup and keeps results alive, so there is no need for `b.ResetTimer()` or `sink` variables.
- **Constant per-iteration work.** State that grows across iterations (appending to a shared slice, a map never reset) makes late iterations measure different work. Reset per iteration. For a reused output buffer, warm it once before the loop (`buf, err := Append(nil, v)`), then write into `buf[:0]`, so the loop measures the steady state and not the first growth.
- **In-place mutation.** When the call changes its input (renames keys, sorts, consumes a buffer), the second iteration sees different input than the first — and an undo step inside the loop gets timed with the call. Build a pool of fresh inputs before the loop and refill it with the timer stopped once every N iterations (e.g. 256); a `b.StopTimer`/`b.StartTimer` pair on every iteration costs more than sub-microsecond work. Check before the loop that one call does the work the name claims (the keys were renamed, the records were consumed).
- **Scaffolding does not dominate.** Input building or output assertions inside the loop hide the hot call. Measure the hot call as directly as the caller's entry point allows.
- **Allocations and throughput visible.** Every benchmark calls `b.ReportAllocs()`. Byte-oriented work (parse, encode, compress, scan) also calls `b.SetBytes`, so MB/s compares across sweep points.
- **Determinism.** Fixed seeds, pinned inputs, no wall clock, no map-order-dependent work.
- **Placement and intent.** Benchmarks live in `<topic>_bench_test.go` (or `bench_test.go` in a small package) — move one found in a unit-test file. A doc comment states what one iteration measures, the cost axis, and the production caller it stands for, so a regression report reads without the implementation.

### 4. Add missing benchmarks

For each uncovered hot path, write a benchmark that passes the step 3 audit by construction:

```go
// BenchmarkParseZeroCopy measures parsing concatenated CloudWatch metric
// records, as cloudwatch/metrics.go does per request.
// Cost axis: record count.
func BenchmarkParseZeroCopy(b *testing.B) {
	for _, n := range []int{1, 100} {
		b.Run(fmt.Sprintf("records=%d", n), func(b *testing.B) {
			data := []byte(strings.Repeat(cloudwatchRecord, n))
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for b.Loop() {
				tail := data
				for range n {
					var out datapoint
					var err error
					if tail, err = Parse(tail, &out, ZeroCopy); err != nil {
						b.Fatal(err)
					}
				}
				if len(tail) != 0 {
					b.Fatal("unread records")
				}
			}
		})
	}
}
```

- **Sub-benchmark names** use `key=value` (`records=100`, `attrs=256`, `size=16KiB`). benchstat reads `key=value` path segments as configuration, which makes `-row`/`-col` projections work across the sweep.
- **Fixtures**: reuse the package's realistic fixtures and generators. Extract a helper when several benchmarks share nontrivial fixture building; a family of packages can share a harness package (precedent: `anomaly/detectors/dettest`, `pkg/zbench`). Do not start ClickHouse/Postgres/Redis.
- **Large external corpora** (e.g. the Silesia files) are fine as an extra case; skip the case with `b.Skip` when the files are absent, and keep a self-contained case that always runs.
- **Steady state vs cold**: when production amortizes construction (a long-lived tokenizer, a pooled encoder), construct once and measure per-item work. Add a construction benchmark only if construction itself runs per batch.
- **Concurrency**: add a `b.RunParallel` variant only when the hot path runs under contention in production (a shared pool, a mutex-guarded cache).

### 5. Verify

```bash
go test -bench=. -count=1 -run=^$ <tags> <pkg>
```

- **No failures**, and nothing under ~1 ns/op (the work was optimized away).
- **Plausibility**: the numbers fit the work each name claims. For any benchmark whose number surprises you, run the coverage check from step 3.
- **Scaling**: divide each sweep point's time by its item count. When the per-item cost grows by more than ~1.5x across the sweep of a linear axis, report it as super-linear with the per-item numbers; do not explain it away as noise or cache effects without a measurement. It is a finding for `/go-optimize-bench` — report it, do not fix it.
- **Suite runtime** within the budget above; time the whole run.
- **Regular tests still pass**: `go test <tags> <pkg>` (benchmark files can break compilation, or trip `goleak` in packages with `main_test.go`).

Do not capture a statistical baseline (`-count=10`, benchstat) here; that is the first step of `/go-optimize-bench`. When a repaired benchmark now measures different work, its old numbers no longer compare — say so in the report and in the commit message.

## Output to the user

- **Hot-path map**: a table of hot path → production call site (`path/to/file.go` plus the calling function — a function name alone, or a line number that drifts, is not enough) → configuration → cost axes → status (`covered` / `added` / `repaired` / `skipped: <reason>`). Include paths judged cold that the user might expect to be hot, with the reasoning.
- **Removals and consolidations**: each benchmark deleted or merged and why, plus the net change in count and suite runtime.
- **Quality fixes**: each defect found in an existing benchmark and what changed. Flag the ones that invalidate earlier numbers.
- **Single-run numbers** for added and repaired benchmarks, labeled informational (count=1). Take every number and count from one final run of the final code; a figure from an intermediate version misleads.
- **Findings for `/go-optimize-bench`**: super-linear scaling, surprising allocs/op — listed, not acted on.
- **Production code left alone**: for example, a function that became unreferenced when its only benchmark went away. Report it; do not delete it here.

Stay terse. The benchmark files are the artifact; the summary routes attention.

## Relationship to go-optimize-bench

`/go-optimize-bench` writes benchmarks ad hoc when its target lacks them; this skill covers the whole package's hot surface and audits existing benchmarks rather than trusting them. Typical sequence: `/go-bench-coverage` once to establish the surface, then `/go-optimize-bench` any number of times against it.

## What NOT to do

- Don't optimize or otherwise edit production code, even an obvious win or dead function — report it.
- Don't benchmark cold paths, standard-library functions, or upstream workloads to fill the table.
- Don't use a configuration or entry point you cannot trace to a production caller.
- Don't let a case's name promise input it does not process.
- Don't discard errors in the loop, and don't add `sink` variables or `b.ResetTimer()`.
- Don't inflate the suite: no duplicate coverage, no sweep point that does not change the cost, no input so large the suite runs for minutes.
- Don't delete an existing benchmark without saying so, and don't delete one a document cites as evidence.
- Don't spin up the DB testbed; benchmark the Go-side work only.
