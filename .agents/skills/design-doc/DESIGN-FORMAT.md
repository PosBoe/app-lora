# DESIGN.md Format

A `DESIGN.md` states a scope's design as it stands today: what the scope must achieve, and
why it achieves it the way it does. It is always written in the present tense and is always
true now. It is not a changelog, not a backlog, and not a record of decisions that no longer
bind — when the design changes, the affected section is rewritten in place in the same
change, and git holds the history.

No test checks this format yet. Check a file against it by reading. When the repository grows a
`design_doc_test.go` in the root, that test becomes the enforcement.

## The marker

A conforming file opens with `<!-- lora-design:v3 -->`. The marker is what a machine
reads: the test fails a `DESIGN.md` without it, and fails a marked file that does not hold
the shape below. Write a new file to the full shape or do not write it. An earlier marker
named a file the trim had not reached; no file carries it and the test admits none, so
there is one marker and one shape. The test checks the shape. The trim mode of `/design-doc`
reads records for what the shape cannot show; [TRIM.md](./TRIM.md) holds its procedure.

The rules the marker carries are: `## Frozen contracts` is always present, Motivation
stands alone, a record carries a `D` id, a record holds the five parts one time each and no
other part, a record fits the line budget, a record holds no date and no commit hash, and a
measured figure cites the evidence file.

A file that is not an architecture design document at all — a style guide, a token spec, a
product brief that happens to be called `DESIGN.md` — does not get the marker and is not
turned into one. It is moved out of the way, as the product's visual design system was.

## Section order

```md
<!-- lora-design:v3 -->

# {Title}

Scope: `path/one/`, `path/two/file.go`

## Motivation
## Solution
## Primary goals
## Secondary goals
## Non-goals
## Frozen contracts        ← always present (v3); states `None:` when the scope holds none
## Decisions               ← only when records exist
```

The sections down to `## Frozen contracts` are always present. `## Decisions` appears only
when the file has records.

## Scope

**The `Scope:` line names what the file binds**, as repo-relative paths, on the first line
after the title, and it is the only thing that says so. A file's own path binds nothing:
`devices/DESIGN.md` binds `devices/registry/` and `devices/uplink/`, and not the rest of
`devices/`. Read a directory as where a document is filed, never as what it binds. There is
no `Parent:` line, because a format that read ancestry off the path would give the repo root
exactly one cross-stack topic, for ever.

**A document sits in the directory that holds its whole scope, and in `docs/design/` when no
directory does.** A scope no one directory can hold, because it crosses contexts or crosses
the stack, goes to `docs/design/` under a name for its topic:
`docs/design/monitor-resolution.md`. The repo root cannot be that home — it holds one
`DESIGN.md`, so the first cross-stack topic would take the slot and the second would have
nowhere to go.

**The binding set is containment over `Scope:` paths.** Four relations decide how a document
binds a scope, and a directory decides none of them:

- The **anchor** holds every file of the scope and fits it most closely. Its goals bind
  absolutely. A scope larger than every document has no anchor.
- An **ancestor** holds every file of the scope. Its goals bind absolutely: a child adds goals
  about its own subject and never relaxes its parent's.
- A **descendant** binds only paths the scope holds. Its goals bind by default. A redesign of
  the scope can retire one with the user's approval, because the redesign may replace the
  mechanism the goal describes.
- An **overlapping** document shares files with the scope, and neither holds the other. It
  binds the shared files absolutely, as an ancestor does, and it gives no licence to retire.

[SCOPE.md](./SCOPE.md) holds the procedure that finds the set.

## Motivation and Solution

**Motivation** — what problem the scope exists to solve and what forces shape it. Not a
summary of the code; the statement of need that would survive a rewrite.

**Solution** — a short present-tense sketch of how the design meets the motivation: the shape
of the approach, its main moving parts, the one or two structural choices that make it work.
No operative detail — contracts, rules and structure stay in `README.md`/`doc.go`, and
rationale stays in the decision records.

