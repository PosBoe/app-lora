---
name: design-doc
description: Distill a repository scope's purpose, solution, tiered goals, non-goals and frozen contracts into its binding DESIGN.md, asking the user about uncertain tiers. `trim` mode instead audits one document's decision records for residue and rewords them with the user's approval. Run it before /redesign when no goal list covers the scope. Invoke only when the user runs `/design-doc`, names this skill, or /redesign directs it — not for generic requests to document goals or update DESIGN.md.
---

# design-doc

Work out what the code in a scope is *for* — the outcomes someone would miss if it were gone — and how its current design serves that purpose. Persist both in the anchor `DESIGN.md`: **Motivation**, **Solution**, the tiered goals, the non-goals and the frozen contracts.

The goal list is a contract. `/redesign` later holds every primary goal fixed, prices every secondary trade, and treats any observable behavior no goal covers as incidental and free to change. So completeness is this skill's core duty: **a goal missed here is a behavior lost later.**

**Two modes.** The default run is the steps below. `/design-doc trim <scope>` leaves the goals alone and audits the decision records instead; [TRIM.md](./TRIM.md) holds its procedure. Run trim only when the invocation says `trim`.

**Where the rules live.** [DESIGN-FORMAT.md](./DESIGN-FORMAT.md) is the one home of the file's shape — the marker, section order, the goal, non-goal and frozen-contract entries, the `Need:` kinds, citations, records and the evidence file. No test enforces it yet, so check the shape by reading. Read it before Step 2; this file does not restate it, and holds only the procedure and the judgment the format cannot express.

## Step 0 — Resolve the scope and the anchor

Follow [SCOPE.md](./SCOPE.md): resolve the scope to files, ask if it has two readings, and find the binding set with the `Scope:` grep. List the set in your notes, marked by relation.

The **anchor** is the document this run writes. When no document binds exactly this scope, the run creates one. File it where DESIGN-FORMAT.md § Scope says: in the directory that holds every path it binds, or in `docs/design/` under a topic name when no directory does. Put every path the scope covers on its `Scope:` line, whichever stack each path is in.

**The anchor holds only what is true of the whole scope.** A goal that belongs to one package below the anchor stays in that package's document. When the anchor and a descendant would state the same goal, the descendant keeps it and the anchor drops it. One goal has one home, so `/redesign` never holds two copies that drift apart.

## Step 1 — Read the scope and its world

Collect, don't diagnose. A mechanism that looks baroque on first read often carries a requirement you have not met yet. For a large scope, keep running notes in a scratch file instead of trusting recall.

1. **House rules and domain language** — the nearest `AGENTS.md`/`CLAUDE.md` and `README.md` of every directory in scope, `CONTEXT-MAP.md`, each modeled context's `CONTEXT.md`, every document in the binding set (the anchor's above all), and `COMPROMISE:` comments.
2. **Every file in scope, tests and docs included.** Tests show which behaviors someone cared enough to pin. User-facing docs and changelog entries show what was promised to users.
3. **The data's whole journey** — for anything stored or transported: the Postgres schema and migrations (`cmd/lora/migrations/`), the HTTP or queue contracts it crosses, and the frontend code that parses and renders it.
4. **Consumers, found mechanically** — grep imports, endpoint paths, config keys and emitted events. Classify each as production, wiring or test. A production consumer is primary-tier evidence.
5. **Variant dimensions** — build tags (`pkg/featflag/`), config switches, and any deployment split the product grows. A goal that exists in only one variant says so.
6. `git log --oneline -20 -- <paths>` — recent shape-changing work shows what someone recently cared about.

## Step 2 — Distill Motivation, Solution, goals and frozen contracts

- **Motivation** — the problem the scope exists to solve and the forces that shape it, in a short paragraph. Write the statement of need that would survive a rewrite, not a summary of the code.
- **Solution** — a few present-tense paragraphs: the shape of the approach, its main moving parts, and the one or two structural choices that make it work. Leave contracts, rules and file maps to `README.md`/`doc.go`, and rationale to the records.
- **Goals** — for each mechanism, ask what user, product or system outcome it exists to produce, and write that outcome. Write "alert state survives a restart", not "alerts are stored in Postgres".
  - **Test each sentence**: name the observation that would show a broken design. If the sentence does not hold that observation, rewrite it until it does. A sentence that names only a mechanism fails — write what a consumer can do because the mechanism exists ("a test run gets a private keyspace with no change to production wiring", not "the client is a seam").
  - **Find the need behind a test.** A test proves today's behavior exists. It does not show that a design chosen from scratch must keep it. When a test is the only thing behind a goal, look for the obligation it serves — the caller, the contract, the incident it was written after — and record that as the `Need:`. When none turns up, the goal is a question for Step 3, not a primary goal.
  - **A promise binds when someone outside the scope reads it.** A README or doc comment that only describes the scope's own behavior is not by itself a `[contract]` need — for a small package it would make every tested behavior primary. It counts when a consumer outside the scope is written against it, or it is the published interface of a library others import; name that reader. Otherwise look for the consumer, or ask.
  - **Evidence you cannot find is a fact.** Write `Evidence: none` rather than a weak citation; the next reader should see that nothing checks it.
  - **Do not invent a budget.** A secondary goal with no measured figure says so in `Measure:`. An invented figure is a fiction a later run will price a trade against.
- **Frozen contracts** — sweep the schema, the wire and the consumers from Step 1 for shapes an outside party depends on: a wire key an external client reads, an enum value in persisted rows, a migration version, a route or field an OpenAPI spec pins. A shape nobody outside the scope reads is not frozen, however old it is. When the sweep finds none, still write the section: its `None:` line says what the sweep read.

