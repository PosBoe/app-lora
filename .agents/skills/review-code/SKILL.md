---
name: review-code
description: Review code of a repository for real defects in one broad pass — a Go package, a vue/ module, a set of files, a topic, or a change (a revision range, a branch, or a PR). Builds a model of the scope's invariants, mutation contracts, and lifecycle, hunts for correctness/concurrency/edge-case/design problems with the checklist of each territory the scope touches (territories/*.md — Go, Vue/TypeScript, SQL and migrations, the openapi seam, CI tooling, Ansible, Terraform, the application config), actively refutes each candidate, widens each survivor to every mirror that restates the same rule, and delivers a short severity-ranked list with concrete failure scenarios. At rest it reports every defect; in change mode it reports only what the change introduces or exposes, with a before/after causal trace. The AI review pipeline runs this skill in change mode. Reports only; it never fixes. Invoke ONLY when the user explicitly runs `/review-code`, explicitly names this skill, or when a review prompt tells a session to use it — do not infer it from generic phrasing like "audit this package", "find bugs here", "review this diff", or "is this sound" alone; treat those as ordinary requests instead.
---

# review-code

Review one scope — code at rest or a change — in a single broad pass. The deliverable is a short, prioritized list of real problems — not a tour of the code, not a refactoring proposal. Report findings and stop; fixing is a separate decision the user makes.

The skill has three parts. This file is the core: the steps every review takes. A **mode** file says what the scope is and which defects count. A **territory** file adds what one area of the tree needs: what to read, the probes, the hunt items, the repo contracts, and the mirrors.

`plan-ideal-refactor` shares this skill's vocabulary and its first steps: *scope*, *territory*, *consumer*, *mirror*, *repo contract*, and Step 0 Resolve → Step 1 Read → Step 2 Model. Keep the two aligned when you edit either.

## Territories

| Territory | Document | Covers |
| --- | --- | --- |
| `go` | [territories/go.md](territories/go.md) | `*.go`, `go.mod`, `go.sum` |
| `vue` | [territories/vue.md](territories/vue.md) | everything under `vue/` |
| `sql` | [territories/sql.md](territories/sql.md) | `*.sql`, `cmd/lora/migrations/`, `pkg/pgdb/`, Go files that build a query |
| `ci` | [territories/ci.md](territories/ci.md) | `.forgejo/`, `.github/`, `Taskfile*.yml`, `docker-compose.yml`, `vue/scripts/`, `go.mod` |
| `config` | [territories/config.md](territories/config.md) | `pkg/appconf/`, `config/*.yml` |

A territory name means the same slice of the tree in both skills. `plan-ideal-refactor` calls the `sql` territory `pg`, and it has no `ci` territory: a CI scope runs on its core file alone.

The **Covers** paths are paths of the repository under review. More than one repository uses this skill, so a territory whose paths a repository does not have never loads there.

**How territories combine.** Load every territory whose **Covers** list matches at least one file in scope. A file can match several territories; load all of them. Load no other territory. The user may force a territory with a prefix: `/review-code sql: pkg/pgdb`.

Each territory file uses fixed section names, each feeding one step:

| Section | Feeds |
| --- | --- |
| *Scope* | step 0 — what the territory covers and what it excludes |
| *Read* | step 1 — its documents, and how to list its surface, consumers, and variants |
| *Probes* | step 3 |
| *Hunt* | step 4 — the items particular to the territory |
| *Repo contracts* | step 4 — rules whose breach is always reported |
| *Mirrors* | steps 1 and 5 — where the territory's rules are restated |

## When not to apply

Apply when someone wants defects found in code: correctness bugs, races, leaks, violable invariants, footgun designs, documents that contradict the code — in code at rest or in a change. Do not apply when:

