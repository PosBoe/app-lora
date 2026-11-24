---
name: go-test-to-datadriven
description: Convert existing Go tests — standalone, table-driven, or snapshot-based — into datadriven tests in this repo's house style, translating each case into a testdata directive whose assertions become rendered golden output, preserving the exact case set and coverage. Invoke ONLY when the user explicitly runs `/go-test-to-datadriven` or explicitly names this skill — do not infer it from generic phrasing like "convert this test to datadriven", "make these tests datadriven", or "move these tests to testdata files" alone; treat those as ordinary requests instead.
---

# go-test-to-datadriven

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
- Only the **pure** tier of `/go-test-datadriven` exists here (no `pkg/ddtest`).

Convert the tests the user names — a test function, a file, or a package's tests — into datadriven shape (`pkg/datadriven` + `pkg/ddtest`). Each existing case becomes a directive case in a `testdata/<topic>/<behavior>` file; its assertions are re-expressed as rendered golden output. Conversion is a faithful translation: the case set and coverage must come out identical. Adding cases is `/go-test-coverage`'s job; fixing what assertions target is `/go-test-decouple`'s.

Relationship to siblings: `/go-test-reshape` surveys a whole package and *decides* the shape per test group — it may pick table-driven, split, move, or leave alone. Here the shape decision is already made by the user: datadriven. This skill executes that conversion; its only judgment call is flagging tests where datadriven is genuinely a bad fit rather than forcing them. `/go-test-datadriven` owns the authoring idioms (harness tiers, rendering rules, testdata layout) — follow it for every idiom question; this skill only adds the translation procedure.

## Read first

- `.agents/skills/go-test-datadriven/SKILL.md` — harness tiers, Go-side idioms, testdata idioms. Everything you author must satisfy it.
- `pkg/datadriven/README.md` — file format, args, `---- BEGIN/END`, `-rewrite` / `-focus`.
- The target package's `README.md` / module `AGENTS.md` — some packages constrain goldens (e.g. `tracing` querygen goldens need a live ClickHouse to `-rewrite`).

## Eligibility

Convert a test when its observable output is text or renderable as text: parsed ASTs, formatted/normalized strings, error messages and typed error fields, HTTP responses, query results, small structs that YAML-render cleanly.

"Renderable as text" is necessary, not sufficient — any bool renders as text. Datadriven earns its keep through golden output: `-rewrite` amortizing expectation churn across many cases, multi-line/structural results reading better as golden blocks than as Go literals, and a corpus that grows without Go edits. A conversion that captures none of those pays the flat harness tax (`Walk` + `RunTest` + arg `Scan` + rendering) and the indirection of a second file for nothing.

Do NOT convert — leave in place and report with a one-line reason:

- **Small stable tables with scalar expectations.** When the expected values are scalars or short strings (`require.Equal` on a number, bool, enum, or error/no-error), inputs are typed Go values that testdata would degrade into a stringly `Scan` layer, setup is trivial, and the case set is small and not expected to grow (roughly: fits on one screen), the table is already the ideal shape. Rule of thumb: if the handler + parsing + rendering would be about as long as the table it replaces, the conversion is net-negative code with worse type safety.
- **Single-scenario tests** — one case pays the full harness tax for one golden block; never convert, regardless of output shape.
- **Deliberate prescriptive pins** — a test that spells out an exact literal *so that a human must consciously edit it* (wire formats, spec constants, curated orderings), usually with a comment saying why. Golden output inverts the mechanism: `-rewrite` silently re-pins exactly the change the test exists to make loud. Relatedly, cases built from symbolic references (`coreAttrs[0]`) assert a *property* over whatever the data is; hardcoding those values into testdata replaces the property with a snapshot of today's data.

- **Multiple parallel Go values per case** where rendering would be tortured rather than clarifying (the `pkg/ch/query_test.go:TestQueryLocal` situation) — table + snapshot is the sanctioned shape there.
- **Property-based / fuzz / benchmark** functions — they have their own shape.
- **Per-case setup that can't become directives**: custom clocks per case, goroutine orchestration, lifecycle hooks that `seed`/`insert`/`load_*`/`adjust_clock` can't express.
- **Tests that materially depend on `t.Parallel()`** for the package's time budget, when the converted suite would blow it — datadriven cases run in order. Flag `/go-test-speedup` instead of skipping silently if conversion is otherwise clearly right.

