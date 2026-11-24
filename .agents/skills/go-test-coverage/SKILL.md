---
name: go-test-coverage
description: Increase test coverage for a Go package by adding cases that exercise uncovered public contracts. Targets only behaviors observable through the public API; skips internal rewraps and dead defensive code. Invoke ONLY when the user explicitly runs `/go-test-coverage` or explicitly names this skill, or when go-test-groom's pipeline explicitly directs it — do not infer it from generic phrasing like "increase coverage", "add missing tests", or "fill coverage gaps" alone; treat those as ordinary requests instead.
---

# go-test-coverage

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

Add test cases to one Go package until every behavior of its public contract is exercised and asserted. Uncovered blocks in a coverage profile point at untested behavior, and an edge-case sweep finds the behavior no block points at. The percentage is a by-product: a case earns its place by pinning a behavior a caller depends on, not by turning a line green.

## The one invariant

**A new case asserts what the contract promises, not what the code happens to do.** Write each case's expected value from the contract — the doc comment, the package `README.md`/`DESIGN.md`, the callers, the spec or external API the code serves — **before** you run the case. Then run it.

- The output matches: the case pins the behavior. Keep it.
- The output differs, and the contract is clear: you found a defect. Do not change the expectation to match the output, and do not fix the production code. Report the defect with its input, the output, the expected value, and the source of that expectation (see "Output to the user"). Leave the case out of the diff, or add it only if the user asks.
- The contract is silent or undecided: do not pin the behavior. A case written against a quirk makes it a de-facto contract (`docs/writing-good-tests-ai-era.md`, "Don't lock in what isn't decided"). List it for a human decision.

The contract is wider than this package's own doc comments. Before you call a contract silent, check these sources. Each one makes a violation a defect, not a decision:

- **The destination.** When the code writes to another system — a vendor API, a chat markup, SQL, a file format — that system's documented limits and grammar are the contract, even when no comment here restates them. A title longer than the vendor accepts, or text that the destination reads as a command, is a defect.
- **The purpose of the code.** An escaper or a sanitizer exists to make untrusted text inert in its destination. A construct that still acts there is a defect, even if the escaper's doc lists only other constructs.
- **Peers.** If the package refuses a bad value of one parameter, a bad value of its peer that passes is a defect. If one sibling implementation guards an edge case, a sibling without the guard is a defect.
- **Scoped rules.** The `AGENTS.md` files on the path to the package, and the contract of an interface the code implements.

A defect also needs a path to it. Name the caller, the config, the stored row, or the external sender that can produce the bad input. When no production path reaches it — a precondition every caller keeps, a code path no caller takes — report it as **latent**, apart from the defects. When the defect is real but its consequence is small, say so, and still report it as a defect.

This skill is where a golden has no earlier value to diff against, so it is where `-rewrite` and `UPDATE_SNAPS=true` lock bugs in most easily. Regenerate only after you wrote the expectation down, and compare each regenerated line with it.

## When not to apply

- The package has no tests at all. Write the first test by hand, so the package's test shape is a decision, not an accident. Then use this skill.
- The tests are fragmented, switchboard-shaped, or in the wrong files. Run `/go-test-reshape` first, so new cases land in the final structure.
- The request is to check whether the existing expectations are right. That is `/go-test-lockin`.

## What counts as the public contract

In a library package the contract is the exported API. In an application package much of it is not an exported Go symbol. It includes:

- exported functions, methods, and types;
- HTTP routes that the package's handlers serve — handler constructors are unexported by the repo's convention, so trace a handler to its route, not to an exported caller;
- services and hooks that the package's fx module provides to other packages;
- CLI commands, cron jobs, workers, and consumers the package registers;
- persisted rows and wire payloads that other code reads.

A block is dead only when **no** entry on this list reaches it, in **either** build lane.

## Procedure

`.agents/skills/go-test-groom/references/verification.md` holds the suite's baseline and gate protocol. This procedure cites it and does not restate it. Drop `-race` from its commands: the root `CLAUDE.md` runs the race lane only on request.

### 1. Baseline

Follow verification.md §2. Decide the lanes first: take the invariants lane too when the package has build-tagged files or reads `featflag`. For an integration package (its tests dial PostgreSQL), check §4 first — the databases must be up, and every command takes `-p 1`.

```
mkdir -p /tmp/gotest-skills/<pkg-slug>
go test -p 1 -count=1 -coverprofile=/tmp/gotest-skills/<pkg-slug>/cover.before.default.out ./<pkg>/...
go test -p 1 -count=1 -tags=invariants -coverprofile=/tmp/gotest-skills/<pkg-slug>/cover.before.invariants.out ./<pkg>/...  # invariants lane only when it applies
```

The invariants lane matters for this skill. Under `-tags=invariants`, `featflag.Invariants` is true. A branch that the default profile shows as uncovered can be fully covered in the invariants lane, and a single-lane profile makes it look dead.

