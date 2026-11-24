<!-- lora-design:v3 -->

# datadriven design

Scope: `pkg/datadriven/`

## Motivation

Tests across the repository need a compact way to state many behavior cases and review their inputs and results.
Authors also need safe result updates because mechanical rewrites can hide defects or change unrelated corpus text.

Stateful workflows add another force.
Earlier cases prepare later cases, while each corpus file needs isolated state and a precise failure location.
Large suites must also remain easy to narrow when one case fails.

## Solution

A parser turns each plain-text corpus into an ordered document of commands, arguments, inputs, expected outputs, and preserved trivia.
The runner sends each case to a caller handler and compares the returned text.
It keeps source order and stops a file after failure.

`Walk` maps a corpus tree to named Go subtests without starting concurrent file runs.
The formatter replaces selected results while it preserves all other source text.
Argument helpers retain raw expressions and decode structured YAML values.

Focus keeps a state-building prefix while it selects one file or case.
Positioned errors and progress logs connect failures to corpus source.
The document model also lets specialized tests inspect cases without running handlers.

## Primary goals

- **G1** — A reviewer can understand each case's command, input, and expected result from its plain-text corpus.
  Need: [consumer] [contract] The repository chooses this style so humans review behavior in text diffs instead of Go plumbing.
  Evidence: `docs/writing-good-tests-ai-era.md`, `pkg/datadriven/README.md`.

- **G2** — Each executed case fails when its handler result differs from the expected output.
  Need: [safety] [contract] A changed result must fail until a reviewer accepts the new expectation.
  Evidence: `pkg/datadriven/README.md`, `pkg/datadriven/run_test.go:TestMultiLineTest`.

- **G3** — Each case can observe state that earlier cases in the same file created.
  Need: [consumer] `pkg/ddtest` runs command workflows whose later cases read state from earlier cases.
  Evidence: `pkg/ddtest/runner.go:Run`, `pkg/datadriven/focus_test.go:TestFocusLine`.

- **G4** — A failed case prevents every later case in its file from running.
  Need: [safety] Later commands can require valid state from every earlier command.
  Evidence: `pkg/datadriven/README.md`, `pkg/datadriven/run.go:runCases`.

- **G5** — Directive argument decoding preserves the author's scalar text, field order, and list shape.
  Need: [consumer] `pkg/ddtest/ddhttp` builds paths, headers, queries, and ordered extracts from these distinctions.
  Evidence: `pkg/datadriven/args_test.go:TestArgText`,
  `pkg/datadriven/args_test.go:TestArgFields`, `pkg/ddtest/ddhttp/request.go:runRequest`.

- **G6** — A handler can decode directive scalars, slices, maps, and durations into its selected Go types.
  Need: [consumer] Repository test commands use typed arguments instead of repeating parse logic.
  Evidence: `pkg/datadriven/args_test.go:TestArgScan`, `pkg/datadriven/README.md`.

- **G7** — A case can compare expected output that contains blank lines.
  Need: [contract] Query plans, diffs, and formatted documents need multi-paragraph expectations.
  Evidence: `pkg/datadriven/README.md`, `pkg/datadriven/run_test.go:TestRewrite`.

- **G8** — Refreshing results changes only the selected expected-output sections.
  Need: [safety] [contract] Golden updates must not churn source text or bless unexecuted cases.
  Evidence: `pkg/datadriven/format_test.go:TestFormatParseRoundTrip`,
  `pkg/datadriven/run_test.go:TestRewrite`, `pkg/datadriven/focus_test.go:TestFocusLineRewrite`.

- **G9** — Each corpus file, and no adjacent documentation or editor file, maps to its own named Go subtest.
  Need: [consumer] `pkg/ddtest` attaches fixture reset and cleanup to each file subtest.
  Evidence: `pkg/datadriven/run_test.go:TestWalk`,
  `pkg/datadriven/focus_test.go:TestFocusWalk`, `pkg/ddtest/runner.go:Run`.

- **G10** — A shared runner can walk corpus files without concurrent access to its mutable file state.
  Need: [safety] [consumer] `pkg/ddtest.Runner` and its HTTP command keep mutable state across one file.
  Evidence: `pkg/ddtest/runner.go`, `pkg/ddtest/ddhttp/command.go`.

- **G11** — A caller can inspect ordered corpus cases without executing handlers or reading runner flags.
  Need: [consumer] The trace-query agreement test reuses one corpus for a second comparison path.
  Evidence: `tracing/spanquery/tql_parse_test.go:TestParseResolverAgreement`,
  `pkg/datadriven/parse_test.go:TestParseTrivia`.

