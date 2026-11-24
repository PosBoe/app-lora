---
name: go-test-cleanup
description: Clean up Go tests for clarity, self-containment, and human-readable assertions. Makes types test-scoped where possible, simplifies test logic, and rewrites assertions so a reader can understand what's being tested without jumping between files or decoding opaque comparisons. Invoke ONLY when the user explicitly runs `/go-test-cleanup` or explicitly names this skill — do not infer it from generic phrasing like "clean up tests", "simplify tests", "make tests readable", or "polish tests" alone; treat those as ordinary requests instead.
---

# go-test-cleanup

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

Polish the Go tests of one package so each test reads top-to-bottom as arrange, act, assert, without a jump to another file or a decode of an opaque comparison. Tests are read more often than the code they test — by the human who reviews agent output and by the next agent that learns the contract from them — so a readable test pays off on every later change.

## The one invariant

**A cleanup never changes what a test verifies.** Every edit is a transcription: the same inputs reach the code under test, the same values are checked, and the test keeps its identity. If an edit would add, drop, loosen, or retarget a check, it is not a cleanup — stop and list it for the owning skill instead.

"What a test verifies" is wider than its assertions:

- **The checked values.** A whole-struct `Equal` checks every field, including the ones that are zero in `want`. Splitting it into two field checks drops the rest.
- **The inputs that make a case discriminating.** A field the assertions never mention can still be what the case is about — the row a filter must exclude, the non-zero value that proves a field is copied.
- **The test's name.** In this repo a name is data. `pkg/testbed/runenv.go` derives the fake seed of a testbed run from `t.Name()`, go-snaps keys a snapshot by it, and Taskfiles, docs, and `-run` patterns cite it. §6 has the procedure.
- **The failure behavior.** `assert` keeps going after a failure and `require` stops. Keep the package's choice.

Ownership of the neighboring work:

| Work | Skill |
|---|---|
| What an assertion targets (contract vs. internals), splitting a whole-struct compare because its incidental fields are not the contract | `/go-test-decouple` |
| Test granularity, table/datadriven shape, which file a test lives in | `/go-test-reshape` |
| New cases | `/go-test-coverage` |
| `testdata/` corpora and datadriven directives | `/go-test-datadriven` |
| Whether an expectation is correct at all | `/go-test-lockin` |

## When not to apply

- The test already reads cleanly. Rephrasing a clear name or reordering clear assertions is churn: it costs a reviewer attention and buys nothing. A package whose tests are already clean gets a short report and a small diff, or none.
- The problem is structure — too many behaviors in one test, the wrong shape, the wrong file. Use `/go-test-reshape`.

## Cleanup catalog

### 1. Scope types to the narrowest context

A package-level type in a test file that one test uses moves into that test's body, with a short generic name (`Model`, `Input`, `Case`) — the function scope supplies the context.

```go
func TestRoundTrip(t *testing.T) {
    type Model struct {
        Name string `msgp:"name"`
        Age  int    `msgp:"age"`
    }

    m := Model{Name: "alice", Age: 30}
    // ...
}
```

- **Shared by several tests.** If the type is small (about five fields or fewer, no nesting), declare a copy inside each test that uses it; a short literal is cheap to repeat and keeps each test standalone. Keep a larger or nested type at file level.
- **Keep at package level** a type that needs methods (a function-scoped type cannot declare them), a type a helper outside the test must name, and a type used as a type-parameter constraint. If the only method is a one-line interface implementation, consider whether a function-typed field or a local fake reads better.
- **Check the reach first.** Grep the whole package, `_test` package included, for the type name before you move it.

### 2. Inline trivial helpers

Inline a helper of one or two lines that one test calls, and delete it. With two callers, inline when the body is trivial or the callers differ only by a literal. Keep a helper that three or more tests call, or whose body has several steps, error handling, or cleanup. A kept helper that calls `require.*`, `t.Fatal`, or `t.Error` starts with `t.Helper()`, so a failure points at the caller.

### 3. Remove dead setup

A field or variable is dead only when **the code under test never reads it on this path**. Prove it by reading that path, not by deleting the field and watching the test pass: a green run proves only that the assertions still hold, not that the case still means what it meant.

Keep:

- a field the code under test reads, even if no assertion names it;
- a value that makes the case discriminating — the excluded row, the second tenant, the non-default that proves a copy;
- a field whose zero value the code treats differently (a zero `Timeout` that means "no timeout").