## Step 3 — Tier the goals, and ask about the doubtful ones

The tiers have operational meaning for `/redesign`: a primary goal disqualifies a design that drops it, a secondary goal is traded at the price its `Loss:` states, and a non-goal's kind decides what a design may add. DESIGN-FORMAT.md § Goals states each tier's required lines.

- **The need decides the tier, never the proof.** A goal whose only need is `[test]` or `[maintenance]` is not primary; the test fails one.
- **Type a non-goal by one question**: should a design built from scratch refuse this capability? If yes, `[excluded]`. If the honest answer is "the current code cannot do it", it is `[no guarantee]` — `[excluded]` would tell a candidate to cut something it gets for nothing.
- **Read the whole list once for overlap.** Two entries that hold one observable are one goal written twice. One observable in two tiers is a contradiction: a run would disqualify a candidate for it and price it as a trade at once. Decide which entry owns the observable, and cut it from the other.
- **Sweep for cross-cutting properties** — durability, isolation, multi-tenancy — that no single file states but many uphold. The list is closed, so these are the goals most often missed.
- **A magnitude needs a number.** A primary goal that turns on a magnitude states it. Where nobody has measured one, the entry is secondary and its `Measure:` says no budget is set.

**When in doubt, ask.** The doubts that matter are tier placement and goal-versus-incident: a behavior you cannot trace to a consumer, test or doc, yet which looks deliberate. Batch every doubt into one `AskUserQuestion` call; each option states the behavior and the candidate tiers, with your recommended reading first. When the evidence is unambiguous, do not ask — cite it and proceed. An answer that confirms a goal becomes a `[user]` need.

## Step 4 — Write the anchor

The default run writes one file, the anchor, and changes no code. Write it to the full shape DESIGN-FORMAT.md describes. The file either leaves the run in that shape or does not change — do not leave it half-written.

- **A new file** holds the marker, the `Scope:` line, Motivation, Solution, the three tiers and the frozen contracts. A goal list earns the file alone; it needs no decision record.
- **An existing file** is updated in place. Reword Motivation and Solution to what is true now, and add, reword or retire goals. Retiring, promoting or demoting a goal that the file already lists goes through the user.
- **Ids are permanent.** A new goal takes the next unused `G` number in the file, counting both tiers. A retired id is never reused, and a list is never renumbered to close a gap: decision records, `/redesign` reports and earlier sessions cite the ids.
- **Decision records are not this mode's prose.** Leave every record's text byte for byte; only trim mode rewords one. This mode may add the links around a record, because they are links and not prose: its `D` id, its `Serves:` and `Trades:` lines (read `Trades:` out of the record's own **Consequences.**, never invent it), and its line in the `## Decisions` index. `Serves: none` is a signal to look at the goal list again.
- **Report records that look dead, and delete none.** Note each record whose losing alternative is gone — the code, the dependency or the use case it needs — and each that names no cost or no observable trigger. List them as delete candidates with the evidence. Deletion is the user's decision.
- **A file that is not a design document** — a style guide, a token spec, a product brief named `DESIGN.md` — gets no marker. Say so, and propose moving it out of the anchor path, as the visual design system moved to `vue/DESIGN-SYSTEM.md`.

## Step 5 — Prove it and commit

1. Run `go test -run 'TestDoc' .` from the repo root. It checks every cited path. Then read the anchor once against DESIGN-FORMAT.md for the shape, every `Serves:`/`Trades:` reference, and the placement, because no test checks them yet.
2. Fix each failure the anchor owns. A `TestDoc` failure in a file this run did not change is not this run's — a fresh worktree lacks generated and submodule files, for example. Report it and leave it. When a failure is in a record this mode may not reword, change nothing in that record: report `run /design-doc trim <scope>` with the record and the rule it fails.
3. Commit on a branch, never on `dev`. When the current branch is `dev`, create one first: `git switch -c docs/<scope>-design dev`. Use a Conventional Commit, such as `docs(system): distill the health goals`.

## Output to the user

Report the document, not only "file updated":

```
# Design doc: <scope>

Anchor: <path> (<created | updated>), committed on <branch>
Scope: <the paths the anchor's goals bind>
Files considered: <the file list, or the directories it spans>
Coverage: <covered files, and the uncovered remainder no Scope: line names, or "all covered">
Binding set: <each other document, marked ancestor | descendant | overlapping; or "none">

Motivation: <one-sentence restatement>
Solution: <one-sentence restatement>

Primary goals
G1 — <outcome>. Need: [<kind>] <why it binds>. Evidence: <path[:Symbol] | none>.
Secondary goals
G2 — <outcome>. Need: [<kind>] <…>. Loss: <observable>. Measure: <figure | no budget set>. Evidence: <…>.
Non-goals
N1 [excluded | boundary | no guarantee] — <outcome not served>. Reopens when: <…> | Owner: <…>.
Frozen contracts
F1 — <shape>. Frozen by: <authority>. A break costs: <cost>. Evidence: <path | none>.
(or) None: <what the sweep read>.

Tests: <pass | fails: run /design-doc trim <scope> — each failing record and its rule>
Delete candidates: <record — dead alternative | no cost | no observable trigger — evidence; or "none">
Changes: <entries added, promoted, demoted or retired, and sections rewritten; or "new file">
```

End with any doubts the user still owes an answer on; if there are none, say so.