**Motivation stands alone (v3).** It cites no file, names no `G`, `N`, `F` or `D` id, and
states no measured figure. A reader who stops after Motivation knows why the scope exists,
and the paragraph stays true when every file under it is rewritten. The test fails a
backticked path or an id in the section.

## Goals

The goal list is closed: observable behavior no listed goal covers is incidental and free to
change. It closes over the paths the `Scope:` line names, and no further. Behavior in an
uncovered file is unbound, not incidental.

```md
- **G3** — {one outcome, stated so that one observation can show a design broke it}.
  Need: [consumer] {why it binds}
  Evidence: `path/to/file_test.go:TestName`
```

- **Every goal carries a permanent id**, written at the start of the entry. Ids are
  **tier-neutral**: goals run on one `G1..` sequence across Primary and Secondary together,
  and the section the entry sits in states its tier. An id is claimed once, never reused and
  never renumbered; a promotion or a demotion moves the line and changes no id. Non-goals run
  on `N1..`, frozen contracts on `F1..`.
- **A goal's sentence holds its own observable**, and it gets no second field. Write "wait it
  out and the next attempt succeeds", not "the retry hint is correct". A sentence that names
  only a mechanism restates implementation; rewrite it as what a consumer can do because the
  mechanism exists. Do not add an `Acceptance:` or `Falsified-by:` field for the same fact.
- **One goal states one outcome** a candidate can keep or drop on its own. No observable
  appears in two entries, and none appears in two tiers at all. A goal states an outcome and
  never the absence of a mechanism: "memory for an abandoned key returns with no operator
  action", not "nothing sweeps them".
- **`Need:` and `Evidence:` are different facts.** `Need:` says why the goal binds.
  `Evidence:` says where the current behavior is checked. A test proves a behavior exists
  today; it does not prove a design chosen from scratch must still produce it.

### Need kinds

`Need:` declares its kind in brackets before the reason, and a line may carry more than one.

| Kind | Meaning |
| --- | --- |
| `[consumer]` | a caller, an operator or a frontend depends on the outcome; name it |
| `[contract]` | a public doc, README, config schema or third-party agreement promises it |
| `[safety]` | it holds correctness, durability, security, isolation or multi-tenancy |
| `[user]` | the user confirmed that it binds — the `confirmed by user` escape |
| `[test]` | a test is the only thing that asks for it |
| `[maintenance]` | it keeps the code cheap to change, port or review |

**A primary goal needs at least one of `[consumer]`, `[contract]`, `[safety]` and `[user]`.**
A test pins what the code does now, and a maintenance argument prices work rather than an
outcome. A primary goal offering only `[test]`, only `[maintenance]`, or the two together is
a question for the user, and the test fails it.

### Evidence

`Evidence:` holds repo-relative paths, not bare names:
`system/health_handler_test.go:TestHealthDatabaseDown`, not the bare test name. A path is
one read; a bare name costs a repo-wide grep. Cite a promise or a consumer the same way. A
goal with a need and no evidence is legal and honest: write `Evidence: none` and let the next
reader see that nothing checks it. Those are the only two shapes — the line either opens with
the literal `none` or holds at least one backticked repo-relative path. Prose beside a
citation stays legal.

### Secondary goals

A secondary goal adds two lines:

- **`Loss:`** — what a redesign gives up by dropping it, as an observable: "seed time then
  grows with the row count", not "seeding gets slower". A stated loss is what lets a redesign
  compare priced trades instead of counting kept goals.
- **`Measure:`** — honest rather than numeric. Give the figure where one exists; write
  `Measure: no budget is set; prototype if this separates two candidates` where a number
  could exist and nobody has taken it; write `Measure: none` with the reason where no number
  can exist. Do not demand a budget nobody has measured.

A primary goal that turns on a magnitude states the magnitude. Where nobody has measured one,
the entry is secondary and its `Measure:` says so.

### Non-goals

A non-goal declares its kind in brackets after the id: `- **N2** [boundary] — …`. Three
kinds, and a redesign treats them differently.

