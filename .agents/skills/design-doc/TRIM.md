# Trim mode

`/design-doc trim <scope>` audits the decision records of one design document and rewords
the ones that carry residue. It is the only mode that rewords a record, and it rewords none
without the user's approval. [DESIGN-FORMAT.md](./DESIGN-FORMAT.md) holds the rules; this
file holds the procedure.

Every document already holds the shape DESIGN-FORMAT.md states. A reader of the shape sees the shape: the five parts,
the budget, a date, a hash, an unresolved id. It does not see what a record *says*. Trim mode
reads for that. The work list comes from reading, not from test failures.

Trim leaves the goals alone. Do not mix it with a default run. When the goal list is
incomplete, stop and run the default mode first, because `Serves:` and `Trades:` need the ids.

## What residue looks like

A record states one decision in the present tense: the choice and its decisive reason, each
alternative with the one reason it lost, the cost, the files that uphold it, and the trigger.
Anything else is residue, and each kind has one home.

| Residue | Signs the test cannot see | Where it goes |
| --- | --- | --- |
| A second decision | two claims that reopen on different observations; an alternative that answers a different question | a record of its own |
| A count | "about 19 call sites", "three test files" — a number a reader must recount (the repo's CLAUDE.md forbids it) | the rule the count stands for, or the named exceptions |
| History | "previously", "we first tried", "was replaced by", "the earlier policy", a story of what another record did, a figure explained by how it changed | deleted; git holds it |
| Operative detail | a stage list, a reporting spec, an API shape, sizes and function names that say *how* | the scope's `README.md` or `doc.go`, or dropped when it already says so |
| Proof | a table, narrated measurement passes, more figures than the one that decided, rejected experiments | the evidence file |
| A stale term | a name the code or `CONTEXT.md` no longer uses, in the title, the index line or the prose | the current term |
| A weak part | **Decision.** with no "because"; an alternative with no italic name, or with more than one reason it lost; a **Consequences.** that names no cost; an *Upheld by* file that no longer upholds the invariant | the part rewritten |

## Step 1 — Read the document and its world

Resolve the scope to one document, as the default mode does ([SCOPE.md](./SCOPE.md)). Trim
one document in one run. Read:

- the whole document, and its evidence file if one exists;
- every file a record names in *Upheld by*, to check that each still upholds its invariant;
- the scope's `README.md` and `doc.go` — operative detail goes there, and you cannot tell
  whether they already say a thing without reading them;
- the nearest `CONTEXT.md`, and `git log --oneline -20 -- <scope>`, to find terms that were
  renamed after a record was written.

Run `go test -run 'TestDoc' .` from the repo root as a baseline. A failure that
names this document is part of the work list. A failure in any other file is not this run's:
note it for the output and leave it.

Note what you see outside the records too — a goal line with history or a dead citation, a
record that serves no goal. Trim does not touch goals, so each one goes to the output as a
follow-up for a default run.

## Step 2 — Classify every record

Read each record against the residue table and give it one verdict:

- **keep** — it holds no residue.
- **trim** — it holds residue. Mark each passage with its kind and destination.
- **split** — it holds more than one decision. Name each decision and the trigger that
  separates them. The first keeps the record's id; each other takes the next unused `D`
  number.
- **delete candidate** — the losing alternative is dead (the code, the dependency or the use
  case it needs is gone), or the record names no cost, or no observable trigger. State which,
  with the evidence. Offer the other ways out where they fit: the user supplies the missing
  part, or the record moves to a `COMPROMISE:` comment when one file upholds it.

Read each record with `Trades: none` against its **Consequences.** When the cost there
falls on an outcome a consumer sees, and no goal states that outcome, the record hides a
trade-off. Trim does not add the goal. Write the goal it would trade, worded as a secondary
goal with its `Loss:`, and put it in the output as a follow-up for a default run. The user
decides each one.

A passage that looks like residue but carries the decisive reason stays. Trim cuts what the
decision does not need; it does not shorten a record for its own sake.

## Step 3 — Show the plan and wait

Present one plan before you write anything. For each record that is not **keep**, show:

- the verdict, and each passage that leaves with its residue kind and destination;
- the record as it will read, in full — and for a split, each new record in full;
- for operative detail, the target file and whether it already says the thing. Move the text
  when it does not; drop it when it does.

Then ask one `AskUserQuestion` call. Offer the whole plan as the recommended option and, where
records differ in how much they change, subsets by record. Ask each delete candidate as its
own question, with your recommended reading first. A record the user does not approve stays
byte for byte.

## Step 4 — Write

Make the approved changes in one pass, in this order.

1. **Evidence first.** Create the evidence file when the first record needs one. Move each
   proof passage under the record's `## D<n>` heading as a dated entry, verbatim — an entry
   describes a past state, and this is the only time it is written. Take the date and commit
   from the passage. Where it states none, take them from `git log -S` on a figure it holds,
   and say in the entry that the date is the commit's. Write `Workload:` from the passage, or
   `Workload: not recorded`. Never reword an existing entry, even for a stale term. Move only
   the proof: a sentence that also states the decision stays in the record, and the entry
   opens with a subject of its own rather than "It…". A record that gains an evidence heading
   carries an `Evidence:` line, whether or not a figure stays in it — the test requires it.
2. **Operative detail second.** Move each passage to its approved target, verbatim, under a
   heading that fits. This is the one case in which `/design-doc` writes a file that is not a
   design document or its evidence file.
3. **Records third.** Write each record to the five parts. Open **Decision.** with the choice
   and its reason in one sentence. Cut each alternative to the one reason it lost. Keep the
   figure that decided, with the `Evidence:` line. Write each record to 58 lines or fewer: the
   test admits 60, and a record at the limit breaks on the next word someone adds. A split-off
   record that no goal covers takes `Serves: none`; do not invent a goal for it, and name it
   as a follow-up.
4. **The index last.** Rewrite each changed title in its index line, and add a line for each
   split record.

Read each rewritten record once more for a doubled or badly wrapped sentence before you test.

## Step 5 — Prove it and commit

Run `go test -run 'TestDoc' .` from the repo root. Every moved citation must still
resolve. When a record cannot pass without an answer from the user, revert that record and
report what blocks it — never leave the file failing.

Commit on a branch, never on `dev`, with a Conventional Commit such as
`docs(zbench): trim the decision records`. When the current branch is `dev`, create one
first: `git switch -c docs/<scope>-trim dev`.

## Output to the user

```
# Trim: <document>

Tests: <pass before and after | what still fails, and why>
Records: <kept> kept, <trimmed> trimmed, <split> split into <n>, <deleted> deleted
Lines: <before> → <after>
Evidence file: <path> (<created | updated | not needed>)
Moved to README.md/doc.go: <each target and the heading it landed under, or "nothing">
Removed: <per record, one line: the history, stale terms and weak parts it lost>
Still open: <records the user has not decided, or "none">
Follow-ups for a default run: <goal lines with history or dead citations, records serving no goal, records whose `Trades: none` hides a cost with the goal each would trade; or "none">
```