```go
// Before — AgeCategory reads only Age
u := User{Name: "alice", Age: 30, Tags: []string{"admin"}, CreatedAt: time.Now()}
require.Equal(t, "adult", u.AgeCategory())

// After
u := User{Age: 30}
require.Equal(t, "adult", u.AgeCategory())
```

### 4. Prefer composite literals

Collapse a struct built by successive field assignments into one literal. Keep the assignments when a later value depends on an earlier one, or when the test is about mutation.

### 5. Make assertions read as the contract

**Move `want` to the assertion; do not split it.** An `expected` declared 40 lines above its assertion forces a scroll. Move the declaration to just before the assertion and keep the whole-value compare:

```go
// Before
expected := Model{Name: "alice", Age: 30}
// ... 20 lines of arrange and act ...
require.Equal(t, expected, got)

// After
// ... arrange and act ...
want := Model{Name: "alice", Age: 30}
require.Equal(t, want, got)
```

Split into per-field checks only when the fields you list are **every** field of the type — then the checks are identical to the whole compare. Otherwise the split stops checking the unlisted fields; if those fields look incidental, list the test for `/go-test-decouple`.

**Semantic replacements.** Each row checks the same thing, and testify prints a better failure:

| Before | After |
|---|---|
| `require.Equal(t, nil, err)` / `if err != nil { t.Fatal(err) }` | `require.NoError(t, err)` |
| `require.Equal(t, true, ok)` | `require.True(t, ok)` |
| `require.Equal(t, 3, len(s))` / `if len(s) != 3 { t.Fatalf(...) }` | `require.Len(t, s, 3)` |
| `require.Equal(t, 0, len(s))`, `require.Equal(t, "", s)` | `require.Empty(t, s)` |
| `require.True(t, len(s) > 0)` | `require.NotEmpty(t, s)` |
| `require.NotEqual(t, nil, x)` | `require.NotNil(t, x)` |
| `require.Error(t, err)` then `require.ErrorContains(t, err, "x")` | `require.ErrorContains(t, err, "x")` — it already fails on a nil error |
| `require.Error(t, err)` then `require.ErrorIs(t, err, ErrX)` | `require.ErrorIs(t, err, ErrX)` |
| `require.Equal(t, got, 42)` | `require.Equal(t, 42, got)` — testify reads the first argument as expected, so swapped arguments print a failure that lies |

Two rules keep a replacement a transcription:

- `require.NotNil` is stricter than `NotEqual(nil, x)`: it also rejects a typed nil pointer inside an interface. If the gate then fails, the test was passing on a typed nil — report it, do not revert silently.
- `ElementsMatch` in place of sort-then-`Equal` is a transcription only when order is not part of the contract. When the code under test promises an order, the sorted compare is the weaker check; list it for `/go-test-decouple`.

**Failure messages.** Delete a message that restates the assertion (`"expected no error"`). Keep one that says which contract broke, or which iteration of a loop failed.

### 6. Name tests for behavior — and rename safely

A name should say what the test proves: `TestParse_rejectsTrailingComma`, not `TestParse2`; the case name `"negative duration returns error"`, not `"case1"`. Follow the naming style the package already uses. Leave a name that already describes behavior, even if you would phrase it differently.

Before you rename a test or a subtest:

1. **Grep the whole repository** for the old name — Taskfiles, CI workflows, docs, `AGENTS.md`, comments, other tests (`TestSubtestsReportOnTheirOwnT`-style citations). Update every citation in the same change.
2. **Testbed seed.** If the test boots a bed (`run(t)`, `Run(t)`, anything that reaches `testbed`), the rename changes the fake seed and so the seeded data. The determinism rules say no test asserts on a literal fake value, but a test that breaks that rule now fails, or passes only by luck. Run the renamed test several times (`-count=5`) and report the rename.
3. **Snapshots.** A go-snaps test keys its snapshot by `t.Name()`. After a rename the old entry is orphaned and the new one re-records green — a silent loss of the check. Do not rename a snapshot test unless you also move its snapshot entry by hand, then diff the new entry against the committed one.
4. **Datadriven.** `pkg/datadriven` names each subtest after its `testdata/` file, so those subtest names change only with a file rename — which is `/go-test-datadriven` work, not this skill's.

### 7. Transcribe raw error checks