- The user wants a design-level restructuring plan — use `/plan-ideal-refactor`. (A review may *find* design flaws; it doesn't design the replacement.)
- The concern is test quality or coverage — use the test skills.
- The user already identified the bug and wants it fixed — just fix it.
- Someone else's findings need adjudicating — use `/fix-or-refute`.
- The user wants review comments posted on a PR — use `/code-review`.

## Step 0 — Resolve the scope, the mode, and the territories

The scope comes from the invocation argument (e.g. `/review-code pkg/appconf`, `/review-code vue/src/system`, `/review-code dev..HEAD`). If none was given, ask what to review before reading anything. Several scopes in one invocation get one pass each and one combined list; a defect that spans them is one finding.

**Pick the mode**, then read its file in full:

- The argument is a revision range (`base..head`, `base...head`), a branch, a PR, or says `change` → [change mode](modes/change.md).
- Anything else — a package, a directory, files, a topic → [rest mode](modes/rest.md).

**Pick the territories.** List the files in scope (for a change, `git diff --name-only <base> <head>`), and read each territory file the table above selects, in full.

**The argument may also carry context** from whoever invoked the skill — constraints ("persisted strings are frozen"), a known finding to confirm, or leads worth a look. Constraints bind. A handed-in finding or lead is a starting point, not the scope: confirm it, then widen it like any other finding (step 5). Past runs confirmed a handed-in finding in one file and missed the identical defect in its sibling.

**The invoking prompt wins.** When the prompt that invoked the skill names an answer format, a probe it forbids, or a limit of its environment ("the container has no databases", "never run race suites"), follow the prompt. Skip that probe, use the evidence the prompt offers in its place (for example CI records), and name what you skipped in the coverage. The prompt's answer format replaces this skill's output section; the content of each finding stays the same.

## Step 1 — Read

Read before hunting. Findings made while still reading are guesses; half of them dissolve once you know the whole scope. The mode file says how far the reading reaches; each territory's *Read* section names its extra documents and the commands that list a surface or its consumers.

1. **The documents first**: the `DESIGN.md`, `AGENTS.md`, `README.md`, `CONTEXT.md` (and `doc.go` in Go) of each directory in scope, and the `AGENTS.md` and `DESIGN.md` of its parent directories. A `DESIGN.md` records goals, frozen contracts and decisions. A behavior a decision record chose on purpose is not a defect by itself. Code that contradicts a record, or a record whose evidence (a test, a file) no longer exists, is a finding. A record says what someone chose, not that the choice is safe: a goal that promises a field or a flow says nothing about who may reach it, and it does not settle the questions step 4 asks.
2. **The public surface**: every export, prop, route, schema or job the scope offers to others. Read each doc comment against the code it describes — this is where doc drift lives (see step 3).
3. **The implementation**, top-down, then the tests. Tests document the intended invariants and pin behavior: a candidate bug on a path that a passing test exercises is usually a misreading — re-trace before keeping it. A test or a comment that *pins* the suspect behavior means someone chose it; read the commit that added it (`git log -S`) to learn why. Chosen is not the same as correct. When the pinned behavior breaks a limit or a promise stated somewhere else — a declared cap, an API doc, another document — two records disagree, and that is a finding: report the contradiction and cite both. A past run dropped a real bug because a comment said the overflow past a cap was deliberate; a later session fixed it, because the cap was the contract.
4. **Variants together**: build tags, deployment targets, app entries — whatever the territory names. Behavior that silently diverges between variants is a finding.
5. **Consumers and mirrors.** List the consumers of the surface — call sites, parse sites, jobs, tests — and read the few lines around each use, enough to know how the surface is actually used. You need them twice: in step 5 to refute "a caller could…" scenarios, and to find the **mirrors** — the places that restate one of the scope's rules. A repository often restates a rule: in a client-side form, across sibling kinds or files of one package, in each environment's variables, or in parallel `testdata/` corpora. Each territory lists its usual mirrors. Note each mirror now; step 5 checks every finding against them.

   A mirror can be **missing**, and grep finds only the mirrors that exist. So for each guard in scope — a refusal, a check, a lock, an escape or a normalization — list every path that reaches the state it protects or the value it encodes: each command or verb, each handler, each job, each writer and reader of the key. List them from the state, not from the diff or from the guard's call sites. A path on that list with no guard is a lead. For example, a check added to the create handler also belongs on update and import. An escape added to one encoder of a key also belongs on every other encoder whose output is compared with it.
6. **Other repositories.** The repository's `AGENTS.md` names the repositories that import it or that it deploys; When such a checkout sits beside this one (`../app-billing`) and the scope changes an exported Go signature, a wire shape, or an interface another repository calls, grep it: a consumer there can expose a defect this repository compiles clean.
7. **Recent churn**: `git log --oneline -20 -- <path>` — weight your attention on recently changed code.

## Step 2 — Model the current state

Write the model down before hunting — a few lines each, in a scratch note or in your working notes:

- **Responsibilities and public surface** — what the scope does and promises, and who consumes each part.
- **Invariants** — what must always hold (sorted, non-nil, registered exactly once, bounded by X, …), and which file enforces each.
- **Ownership and mutation** — who mutates what, and when; which values are shared vs owned.
- **Lifecycle** — construction → use → shutdown of the main types, components or jobs; who stops each goroutine, timer or listener, who closes each resource.
- **Concurrency** — which goroutines, requests or async callbacks touch which state, and what protects it; for each read-modify-write, what two concurrent callers produce.

The point of writing it down: bugs live where the code contradicts the model it implies. An explicit model turns "this feels off" into "line X violates invariant Y". An invariant with no enforcing file is itself a lead.

## Step 3 — Probes

Run the probes of each territory you loaded, unless the invoking prompt forbids one. They are cheap signal before manual hunting. Linter and type-checker output is excluded from the report (CI catches it) but points at code worth a closer look. A failure of an existing test, a race report, or a type error on code in scope is a confirmed finding — trace it and report it like any other. If a probe times out, say so in the close-out; do not retry it repeatedly.

**Doc drift probe.** A repository may check citations mechanically. In this repository, `docs_policy_test.go` fails a document that cites a dead path, and a Go comment that cites a dead `pkg.Symbol`, member or `[Name]` doc link. Such a check cannot check a bare name in prose ("the `TrainingPeriod` field", "checks minBreachPoints"), and it reads no TypeScript comment at all. Past reviews found stale doc comments that survived earlier reviews for exactly this reason. For each doc comment on the surface that names a field, function, prop, check or limit, confirm with `grep -w` that the scope still has it and still does what the comment says.

## Step 4 — Hunt

Work the scope against your model, in this priority order. Each territory's *Hunt* section adds the items particular to its area; hunt both lists.

- **Correctness**: wrong boundary conditions, null or nil dereferences, swallowed or misclassified errors, invariants violable through the public surface, state that goes stale or inconsistent across calls, validation that accepts a value the rest of the system cannot handle (a range that can never resolve, a default below a configured floor). Repo contracts count as correctness — each territory lists its own.
- **Trust boundaries**: for each entry point a caller outside the tenant or without a session can reach, name the credential that authorizes it and who holds that credential. An id or token handed to one party that authorizes an action for another, a check that scopes by one id but acts on another, and a secret returned by a list or create response are all findings. Ask this even where a design goal describes the flow — the goal records what the flow does, not who can drive it. Trace a secret by its **sinks**, not only by its source: list each place that others can read — an uploaded artifact, a log, a published comment, a response body — and each writer into it. A writer that copies the output of a process that can read the secret (an agent, a shell, a CLI in the same environment or container) can publish the secret. It does not matter that the code itself never passes the secret there.
- **Concurrency and async**: shared state without synchronization, leaks (every goroutine, timer, listener or subscription started needs a proven stop path; every resource an owner that closes it), a late result that overwrites a newer one, misuse of cancellation.
- **Resources on error paths**: follow every acquire — open, dial, lock, begin, subscribe — through each return that follows it, **including the acquire's own error return**. Some constructors return a live resource *together with* an error; when the error path does not close it, the resource leaks. If you cannot tell from the call site, read the dependency's source. A path you list as sound in the close-out must have had each of its error returns traced.
- **Edge cases**: empty/zero-value inputs, single-element collections, overflow, a default applied without the bounds check an explicit value gets, partially-constructed values escaping (returned or published before fully initialized).
- **Variant divergence**: the default vs the `invariants` build, fresh install vs upgrade, one environment's config vs another's — where the divergence is clearly unintentional.
- **Doc drift**: a comment, `README.md` or `DESIGN.md` that contradicts the code — one of the two is wrong. Also a test whose name, comment or corpus case claims a check the test does not make: it gives false assurance, which is different from a missing test.
- **Design flaws**: footgun defaults, unclear ownership/mutation contracts, exported mutable state, abstractions that leak or force consumers to know internals, two code paths kept in sync by hand, dead or unreachable code.
- **Performance**: only the clearly pathological — accidental O(n²), unbounded growth, allocation or a request in a hot loop, a render that refetches on every keystroke. No micro-optimizations.

## Step 5 — Refute and widen

**Refute.** For every candidate finding, try to kill it before it reaches the report:

- Re-read the failing path end to end. Check the consumers: is the "dangerous" input actually reachable? Check for guards elsewhere — validation upstream, a mutex at a higher layer, an init that establishes the invariant, a schema that rejects the value before the handler sees it.
- For a size, rate or growth claim, find the bound the system *enforces* — a validation max, a queue capacity, a config default — and cite the file that sets it. A typical or observed value refutes nothing; no enforced bound means the claim stands.
- Check the tests: if one exercises exactly this path and passes, your trace is probably wrong — find where.
- A plausible reason you infer is not a guard. A commit that touched the lines for another reason refutes nothing. Quote the guard, the test, or the record that kills a candidate.
- Refute the candidate at its strongest. A guard on one path, one source or one sink kills only that version. Before you drop the candidate, restate it for the other paths, sources and sinks from step 1 and check each one. For example, "the handler never logs the token" does not refute "the token reaches the log" when a library the handler calls logs the request.
- **Every finding needs a concrete failure scenario**: what input or sequence of calls leads to what wrong behavior. Can't construct one? Drop it, or reclassify as a design smell — whose "scenario" is the specific caller mistake the design invites, not a failing execution.
- When confirmation is cheap, buy it: write a throwaway test or scratch program, run it, delete it. A reproduced failure is worth ten traced ones.
- Label every finding: **CONFIRMED** (traced or reproduced end to end) or **PLAUSIBLE** (looks wrong but depends on caller behavior or environment you didn't verify).

Candidates you refuted that looked real get one line each in the close-out, with the guard that killed them — so the reader sees they were checked.

**Widen.** A surviving defect is rarely alone. Before reporting it, check every mirror from step 1 and the mirrors each territory lists: the sibling kinds and files, parallel `testdata/` corpora, a client form or spec that restates a server rule, the server validator a form restates, the same value in each environment, and other places grep turns up for the same rule or constant. In the first recorded run of the Go review, every correction a later session made was a finding the review had reported too narrowly — the same gap in a sibling kind, twice as many bad corpus cases, a Vue form with the identical rule. Report one finding with every site listed, not the first site found.

**Check the fix direction.** It gets less scrutiny than the finding, and it is the part that gets rejected. If it names a helper, confirm the helper exists and has production consumers (`grep -rn`), and that applying it restores the invariant you stated — not just half of it.

## Step 6 — Verify, then report

A reader checks a report by its citations, and one wrong citation discredits the finding it supports. As the last act before reporting:

- **Citations** are repo-relative (`core/attrorder/handler.go:105`, never a bare `handler.go:105`). Cite the line that holds the code, not the doc comment or the blank line above it. Save the draft report to a scratch file, and run [the citation check](scripts/check-citations.sh) on it from the repository root. It prints the line each citation points at, and it marks a missing file or a blank line. Fix each citation whose printed line does not show what the report claims. In change mode, run it a second time with the base revision as its second argument, and read only the base-side citations in that output. Past runs that re-checked by hand still shipped citations one to three lines off.
- **No counts of the tree.** They are the claims most often wrong, and the repository's `AGENTS.md` may forbid them in prose. List the sites, or give the grep that lists them.

**What to report.** Report every surviving finding a maintainer would fix, ranked by severity, even when it is minor; a repo-contract breach is always reported. Past runs filed a `(nil, nil)` return and an unbounded default "below the bar", and both were fixed as bugs the next day. The mode file narrows this: change mode reports only what the change introduces or exposes. Do not report: style or formatting, anything a linter or the type checker would catch, generated code (but do report a generator that emits a defect), missing tests, speculative "could be nicer" refactors, or documentation gaps where nothing is wrong. Do not manufacture findings to fill space.

Severity:

- **high** — wrong results, data loss or corruption, a hang, a leak, or a crash reachable in normal use.
- **medium** — reachable only through an unusual but valid input or configuration, or a footgun that has already misled a caller.
- **low** — doc drift, dead code, a design smell with a concrete invited mistake.

## Output

When the invoking prompt names an answer format, use it (step 0) and skip this section.

Rank findings most-severe first.

If the harness exposes a `ReportFindings` tool, report through it — one call, all verified findings, ranked — mapping each finding to its fields: `file`, `line`, `summary` (the one-sentence defect, with every widened site named), `short_summary` (the claim alone, **at most 60 characters** — the tool rejects longer), `failure_scenario`, `verdict` (CONFIRMED/PLAUSIBLE), `category` (correctness, trust, concurrency, edge-case, cross-build, doc-drift, design, performance). The tool has no severity field, so rank carries it. Don't duplicate the findings as text; the prose response then carries only, per finding number, its severity and the fix direction (one or two sentences — no patches), then the close-out below.

Otherwise, use exactly this structure:

```
## Findings (most severe first)

1. <file:line> — <one-sentence defect> [high|medium|low] [CONFIRMED|PLAUSIBLE] (<category>)
   Also at: <other file:line sites of the same defect, or "none — checked <mirrors>">
   Scenario: <input or call sequence → wrong behavior>
   Fix direction: <one or two sentences — no patch>

2. …
```

Close with a short paragraph:

- **Coverage** — the mode and the territories you loaded, what you read in full, what you only skimmed or skipped (required for a large scope), and which mirrors you checked.
- **Probes** — what ran, what it showed, and what you skipped and why.
- **Refuted** — one line per candidate that looked real, with the guard that killed it.
- **Sound** — what you traced and found clean (e.g. "shutdown path and error propagation traced clean").

A short or empty list then reads as evidence, not omission. If the scope is sound, say so plainly.