When the user names a scope where only some tests are eligible, convert the eligible ones and list the skips — never force an awkward conversion to make the scope "complete", and never quietly drop a test.

## Procedure

### 1. Baseline

Capture coverage and the case inventory before any edits — the reference for proving no case was lost. Follow `.agents/skills/go-test-groom/references/verification.md` §1–§4, including the `-tags=invariants` lane when the package has build-tagged files or reads `featflag`:

```
mkdir -p /tmp/gotest-skills/<pkg-slug>
go test -coverprofile=/tmp/gotest-skills/<pkg-slug>/cover.before.default.out ./<pkg>/...
go test -v ./<pkg>/... 2>&1 | grep -E '^--- (PASS|FAIL)' > /tmp/gotest-skills/<pkg-slug>/tests.before.default.txt
```

### 2. Plan the conversion units

Group the target tests by subject (the API under test). Per group decide:

- **The test function(s)**: one `TestXDatadriven` per behavior cluster, ~1–2 commands per handler. Several fragmented `TestX_*` originals usually collapse into one; one broad original may split into two. Don't build a `switch tc.Name` switchboard.
- **The harness tier**: pure handler for pure functions; testbed runner when the original needed the DB/router. Follow `/go-test-datadriven`'s tier rules — do not boot the testbed for a pure function, do not hand-mock half the app when the testbed exists. The tier serves the *whole unit*, sized to its heaviest case: originals that drive the same method with the same output but differ only in constructed dependencies (a nil vs real bridge, stubbed vs live storage) collapse into **one** datadriven test at the heavier tier — the dependency difference is invisible in the directives, so it must not shape the test boundaries or the testdata layout.
- **The testdata layout**: `testdata/<topic>/<behavior>`, one directory per test function, one file per behavior. Original test names and `t.Run` names become file names or `#` comments — nothing is lost, it just moves into the corpus.

### 3. Translate

Translate case by case using the assertion map below (worked examples in `references/conversions.md`):

| Original | Datadriven equivalent |
|---|---|
| `require.Equal(t, want, got)` | Render `got`; the golden pins the value |
| `require.NoError` + result checks | Render the result; success is implied by the rendered output |
| `wantErr bool` / `require.Error` | The error is *output*: render its message (and typed fields) into the golden |
| `require.ErrorIs` / `errors.AsType` checks | Render type + position + message as separate lines so the golden pins the taxonomy, not just the string |
| `require.Contains(err.Error(), …)` | Render the full message — the golden pins the whole string, strictly stronger |
| `require.IsType` | Render the full structure — strictly stronger |
| Partial field checks / `require.Len` | Render a purpose-built YAML struct holding exactly the contract fields the original checked plus the rest of the value if small; don't reproduce "ignore most fields" as a lossy golden |
| `snaps.MatchSnapshot(...)` | YAML-render the same content in the golden; delete the `.snap` entries in the same edit. A snapshot that bundled several values per case (a map like `{in, out, parts}`) is not automatically one golden — values answering different questions become separate tests with their own corpora, not sections of one output |
| Round-trip / idempotence / fixed-point checks | Stay in Go inside the handler with `tc.Fatalf`, documented in the test's doc comment — the golden pins the value, Go pins the invariant |
| `t.Run` / table `name` field | `#` comment above the case (or the file name, if it names a behavior cluster) |
| Table input fields | The *primary textual input* — the string the code under test parses or transforms (query text, an expression, a payload) → the case input block; **every other scalar** (identifiers, keys, options, modes) → named args (`> describe metric=m attr=a`) with defaults declared before `Scan`. Never split sibling scalars between the input block and args (`> describe metric=m` with the attr as input is wrong); a case whose fields are all scalars has no input block |
| Go-side fixture setup (testbed tier) | `> seed` / `> insert` / `> load_*` / `> adjust_clock` directives, with `{entity:key.field}` interpolation — the whole scenario must read from the file |
| `mock.On(...)` expectations | A step assertion, not an output — don't encode mock call logs into goldens. Convert the observable output; if the test *only* asserted mock calls, skip it and point at `/go-test-decouple` |

