---
name: go-test-datadriven
description: Authors new datadriven tests in this repo's house style, or audits existing datadriven tests/testdata against it. Invoke ONLY when the user explicitly runs `/go-test-datadriven` or explicitly names this skill — do not infer it from generic phrasing like "write a datadriven test", "clean up datadriven tests", or "datadriven conventions" alone; treat those as ordinary requests instead.
---

# go-test-datadriven

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
- Only the **pure** tier exists here. For an HTTP handler, build the mux in the
  test, serve each case through `httptest.NewRecorder`, and render the status and
  body as the output. The exemplars are `pkg/datadriven`'s own tests.

Author new datadriven tests (`pkg/datadriven` + `pkg/ddtest`) in this repo's house style, or bring existing ones in line with it. The house style makes testdata files the reviewable artifact: each file reads top-to-bottom as a documented behavior spec, and the Go handler is a thin, deterministic renderer.

A datadriven test has failure modes a green run hides: a case that silently never runs, a corpus that `-rewrite` only half regenerates, a golden that is true on one build or one CPU only, a directory no test reads. An audit that only checks style misses all of them, so the audit below checks them first.

Related skills: `/go-test-reshape` decides shape package-wide (it converts to datadriven where that's the right shape, splits switchboards, and moves tests to the right file); `/go-test-to-datadriven` executes a user-directed conversion of named tests into datadriven shape; `/go-test-groom` runs the whole `go-test-*` pipeline. This skill owns the datadriven idioms themselves and is the reference the others lean on.

## Read first

Before writing or editing anything, read:

- `pkg/datadriven/README.md` — file format, `#` comments, args, `---- BEGIN/END`, `-rewrite` / `-focus` semantics, `TestCase` API.
- `pkg/ddtest/README.md` — the `Command` interface, reset lifecycle, built-in commands, the `http` DSL (only needed for testbed-tier tests).
- `docs/go-test-conventions.md` § Determinism and § Fixtures and corpora — the fixed clock epoch, float rendering, and the unterminated-command trap.
- `cmd/gotest/README.md` § "A `-short` gate must not narrow a corpus behind `-rewrite`".

Also read the target package's `README.md` / module `AGENTS.md` — some packages have extra constraints (e.g. `tracing` querygen goldens under `json_on/`/`json_off/` need a live ClickHouse and cannot be `-rewrite`n offline).

## Pick the harness tier

Two tiers exist; pick the lightest one that exercises the behavior.

1. **Pure handler** — `datadriven.Walk` + `datadriven.RunTest` with a `switch tc.Name` handler. No fx, no DB. For parsers, tokenizers, formatters, pure transforms. Exemplars: `pkg/logparser/loglexer/tokenizer_test.go`, `tracing/spanquery/tql_parse_corpus_test.go`.
2. **Testbed runner** — the package's own `Run(t)` entry point (declared over a `testbed.Bed` in its `testbed_test.go`), which boots the full app (Postgres, ClickHouse, Redis, router) and returns a `*testbed.App`. `app.Runner()` is a `*testbed.Runner` (a `*ddtest.Runner` plus the `http` command) whose fixture is pre-seeded with the default org/user/project; register your commands and formats on it, then run the corpus with `app.RunDatadriven("testdata/dir")`. Application values the commands need come out with `fx.Populate`. For HTTP handlers, repos, and query paths. Exemplars: `core/tenancy/org_user_handler_test.go` (http DSL), `tsmetric/metricread/query_handler_test.go`. For a test that runs two corpora or needs a custom router/fixture, use `app.NewRunner(testbed.WithRouter(r))` or `app.NewRunner(testbed.WithEmptyFixture())` and run `runner.Run(t, dir)`. For a corpus over pure code — no app, no database — use `ddtest.NewRunner()` and register only what it speaks.

Do not boot the testbed for a pure function; do not hand-mock half the app when the testbed gives you the real thing.

The tier is chosen **per test function, for its heaviest case** — and test boundaries follow the observable contract (the directive vocabulary and the rendered output), never the harness wiring. If two groups of cases drive the same command with the same rendering, they are one test walking one `testdata/<topic>/` directory (split into *files* by behavior domain), even when only some cases need the app: a dependency the code under test consults internally (a bridge, a repo, a cache) is invisible in the corpus and must not split it. Running contract-identical pure cases under the testbed harness is fine; splitting one contract into a pure twin corpus and a testbed twin corpus is not — the corpus layout would then document how the unit is constructed instead of what it does. Exemplar: `tsmetric`'s `TestNameResolverAttr` resolves star/billing/custom attrs (no app needed) and tracing attrs (bridge needed) in one corpus at the testbed tier.

**Pin a dependency the corpus cannot observe, then prove the pin.** Check what the app actually contributes to the output. Sometimes a testbed test's only app dependency returns the same state for every case, because the corpus seeds nothing that dependency reads (a lookup over an empty table, a cache that stays cold). Then the corpus is pure work paying for a full app boot. Run it at the pure tier with that dependency built in its observed state. Keep the rest of the production path real, and state the pin in a doc comment. Add **one** integration test that reads every case of the same corpus, runs it through both the live and the pinned dependency, and requires equal output. That test has no goldens of its own, so it is not a twin corpus: it fails the day the pin stops matching production. The existing goldens must pass byte for byte after the move. Exemplar: `TestParse` in `tracing/spanquery/tql_parse_corpus_test.go`, and its agreement test `TestParseResolverAgreement` in `tracing/spanquery/tql_parse_test.go`.

**Not every test is datadriven.** A corpus that holds one scenario — one case, or one fixed input whose output is a single scalar — gains nothing from a golden and pays the Walk/RunTest overhead. A deliberate literal pin of a wire format must not live in a golden at all: `-rewrite` re-pins it silently, which defeats the pin. Keep those direct or table-driven; an audit reports them as over-conversion, and `/go-test-reshape` owns reverting them. A corpus of several cases stays datadriven even when each output is one short line: the cases read as a spec, and adding one is a three-line edit.

## Go-side idioms

- **Doc comment on the test function**: what commands exist, what the rendered output is, and any invariant the handler enforces beyond the golden. `TestParse` in `tracing/spanquery/tql_parse_corpus_test.go` is the model.
- **`Walk` over a directory**: `datadriven.Walk(t, "testdata/<topic>", ...)` — each file is its own `t.Run` subtest. Don't hand-list files with individual `RunTest` calls.
- **Dispatch on `tc.Name`** with a default arm of `tc.Fatalf(t, "unsupported command: %s", tc.Name)`.
- **`tc.Fatalf`, never `t.Fatalf`**, inside handlers — it prefixes the failing case's `file:line`.
- **The per-file `t` reaches every helper.** A handler, a command's `Run`, and any hook or callback a command calls take the `t` that `Walk`/the runner passes for the current file, never the parent test's `t`. A failure then lands on the file's subtest, and `t.Cleanup` runs when that file ends. A hook field on a shared command type needs a `*testing.T` parameter for this reason.
- **A command that takes no input refuses one.** `if tc.Input != "" { tc.Fatalf(t, "%s takes no input; end the directive with ----", tc.Name) }`. The parser gives such a command the directives after it when its `----` is missing, and a command that ignores its input hides that loss. When an audit finds a swallowed directive, add this check to the command that swallowed it — built-in or custom — so the next one fails at once.
- **Arguments**: declare the default value first, then `tc.NamedArg("x").Scan(&x)` — `Scan` is a no-op when the arg is absent, so the default survives. Check the returned error with `tc.Fatalf`. Positional args via `tc.Arg(0).Name`.
- **Input block vs args**: the input block carries the case's *primary textual input* — the string the command parses or transforms (query text, an expression, a payload). Short scalars (identifiers, keys, options) are named args, even when one of them is the command's main parameter — `> describe metric=m attr=a`, never `> describe metric=m` with the attr as the input block. A command whose parameters are all scalars takes no input.
- **Error taxonomy**: an error that is itself the behavior under test (parse error, validation rejection, 4xx) is *output* — return `err.Error()` (or render its typed fields) so the golden pins the message. An error that means the harness broke (fixture load failed, YAML didn't parse, scan failed) is `tc.Fatalf`.
- **Invariants beyond the golden** (round-trips, idempotence, fixed points) are checked in the handler with `tc.Fatalf` and documented in the test's doc comment. The golden pins the value; Go pins the invariant. See the re-parse and `Prepare` fixed-point checks in `TestParse`.
- **`-short` never narrows a `-rewrite`.** A test may run a subset of its corpus under `testing.Short()` when the full corpus is slow, but the gate is `testing.Short() && !datadriven.Rewriting()`. A rewrite runs under `-short` in every lane except race, so without the escape it regenerates part of the corpus and leaves the rest stale. Keep in the short set every file that pins a value the code can change on its own (a content hash, a generated id, header text). A skip of one file under `-short` is the same narrowing. `tsmetric/metricread/engine_test.go` shows the shape.
- **Rendering** — the golden must be deterministic and minimal:
  - YAML (`github.com/goccy/go-yaml`) for structured output. Marshal a purpose-built struct with explicit `yaml` tags and `omitempty`, not the domain model — the struct defines field order and keeps irrelevant fields out of the golden.
  - Plain text for inherently textual output (normalized query strings, error messages, token streams).
  - When the output is the input transformed within the same language (a normalized query, a rewritten expression), render it *as* that language — the case then reads as a rewrite rule, input above, result below. Don't wrap it in a YAML list of clauses; that format mismatch hides the rewrite. Exemplars: `TestParse`, `tsmetric`'s `TestQueryNormalization`.
  - `tabletest.RenderColumns` for tabular query results (columns + rows as an ASCII table); `tabletest.Render` when you format the cells yourself.
  - **Render a computed float at `%.12g`** (`strconv.FormatFloat(v, 'g', 12, 64)`), never `%g`, `%v`, or `'g', -1`. Full precision pins the last digits, and those differ between amd64 and arm64 where a multiply-add contracts. A renderer that chose a tighter verb on purpose (`%.6g`) keeps it. `pkg/tabletest` already renders at `%.12g`.
  - Never render wall-clock time, random IDs, or map iteration order. A time from the fixture's fake clock is deterministic — its epoch is fixed — so a golden may show it. Sort slices, and render only deterministic facts.
  - **Output that differs by build gets one golden per build.** When the rendering depends on `featflag.JSON` or the `cloud` tag (SQL over map vs JSON storage, a cloud-only field), a single golden is true for one build only. Split it into per-mode files or dirs (`_json_on`/`_json_off`, `json_on/`/`json_off/`) and select the one to walk with the flag. Scope the split to the corpora whose output depends on the flag.
  - When the full rendering is dominated by a case-invariant block and the cases differ in one fact (e.g. long SQL where only the table name varies), golden the varying fact and enforce the invariant remainder in the handler — a `tc.Fatalf` shape check (exact-match template or anchored regexp) that fails with the full value in the message. Same strength, no boilerplate repeated per case; `tsmetric/metricread/catalog_scan_test.go` is the model.
  - When the pinned artifact is genuinely large *and* all load-bearing (assembled SQL, a rendered plan), the fix is layout, not omission: pretty-print it — for SQL use the in-repo `pkg/sqlpretty`, and in general look for an existing formatter before hand-rolling one — and verify in the handler that the formatted text round-trips to the rendered original (whitespace-stripped and case-folded, since layout spacing and keyword casing are the formatter's own domain; quoted literals, idents, and structure must survive untouched). The round-trip check is what licenses the formatter — without it, formatting is a second renderer that can drift. Goldens become reviewable and `-rewrite` diffs localize to the clause that changed; `chScanSQL` in `tsmetric/metricread/storage_test.go` is the model. A non-SQL placeholder like `<time range>` is inserted *after* formatting — the formatter would lex it as garbage.
  - Pin an absence structurally when the output would show the thing if it happened: a rejected query goldens as its bare error line, and any SQL executed anyway would render below it — no `queries: []` scaffolding needed.
- **One behavior, one command.** Like a table-driven test, a datadriven test exercises one behavior through one command, parametrized with a few args/options — not a vocabulary. If you find yourself defining several custom commands (or a `switch tc.Name` with many arms), you are building a switchboard: split into separate test functions with their own `testdata/<topic>/` directories instead. The exception is commands that load data/fixtures into the database (`seed`, `insert`, `load_metrics`, `load_traces`, `adjust_clock`, …) — any number of those may accompany the single command under test. A command name that names a scenario (`> zero_budget`, `> unknown_kind`) with the scenario built in Go is the same smell: the case is invisible in the file.
- **One behavior applies to the output too.** A single command whose golden has sections answering unrelated questions (e.g. `parts:` — how the query normalized — above `queries:` — what SQL it executed) is two behaviors sharing a directive: each section churns for its own reasons and drowns the other in review. Split into two test functions over their own corpora, even when both run the same harness path. Scope feature-flag mode dirs (`json_on/`/`json_off/`) to the test whose output actually depends on the flag — after the tsmetric split, the SQL corpus kept its mode dirs while the normalization corpus needs only one.
- **Custom commands** (testbed tier): `ddtest.CommandFunc` when stateless; implement `ddtest.Resetter` alongside `Run` when the command accumulates state a file reset must undo — and only then; a command with no state implements no `Reset` at all. Register with `runner.Register(name, cmd)`; command names are `snake_case`. Name the command for the question its output answers, not the Go method it wraps: a command that builds a catalog query but renders only the table it targets is `scan_table`, not `build` — when the rendering narrows (see the rendering rules above), re-check that the name still describes the output.
- **Fixture access**: use the `f *dbfixture.Fixture` passed into `Run` — never cache fixture-keyed entities (project, user, token) across files; the fixture is reseeded per file.

## Testdata idioms

- **Layout**: `testdata/<topic>/<behavior>` — one directory per test function, one file per behavior. Corpus files are extensionless, and the filename names the behavior (`errors`, `aliases`, `grouping`, `create_requires_admin`; no `.txt`). There is no in-file subtest mechanism; when a file needs `# ====` section dividers to stay navigable, that's the signal to split it into files. dbfixture seed `.yml` files live under `testdata/fixtures/` so they read apart from the corpora.
- **Every corpus directory has a test that walks it.** When a test stops walking a directory — renamed, split, or retired — delete the directory in the same change. A corpus no test reads still looks like a spec, and it rots silently: it keeps a format the code no longer speaks.
- **Every command ends with `----`, even with no output.** The input runs to the `----` line or EOF, and a blank line does not end it. So a bare `> adjust_clock 5m` followed by a blank line and `> load_traces …` is **one** case: the loader and everything up to the next `----` become the clock command's input and never run, and the case after it asserts against state that was never built. The test then fails as if the product were broken, or passes while asserting nothing. Write a no-output command as the directive, then `----`, then a blank line.
- **Comment every case** (or tight group of cases) with a one-line `#` comment stating the behavior — what the case demonstrates and, when non-obvious, *why* the output is what it is. Not a restatement of the directive. `tracing/spanquery/testdata/parse/aliases` is the model:

  ```
  # a non-aggregate alias in having is demoted to where and reads as an attribute
  # there; parse kinds it but never expands the alias.
  > parse
    lower(_status_code) as code | having code = "error"
  ----
  lower(_status_code) as code | where code::str = "error"
  ```

- **A comment stays true, and an input discriminates.** Choose an input where the behavior the case names changes the golden: if the code regressed to the old or the rejected behavior, the output would differ. When the code or input shape changes, re-read the comments of the cases it touched. A comment that claims a check the input no longer exercises is a false spec.
- **One primary command per file**, parametrized with args. Two sanctioned exceptions: setup/seed commands before the case under test (`seed`, `insert`, `load_metrics`, `load_traces`, `adjust_clock`), and HTTP workflow files that drive a `> http …` sequence against one endpoint or workflow.
- **Minimal inputs**: the smallest input that demonstrates the behavior. Seed only the entities the case touches. A leading `> reset` is redundant: the runner resets between files.
- **Format mechanics** (details in `pkg/datadriven/README.md`): lists as `key=[a, b]`, quoted scalars for spaces, `>`-prefixed continuation lines for long directives (which also means a case's input can never start with `>`), `---- BEGIN` / `---- END` only when the output contains a blank line (`-rewrite` picks the delimiter and normalizes a needless one away), never repeat an arg name.

## Testbed-tier idioms

- **Built-ins are pre-registered** on a testbed runner: `reset`, `seed`, `insert` / `update` / `delete` / `select`, `adjust_clock`, and `http`. A bare `ddtest.NewRunner()` has only `reset`. Register only what's missing — reuse the shared loaders `tsmetricddtest.LoadMetricsCommand`, `tracingddtest.LoadTracesCommand`, `tracingddtest.LoadEdgesCommand` before writing a bespoke command for the unit under test.
- **Seed in the testdata file, not in Go**: prefer `> seed` / `> insert user` directives over Go-side fixture calls — the reader sees the whole scenario in one file. `fake_data=true` fills plausible defaults.
- **Interpolate fixture values** with `{entity:key.field}` (`{project:default.id}`, `{user_token:default.token}`) instead of hardcoding IDs.
- **Capture created IDs** with `> http store_key entity:key path=$.jsonpath`, then use `{entity:key.id}` in later directives; assert DB state after mutations with `> select entity key=k` plus an expr-lang expression as input.
- **Keep HTTP goldens small**: set `> http base` and `> http bearer` once at the top; use relative paths (`./alerts`); `format=status` when the body is irrelevant; `extract={<name>: '$.jsonpath'}` to golden only the relevant fields. When cases need the same trimming, register a named format (`runner.HTTP().RegisterFormat("view", …)` + `SetDefaultFormat("view")`) that marshals a purpose-built YAML struct, and select a non-default view per directive with `format=<name>` — see `core/tenancy/org_user_handler_test.go`. Error responses (status >= 400) auto-render as the standard envelope; use `SetErrorFormat` only to post-process it.
- **Time**: `> adjust_clock 1m` advances the fixture's fake clock (loaders pin rows to it); the clock rewinds to its epoch on every per-file reseed. It renders nothing, so it still needs its `----`.

## Workflow for new tests

1. Write the Go test + testdata directives with empty expectations (directive, input, `----`, blank).
2. Materialize goldens: `go test ./<pkg> -rewrite` — the flag must come **after** the package list; `go test -rewrite ./...` silently tests nothing. When any output depends on the build, rewrite under both builds: `go test -tags=invariants ./<pkg> -rewrite` as well.
3. Review every rewritten golden as an assertion you are authoring, not output you are accepting — `-rewrite` happily encodes bugs as "expected". If a golden looks wrong, fix the code or the case, not the golden. Never make a failing test pass by regenerating its expectation unless you can say why the behavior legitimately changed. A golden changes only through `-rewrite`: never convert goldens with a script or by hand in place of a run, because the run is what proves the new text is what the code prints. When the run that would rewrite a golden needs a database you cannot use, keep the rendering change out of the patch and report it with the command to run.
4. Iterate on one case with `-focus <path>:<line>` (copied from the failure message; cases above it still run to build state but aren't compared). Target a file with `-run TestX/<file>`.
5. If a systematic bug garbled every golden, wipe with `datadriven.ClearResults`, fix, then rewrite cleanly.
6. Run `scripts/check_corpus.py <pkg>` (next section).
7. Finish at gate 2 of the suite's gate list: `task test:pkg -- ./<pkg>/...` (both build tags). Escalate to `task test` only when the work reached shared fixtures or helpers outside the package. Integration-level corpora need the databases up before any of this — `task up`, then `task test:setup`; see `.agents/skills/go-test-groom/references/verification.md` §4–§5.

## Auditing existing datadriven tests

Do the steps in order. Steps 1–3 find the defects that a green run hides, and they are cheap, so never skip them for the style pass.

1. **Run the corpus checker** from the repository root: `python3 .agents/skills/go-test-datadriven/scripts/check_corpus.py <pkg> [<pkg> …]`. It reports each *swallowed* directive (a command with no `----` whose input holds a later directive) and each *unwalked* corpus directory (no `*_test.go` of the package names it). A swallowed finding is always a defect. For an unwalked finding, confirm with `git grep` that no test builds the path, then delete the directory.
2. **Grep the test files** of the package for the hidden-defect shapes:
   - `testing.Short()` in a test that walks a corpus — the gate must also check `!datadriven.Rewriting()`, and a per-file skip under `-short` is the same narrowing;
   - float formatting in a handler or renderer (`%g`, `%v` on a float, `FormatFloat(…, -1, …)`) — use `%.12g`;
   - `featflag.JSON` or a `cloud` build tag near a golden — a golden whose output differs by build needs one file per build;
   - a testbed `Run(t…)` in a test whose corpus seeds nothing the code reads — a candidate for the pure tier with a proven pin.
3. **Read each case's comment against its input.** Flag a comment that claims a behavior the input cannot distinguish, and an input whose golden would be the same under the regressed behavior. When a golden pins behavior that looks wrong — a comment explains away a contradiction, or the doc of the code says otherwise — report it as a suspected locked-in bug with the evidence, and leave the product code and the golden as they are. An audit changes tests; whether the behavior changes is the user's decision (`/go-test-lockin` investigates such a case).
4. **Walk the smell table** below, top-down.
5. **Fix, then prove.** After edits, run `-rewrite` (both builds when output depends on the build) and verify the diff contains only intended churn. The formatter round-trips untouched cases byte-for-byte, so any other churn is a real behavior change. A change that alters goldens you cannot regenerate stays a finding (see step 3 of the workflow). A fixed swallowed directive makes the hidden commands run for the first time: their goldens may fail, and each failure is a finding to report, not an expectation to regenerate blindly. Run `scripts/check_corpus.py` again; it must report nothing.

| Smell | Fix |
|---|---|
| A command with no `----` followed by more directives | Terminate it with `----`; make the command refuse input; review the goldens of the now-running directives |
| A corpus directory no test walks | Delete it in the change that stopped walking it |
| `-short` narrows the walked set with no `datadriven.Rewriting()` escape | Gate with `testing.Short() && !datadriven.Rewriting()` |
| A float rendered at full precision (`%g`, `%v`, `'g', -1`) | `%.12g`, then `-rewrite` |
| One golden for output that differs by build | One golden per build, selected by the flag; rewrite under both tags |
| Testbed boot for a corpus whose app dependency is constant | Pure tier with the dependency pinned, plus one agreement test over the same corpus |
| A corpus holding one scenario, or a literal wire-format pin | Report as over-conversion; `/go-test-reshape` reverts it to a direct test. Several cases with short outputs are not this smell |
| One giant file navigated by `# ====` sections | Split into per-behavior files under the same `testdata/<topic>/` dir |
| Bare cases with no `#` comments, or comments the input no longer supports | Add or correct a one-line behavior comment per case/group |
| `t.Fatalf` inside the handler | `tc.Fatalf` (adds `file:line` of the case) |
| A helper or hook called from a handler gets the parent `t` | Pass the per-file subtest `t` |
| Handler returns `""` and asserts via `require` on values | Render the values into the output — goldens are the reviewable assertion; keep only cross-case invariants as Go checks |
| Goldens full of irrelevant response JSON | `format=status`, `extract={…}`, or a default formatter marshaling a trimmed YAML struct |
| One command's golden mixes sections answering unrelated questions | Two behaviors sharing a directive — split into separate test functions with their own corpora; mode dirs stay only with the output that depends on the flag |
| A same-language transformation rendered as a YAML clause list | Render the transformed input in its own language, so the case reads as a rewrite rule |
| Goldens are long one-line strings (SQL, plans) that can't be read or diffed | Pretty-print with an existing formatter (`pkg/sqlpretty` for SQL) whose output the handler round-trips back to the original |
| Goldens churn under `-rewrite` (wall-clock timestamps, IDs, map order) | Fake clock, `store_key` + interpolation, sorted/struct-ordered rendering |
| Setup done in Go, invisible in the file (including a command named for a scenario) | Move to `> seed` / `> insert` / `> load_*` directives |
| Unrelated commands mixed in one file | Split; keep setup + one primary command per file |
| Many custom commands registered on one test (a `switch tc.Name` switchboard) | One behavior/command per test function, each with its own `testdata/<topic>/` dir; only data-loading commands are shared |
| Hand-listed `RunTest` calls per file | `datadriven.Walk` over the directory |
| Output is a Go struct dump (`%+v`, `fmt.Sprint`) | YAML with explicit tags, or one `key: value` fact per line |
| Hardcoded fixture IDs in directives | `{entity:key.field}` interpolation |
| `---- BEGIN`/`---- END` around output with no blank line, or a leading `> reset` | Plain `----`; drop the reset |

## Output to the user

When done, report:

- Harness tier chosen (pure / testbed) and why, in one sentence.
- Commands registered (built-in vs custom) and the testdata layout created or changed.
- For audits: what `check_corpus.py` and the step 2 greps found, then which smells were found and fixed, file by file. List separately what you found and did not fix, with the reason.
- Any golden worth a human sanity-check (behavior pinned for the first time, output that changed meaning during cleanup, or a directive that runs for the first time after a missing `----` was added).

## Additional resources

- `scripts/check_corpus.py` — finds swallowed directives and unwalked corpus directories.
- `references/examples.md` — full copy-paste templates: pure handler, testbed runner with a custom command, HTTP handler test with a formatter, and exemplar testdata files.
- `pkg/datadriven/README.md`, `pkg/ddtest/README.md`, `pkg/testbed/README.md`, `docs/go-test-conventions.md`.
- `.agents/skills/go-test-groom/references/verification.md` — the suite-wide baseline, database-prerequisite, gate-list and golden-regeneration protocol.
- Exemplars in-tree: `tracing/spanquery/tql_parse_corpus_test.go` + `tracing/spanquery/testdata/parse/` (pure tier with a proven pin), `core/tenancy/org_user_handler_test.go` + `core/tenancy/testdata/org_user_handler/`, `tsmetric/metricread/testdata/query_handler/`, `pkg/logparser/loglexer/tokenizer_test.go`.