- **G12** — Malformed or ambiguous corpus text fails before any handler receives it.
  Need: [safety] Silent acceptance can omit assertions or change a command's meaning.
  Evidence: `pkg/datadriven/parse_test.go:TestParseErrors`,
  `pkg/datadriven/args_test.go:TestParseLine`.

- **G13** — Comment lines never invoke handlers or create case groups.
  Need: [contract] Authors use comments and dividers to explain cases without changing execution.
  Evidence: `pkg/datadriven/run_test.go:TestComments`, `pkg/datadriven/README.md`.

## Secondary goals

- **G14** — An author can compare or rewrite one case while its earlier state-building cases still run.
  Need: [maintenance] [contract] Focus is the supported workflow for one failing case in a stateful file.
  Evidence: `pkg/datadriven/focus_test.go:TestFocusLine`,
  `pkg/datadriven/focus_test.go:TestFocusLineRewrite`,
  `tsmetric/metricread/testdata/query_timeseries/README.md`.
  Loss: A targeted rerun then compares unrelated cases and executes the target's tail.
  Measure: no budget is set; compare end-to-end feedback time if this separates two candidates.

- **G15** — A failure reports the corpus path and source line for its case.
  Need: [maintenance] [contract] Authors copy this position into focus and use it to locate the failing source.
  Evidence: `pkg/datadriven/README.md`, `pkg/datadriven/file_test.go:TestCaseAt`,
  `pkg/datadriven/parse_test.go:TestParseErrors`.
  Loss: Authors must search the corpus for the active directive after each failure.
  Measure: no budget is set; measure failure-location time if this separates two candidates.

- **G16** — A long output mismatch shows a unified diff near changed lines.
  Need: [maintenance] Authors must distinguish a small behavior change inside a large result.
  Evidence: `pkg/datadriven/run.go:reportMismatch`.
  Loss: Long mismatches then print two complete blocks that authors compare manually.
  Measure: the current threshold is more than five expected-output lines.

- **G17** — A case that runs for ten seconds reports its position before the test timeout.
  Need: [maintenance] Authors must identify the active case when a suite hangs.
  Evidence: `pkg/datadriven/run.go:watchSlowCase`.
  Loss: A timeout then omits the active corpus position.
  Measure: the first report and each later report use a ten-second interval.

- **G18** — An author can remove all expected results while preserving commands, inputs, and comments.
  Need: [maintenance] [contract] A broken serializer can require clean results before a safe rewrite.
  Evidence: `pkg/datadriven/run_test.go:TestClearResults`, `pkg/datadriven/README.md`.
  Loss: Authors then remove result sections manually before a clean rewrite.
  Measure: none; this is a binary file transformation.

- **G19** — An author can suppress per-case success logs without changing execution or failures.
  Need: [maintenance] [contract] Verbose runs can otherwise obscure the failure in a large corpus.
  Evidence: `pkg/datadriven/README.md`, `pkg/datadriven/datadriven.go`.
  Loss: Verbose runs then print one success log for every executed case.
  Measure: no output-volume budget is set; compare logs if this separates two candidates.

## Non-goals

- **N1** [boundary] — The package interprets command names or owns application test state.
  Owner: The caller-provided handler; `pkg/ddtest/` owns the shared application test DSL.

- **N2** [excluded] — One corpus file contains nested Go subtests.
  Reopens when: Separate files cannot express required nested setup, cleanup, or `go test -run` selection.

- **N3** [no guarantee] — The package remains compatible with future `cockroachdb/datadriven` syntax or APIs.

- **N4** [no guarantee] — A plain `----` expected-output section contains blank lines.

- **N5** [no guarantee] — A case input starts with `>` without a leading blank line.

## Frozen contracts

- **F1** — The corpus grammar: a line opening with `>` is a directive and a
  following `>` line continues it, a line opening with `#` is a comment, `----`
  opens an expected-output section, and `---- BEGIN` and `---- END` bound output
  that holds a blank line.
  Frozen by: every committed corpus of this repository. Refreshing rewrites an
  expected-output section and never a directive, an input or a comment, so the
  hand-written half of a corpus is text no tool can restore.
  A break costs: every corpus spelling the old grammar fails to parse, and an
  author rewrites by hand what the refresh cannot. A grammar change that still
  parses is worse — a case then runs a command it does not name.
  Evidence: `pkg/datadriven/parse.go`, `pkg/datadriven/README.md`
