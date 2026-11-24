---
name: trim-agents-md
description: Audit the AGENTS.md files of a scope against the repo's routing doctrine — AGENTS.md holds only operative rules — and trim them to the minimal essential set. Split each fused bullet into its parts, keep the imperative payload, and route the residue to the document type that owns it (README.md/doc.go for mechanism, CONTEXT.md for vocabulary, a COMPROMISE comment for a file-local decision, a shared guide for a repo-wide rule); move rules up or down the scoped AGENTS.md hierarchy to their one home; delete what restates a shared guide or an ancestor; then execute the approved moves and prove them with the docs-policy tests. Never writes DESIGN.md — design rationale is parked verbatim for /design-doc; vocabulary defects found on the way are /audit-terms' work; capturing new rules after a work session is /revise-claude-md, not this. Invoke ONLY when the user explicitly runs `/trim-agents-md` or explicitly names this skill — do not infer it from generic phrasing like "clean up AGENTS.md", "the instructions file is too long", "update CLAUDE.md", or "reorganize the docs" alone; treat those as ordinary requests instead.
---

# trim-agents-md

Trim a scope's AGENTS.md files to operative rules and nothing else. The deliverable is each file reduced to the rules an editor must obey — one bullet per rule, payload intact — with everything else moved to the document type that owns it, or deleted, and every move proved safe by the repo's own policy tests. Not a documentation rewrite, not a DESIGN.md author, and never a place to invent rules the tree does not already state.

The cost model behind every verdict: an AGENTS.md line is read by every agent on every task in that subtree, forever. The lines that pay for that are executable commands with their exact flags, short code examples that pin a pattern, explicit boundaries, and non-obvious gotchas. Overviews, mechanism tours, and rationale prose do not pay — they measurably slow the reader without changing what it writes — and each has a document type of its own. Target the shape `docs/maintaining-agents-md.md` asks for: the leanest files under 60 lines, none over 300.

One kind of content is inviolable. A **deliberate documented divergence** from a shared guide or an ancestor — a bullet that says "here X, not the guide's Y, because Z" — is the highest-value content an AGENTS.md holds, because it is the one thing an agent cannot learn anywhere else. It is never trimmed, never moved into prose, and never paraphrased.

**The audit runs in plan mode; moves follow approval.** Enter plan mode with `EnterPlanMode` before step 0 reads anything, unless the session is in it already. Steps 0 through 4 are read-only: no edits, no commits, no branch. Present the ledger with `ExitPlanMode` and wait. The user's approval starts step 5, and it approves per file. If the harness has no plan mode, still make no edit until the user approves.

## Step 0 — Resolve the scope

The scope comes from the invocation argument and takes one of four shapes:

- **One AGENTS.md, or the directory that holds it** (e.g. `/trim-agents-md openapi/`) — scope = that file, **plus** every AGENTS.md above it up to the repo root, **plus** every AGENTS.md below it. The ancestors and descendants are read, not trimmed: a level move needs both endpoints readable, and "duplicate of an ancestor" is undecidable without the ancestors.
- **A subtree or bounded context** (e.g. `/trim-agents-md vue/`) — scope = every AGENTS.md under it, plus the root AGENTS.md as the read-only ancestor.
- **The whole repo** — expensive: every AGENTS.md in the tree. Confirm the user means it, then work worst-offender-first by line count and report per file.
- **No argument** — ask before reading anything.

Whatever the shape, the rubric sources below are always read. They are edited only through a ledger item the user approved, never as a side effect.

## Step 1 — Load the doctrine and the destinations

The rubric is the repo's own, not this skill's invention. Read, in this order:

1. The root `AGENTS.md` — the one-line charter ("`AGENTS.md` holds only operative rules") and the four-document division: `README.md` holds the directory map and file tour, `DESIGN.md` the purpose, goals and rationale, `CONTEXT.md` the domain terms.
2. `docs/maintaining-agents-md.md` — what belongs (conventions, gotchas, build/test commands, scope boundaries) and what does not (architecture prose, ephemeral notes, duplicated rules).
3. `docs/go-package-conventions.md`, section **"Where the reasoning goes"** — the routing authority step 3 operationalizes, including the fifth destination: a `COMPROMISE:` comment on the code itself.
4. `docs/go-test-conventions.md` — the second shared guide; a test rule in a subtree AGENTS.md is judged against it.

