# Writing good tests in the AI era

Good tests assert on what the code produces, not how it produces it. That has always been true;
coding agents raise the stakes. An agent validates its change by running your tests, reads them as
the spec of what the code is supposed to do, and can regenerate expectations mechanically. Three
consequences:

- **Tests are the constraint.** With good tests an agent writes good code — and can rewrite the
  internals freely while proving the behavior is preserved. That flexibility is exactly what makes
  agents useful, and implementation-pinning tests take it away.
- **Tests are read more than code.** With agents, the implementation churns while the assertions
  stay put, and both you and the next agent read the tests to learn what the code is supposed to do
  (see [agent-harness.md](agent-harness.md)). Optimize tests for reading, not just for running.
- **Tests get weakened under pressure.** When a coupled assertion fails under a correct refactor, a
  human investigates; an agent pushing to go green will weaken the assertion or regenerate the
  expectation instead. Every implementation-coupled assertion is an open invitation to degrade the
  suite.

## The litmus test

An assertion is sound when it survives every behavior-preserving refactor. Check it against four
imaginary refactors that change nothing callers can observe:

1. **Rename/reorganize** — every unexported identifier renamed; private helpers merged, split, or
   inlined.
2. **Re-algorithm** — batching, caching, or the order of internal steps swapped for something
   different with the same outputs.
3. **Reword** — every error message and log line rephrased.
4. **De-instrument** — every metric and log statement removed or renamed.

If any of these would break the assertion, it tests the implementation, not the contract. "The
current code happens to do X" is not a contract; a contract is written down — in godoc, a README,
or the exported type system.

## What to assert on

Observable results: return values, final public state, rendered output, error semantics.

- **Errors**: assert the type and its structured fields (`errors.AsType[*ParseError](err)`), or the
  sentinel (`errors.Is(err, fs.ErrNotExist)`). Never match substrings of messages this package
  doesn't own — wording changes across library versions and operating systems.
- **Unordered results**: `require.ElementsMatch`, or sort before comparing. Fixed-order comparison
  of a map-derived slice passes locally and fails in CI.
- **Async outcomes**: `require.Eventually` on the observable result — never `time.Sleep` and hope.
- **Internal details that matter**: promote them to a contract first — a return value, a public
  method, a documented side effect — then assert on that. If the API can't express the behavior,
  that's an API gap to fix, not something to reach with reflection.

## What not to assert on

- Whether a cache was populated, how many times a helper was called, the order of internal steps.
- Log output and metric values — observability, not behavior (unless logging or metrics is the unit
  under test).
- Generated SQL by substring — assert on the query result, or golden the whole rendered query so
  the diff is reviewable.
- Unexported fields and functions — if the behavior is real, it is observable through the public
  API.
- `runtime.NumGoroutine()`, wall-clock timing, scheduler behavior.

All of these break under the litmus test's refactors — and each break either blocks a correct
change or trains an agent to weaken the test.

## One test, one behavior

- Each test function owns one behavior or one tight cluster of behaviors; cases inside share one
  assertion shape, so a reader can predict what dimension the next case adds.
- Name tests and cases for behavior, not mechanics: `TestParse_rejectsTrailingComma`, `"negative
  duration returns error"` — not `TestParseV2`, `"case 7"`.
- Tests live with their subject: the tests for `foo.go` go in `foo_test.go`.

## Which test style to use

**Datadriven tests** (preferred) — cases live in plain-text `testdata/` files that read
top-to-bottom as a documented behavior spec: directive, input, expected output. This is the sweet
spot with agents: the Go plumbing is written once, expectations regenerate mechanically with
`-rewrite`, and a human reviews behavior by reading text diffs instead of test code. See
[`pkg/datadriven/README.md`](../pkg/datadriven/README.md) for the file format and `/go-test-datadriven`
for the house idioms.

**Table-driven tests** — when datadriven doesn't fit and the expected output is a small Go value.
When the expected output is large or structured (multi-line strings, JSON, YAML), keep it in a
datadriven file instead of inlining it — inline blobs obscure the test logic and make diffs
harder to review. `cmd/serve/vue_test.go` is a table-driven test.

**Plain standalone tests** — only for corner cases or complex behavior that needs unique
setup/teardown or asserts on side effects across multiple calls. Include a comment describing the
scenario.

## Keep goldens trustworthy

Mechanically regenerated expectations are the sharpest risk of agent-written tests: `-rewrite`
materializes whatever the code *currently does* as "expected" — including bugs.

- Review every regenerated golden as an assertion you are authoring, not output you are accepting.
- Never make a failing test pass by regenerating its expectation unless you can say why the
  behavior legitimately changed.
- Keep goldens deterministic — fake clock, sorted rendering, no timestamps or random IDs — so a
  regeneration diff contains only intended changes.

## Don't lock in what isn't decided

Agents will happily write a test that pins controversial or accidental behavior —
[README.md](README.md) lists this as one of the two things agents are reliably bad at. Whether a
behavior *should* be guaranteed is a trade-off call, and trade-off calls belong to humans:

- Test the behaviors that are intended and documented. When a behavior looks accidental or
  undecided — a quirk callers shouldn't rely on — flag it for a human instead of enshrining it in
  an assertion.
- Golden only contract fields: render purpose-built structs, not whole domain objects, so
  incidental fields never become de-facto contracts.

## Skills for improving tests

The `go-test-*` skills in `.agents/skills/` compose as a pipeline over one package, in this order:

**`/go-test-reshape`** — fixes granularity and file layout: consolidates many small, ad-hoc test
functions into datadriven or table-driven tests, splits overly broad "switchboard" tests into
focused behavior-scoped ones, and moves tests into the file that matches the code under test.

**`/go-test-decouple`** — audits every assertion for pinned internal implementation (mock call counts,
unexported fields, error substrings, log output, white-box calls to unexported functions, etc.) and
retargets them onto the public contract. Use when tests break under correct refactors.

**`/go-test-coverage`** — adds test cases for uncovered public-API behaviors, driven by coverage
diff. Extends existing test structures rather than creating new ones.

**`/go-test-cleanup`** — readability polish: scopes test types narrowly, inlines trivial helpers,
removes dead setup, and rewrites assertions to be self-documenting.

**`/go-test-groom`** — runs the pipeline (reshape → decouple → coverage, optionally cleanup) end-to-end
in one prompt with a shared baseline and a single combined report.

**`/go-test-datadriven`** — owns the datadriven authoring idioms (harness tiers, testdata layout,
rendering rules) that the other skills lean on.

**`/go-test-to-datadriven`**, **`/go-test-lockin`**, **`/go-test-determinism`** and
**`/go-test-speedup`** — convert named tests to datadriven, pin a behavior before a refactor,
remove flakiness, and make a slow package fast.