Author the directive cases with empty expectations (directive, input, `----`, blank line) — do not hand-write goldens.

### 4. Materialize and review

```
go test ./<pkg> -rewrite        # flag AFTER the package list; `go test -rewrite ./...` silently tests nothing
```

Review every golden against the assertion it replaced — `-rewrite` happily encodes bugs as "expected". Each golden must pin *at least* what the original asserted: if `TestParseError` required a specific message, the golden must contain that message. A golden that surprises you means the code, the case, or your renderer is wrong — never accept it to move on. Iterate on one case with `-focus <path>:<line>`.

Also review the goldens *across* cases: when they repeat a large case-invariant block and differ in one fact (an original `require.Contains` becomes full SQL where only the table varies), don't ship the repetition — golden the varying fact and move the invariant remainder into a handler shape check (`tc.Fatalf` on an exact template or anchored regexp), then rename the command for what the narrowed output answers. Strictly-stronger-than-the-original still holds; it just lives in Go instead of being pasted per case. When the golden is large and all of it is load-bearing, restructure instead of shrinking: pretty-print it with a lexical formatter round-tripped in the handler (see `/go-test-datadriven`'s rendering rules).

**Delete in the same edit**: the original test functions, helpers that existed only for them (grep first), stale `.snap` entries, obsolete testdata. No `// moved` breadcrumbs; the git diff is the review surface.

### 5. Verify

```
go test -race -count=1 ./<pkg>/...      # gate 1: fresh build catches dead helpers only the old tests used
go test -v ./<pkg>/... 2>&1 | grep -E '^--- (PASS|FAIL)' > /tmp/gotest-skills/<pkg-slug>/tests.after.default.txt
go test -coverprofile=/tmp/gotest-skills/<pkg-slug>/cover.after.default.out ./<pkg>/...
diff <(go tool cover -func=/tmp/gotest-skills/<pkg-slug>/cover.before.default.out) \
     <(go tool cover -func=/tmp/gotest-skills/<pkg-slug>/cover.after.default.out)
```

- **Coverage must not drop.** A per-function drop means a case was lost in translation — grep the deleted tests for that function and recreate the case. Do not add coverage beyond what the originals had. Diff each tag lane against its own baseline.
- **Case accounting**: converted case count ≥ original tests + cases deleted. Old test names legitimately become case names, so compare counts plus the coverage diff, not leaf names.
- Finish at gate 2: `task test:pkg -- ./<pkg>/...` (both build tags). Escalate to `task test` only if the conversion touched shared fixtures or helpers outside the package.
- The conversion just authored a corpus full of `-rewrite`-materialized expectations. Say which goldens are worth a human read, and recommend `/go-test-lockin` if the originals themselves came from regenerated snapshots.

## Output to the user

- Per conversion unit: original tests → new test function(s), harness tier, testdata files created.
- Skipped tests with the one-line reason each.
- Coverage `before% → after%`; any function that regressed and the case recreated for it.
- Goldens worth a human sanity-check — anywhere the rendered output pins more than the original asserted (that's the point of the conversion, but it's new contract surface).

Stay terse. The diff and the testdata files are the artifact; the summary just routes attention.

## Additional resources

- `references/conversions.md` — worked table-driven → datadriven and snapshot → datadriven conversions, and renderer-design notes for lossy originals.
- `.agents/skills/go-test-datadriven/references/examples.md` — full harness templates (pure, testbed, http DSL).
- `.agents/skills/go-test-reshape/references/patterns.md` — a worked standalone → datadriven consolidation.
- `.agents/skills/go-test-groom/references/verification.md` — the suite-wide baseline, two-tag rule, gate list and proof-of-no-loss recipes this skill's steps 1 and 5 follow.
- Exemplars in-tree: `tracing/spanquery/tql_parse_test.go` + `tracing/spanquery/testdata/parse/`, `pkg/logparser/loglexer/tokenizer_test.go`, `core/tenancy/org_user_handler_test.go`.