Then inventory the **destination set** for each directory in scope: does a sibling `README.md`, `DESIGN.md`, `CONTEXT.md`, or `doc.go` exist, and what does each already say? A later MOVE verdict must name a real destination section, or trigger the missing-destination edge case. Find every binding DESIGN.md with `grep -rn '^Scope:' --include='*.md' .`.

Last, read the `allowedCitations` map in `docs_policy_test.go` at the repo root and note every entry whose `doc` is a file in scope. The map is keyed by the *pair* `{doc, cited}`, so a sentence that moves takes its allowance entry with it or breaks the test.

## Step 2 — Inventory the items

Atomize each in-scope AGENTS.md into items. **The unit is the rule, not the bullet.** The tree's common failure shape is a fused bullet — the rule, the mechanism behind it, and the incident that produced it, welded into one paragraph. Split a fused bullet into its parts first, and classify each part on its own; a verdict on the whole bullet gets the payload wrong in both directions.

For each item record:

- the text and its `file:line`;
- every backticked path citation it carries;
- the shared-guide, root, or ancestor line it may duplicate — found by grepping the item's key phrases against `docs/go-package-conventions.md`, `docs/go-test-conventions.md`, and every ancestor AGENTS.md;
- whether it carries a divergence marker: "not the repo-wide default", "unlike the guide", a stated reason attached to a departure.

Record `wc -l` per file as the report's baseline.

## Step 3 — Classify: the payload test, then the routing rubric

### The payload test

Strip the item to the imperative an agent must obey — the sentence that, if deleted, would change what an agent *writes or runs* in this subtree. That sentence is the payload. It stays in AGENTS.md, tightened to one bullet in Simplified Technical English if it is not one already, with its commands, flags, and code spelled exactly as before — the payload is tightened around, never reworded through.

Everything left after the payload is residue, and it routes by the table. Length is not the test: a 40-word bullet that is all payload stays whole; a 10-word architecture note goes. [references/before-after.md](./references/before-after.md) holds three worked splits to calibrate against.

### The routing rubric

`docs/go-package-conventions.md` § "Where the reasoning goes" is the authority; this table only operationalizes it.

| The residue is… | Destination | Executed by |
|---|---|---|
| An operative rule, a gotcha, a command, a boundary | stays — KEEP or TIGHTEN | this skill |
| The package tour, a file map, how a mechanism works | `README.md` or `doc.go` | this skill |
| Why the design is what it is; goals; a decision's story | `DESIGN.md` | parked for `/design-doc` — see step 5 |
| A term and what it means | `CONTEXT.md` | this skill, in the shape `.agents/skills/domain-modeling/CONTEXT-FORMAT.md` states |
| A decision contained in one file or function | a `COMPROMISE:` comment on that code | this skill |
| A rule true of the whole repo, or of Go or tests generally | the root AGENTS.md or a shared guide | this skill, flagged high-blast-radius in the ledger |
| A restatement of a shared guide or an ancestor, with no named divergence | nowhere | DELETE |
| Ephemeral, stale, or contradicted by the code | nowhere | DELETE — stale goes through the user first |

### Level moves

The hierarchy rule comes with the doctrine: a context root keeps only what spans its whole context; each subpackage carries its own rules.

- **Up**: a rule that constrains two or more sibling subtrees moves to the parent AGENTS.md. A rule that constrains the whole repo moves to the root or into a shared guide.
- **Down**: a parent rule that names one subpackage's internals moves into that subpackage's AGENTS.md. Create the child file only when it genuinely diverges from its parent — `docs/maintaining-agents-md.md`'s own bar. A rule that earns no file there was probably not operative at all.
- **Duplicates resolve downward-delete**: where an ancestor and a descendant state the same rule, the higher home wins and the lower copy is deleted — never the reverse. Except when the lower copy *differs*: a difference may be an undocumented divergence, the tree's most valuable and most fragile content. Then the fix is TIGHTEN, into the divergence shape — "here X, not the guide's Y, because Z" — and nothing is deleted.