- **`[excluded]`** — a capability the scope refuses; a candidate must add no mechanism for it.
  Type a non-goal this way only where supporting the capability would be **wrong**, never
  where it is merely unimplemented today. It carries a **`Reopens when:`** line — the
  observable condition under which the scope would take the capability. It vetoes a candidate
  as absolutely as a primary goal holds one, and a veto nobody can lift is a defect.
- **`[boundary]`** — a responsibility that lives elsewhere. It carries an **`Owner:`** line
  naming where, and a candidate keeps it outside unless the scope includes that owner.
- **`[no guarantee]`** — a property the scope does not promise. A candidate need not preserve
  it, and may provide it incidentally at no cost; that is not gold-plating and never
  disqualifies. When the reason for the entry is a limit of the current code, this is the
  kind, not `[excluded]`.

## Frozen contracts

Frozen contracts hold the shapes the scope may not change because it does not own them: a
wire key an external client reads, an enum value already written to persisted rows, a
migration id, a path an OpenAPI spec pins. They are not goals — a goal is an outcome — so
they never enter the tiers.

```md
- **F1** — {the exact frozen shape}.
  Frozen by: {the authority}
  A break costs: {the cost}
  Evidence: `path/to/pin`
```

Each field opens its own line, as on a goal; the test reads a field only at the start of a line.

`Evidence:` names the repo artifact that checks the shape — a wirepin corpus, an OpenAPI
file, a migration — or the literal `none` where nothing in the repo checks it. The `none`
escape keeps the line honest: an enum frozen by rows in a customer's database has its
authority outside the tree, and a forced citation there would be an invention.

**The section is always present (v3).** A missing section cannot be told from a sweep nobody
ran. When the scope holds no frozen contract, the section holds one line that says what the
sweep read:

```md
## Frozen contracts

None: {what was read — the schema, the wire, the consumers — and why no one outside pins a shape}.
```

**Freeze only what an outside party depends on.** A shape is frozen because a consumer
outside the scope reads it, never because the scope prefers one method. A preference is a
decision record, and a record can reopen.

## Citations

**A binding line cites a tracked file of this repository.** `Evidence:`, `Frozen by:` and
*Upheld by* cite a path the tree holds. They never cite a branch, a pull request, a session,
a chat, or a note outside the tree. Such a target can change or go away, and the claim then
rests on nothing a reader can open. Name an outside authority in prose — "rows in a
customer's database" — and write `Evidence: none`.

**An entry cites from the repository root.** A path in a goal, a non-goal or a frozen-contract
entry — and in its `Need:`, `Evidence:` and `Frozen by:` lines — is read from the repository
root and nowhere else. A path in a record's *Upheld by*, or in prose, also resolves against the
document's own directory and as a bare name. Write every citation from the root, so that both
readings agree. `docs_policy_test.go` checks that every backticked path in a document exists.

## Decisions

**What earns a record: a real fork** — a decision where a competent alternative was genuinely
viable, something a future maintainer might plausibly "fix" back. A trade-off two or more
files must uphold together always qualifies; the rejected ideal *is* the alternative. Obvious
decisions with no live alternative stay out.

A distilled goal list also earns a `DESIGN.md`, even without a decision record. Do not create
a `DESIGN.md` when the scope has no distilled goals or qualifying records. Remove the file
when its last goal and record no longer bind.

**Record a decision where its blast radius is.** One contained in one file or one function
belongs in a comment on that code, opened with the grepable marker `COMPROMISE:` — the reader
who has to respect it is already looking at it. `DESIGN.md` is for what spans files: a joint
invariant, something that shapes the package's contract with its callers, or a reason that
lives somewhere the affected code cannot point at. Do not restate in `DESIGN.md` what a
`COMPROMISE:` comment already says in place.

The `## Decisions` heading is followed by an **index**, one line per record in record order:
the id and title in bold and the reopen trigger in a clause. A redesign reads the triggers
first, and the index is what keeps that read cheap in a long file.

````md
## Decisions

- **D12 — {Short title}** — reopens when {the trigger, in a clause}.

### D12 — {Short title}

Serves: G1, G3
Trades: none
Evidence: `path/to/DESIGN-EVIDENCE.md`     ← only when the record states a measured figure

