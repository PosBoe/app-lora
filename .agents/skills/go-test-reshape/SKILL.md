---
name: go-test-reshape
description: Reshape a Go package's tests to the right granularity and file layout — consolidate many small ad-hoc tests into datadriven (preferred) or table-driven tests, split overly broad "switchboard" tests into focused behavior-scoped ones, and move tests into the file that matches the code under test. Invoke ONLY when the user explicitly runs `/go-test-reshape` or explicitly names this skill, or when go-test-groom's pipeline explicitly directs it — do not infer it from generic phrasing like "consolidate tests", "convert to datadriven", or "fix test granularity" alone; treat those as ordinary requests instead.
---

# go-test-reshape

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

Reshape a Go package's tests toward the target shape below. Fragmented tests get consolidated into dense case tables; switchboard tests get split into focused ones; misplaced tests move to the file that matches their subject. These are the same operation seen from different starting points, so one survey decides all three at once.

Reshaping moves and re-expresses cases — it never adds or drops them. Coverage and the case set must come out identical. Adding cases is `/go-test-coverage`'s job; fixing assertion targets is `/go-test-decouple`'s.

## The target shape

- **One test per behavior cluster.** Each test function owns one public API or one tight cluster of behaviors. Aim for ~2–5 focused tests per subject — not one test per case (fragmentation), not one test for everything (switchboard).
- **Uniform cases.** Cases inside a test share one assertion shape; a reader can predict what dimension the next case adds. If adding a case would require a new table column or a new handler branch, the case belongs in a different test.
- **The densest shape that fits**, in order of preference:
  1. **Datadriven** — output is text or renderable as text (ASTs, queries, errors, HTTP bodies). Uses `github.com/waneock/app-lora/pkg/datadriven`; cases live in `testdata/<topic>/<file>` and read as `> cmd args\ninput\n----\nexpected`. Authoring idioms: `/go-test-datadriven`; templates: `references/patterns.md`.
  2. **Table-driven** — output is a small Go value (scalar, short slice, simple struct) that reads cleanly inline.
  3. **Table-driven + `snaps.MatchSnapshot`** — only when the output is a large nested value AND datadriven is a bad fit (multiple parallel values per case, e.g. `pkg/ch/query_test.go:TestQueryLocal`).

  The preference order is not absolute: datadriven must also pay its harness tax. A small stable case set with scalar expectations and typed Go inputs, a single-scenario test, or a deliberate prescriptive pin (an exact literal a human must consciously edit — `-rewrite` would silently re-pin it) stays table-driven or standalone even though its output technically "renders as text" — the gate is spelled out in `/go-test-to-datadriven`'s eligibility section.
- **Tests live with their subject.** The tests for code in `foo.go` belong in `foo_test.go`, and all tests for one subject sit together in that file rather than scattered across the package. Datadriven layout mirrors this: one `testdata/<topic>/` directory per test function, one file per behavior, filename names the behavior.

## When to apply

**Consolidate** when:

- Several `TestX_*` functions share the same setup, call the same function, and differ only in input values or expected output.
- Repeated `t.Run(...)` blocks carry copy-pasted scaffolding around a single function under test.
- Cases for one table-driven test are spread across multiple functions instead of one slice.

**Split** when:

- The handler dispatches on more than ~3 commands (datadriven), or the table body branches on input shape (table-driven). Each branch is materially a different test.
- Case names cluster into prefixes (`parse_int_*`, `parse_err_*`, `parse_roundtrip_*`) — each prefix is a test waiting to be promoted.
- The table struct has many optional fields, mostly zero per case, set only by a subset — polymorphism inside a table.
- One subset of cases asserts on `wantErr`, another on structured output, another on a side effect. Three contracts deserve three tests.

**Move / regroup** when:

- A test lives in a file unrelated to its subject (`TestParse` sitting in `format_test.go`).
- Tests for one API are scattered across several `*_test.go` files.
- A grab-bag file (`util_test.go`, `misc_test.go`) has accumulated tests for many unrelated source files.

**Do NOT reshape:**

- A long-but-uniform table. 50 cases sharing one assertion shape are fine — the shape is what matters, not the count.
- Integration tests with materially different fixtures or lifecycle (real DB seed vs. in-memory). Keep these separate.
- Tests whose per-case setup cost dominates (one-time `*testing.M` setup is fine; heavy per-case setup belongs in its own function).
- Property-based / fuzz tests — they have their own shape.
- A single test that already does one thing well.

If a file mixes reshape-able and opt-out tests, keep the opt-outs in the file that matches their subject, with a one-line comment stating why they stay standalone (`// Standalone: requires custom clock setup`).

## Procedure

Four steps, in order. Skip a step only with a stated reason.

### 1. Baseline

Capture coverage and the case inventory before any edits — this is the reference for proving no case was lost. Follow `.agents/skills/go-test-groom/references/verification.md` §1–§4: namespaced scratch paths, and **both build tags** whenever the package has build-tagged files or reads `featflag` (a hosted-only inventory cannot see `//go:build cloud` tests move or vanish).

```
mkdir -p /tmp/gotest-skills/<pkg-slug>
go test -coverprofile=/tmp/gotest-skills/<pkg-slug>/cover.before.default.out ./<pkg>/...
go tool cover -func=/tmp/gotest-skills/<pkg-slug>/cover.before.default.out | tail -n 1
go test -v ./<pkg>/... 2>&1 | grep -E '^--- (PASS|FAIL)' > /tmp/gotest-skills/<pkg-slug>/tests.before.default.txt
```

Do not use the baseline to decide what new cases to add — that's `/go-test-coverage`'s job. The baseline exists solely to catch regressions introduced by the reshape itself.

### 2. Survey and plan