Raw `if err == nil { t.Fatal(...) }` plus `strings.Contains(err.Error(), ...)` becomes `require.ErrorContains(t, err, "timeout")`. That is a transcription and it checks the same thing. Whether the check should be a substring at all is `/go-test-decouple`'s call; do not guess at an `errors.AsType` target here. When a typed check already exists, use `errors.AsType`, not `errors.As` (root rule).

### 8. Remove stutter and restating comments

Rename a variable that repeats its type (`configConfig` → `cfg`). Delete a comment that restates the next line. Keep a comment that says **why** — why this value, why this order, which bug the case pins. A comment you write or keep follows the repo's ASD-STE100 rule: short sentences, active voice, simple tenses.

### Not cleanups

These look like polish but change behavior. Do not make them here:

- `context.Background()` → `t.Context()`: the test context is canceled before `t.Cleanup` functions run, which changes what the cleanup code sees.
- `assert` ↔ `require`: changes whether later checks run after a failure.
- Adding `t.Parallel()`: the shared databases and process-global state (fake clock, `featflag`) make it a determinism change — `/go-test-determinism`.
- Converting to table or datadriven form: `/go-test-reshape`, `/go-test-to-datadriven`.

## Procedure

`.agents/skills/go-test-groom/references/verification.md` holds the suite's baseline and gate protocol; this procedure cites it rather than restating it. Drop `-race` from its commands: the root `CLAUDE.md` runs the race lane only on request, and a cleanup cannot introduce a race that the baseline lacked.

### 1. Baseline

Follow verification.md §2: decide the lanes (default, and invariants when the package has build-tagged files or reads `featflag`), then write the case inventory and coverage profile for each lane under `/tmp/gotest-skills/<pkg-slug>/` with phase `before`. For an integration package, check §4 first — the databases must be up.

### 2. Clean up

Read every `*_test.go` in the package, and read the code under test on the paths the tests exercise — §3 and §5 need it. Then work one file at a time:

1. Package-level test types: check their reach, then scope them (§1).
2. Each test, top to bottom: dead setup, trivial helpers, literals (§2–4).
3. Each assertion: move `want` down, replace with the semantic form, fix argument order (§5, §7).
4. Names: rename only where the name describes mechanics, and follow the §6 steps for each rename.
5. Stutter and comments (§8).

Edit `*_test.go` files in place. Do not create files, move tests between files, touch `testdata/` or snapshot files, or edit production code. When you find something that needs one of those, list it in the report.

Compile-check after each file with `go vet ./<pkg>/...` (and `-tags=invariants` when the invariants lane applies); it compiles the test files, which `go build` does not.

### 3. Finalize

1. Write the `after` inventory and coverage for each lane, per verification.md §2.
2. **Coverage must be identical.** A cleanup adds and removes no cases. `diff` the `go tool cover -func` output of each lane against its baseline. Any change means an edit changed what runs — find it.
3. **Case accounting.** `diff` the sorted before/after inventories. The only differences allowed are your renames, one old name to one new name. A missing or extra case is a bug.
4. `git status --short` shows only `*_test.go` changes — no new, moved, or deleted snapshot or `testdata/` file.
5. Gate 2: `task test:pkg -- ./<pkg>/...`.

If a test fails, the cleanup changed something. The usual causes are a removed field the code read, a scoped type a sibling file used, and a stricter semantic assertion (§5). Find the edit, then fix or revert it and name it in the report.

## Output to the user

Stay terse — the diff is the artifact, and the report routes attention to it:

- **Types scoped**: each type moved, and where.
- **Helpers inlined**: each helper removed or kept on purpose.
- **Assertions rewritten**: grouped by kind, with the counts from your diff.
- **Tests renamed**: `old → new`, one line each, with the citations updated and the seed/snapshot status.
- **Dead setup removed**: what, and the code path that shows it is not read.
- **Left for other skills**: each edit you did not make because it would change what a test verifies — a whole-struct split, an order-insensitive compare, a substring check that wants a type — with the owning skill.
- **Proof**: lanes run, coverage diff result, case-accounting result, gate 2 result.

## The go-test-* suite

This skill is step 4 of the pipeline that `/go-test-groom` runs on one package: `/go-test-reshape` → `/go-test-decouple` → `/go-test-coverage` → `/go-test-cleanup`. `/go-test-datadriven` owns the datadriven idioms. `/go-test-speedup`, `/go-test-determinism`, and `/go-test-lockin` sit outside the pipeline. `references/examples.md` holds worked rewrites.