**Decision.** {The choice, because {the decisive reason}. Stated positively, not as an apology.}

**Alternatives.** *{The alternative.}* {What it buys, and the one reason it lost. Where it is
still desirable and merely priced too high, say so rather than pretending it was wrong.}

**Consequences.** {What is given up and what it costs. Name one thing that becomes harder.}

*Upheld by* `{file}` ({the invariant it must keep}), `{file}` ({…}).

**What would reopen it.** {The observable condition under which the losing alternative wins.
This is the deletion trigger.}
````

### The record id (v3)

**A record carries a permanent id**, `D1..` on one sequence for the file, written before the
title in the heading and in the index line. The rule is the rule for `G`, `N` and `F`: an id
is claimed one time, never reused and never renumbered. A deleted record frees nothing. The
id is what a second record, an evidence heading and a `/redesign` report cite, so a title can
be sharpened and no reference breaks. Cite a record of another document with its owner, as
for a goal: `devices:D4`. A bare `D` id in a record names a record of the same file, and the
test fails one that names none.

**The title states the decision**, not the topic: "Cache per tenant, not per query", not
"Caching".

### One record, one decision (v3)

**A record holds the five parts, one time each, in the order above, and no other part.** The
parts are **Decision.**, **Alternatives.**, **Consequences.**, *Upheld by* and **What would
reopen it.** The test fails a missing part, a second one, a part out of order, a `####`
heading in a record, and a line that opens with any other bold run-in heading, such as
`**Why.**`. The reason a choice won goes in the first sentence of **Decision.** Each part
opens its own line: an *Upheld by* list at the end of the Consequences paragraph is a part
the test cannot see.

Apply one test when you write: if two claims in a record can reopen on two different
observations, they are two records. A record that needs a sub-heading is two records.

### The parts

- **Decision.** opens with the choice and its decisive reason in one sentence: "{X}, because
  {Y}." A reader who stops there has the decision.
- **Alternatives.** gives each alternative three things: its name in italics, what it buys,
  and the one reason it lost. State the decisive factor only. A supporting reason goes to the
  evidence file when it is a measurement, and goes away when it is not.
- **Consequences.** names a cost. At least one thing becomes harder, slower or impossible.
  A record that can name no cost recorded no fork.
- ***Upheld by*** names the files that must keep the invariant, each with the invariant.
  Write it with asterisks. A document under `vue/` carries `_Upheld by_` instead,
  because prettier formats that directory and rewrites an emphasis to underscores;
  the check reads both markers.
- **What would reopen it.** states an observation, never "nothing" and never "if requirements
  change". A record that states no reopen trigger is a veto nobody can lift, and the test
  fails it.

### A record states the present design (v3)

**A record holds no calendar date and no commit hash.** A date or a hash narrates how the
design came to be, and git holds that. The test fails both. Do not write "previously", "we
first tried" or "was replaced by": write what the design is, and what the alternative is.

**A measured figure cites the evidence file.** A record that states a measured figure — a
percentage, a ratio such as `1.6×`, or a benchmark unit such as `ns/op` or `MB/s` — carries
an `Evidence:` line under `Trades:` that cites the scope's evidence file. The evidence entry
holds the date, the workload and the full result. The record holds only the figure that
decided. A figure in backticks is a configured value, not a measurement, and the test does
not read it: write a threshold as `` `5%` ``. A record whose evidence file holds a heading for
it carries the `Evidence:` line too, figure or not: the test fails a heading whose record does
not cite the file.

**Operative detail stays out.** A stage list, a reporting spec, an API shape or a file map
belongs to `README.md` or `doc.go`. A record says why, and points at the file that says how.

### The line budget (v3)

**A record fits in 60 lines**, counted from its heading to its last line of text. The test
fails a longer one. A record over the budget holds one of four things that do not belong: a
second decision, history, operative detail, or proof. Split the first, delete the second,
move the third to `README.md` or `doc.go`, and move the fourth to the evidence file.

### When a record is deleted