Read every `*_test.go` in the target package. For each test function, note:

- The subject: the API under test and the source file that defines it.
- The setup shape (struct literals, fixtures, mocks).
- The current shape (standalone / table / datadriven) and the file it lives in.

Group by `(subject, setup shape)` and decide per group:

- Fragmented group → consolidate into one test in the subject's file.
- Switchboard test → split by behavior cluster into ~2–5 tests.
- Right-sized but misplaced → move to the subject's file.
- Right-sized and well-placed → leave alone.

Pick exactly one shape per group (preference order above). If a group straddles two shapes (some text-y, some scalar), split the group rather than forcing one shape. Do not force unrelated APIs into one table.

### 3. Reshape

Edit in place, replacing as you go — never leave the originals alongside the new tests.

**Consolidating into datadriven**: one `TestXDatadriven` per group calling `datadriven.Walk` + `datadriven.RunTest` and dispatching on `tc.Name`; the handler returns a `string` rendering the observable output (parsed AST, formatted query, error type + position + message). One file per behavior; group related cases inside a file with `#` comments; split into separate files when behaviors are independent or the file would pass ~200 lines (pattern: `pkg/ast/mqlast/testdata/parse/{selector,errors,roundtrip}`). After translating, materialize goldens with `go test ./<pkg>/... -rewrite` — the flag must come after the package list (`go test -rewrite ./...` silently tests nothing) — and inspect the diff line-by-line: `-rewrite` happily encodes bugs as "expected".

**Consolidating into a table**: declare the table once; field order is identifying `name` first, then inputs, then expectations, then flags (`wantErr`) last; one case per line where it fits. The body is a single `t.Run(tt.name, ...)` with no per-case branching beyond `if tt.wantErr { require.Error(...); return }`. Call `t.Parallel()` at the top of the `t.Run` body when cases share no mutable state — datadriven cases are never parallelizable. A subset needing a setup variant (custom clock, custom fixture) gets its own table or standalone test — never an `if tt.name == "foo"` branch.

**Splitting**: promote each behavior cluster to its own test. Datadriven: one test function per cluster, each walking its own `testdata/<topic>/` directory, so each handler drops back to one or two commands. Table-driven: one `TestX_<behavior>` per cluster with its own table struct scoped to one input/output shape. Don't atomize back to one-case-per-test — that undoes the consolidation win.

**Moving**: relocate tests (plus their helpers and testdata) to the file matching their subject; create `bar_test.go` when `bar.go`'s tests are stuck elsewhere; dissolve grab-bag files entirely when everything in them belongs somewhere else.

**Delete in the same edit**: the original test functions, any helper that existed only to support them (verify with grep before removing), and obsolete `testdata/` directories. No `// moved` / `// removed` breadcrumbs.

### 4. Finalize

```
go test -race -count=1 ./<pkg>/...        # gate 1: race detection, fresh build
go test -v ./<pkg>/... 2>&1 | grep -E '^--- (PASS|FAIL)' > /tmp/gotest-skills/<pkg-slug>/tests.after.default.txt
task test:pkg -- ./<pkg>/...              # gate 2: the package under both build tags
```

`-count=1` forces a fresh test-binary build, which catches dead helpers only the old tests used (regular `go build` does NOT — `*_test.go` is excluded from non-test builds). Escalate to gate 3 (`task test`) only if the reshape moved shared helpers or fixtures out of the package.

Then prove nothing was lost with the two recipes in `.agents/skills/go-test-groom/references/verification.md` §6 — the **coverage diff** (per tag lane baselined) and **case accounting** (leaf-name diff for splits and moves; counts plus the coverage diff for consolidations, where old test names legitimately become case names).

A per-function coverage drop means a case was lost in translation — grep the deleted tests for that function and recreate the missing case in the new shape. Do not add coverage beyond what the originals had; reshaping never changes the case set.

Finally, inspect the diff: originals deleted rather than copied; testdata added/removed to match; no TODO breadcrumbs; case names describe behaviors (`go test -run TestX -v` and skim the names).

## Output to the user

- Per group: what was consolidated / split / moved (test names), the shape chosen, and one sentence why.
- Count deltas: "was N tests across M files; now K tests, J cases".
- Coverage `before% → after%`; name any function that regressed and the case added to recover it.
- Any `-rewrite`d golden worth a human sanity-check.

Stay terse. The diff is the artifact; the summary just routes attention.

## The test-* suite

This skill is step 1 of a pipeline that composes on one package:

1. `/go-test-reshape` — granularity and file layout (this skill).
2. `/go-test-decouple` — retarget assertions from internals onto the public contract.
3. `/go-test-coverage` — add cases for uncovered public behaviors.
4. `/go-test-cleanup` — readability polish: scoped types, inlined helpers, semantic assertions.

`/go-test-datadriven` owns the datadriven authoring idioms the others lean on. `/go-test-to-datadriven` is the directed variant of this skill's consolidation move: the user has already chosen datadriven for named tests, and it executes just that conversion. Siblings sit outside the pipeline: `/go-test-speedup` (time budget), `/go-test-determinism` (flake and cross-test interference), and `/go-test-lockin` (whether the expectations are true). `/go-test-groom` runs the pipeline end-to-end in one prompt; `references/verification.md` under it holds the suite-wide baseline, database-prerequisite and gate protocol.

## Additional resources

- `references/patterns.md` — datadriven / table / snapshot templates in this repo's conventions, a worked consolidation example, and a split example.
- `pkg/datadriven/README.md` — file format, `#` comments, `tc.Fatalf`, `-rewrite` and `-focus` semantics.
- `docs/writing-good-tests-ai-era.md` and CLAUDE.md test rules: assert inputs/outputs not steps; prefer datadriven; otherwise table-driven.