List the uncovered blocks. A profile line that ends in ` 0` never ran:

```
grep ' 0$' /tmp/gotest-skills/<pkg-slug>/cover.before.default.out
```

Each hit reads `file.go:startLine.col,endLine.col numStmts 0`. Read those line ranges in the source. When you took both lanes, the candidate set is the **intersection**: a block that is uncovered in one lane and covered in the other is tested already.

### 2. Classify each uncovered block

For each block, answer in order. Stop at the first answer that removes it.

1. **Is it reachable?** Trace callers up to an entry of the contract list above. Check the guards on the way — `featflag.*`, `//go:build invariants` / `!invariants`. If nothing reaches it in either lane, it is dead code: report it, do not test it, and do not delete it.
2. **Is it observable?** A caller must be able to tell this branch from the others through a return value, an error (`errors.Is`, `errors.AsType`), a side effect on public state, a persisted row, a response, or a released resource. Skip a branch whose only effect is on unexported fields, an unexported `return err` pass-through whose error is already asserted at the public boundary, and a guard that repeats the caller's validation. An error path that an exported function returns, wraps, or replaces is observable — test it.
3. **Is it intended?** Apply the invariant above. An undecided branch goes on the list for a human, not into a test.

A block that survives all three is a **target**. Also check the covered code that no assertion reads: a line can run in a test that never checks its result. Such a behavior is a target too.

### 3. Sweep for behavior that coverage cannot show

Coverage shows only the branches the code has. A missing check has no block to show, and a defect of that kind is found only by input you choose. For each exported entry point, go through its inputs and its outputs:

- **Empty and zero**: empty string, nil or empty slice and map, zero value of a struct, zero duration.
- **Sign and range**: negative where a count, size, or duration is expected; the exact limit and one past it; the largest value (`math.MaxInt`, `math.MaxInt64`). Look for arithmetic on an input — a sum or product that overflows can turn a check inside out.
- **Floats**: NaN, ±Inf, `-0`, subnormal values, where the code divides or compares.
- **Every field of a struct input**: when an entry point takes a config, a params struct, or a request, sweep each field, not a sample — the defect is in the one field that has no guard. Make the list mechanically, not from memory: print each struct with `go doc -all ./<pkg> <Type>`, and write each numeric, duration, string, and enum field down. Mark each one `checked`, `defaulted`, or `unchecked` against the validation code, then test the `unchecked` ones with a negative, a zero, and a bad value. For each bad value that passes, follow the field to the code that consumes it — the client, the query, the allocation. If the consumer gives the value another meaning without an error (a negative timeout becomes "no timeout", a zero limit becomes "unlimited", an empty filter becomes "all"), that is a defect, not an undecided rule. In a big config package this list is long; it is also where this sweep pays most.
- **Restored and decoded state**: a stored blob, a row, a queue message, or a cache entry is input, even when this package wrote it. List what the writer guarantees — order, range, finiteness, non-empty fields, consistency between fields, the version — and give the reader a blob that breaks each one. The reader must refuse it or stay correct without it.
- **Text**: the delimiter, quote, and escape characters that the code itself parses or writes; letter case; a prefix that looks like a keyword or a name; non-ASCII text.
- **Escaping and sanitizing**: judge an escaper by the grammar of its destination, not by its own list. List what the destination reads as special — markup, entities, links, mentions, commands, path separators — and feed each one in. The code's list shows only what its author thought of.
- **Nil** where the function takes a pointer, an interface, or a func.
- **Round trip**: parse then format, encode then decode, save then load — when the API has both halves.
- **Error paths**: on failure, a resource that the call acquired (a connection, a file, a lock, a goroutine) is released.
- **Sibling parity**: when the package holds parallel implementations of one job (dialects, kinds, codecs, drivers), a guard that one sibling has, each sibling needs. Check that each sibling handles the edge cases its peers handle.
- **Combinations**: when the code special-cases some combinations — an override list, a switch over types, a per-kind table — the gaps hide in the combinations the list forgot. Take each special case and put it through each wrapper the package supports (nullable, array, pointer, nested, optional). A combination that works bare and fails wrapped is a defect when its siblings work wrapped.
- **Stated limits**: each limit or rule the docs state — in a doc comment, a `README.md`, a config reference — has a check that enforces it.

Each hit becomes a target. Many sweep hits show a defect rather than a gap; the invariant decides which.

### 4. Rank and stop

Write the target list: `(entry point, behavior, source of the expectation, planned case name)`. Order it by consequence:

1. Behavior that guards persisted data, money, security, or another system: validation, refusals, escaping, limits.
2. Error paths and error types that callers match on.
3. Boundaries and the sweep hits of §3.
4. Formatting and rendering variants.