### Guard lists

Always survives, whatever else the verdict says: divergence bullets; executable commands with their exact flags; short code examples that pin a pattern; explicit boundary statements ("this package must not import…"); non-obvious tooling.

Never survives: a restated delegation — "see `MODULE-BOUNDARIES.md`" followed by a summary of it; the pointer stays, the summary goes; a tiered goal list — that is a DESIGN.md's body, whatever file it sits in; auto-generated content of any kind.

## Step 4 — Refute before the ledger

For each verdict, try to kill it:

- **DELETE as duplicate** must cite the surviving line — the guide or ancestor `path:line` that makes the local copy redundant. Diff the two texts. A difference that is not pure wording blocks the delete: it may be a latent divergence, and that fork goes in the question batch.
- **MOVE** must name a destination that exists, or invoke the missing-destination edge case.
- **MOVE of an item carrying backticked paths** must re-resolve every citation from the destination's directory — `docs_policy_test.go` resolves a citation relative to the *citing* document — and check the `allowedCitations` notes from step 1 for a `{doc, cited}` pair that must migrate with the sentence.
- **DELETE as stale** must be confirmed against the code the rule governs. "A wrong rule is worse than no rule" — but the code, not the rule, may be what is wrong, and that decision is the user's, never made silently.

Batch every surviving doubt into **one `AskUserQuestion` call** before `ExitPlanMode`, each option stating the item, the candidate verdicts, and your recommended reading first. When the evidence is unambiguous, don't ask — proceed and cite it.

## Step 5 — Present the ledger, then execute

Present the ledger with `ExitPlanMode`, grouped per file in the Output shape below: lines before → projected after, every item's verdict with its destination `path § section`, every delete with the surviving line that justifies it, and the parked-for-DESIGN material verbatim. **Approval is per file** — contested items were already settled in the step 4 batch, so what reaches the ledger is a plan the user approves or strikes file by file. A struck file is skipped whole.

Execute each approved file as **one conventional commit** — `docs(<scope>): trim AGENTS.md` — that holds the destination writes, the trim, and any `allowedCitations` migration together, so every commit passes the policy tests on its own. Inside a commit, order matters:

1. Write the destinations first — README/doc.go sections, CONTEXT.md entries, `COMPROMISE:` comments, shared-guide additions — adapting each moved citation to resolve from its new directory.
2. Trim the source only after the payload and residue demonstrably exist at exactly one destination each.
3. Migrate the affected `allowedCitations` entries in the same commit.
4. A whole-file deletion removes the AGENTS.md **and** its sibling `CLAUDE.md` symlink together, after a repo grep for citations of the deleted path.

### The DESIGN.md handoff

This skill never writes, rewords, or deletes a byte of any DESIGN.md. `/design-doc` is that file's sole author, and DESIGN-FORMAT.md states a shape this skill does not produce. Residue verdicted for DESIGN.md takes one of three paths:

- **The material is already in the DESIGN.md.** Verify by reading it, then DELETE the AGENTS.md copy outright — it was a duplicate, not a move.
- **The material is new rationale and a DESIGN.md exists for the scope.** Remove it from AGENTS.md and park it **verbatim** in the run report, under the `Parked for /design-doc` block, closing with the exact invocation to run.
- **No DESIGN.md binds the scope.** Do **not** delete — that would destroy the only copy of the rationale. Leave the item in place and mark it `blocked on /design-doc` in the ledger and the close-out. Never trade knowledge for line count.

## Constraints