A record leaves the file in three cases, and in each one the record, its index line and its
evidence heading go in the same change. Git holds the history.

- **The trigger fired** and the decision is reversed.
- **The alternative is dead.** No maintainer can plausibly "fix it back" any more: the code,
  the dependency or the use case the losing option needs is gone.
- **The record states no fork**: it names no cost, or it names no observable trigger.

Deletion for the second and the third case is the user's decision. `/design-doc` reports
such records and deletes none without an answer.

**`Serves:` and `Trades:`** name the goal ids the decision upholds and the ones it gave up to
uphold them. They are links into the goal list, which is what stops a record from outliving
the goal it served; a reference that resolves nowhere fails the test.

- **Each line holds ids or the literal `none`, and no prose.** A cost written there is a cost
  a reader cannot follow and the test cannot resolve, and **Consequences.** is where a cost
  belongs. A long list may wrap onto the next line. The test fails any other word.
- `Serves: none` is a signal that the goal list misses something and is worth a look.
  `Trades: none` is ordinary.
- **A record is a trade-off when its `Trades:` line names a goal**, and nothing else marks
  one. Every record names a cost in **Consequences.**; a trade-off is a record whose cost
  falls on an outcome the goal list states. `Trades: none` says that no consumer sees the
  cost. When **Consequences.** names a cost to an outcome a consumer sees, and no goal states
  that outcome, add the goal as a secondary goal and trade it. Do not write `Trades: none`
  to cover a missing goal.
- **A `Trades:` line names a secondary goal.** A primary goal is the one thing a design may
  not drop, so a record that trades one states a contradiction: either the trade is
  over-claimed and the cost belongs in Consequences, or the goal sits in the wrong tier — and
  a tier change goes through the user.
- Qualify a cross-document id with the owning document — `devices:G4` for a directory's
  `DESIGN.md`, `docs/design/monitor-resolution.md:G4` otherwise. Direction decides what is
  legal: an **ancestor**'s primary goal binds this scope absolutely, so trading one fails the
  same way; a **descendant**'s primary goal is one a redesign may retire with the user's
  word, so trading one is legal; a sibling's goals bind nothing here.

An **Alternatives** entry is a candidate already weighed and already priced. Whoever proposes
one again owes new evidence, not a new argument — the reopen trigger states what that
evidence is.

## The evidence file

**`DESIGN.md` is rewritten in place; the evidence file is append-only.** It holds what a
record's figure rests on: the measurement, its date, its workload, and the ledger of
experiments that lost. It sits beside the document under a fixed name: `DESIGN-EVIDENCE.md`
beside a `DESIGN.md`, and `{topic}.evidence.md` beside `docs/design/{topic}.md`. Create it
when the first record cites a measurement, and not before.

```md
<!-- lora-design-evidence:v1 -->

# Evidence: {the design document's title}

## D12

### 2026-08-31 — {what was measured}

Workload: {the benchmark, the preset or input, the commit}
Host: {what the figures depend on: CPU level, core count}

{The benchstat output, the table, the rejected experiments and why each lost.}

### 2026-09-15 — {the next run}
```

- **One `##` heading for each record that cites the file**, named by the record's id and
  nothing else. The test fails a heading whose record is gone or does not carry an
  `Evidence:` line, so the file is bounded by the live records.
- **One `###` entry for each run**, opened by its ISO date. Entries under a heading run in
  date order. Each entry states its `Workload:`; a figure without one cannot be compared
  with the next run.
- **Do not reword an entry.** A number here describes a past state, and it stays. A new run
  adds an entry. When the new entry changes a figure that the record states, change the
  record in the same change.
- **Delete the heading with its record.** Evidence exists to back a live record.

## What lives elsewhere

`DESIGN.md` is the one home for rationale. `doc.go`, `README.md` and `AGENTS.md` keep
operative rules, contracts and structure, and point at `DESIGN.md` for the why — do not tell
the same decision's story twice.

Update the record when a change adds, changes, or removes a cross-file decision. Make both
changes together. Add an accepted `/redesign` record with the change that implements it, and
add its measurements as evidence entries in the same change.