Work down the list. Stop when the rest of the targets repeat a behavior a case already pins — but separate code does not repeat a behavior just because it follows the same pattern: a per-type override or a per-kind branch can differ from its siblings, so each one needs its own input. Also stop when every uncovered block left is dead, unobservable, or undecided. Do not add a case whose only effect is a green line.

### 5. Add cases

Extend the tests the package has. Do not start a parallel framework.

- **Datadriven tests exist**: add cases to the testdata files, or add a file in the same directory. Group related cases with `#` comment lines. Follow `/go-test-datadriven` for the idioms.
- **Table-driven tests exist**: add rows. For a function the table does not cover, add a table-driven test with the same struct shape.
- **Neither**: use the repo's order — datadriven when the output renders as text, table-driven when it is a small Go value, standalone only for a lifecycle scenario.

**Placement.** The tests for `foo.go` go in `foo_test.go`. If that file does not exist, create it.

**Integration packages.** Boot through the package's own `testbed_test.go` entry point (`app := Run(t)` or `run(t)`), never `testbed.Bed.Run`. Follow `docs/go-test-conventions.md`: times from the fake clock, randomness from `env.Fake()`, no assertion on a literal fake value, and every query and delete scoped to the fixture's project. A subtest reports only on its own `t`.

**Assertions** target the contract: the return value, the error (`errors.Is`, `errors.AsType`), the state that the public API reads back. `/go-test-decouple` lists what not to assert on — log text, unexported fields, mocks of internals.

**Names** state the behavior: `"negative duration returns error"`, not `"covers the else branch"` or `"line 42"`. A comment on a case says why the input matters, in the repo's ASD-STE100 style.

Edit only `*_test.go` files and `testdata/`. Compile-check with `go vet ./<pkg>/...` (add `-tags=invariants` when the invariants lane applies); it compiles the test files, which `go build` does not.

### 6. Prove that each case pins its behavior

A case that passes proves only that its assertion holds. For the targets of rank 1 and 2, make sure the case can fail. Break the branch under test for a moment — invert its condition, or return early — and run the case. It must fail. Then restore the code, and confirm with `git diff -- ':!*_test.go' ':!**/testdata/**'` that no production change is left.

If the case still passes with the branch broken, the input does not reach the branch, or the assertion does not read its result. Fix the case.

### 7. Verify and finalize

1. Write the `after` coverage for each lane (same commands as §1, phase `after`).
2. Diff each lane against **its own** baseline — profiles from different tag sets do not compare:

   ```
   diff <(go tool cover -func=/tmp/gotest-skills/<pkg-slug>/cover.before.default.out) \
        <(go tool cover -func=/tmp/gotest-skills/<pkg-slug>/cover.after.default.out)
   ```

   Each targeted function shows higher coverage; if not, the input misses the branch. No function drops; if one does, a new case shadows an old path.
3. Run `grep ' 0$'` on the `after` profile. Each block left is dead, unobservable, or undecided, and the report names it.
4. If you regenerated goldens (verification.md §7), compare every changed line with the expectation you wrote in advance. The flag goes after the package list: `go test ./<pkg>/... -rewrite`.
5. `git status --short` shows only `*_test.go` and `testdata/` changes.
6. Gate 2: `task test:pkg -- ./<pkg>/...`. Use gate 3 (`task test`) only if the change reached shared fixtures or helpers outside the package.

## Output to the user

Stay terse — the diff is the artifact. Put defects first, because they are worth more than the coverage:

- **Defects found**: each one with the input, the actual output, the expected output, where the expectation comes from, and the production path that can produce the input. State that no production code changed and no case asserts the wrong value.
- **Latent**: the gaps no production path reaches today, each with the precondition that keeps it unreachable.
- **Coverage delta**: `before% → after%` for each lane, and the per-function delta of each targeted function.
- **Cases added**: each case name and the behavior it pins, grouped by entry point. Mark the cases proven by §6.
- **Not reached**: targets you ranked but did not get to, in rank order. A big package often needs a second pass; this list is where it starts.
- **Left uncovered**: each block with its reason — dead (no entry point reaches it in either lane), unobservable, or undecided. Say so if one looks reachable in principle: the API may need a new entry point.
- **For review**: each regenerated golden or snapshot, and each undecided behavior you did not pin.
- **Proof**: lanes run, coverage diffs, gate 2 result.

## The go-test-* suite

This skill is step 3 of the pipeline that `/go-test-groom` runs on one package: `/go-test-reshape` → `/go-test-decouple` → `/go-test-coverage` → `/go-test-cleanup`. `/go-test-datadriven` owns the datadriven idioms. `/go-test-speedup`, `/go-test-determinism`, and `/go-test-lockin` sit outside the pipeline. A decouple run that removes an implementation-coupled test can lower incidental coverage; when this skill runs after it, compare with the decouple phase's `after` profile, not with the raw baseline.