- Never author or edit a DESIGN.md, and never touch a decision record's prose anywhere.
- Never delete, relocate into prose, or paraphrase a divergence bullet.
- Payload commands, flags, and code examples move verbatim. Tightening rewrites the sentence around them, never through them.
- A CONTEXT.md addition follows `.agents/skills/domain-modeling/CONTEXT-FORMAT.md`; a `COMPROMISE:` comment names only symbols its file's package imports (`TestCommentsNameLiveSymbols` checks).
- An edit to the root AGENTS.md or a shared guide binds the whole repo. Each is its own ledger item, flagged high-blast-radius, never bundled.
- Everything written obeys ASD-STE100 Simplified Technical English, per the root AGENTS.md.
- The net repo-wide AGENTS.md line count goes down. A file may grow only by move-ins.
- Nothing new is invented: every sentence written at a destination is traceable to a sentence removed from a source.

## Edge cases

- **A file that should die.** When every item is deleted or moved, delete the AGENTS.md and its `CLAUDE.md` symlink in one commit — `docs/maintaining-agents-md.md`'s "don't create one unless it genuinely diverges" applied retroactively.
- **A rule the whole repo needs.** The same rule found in three or more subtrees is proposed for a shared guide or the root; the local copies then delete, because the guides bind by default. Highest blast radius in the skill — always its own flagged ledger item.
- **The destination README does not exist.** Creating one is allowed — only DESIGN.md is off-limits — but one paragraph does not earn a file. Prefer `doc.go` for a Go package, or a `COMPROMISE:` comment for file-local residue. A genuinely ambiguous case joins the step 4 question batch.
- **Vocabulary residue with no CONTEXT.md.** Append to the context's CONTEXT.md where one exists; otherwise park it for `/domain-modeling` beside the DESIGN parkings — creating a glossary is that skill's work.
- **A stale rule against live code.** Never resolved silently. Whether the rule or the code is wrong is the user's fork, asked in the step 4 batch.

## Verification

After each commit, and once at the end:

```sh
go test -run 'TestDocCitationsResolve|TestCommentsNameLiveSymbols' .   # citations resolve from their new homes
go test -run TestDesignDoc .                                           # no DESIGN.md was touched
find . -name CLAUDE.md ! -exec test -e {} \; -print                    # dangling symlinks; must print nothing
```

Then the residue greps: for every moved item, a grep of its key phrase finds nothing in the source and exactly one hit at the destination. Record the `wc -l` before/after per file for the report. Run `task test:pkg` over the touched packages only when a `COMPROMISE:` comment touched a `.go` file; a pure `.md` change does not need it.

## Self-check before the ledger, again after execution

- Every divergence bullet in scope is byte-identical before and after.
- Every moved rule has exactly one home: gone from the source, present at one destination, and the destination is the type the rubric names.
- Rerun the payload test on every MOVE: an agent that obeys only the remaining AGENTS.md files loses no instruction.
- Every DELETE cites the surviving `path:line` that makes it a duplicate, or the user approval that makes it a retirement.
- Every command, flag, and code example greps verbatim post-trim.
- No `allowedCitations` entry names a `doc` that no longer carries the citation, and no migrated sentence lacks its entry.
- No DESIGN.md changed, and the parked block holds everything verdicted for one.
- The numbers reconcile per file: items in = kept + tightened + moved + deleted + blocked.

## Output to the user

```
## AGENTS.md trim: <scope>

### <path>/AGENTS.md — <N> → <M> lines · kept K · tightened T · moved V · deleted D · blocked B

Moved
- "<item's first words…>" (<file:line>) → <destination path> § <section>
…

Deleted
- "<item's first words…>" (<file:line>) — duplicates <guide path:line> | retired, approved by user
…

Parked for /design-doc
> <the removed rationale, verbatim>
Run: /design-doc <scope>

Blocked on /design-doc (left in place)
- "<item's first words…>" (<file:line>) — no DESIGN.md binds <scope>

### Files deleted
- <path>/AGENTS.md + <path>/CLAUDE.md (symlink)

### Verification
- TestDocCitationsResolve, TestCommentsNameLiveSymbols: pass
- TestDesignDoc: pass (no DESIGN.md changed)
- dangling-symlink check: clean
- net AGENTS.md lines: <before> → <after>
```

Close with what was checked and found already lean, by name — "`vue/AGENTS.md` (36 lines) is all payload and stays whole" — so a short ledger reads as evidence, not omission. Do not manufacture trims to fill space.
