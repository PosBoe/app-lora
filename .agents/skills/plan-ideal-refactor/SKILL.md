---
name: plan-ideal-refactor
description: Produce an evidence-backed refactoring plan for any scope in the repo — a Go package or topic, vue/ components and composables, the PostgreSQL schema of a set of tables, the Go/Vue API seam (openapi operations, handlers, composables), the AI review pipeline, an Ansible playbook and its roles, the application config (its sections, keys and descriptions), or tooling, CI, scripts, and cross-stack topics. One process for every scope; a territory document per stack (territories/*.md) states what the ideal end state is there, what to read, what to measure on the local lab, and how its migrations run. Reads the whole scope and everything that consumes it, sets genuinely different candidate designs against each other, specifies the winner as if written from scratch today, then plans how everything around it is refactored to fit and lays out a stepwise migration, asking the user on genuine forks. Plans in plan mode, presents the plan for approval, and carries it out once the user approves — nothing is edited before that. When the observable behavior itself is up for redesign (goals kept, contracts free to break) use redesign. Invoke ONLY when the user explicitly runs `/plan-ideal-refactor` or explicitly names this skill — do not infer it from generic phrasing like "plan a refactor", "redesign this", "rearchitect this", "improve this table", "clean up this endpoint", or "propose a better design" on their own; treat those as ordinary requests instead.
---

# plan-ideal-refactor

Prepare a refactoring plan for a chosen scope, and carry it out once the user approves it. The goal is the best design this scope can have, designed ideal-first: work out what it would look like written from scratch today, then reconcile everything around it with that design. The direction of reconciliation is fixed: the surrounding code, schema, jobs, and callers are refactored to fit the ideal design, not the ideal design bent to fit them. Designing the clean version is the easy half; working out how every consumer, wiring site, format, and stored row adapts to it is the plan's real substance, and most of the effort goes there.

`review-code` shares this skill's vocabulary and its first steps: *scope*, *territory*, *consumer*, *mirror*, *repo contract*, and Step 0 Resolve → Step 1 Read → Step 2 Model. Keep the two aligned when you edit either.

**What "ideal" means depends on the territory.** A Go package, a PostgreSQL table, and an HTTP operation are judged by different bars. This file holds the one process every plan follows. Each territory document under `territories/` holds that territory's definition of the ideal end state and the facts the process needs there: what to read, what to measure, what the design collides with, how its migrations run, and how it is verified.

**What is held.** Observable behavior — what a user, an API caller, or a PR author sees — and persisted data are preserved unless the user asked for a change. Everything else is fair game. Each territory states exactly what counts as observable and persisted there, and the price at which it may change (the `api` territory, for example, lets `/internal/v1` break through an approved ledger).

**Planning runs in plan mode; execution follows approval.** Enter plan mode with `EnterPlanMode` before step 0 reads anything, unless the session is in it already. Steps 0–6 write no repo file: no edits, no commits, no branch. The one sanctioned side effect is a territory's evidence lab — scratch databases and captures whose files live in the session scratchpad. Present the finished plan with `ExitPlanMode` and wait for the user. Their approval starts step 7. If the harness has no plan mode, still make no edit until the user approves the plan.

## Territories

| Territory | Document | Covers |
|---|---|---|
| `go` | [territories/go.md](territories/go.md) | Go source: a package, a set of files, or a topic across packages. |
| `vue` | [territories/vue.md](territories/vue.md) | `vue/`: components, composables, TypeScript modules, frontend topics. |
| `pg` | [territories/pg.md](territories/pg.md) | The PostgreSQL schema of a set of tables: keys, constraints, enums, indexes, jsonb. |
| `config` | [territories/config.md](territories/config.md) | The application config: its sections and keys, their descriptions, and how a config file loads. |

The **Covers** column describes scopes of the repository under review. More than one repository uses this skill, so a territory whose kind of code a repository does not have never loads there.

A territory name means the same slice of the tree in both skills. `review-code` calls the `pg` territory `sql`, and its `ci` territory holds the tooling.

A scope no territory covers — build tooling, Taskfiles, CI outside the review pipeline, scripts, docs — runs on this file alone. Its yardsticks are the universal ones in step 3, plus the house rules of the directories it touches.

**How territories combine.** A scope often lands in more than one. Load every territory the scope touches, and let each govern the part of the design that lands in it. Pick by where the *substance* of the change lives, not by which files it edits:

- The tables' schema is the substance → `pg`. The Go code around the tables is the substance and the schema stays put → `go`.
- What crosses the wire is the substance — which operations a view calls, what shape they carry, where the type is declared → `api`, plus `go` and `vue` for the code on each side. A Go or Vue scope that takes the wire as a fixed input does not load `api`.
- A cross-stack scope that is not the seam (how durations travel from config to the UI, error codes across API and UI) → each stack's territory, with the core yardsticks for any part outside them.

The user may force a territory with a prefix: `/plan-ideal-refactor pg: incidents incident_events`. Without one, resolve the territories in step 0 and state them at the top of the plan.

Read each loaded territory document in full after step 0 and before step 1. Its sections use fixed names, each feeding one place in the process; a territory omits the sections it has nothing to add to:

| Section | Feeds |
|---|---|
| *Scope* | step 0 — how the territory's anchors resolve |
| *Ideal end state*, *Palette* | step 3 — the goals or yardsticks candidates are judged by, and the moves to consider |
| *Read* | step 1 |
| *Current state*, *Problems* | step 2 — the territory's inventories and problem categories |
| *Candidates* | step 3 — required candidate shapes and radical moves |
| *Evidence lab* | the lab and what earns an experiment there |
| *Design spec* | step 4 — what "complete" means for the winner |
| *Collisions* | step 5 |
| *Migration*, *Verification* | steps 6 and 7 |
| *Forks* | the questions only the user can answer there |
| *Plan sections* | the plan template's additions and renames |
| *Self-check*, *Constraints*, *Closing* | the self-check, hard limits, and the run's closing report |

## When not to apply

- The user wants localized cleanup of recently changed code — use `/simplify`.
- The concern is Go tests, not production design — use the `go-test-*` skills (`/go-test-groom` for the whole pipeline).
- A Vue component is simply too big — use `/vue-split-components`; renames and folder moves without redesign — `/vue-organize-components`.
- The observable behavior itself may change — what a view shows, what a mechanism guarantees — use `/redesign`. This skill holds behavior.
- One slow query made faster without a schema change, one endpoint added, one field filled, one policy value tuned — ordinary work, not a design plan.
- Something is broken and the user wants it fixed, or wants bugs found — fix it, or use `/review-code`.
- The user already has a plan and asked for implementation — implement it instead.

This skill plans first and implements second. A user who wants the plan alone says so at approval time, and the run stops there.

## Asking the user when in doubt

Ask the user (with `AskUserQuestion`) whenever the doubt is genuinely theirs to resolve and the answer changes the plan. Ask *inline, when the doubt arises* — before writing the section it affects — never by deferring it to the closing list.

Two kinds of doubt, kept distinct:

- **Resolve it yourself** — design-merit calls: which pattern fits, how small a piece is, where a seam goes, naming by repo idiom, which candidate wins on the territory's goals. The territory documents are opinionated here; asking the user to adjudicate design is offloading your job. When two shapes are equal on design merits, break the tie yourself with migration cost and record it.
- **Ask the user** — forks that depend on information not recoverable from the code:
  - **Domain or product intent** — what a concept is *supposed* to mean, whether two things that look alike are one, whether a current behavior is intended or incidental.
  - **A trade-off only the user values** — flexibility for an anticipated future against the simplest shape today; cost against latency; a smaller safer cut against the full redesign.
  - **Acceptable compromises** — when reconciliation hits persisted data, a wire format, or a costly consumer migration, and the clean resolution has a real price.
  - **Reopening a binding record** — a `DESIGN.md` goal, frozen contract, or decision record the winning design contradicts (step 3).
  - **The territory's own forks** — each territory's *Forks* section lists the questions only the user can answer there (dirty data, retention, ledger rows, the review's cost budget, …).

How to ask: first a short brief in prose that grounds the question in what you read, in the scope's own vocabulary; then **one** question per `AskUserQuestion` call, with your recommended option first. Every option you do not recommend carries its reason — what it costs or risks — in its description; the reason is often the whole answer. Independent decisions are separate questions, even when they arrive together; never fold one decision into another's options. Several forks from one step are asked one after another, and none is held past the section it affects: a step 2 defect is asked before step 3 designs around it. If nothing meets the "changes the plan" bar, don't ask — proceed and record your resolution.

## Step 0 — Resolve the scope and the territories

The scope comes from the invocation argument, in one of three shapes:

- **Named files, a directory, a package, tables, or operations** — scope = those, plus enough of their surroundings to understand them in place. The territory's *Scope* section says how its anchors resolve (a table to its full DDL footprint in both deployments, an operation to its handler and composable).
- **A subsystem or feature area** ("the seed pipeline", "the channel forms") — map it to concrete files first.
- **A broad topic** ("how durations travel from config to the UI") — make it concrete before anything else. Grep the key terms: the prose term often differs from the identifiers (a document's "retention period" may be a `TTL` field and a `maxAge` parameter), so grep the term in documents and doc comments first, then the identifiers they name. Check `CONTEXT-MAP.md` for the bounded contexts the topic touches. Follow the data flow end to end, across process and language boundaries: producer → wire or file format → consumer.

Write down the resulting list — files, tables, or operations — and the loaded territories as the scope definition at the top of the plan. The reader must see exactly what was and wasn't considered.

If no argument was given, or the scope has two plausible readings that lead to different file sets, ask the user before reading anything. Otherwise resolve it yourself and record the resolution.

## Step 1 — Read everything first

Do not diagnose while reading; collect. Premature diagnosis is the main failure mode — a "problem" spotted on file two is often load-bearing by file ten. For a large scope, keep running notes in a scratch file instead of trusting recall.

Read, in this order, adding each territory's *Read* section where it slots in:

1. **Binding documents and house rules** of every directory in scope and its ancestors: `DESIGN.md`, `AGENTS.md`, `README.md`, `CONTEXT.md`, and `CONTEXT-MAP.md`. A `DESIGN.md` is binding — its goals, frozen contracts, and decision records are settled answers the design must honor or explicitly reopen (step 3). Read it before the code, so a "problem" the document records as a deliberate choice is recognized as one.
2. **The verification story** — how this code is built, tested, and checked. Record the concrete green-check commands now, from each territory's *Verification* section and, for territory-less parts, from the Taskfile targets, `package.json` scripts, and `.forgejo/workflows/`. Step 6 orders the migration around them.
3. **Every file in scope**, tests included. Test pain is design evidence: heavy mocks, deep fixtures, and assertions on internals all point at seams the design is missing.
4. **Variant dimensions together** — build tags (default vs `invariants`), feature flags, config switches. A design that only works under one variant is broken.
5. **Consumers and mirrors, found mechanically** — grep imports, exported symbols, endpoint paths, table names, config keys, file names, job ids. Classify them: production code, wiring and registration, tests. The far side of every format is a consumer too: the frontend parsing a response, the reader of a generated file, the job invoking a script.
   - **Test files are consumers**, including in-package tests and helpers that wrap a symbol. Past runs under-listed exactly these and found them only when the build broke mid-execution.
   - **Read around each use, not only the use** — the five or so lines before and after. When two or more consumers wrap the call in the same lines (the same branch, conversion, or fallback), that logic belongs to the scope, and it cannot be seen from inside it.
   - **Mirrors** are the places that restate one of the scope's rules: a Vue form that repeats a Go validation rule, sibling kinds or files of one package that repeat a check, parallel `testdata/` corpora per kind. The compiler does not connect a mirror to its rule, so a moved or changed rule leaves it behind. Grep the rule's constant and its error text, and list each mirror beside the consumers.
   - **How much to list.** Read every distinct usage pattern; when consumers number in the dozens, read one representative site per pattern, but still list every file. For a change the compiler points at (a rename, a changed parameter or return type), the plan may give the grep and the file list instead of every line. A change the compiler cannot see (a changed meaning, a new zero-value behavior) is listed per site.
6. **Other repositories.** The repository's `AGENTS.md` names the repositories that import it or that it deploys; When such a checkout sits beside this one (`../app-billing`), grep it like the rest: its uses are consumers the plan must name, though this run cannot compile them.
7. `git log --oneline -20 -- <paths>` — recent shape-changing work explains why the code looks the way it does and reveals in-flight refactors the plan must not collide with.

## Step 2 — Model the current state

Establish, with evidence:

- **Responsibilities** — what the scope actually does. Several unrelated ones is a finding; so is the same responsibility implemented differently in two places.
- **Public surface and its users** — each exported symbol, prop, operation, column, input, config key, or file format mapped to who consumes it. Surface nothing consumes is a finding, and its default resolution is deletion. So is logic several consumers write around the surface in the same way: name each copy by `file:line`.
- **Dependency direction** — what the scope depends on and who depends on it; any direction that feels inverted (domain logic depending on transport, storage, or tooling; two sides of a boundary each reaching into the other).
- **Invariants and lifecycle** — what must always hold, and which file enforces each; who starts and who stops each goroutine, timer, listener, or job, and who closes each resource. An invariant with no enforcing file is a lead for a problem or a defect.
- **Concurrency** — for each read-modify-write and each multi-statement write in scope (a check then an insert, a delete then a re-insert, a cache then its source), what two concurrent callers produce. A result neither caller asked for is a problem; where the territory has a lab, reproduce it there.
- **The territory's inventories** — its *Current state* section names what else to establish (the query inventory of a schema, the calls per view of a seam, the write semantics of a table).
- **Design problems**, numbered P1…Pn so later sections can cite them. Each needs a category (the territory's *Problems* section lists the common ones), `file:line` evidence, and a concrete consequence — what it makes hard, risky, slow, or duplicated today. Where the territory states goals, name the goal each problem violates. A problem without a consequence is a style preference; leave it out. Take each `file:line` from a `grep -n` or a read made while writing the problem, not from recall — a citation ten lines off sends the executor to the wrong code.
- **Try to refute each problem before you number it.** Two pieces of code that look alike are often two rules that share a shape: a past run merged two "duplicate" gates that applied different policies, and only a test caught it during execution. For every claim of duplication, dead code, or needless indirection, look for (a) a test that asserts the two paths differ, (b) the commit that introduced the difference (`git log -S '<symbol>'`, `git blame`), (c) a `DESIGN.md` decision or a comment that names it, and (d) a measurement that shows why it exists. Quote what the evidence says; a commit that touched the lines for another reason refutes nothing, and a plausible reason you infer is not evidence. A "kept for speed" reason is a claim: measure it, or keep the problem. If the difference is deliberate, drop the problem or restate it as what it is (often a naming problem: two rules that deserve two names). Record each refuted suspect in one line, with the evidence that refuted it. A suspect you set aside on reasoning alone is not refuted: list it as unverified, or keep it as a problem. The same holds the other way: a claim that code never runs — a dead branch, a filter that never removes anything, a guard nothing can reach — often rests on another piece's behavior (a validation rule, a lifecycle, a library default). Check that behavior in its source or with a test before the plan deletes the code; while it is unchecked, the deletion waits behind a test that proves it.
- **Defects found** — real bugs noticed while reading, kept apart from design problems. Each carries a concrete failure scenario (what input or call sequence leads to what wrong behavior) and a label: **CONFIRMED** when traced or reproduced end to end, **PLAUSIBLE** when it depends on caller behavior or an environment you did not verify — the labels `review-code` uses. Widen each one to its mirrors before you record it: the same gap usually sits in a sibling kind or in the Vue form that restates the rule. Try to refute each one the same way: a comment, test, or `DESIGN.md` record that states the behavior is intended makes it a decision, not a bug — present it as a decision to reopen, quoting the record. A defect fix changes observable behavior, which this skill otherwise holds. When the fix is local, inside the scope, and has one obvious right answer that a test can pin, do not ask: list it as a fix the plan makes, and let approval of the plan cover it. Ask only when the fix reaches beyond the scope, changes a contract or a stated promise, or has more than one reasonable answer; recommend a follow-up when it needs a decision the code cannot settle. An accepted fix becomes its own `[behavior]` migration step, ordered before the restructures that touch the same code, with a test that fails first and fails cleanly (bound a hang inside the test, not with a timeout around a goroutine that keeps running). It is never folded into a restructure step.

## Step 3 — Candidates

Design from the domain, not by rearranging the current code. The inputs are what the scope is *for* (the `CONTEXT.md` glossary names the concepts), the inventories from step 2, and the territory's *Ideal end state*.

**Favor the best greenfield design.** The cleanest from-scratch shape is the default. Any pull back toward the current structure carries the burden of proof, and only a hard external constraint earns it: persisted data or a wire format that must still load, a frozen contract, or a user-declared scope limit. Migration cost never does — it is reconciliation work for step 5. If you catch yourself keeping a shape "because consumers expect it" or "to keep the diff small", you are rearranging, not designing.

**Compete before you commit.** The current code is the strongest anchor in the room. A plan that concludes "rebuilt from scratch, it would keep its shape" may be right, but it must win that result in the open. Sketch genuinely different whole-scope candidates, a few lines each (layout, who owns what, the central type, table, or operation):

- **The repair** — today's structure with every P-problem fixed in place. It exists to be beaten, and it wins only on the goals, never on the size of its diff.
- **The redesign** — the boundaries redrawn from the domain: pieces merged or split, ownership moved. What would you create today if this scope did not exist? Include it when the domain suggests an ownership that differs from both others; the territory may require it.
- **The radical** — at least one candidate that questions an assumption the others share: merge or delete a layer, invert who owns a responsibility, change what a row or an operation *is*, replace an object with data, move the logic to the caller that already has the context. Most radicals lose. A radical that loses must lose to a named goal or a hard constraint, not to vague discomfort.

"Genuinely different" means a different answer to who owns what; two spellings of one layout are one candidate. The territory's *Candidates* section may add rules. For a small scope the radical sketch may be two lines — its job is to make the winner earn its place.

**Judge** by: the hard constraints first; then the territory's goals in its stated order, or the P-numbers each candidate removes where the territory gives yardsticks rather than goals; then less mechanism (fewer pieces, exports, operations, tables, indirections); only then migration cost. Two candidates are equal on design merits when they serve the same goals and remove the same P-numbers with about the same mechanism. When candidates tie on paper and the territory has an evidence lab, the lab breaks the tie — never incumbency. Record the tournament: per candidate, what it wins and loses, and the sentence that decided it.

**`DESIGN.md` binds the design.** A frozen contract holds like persisted data. A goal or decision record the winner contradicts is not quietly overruled: ask the user whether to reopen it, and if they agree, the migration updates the document in the same change (the `design-doc` skill owns its format). When a record's stated reason is a prediction the lab can measure, a prototype and its number are the way to reopen it, not an argument. A candidate that honors every record wins ties. Separate a contradiction from a stale reference: a record whose only stale words are names (a moved file, a renamed test) needs no question — it is a document edit in step 5. Everything else is a contradiction and is asked, even when no caller uses the stated behavior today — "unused" is a finding to show the user, not a license to drop a promise.

If a fork here turns on intent the code doesn't settle, ask before committing the design — don't design both variants speculatively.

**Universal yardsticks** — every territory's bar sits on these:

- Each piece has one nameable job and one reason to change; high-level entry points read before their helpers.
- Interfaces are defined by the consumer and stay small; expose only what someone outside actually uses.
- Prefer plain functions and data over indirection; composition over hierarchies.
- Knowledge lives in one place: a format, a constant, a formula, a rule is defined once and consumed everywhere else — never re-derived on both sides of a boundary. Before planning a derivation on one side, check what the other side already computes.
- Smaller is a valid outcome. Deletion and merging are first-class design moves.
- Repo idiom is part of "ideal". Greenfield means designing the *structure* from scratch, not importing a framework or idiom foreign to this repo. A convention the scope's own `README.md` or `AGENTS.md` states overrides a default of this skill or its territory; say so where it applies. A *design reason* written in a `README.md` or a comment is different: it is evidence to weigh like any other, not a record that binds. Only `DESIGN.md` binds, and a candidate that removes a README's reason says why that reason no longer holds.

## Step 4 — The ideal design, in full

Specify the winner concretely enough to implement from without reopening design questions. Losers stay at sketch level. Spell out:

- **Layout** — every file, package, module, table, operation, or job in the ideal version, each with a one-line purpose and its placement rationale.
- **The full surface** — in the shape it would take in the final code, as the territory's *Design spec* section defines it: Go signatures in `go doc` form, component props and composable signatures, complete DDL, complete openapi YAML, action manifests. Sketch-level bodies are fine; the surface is not optional.
- **Boundary contracts** — for each producer/consumer boundary: the exact format, which side owns it, and how the other side consumes it.
- **Data flow** — walk the two or three main scenarios end to end through the new design, across every boundary: which piece is invoked with what, what it hands to whom, where state lives, where errors surface. The territory says what else a walk must show (the index each query uses, the constraint that catches each race, the rendered state of each view). If a scenario can't be walked cleanly, the design isn't done.
- **Ownership of every current responsibility** — each responsibility, column, field, operation, or input from step 2 mapped to its new home, or explicitly to deletion. Nothing silently falls on the floor.
- **Cross-cutting story** where the scope has one — errors, concurrency and lifecycle, idempotence of scripts and jobs, failure modes.
- **Rationale** — every element cites the P-numbers it removes, the territory goal it serves, and any `DESIGN.md` goal or decision it touches. An element citing nothing gets cut: it is there for symmetry or speculation. Where a named pattern applies, say why it fits *this domain* — what varies, what stays fixed.
- **Alternatives rejected** — for each major decision the candidates did not already settle, the one serious alternative and the sentence that killed it.

## Evidence lab

Some claims are empirical: which index the planner picks, how many rows a sort key reads, how many bytes a codec saves, how many requests a view makes, whether an unfamiliar feature behaves as the plan assumes. A territory with an *Evidence lab* section says where its lab is and what earns an experiment there. The rules hold for every lab:

- **Measure what the plan rests on.** A performance, planner, size, or runtime claim cites an experiment with its numbers, or an explicit "untested because …". Never just confidence. Anything the repo does not already use gets proved to work at all before the design leans on it — including its error shape, which reconciliation code must handle.
- **No theater.** A pure correctness claim (a unique constraint enforces at-most-one) or a pure reading claim (no template reads this field) needs no experiment.
- **Measure what you recommend.** A number taken on a different variant — another tier length, sort key, or batch size — does not stand for the winner. Re-measure it on the winner's configuration, or label it with the variant it came from.
- **Scale honestly.** Load data at a scale where the effect can show; a 1k-row table proves nothing about a planner. Compare candidates side by side on the same box and the same data: absolute numbers from a dev machine are indicative, comparisons are the signal. Name what the lab cannot show (replication, production scale) and carry it as unverified.
- **Leave no trace, and do not take the lab down.** Work in scratch databases, never the dev or shared test databases in place, and drop them when done. The local servers are shared with other sessions: bound each experiment's memory (generated data in batches, `SET work_mem` per session) — one unbounded insert has already made the out-of-memory killer stop a shared ClickHouse. Driver scripts and captures live in the session scratchpad, not the repo.

## Step 5 — Reconciliation

This is the bulk of the plan. "Most detailed" is relative to the change, not a length target: when the repair wins and the plan is a few local fixes, a short reconciliation is the honest result. The ideal design is now fixed; confront it with every collision surface — the common ones below, plus each territory's *Collisions* section:

- **Consumers** — every usage pattern from step 1: call, parse, and invocation sites, test files included.
- **Wiring and registration** — fx providers and invokes, route tables, CI job references, task dependencies, generated-file consumers.
- **Config** — YAML options (a rename follows the deprecated-field rule in `pkg/appconf/AGENTS.md`), env vars, flags.
- **Persisted and wire formats** — rows in Postgres (jsonb shapes included), HTTP responses the frontend parses, saved URLs and stored user state. Watch for wire names derived from code names: a type or field rename can change a JSON key, a schema name sent to a provider, or a cache fingerprint. Grep the name as a string, not only as a symbol, including `*.yaml`, `*.sql`, and `vue/src/`.
- **Cross-boundary contracts** — a format change needs both sides sequenced: who ships first, what tolerates both shapes in between.
- **Variants** — build tags, feature flags, config switches.
- **Mirrors** — every place from step 1 that restates a rule the design moves or changes. A mirror left behind compiles and passes its own tests, and then disagrees with its rule.
- **Documents** — every `README.md`, `AGENTS.md`, `CONTEXT.md`, `DESIGN.md`, `doc.go`, and comment that names a symbol or path the plan moves, renames, unexports, or deletes. A repository may check citations mechanically. In this repository, `docs_policy_test.go` fails a cited path or `pkg.Symbol` the tree no longer holds, in documents, Go comments, and `.sql` comments alike — but such a check tests only that a name exists, so prose that still describes the old shape passes it; read each hit. Grep the whole repository, not the scope — a name is most often cited from a parent's or a neighbor's `README.md`: `grep -rn --include='*.md' --include='*.go' --include='*.sql' --include='*.vue' --include='*.ts' --include='*.yaml' -w '<Name>' . | grep -v -e node_modules`, excluding vendored trees, once per changed name — a table, a type, or a route is named in `.vue`, `.ts`, and `.yaml` comments too; for a short or common name, grep the qualified forms. An edited sentence follows the documentation rules in the repository's `AGENTS.md` (here: Simplified Technical English, and no count of what the tree holds). Every past run ended with document edits its plan did not name.
- **Other repositories** — a repository that imports this one, or that this one deploys, cannot be compiled from here; Every exported signature, and every wire or schema shape it reads, that the plan changes is a follow-up for it: name the sites the step 1 grep found, or, without a checkout, say the check is owed. The package's `DESIGN.md` may state that nothing outside this repository imports it.

For each collision, the default resolution is a concrete change to the consumer, wiring, or format that lets the ideal design stand unchanged. This section reads as a per-consumer refactoring plan — which files, which sites, what they become — not a list of risks. Only when adaptation is genuinely impossible record a compromise with a one-sentence justification. The empty compromise list is the goal. Persisted data is the one constraint that usually earns a legitimate compromise (old rows, saved URLs, and in-flight messages must still load); a compromise that exists only to avoid editing a consumer, however many consumers, is not legitimate.

When a collision forces a real trade-off whose acceptability you can't judge from the code, surface it to the user with the resolution you recommend and its cost.

## Step 6 — Migration plan

Order step 5's reconciliation, plus building the ideal scope itself, into steps where **each step leaves every affected territory green** under the commands recorded in step 1. The territory's *Migration* section adds its mechanics (atomic seam commits, repair-then-constrain, new-table-backfill-swap, phased schema migrations).

- **Strangler order** — introduce the new seam, migrate consumers, delete the old — over big-bang rewrites. A temporary bridge between steps (an adapter, a forwarding function, a deprecated alias) is fine when the plan names it and the step that deletes it.
- **Tolerant reader first** for a format change across a boundary that deploys separately: the consumer accepts both shapes, the producer switches, the tolerance goes.
- **Cheapest risk first** — pure moves, renames, and deletions of dead surface; then new seams; then consumer migrations; then wire, config, and schema changes.
- **Every step compiles on its own**, test files included: a step that changes a signature carries the edits to every test caller and helper that uses it.
- **Size** — one reviewable commit per step. An ambitious design does not license an ambitious rollout: the boldest design still arrives as a sequence of boring, individually safe steps.

Flag on every step:

- **Kind** — `[restructure]` is a pure move, rename, or reshape; `[behavior]` is anything an outside observer could notice, present only when the user asked for it or accepted a defect fix; `[test]` adds or changes only tests. A change to what a reader sees — resolution, retention, rounding, ordering, an error status, an outage window such as a paused job — is `[behavior]` even when only the schema moves. A `[restructure]` label is a claim: check it against the edge cases existing tests pin — zero values, empty inputs, range limits, nil — and relabel the step, or change the design, where the new shape answers one differently. The territory may use its own kinds (`[schema-only]`, `[contract|cutover|removal]`).
- **Consumers** changed with it.
- **Coverage** that guards it, and gaps. A step guarded by nothing gets its missing test as an earlier step — never migrate blind.
- **Migrations and config** it needs; a migration version uses the actual current timestamp.
- **Documents** it edits, from step 5.
- **The territory's flags** — benchmark checks, ledger rows, stale-tab views, corpus re-recordings, cost effects.

## Constraints

- No new dependencies in any territory: `go.mod`, `vue/package.json`, the engine's standard library, and the installed CLI tools as they stand are the toolbox.
- Public surface may change freely when the same plan updates its consumers.
- No edits before approval — write no code, SQL, YAML, or migration into the repo, not even an "illustrative" snippet. Signatures, DDL, and schema sketches inside the plan text are required. After approval, the edits are the approved migration steps and nothing else.
- Each territory's *Constraints* add its hard limits (generated files never edited by hand, `pkg/pg` and `cmd/apigen` left alone).

## Verify the plan's facts before presenting

A reviewer checks a plan by its claims, and one false load-bearing claim sinks it. Past plans lost to citations 200 lines off, to a "nothing reads this" that a lazy-mounted panel read, and to a file count that was simply wrong. So, as the last act before presenting:

- **Citations** are repo-relative (`core/attrorder/handler.go:105`, never a bare `handler.go:105`). Re-open every cited line — a loop of `sed -n '<line>p' <path>` over the plan's citations is enough — and fix each one that drifted.
- **Load-bearing claims** — each defect, each "dead" or "unused" or "only caller" claim, each claimed duplicate — get one fresh grep or read now, not trusted from step 2. A claim that no longer holds changes the design, not only the sentence.
- **No counts of the tree.** They are the claims most often wrong, and the repository's `AGENTS.md` may forbid them in prose. List the sites, or give the grep that lists them. A number from a measurement (a lab result, a benchmark) keeps its command beside it.

## Self-check before presenting

Walk the draft once against this list, and each loaded territory's *Self-check*; fix what fails:

- The scope definition is explicit: the reader can list what was considered, on every side of every boundary, and which territories governed.
- Every problem has fresh `file:line` evidence, a consequence, and survived a refutation attempt; refuted suspects are listed; defects are apart from design problems, each with its scenario, its label, its widened sites, and the user's decision.
- The candidates include a radical that differs in who owns what, and the winner beat it on goals or mechanism, never on migration cost.
- No `DESIGN.md` goal, frozen contract, or decision record in scope is contradicted without the user's word.
- The winner has a full surface, layout, walked data flows, boundary contracts with one owner each, and a home for every current responsibility — no "details TBD".
- Every design element cites a problem or a goal; anything citing none was cut. No shape was kept "because consumers expect it".
- Every empirical claim cites a measurement or an explicit "untested because …"; every scratch database is dropped.
- Every fork that turned on intent or a user-owned trade-off was asked inline, one question at a time; design-merit calls were resolved without asking.
- Reconciliation resolves every collision to named files and sites; every renamed or deleted name was grepped across documents, comments, and `.sql`; every mirror of a moved rule has a step; every compromise has a justification.
- Every migration step leaves every affected territory green; behavior steps are flagged.
- The plan is executable by someone who has not read the code: paths, symbols, tables, operations, and consumer lists are concrete, not "update the relevant callers".

## Present the plan for approval

Present the plan with `ExitPlanMode`, in this structure. A loaded territory's *Plan sections* adds or replaces the subsections it names; a territory-less scope uses the skeleton as is.

```
# Refactoring plan: <scope>

## 0. Scope
<territories loaded; the files, tables, or operations considered, on every side of every boundary; how a topic was mapped>

## 1. Current state
### Responsibilities
### Public surface and its users
<territory inventories>
### Design problems
P1 (<category>[, violates <goal>]) — <one-line statement>. Evidence: <file:line>. Consequence: <what it costs today>.
### Refuted suspects — suspects the evidence showed deliberate, one line each with the test, commit, or record
### Defects found — real bugs, kept apart from design; the user's decision on each

## 2. Ideal design
### Candidates    — repair, redesign where warranted, radical: a few lines each; what each wins and loses; the winner and why
### Layout        — one-line purpose each, placement rationale
### Surface       — as it would appear in the final code
### Boundary contracts — per boundary: format, owning side, consuming side
### Data flow     — main scenarios walked end to end
### Experiments   — what the lab measured, at what scale, the numbers; claims left untested and why (omit if none)
### Rationale     — per element: P-numbers removed, goals served, DESIGN.md IDs touched
### Alternatives rejected

## 3. Reconciliation — the most detailed section of the plan
### Consumer and wiring changes — per collision: the files and sites adapted, and what they become
### Documents     — each document and comment edit, by file
### Compromises   — goal: empty; each with a one-sentence justification
### Follow-ups for other repositories — or why none apply

## 4. Migration
Step 1 — <name>. [<kind>] [consumers: <list|none>] Coverage: <what guards it>. Gaps: <missing tests>. Docs: <files|none>. <territory flags>
Step 2 — …
```

End with the open questions that remain the user's to decide (naming, scope cuts, sequencing), plus any standing question a territory asks at approval. A fork that would change the design or the migration order was already asked inline — this list is the low-stakes remainder, not a dumping ground. If there are none, say so.

The approval starts step 7. If the user rejects the plan or changes its shape, revise it and present it again — do not edit under a rejected plan.

## Step 7 — Execute the approved plan

Build only what the approved plan describes, in its migration order. A change the plan does not name needs the user's word first.

1. If on `dev`, branch first: `git switch -c refactor/<topic> dev`.
2. Implement one migration step at a time, and commit each on its own — a Conventional Commit whose message names the step, for example `refactor(system): give the health check its own seam`.
3. Check each step cheaply, with the per-step checks of every territory it touches (compilers, type checks, linters, generators). Skip the slow lanes — database-backed tests and snapshot re-recording belong in one pass at the end.
4. Verify fully once, after the last step, with the final checks of every affected territory. Include the repository's doc-policy check whenever a document or comment changed (here, the root package `.`, where `docs_policy_test.go` lives). Re-record datadriven corpora (`-rewrite`, package list first) and snapshots (`UPDATE_SNAPS=true`) in this pass.
5. If a step shows the plan is wrong — a consumer the inventory missed, dirtier data than assumed, a benchmark past its threshold — stop before improvising: report what the code showed, propose the amended step, and get the user's word. Do not silently redesign mid-execution.

Close the run with the commits made, the verification output, each territory's closing items, and every deviation from the approved plan.
