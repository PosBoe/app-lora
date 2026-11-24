---
name: redesign
description: >-
  Redesign how a scope achieves its goals: discover every decision its code
  makes, flip each one, and price the flip with evidence — probes, cited
  research, paired measurements, a prototype only where nothing cheaper
  settles it. Holds the primary goals and frozen contracts that /design-doc
  recorded in the binding DESIGN.md files; everything else, public contracts
  included, may break at a priced cost. Delivers a verdict-first report ready
  for /plan, and never edits production code. For behavior-preserving
  restructuring use /plan-ideal-refactor; for new capability use
  /feature-design. Invoke only when the user runs `/redesign` or names this
  skill — not for generic "is there a better way" or "rethink X" requests.
---

# redesign

Given a scope with a recorded goal list, find the set of decisions its code has made, flip each one, and price the flip with evidence. The current implementation is a set of choices, never the frame: this skill is licensed to conclude that the right answer stores the data differently, exposes a different API, uses a different library, or deletes a mechanism. Everything about the current code is negotiable — public API, wire contracts, persisted rows, config shape, dependencies — *except the primary goals and the frozen contracts*. Breakage is a cost to be priced and reported, not a sin to be avoided; unpriced breakage is a defect in the output.

The unit of work is a **decision**, not a whole-design candidate. A decision is one choice the scope makes — two cache tiers rather than one, msgpack rather than gob, Redis rather than memcache, S3-FIFO rather than TinyLFU — with a **keep** option and one or more **flips**. Decisions are judged one by one in dependency order, and the recommendation is one consistent assignment across all of them. This differs from `plan-ideal-refactor` on one axis: that skill holds observable behavior fixed and redesign the structure underneath; this skill holds the *goals* fixed and redesigns the choices on top of them. It differs from `feature-design` on another: that skill adds to the product; this one re-derives how an existing purpose is served. The deliverable is a recommendation with evidence — not an implementation and not a migration plan. In practice the user pastes the report into `/plan` in a fresh session, so the report must stand on its own and must not leave that session questions to re-ask.

**The user owns the goals; the run owns the evidence.** The user decides what the scope must achieve and what may break. How much to probe, measure or prototype is the run's call, never a question: a run that asks permission to measure hands the user work it was invoked to do. Past runs showed this plainly — the "which decisions should be struck?" question was answered "up to you, gather evidence" or left blank nearly every time.

## Step 0 — Resolve the scope and pass the entry gate

Follow [`design-doc/SCOPE.md`](../design-doc/SCOPE.md): resolve the scope to files, ask if it has two readings, and find the binding set with the `Scope:` grep. [`design-doc/DESIGN-FORMAT.md`](../design-doc/DESIGN-FORMAT.md) § Scope says how each relation binds; this run applies it as follows.

- **Anchor and ancestors** bind absolutely.
- **Descendants** bind by default. A flip may drop a descendant goal only by naming it and asking at the checkpoint. A descendant goal dropped in silence is the same defect as a silently re-tiered goal.
- **Overlapping documents** bind the shared files absolutely. Name those files in the output: that is where the binding applies, and the rest of the document's scope is outside the run.
- **A boundary inside the binding set is open.** When a `[boundary]` non-goal's `Owner:` is itself in scope, a flip may move the work between the two, priced like any other.

The output lists the whole binding set marked by relation, and the covered and uncovered parts of the scope.

**The entry gate reads coverage.** The run proceeds when the binding documents cover any file in scope, and their union binds. **If no document covers any file in scope, stop** and direct the user to `/design-doc`. Do not distill goals inline: the user reviews and owns the goal list before this skill judges decisions against it. Offer `/design-doc` on the uncovered remainder as a follow-up.

**The guard on a partial run.** The closed-list rule — unlisted behavior is incidental — holds over the covered files only. Behavior in an uncovered file is **unbound, not incidental**: a flip that changes it names the change and asks at the checkpoint. Without this guard, a scope that reaches undocumented code licenses a flip to discard whatever nobody has tiered yet.

**Route away when another skill fits better.** If the user wants the current behavior kept and only the structure improved — `/plan-ideal-refactor`. A new capability — `/feature-design`. Bugs — `/review-code` or `/code-review`. Performance of Go code with the approach kept — `/go-optimize-bench`. Schema-only redesigns — `/plan-ideal-refactor` with the `pg` or `ch` territory. This skill is for "should this even work the way it works".

## Step 1 — Read the scope and its world

Collect, don't diagnose; a mechanism that looks baroque on first read is often carrying a requirement you haven't met yet. Keep running notes in a scratch file from the start — the run fans out in step 5 and the tree must survive it. For a scope larger than you can read in one pass, split the read across `Explore` sub-agents by directory and keep the notes in the main context.

1. **House rules and domain language** — the nearest `AGENTS.md`/`CLAUDE.md` and `README.md` of every directory in scope, `CONTEXT-MAP.md`, and each modeled context's `CONTEXT.md`. `DESIGN.md` files and `COMPROMISE:` comments are prime evidence: a decision record states the alternative that lost and the condition that would reopen it. Read the index under `## Decisions` first — it pairs each record with its reopen trigger — and open in full every record whose `Serves:` or `Trades:` line names a goal in play. A `Trades:` line is a price the current design already paid — a flip that keeps the traded goal must say what it does differently to afford it.
2. **Every file in scope, tests, benchmarks and docs included.** Tests encode which behaviors someone cared enough to pin; benchmarks are the instruments step 5 measures with; user-facing docs and `vue/src/CHANGELOG.md` entries show what was promised to users.
3. **The data's whole journey** — for anything stored or transported: the Postgres schema and migrations behind it, the HTTP/queue contracts it crosses, the frontend code that parses and renders it. A redesign that only reads the Go side of a cross-stack mechanism will misjudge both the goals and the cost of breaking the contract.
4. **Consumers, found mechanically** — grep imports, endpoint paths, config keys, emitted events. Classify production code vs wiring vs tests. Consumers are the audience for any breakage.
5. **Variant dimensions** — build tags (`pkg/featflag/`), process roles (`serve` and any later worker), and config keys whose absence changes the wiring. A flip that only works in one deployment must say so, and step 5's "every"/"none" probes cover this list.
6. **The available stack** — record what this app actually runs on, because a flip must be buildable from it: Go backend (fx-wired), PostgreSQL, Vue.js frontend, and the existing dependency set (`go.mod`, `vue/package.json`). A flip that assumes infrastructure or dependencies outside this list isn't disqualified, but carries the assumption as an explicit cost.
7. **Prior measurements** — earlier figures in decision records and in the evidence file beside each `DESIGN.md`, benchmark files, and the session memory. A flip that was measured and rejected before is not re-measured without a new fact.
8. **Trust boundaries** — every entry point in scope that runs without authentication or before it, and every value that grants access (an id used as a bearer credential, a token, a signed link). For each, trace who can learn it: which responses, lists, logs and emails carry it. A redesign that reviews how a credential is stored but not who receives it misses the exposure that matters.
9. `git log --oneline -20 -- <paths>` — recent shape-changing work and in-flight directions the recommendation must not collide with blindly.

## Step 2 — Load the goals and find the contradictions

The binding set's goal lists are authoritative — do not re-distill them. Load every goal from every file in the set, and carry each one as `<document>:<id>` — `devices:G4`, the owning document in full, not "the cap goal" — so a goal keeps one name through decisions, ledger and output. The qualifier names the document: a bare directory for its `DESIGN.md`, which `design_doc_test.go` resolves as `<dir>/DESIGN.md`, and the path itself for one in `docs/design/` — `docs/design/monitor-resolution.md:G4`. **Never renumber**, and never invent an id. The tiers carry the operational rules:

- **Primary goals** — a flip that drops one is out, not scored. A **descendant** file's primary goal is the one exception: it may be dropped, but only with the user's word, asked at the checkpoint before any decision is judged. A flip that deletes the mechanism a goal describes drops that goal, even though nothing is left to violate it.
- **Secondary goals** — a flip may trade one away, but must price the loss in the breakage ledger like any other cost. The goal's own `Loss:` line is the price; where its `Measure:` records that no budget exists, the ledger says so too, and a measurement is the way to settle a decision that turns on it. Never invent a figure the document does not have.
- **Non-goals** — read the kind, and apply it. **`[excluded]`**: a flip that builds mechanism for it is gold-plating; an absent-capability decision that serves it is not listed unless its `Reopens when:` line is met, in which case the exclusion is a question for the checkpoint and not a veto to obey in silence. **`[boundary]`**: the work belongs to the named `Owner:`, so keep it there unless the scope includes that owner. **`[no guarantee]`**: nothing is required, and nothing is forbidden — a flip that provides the property incidentally, at no extra mechanism, is not gold-plating and is never disqualified for it.
- **Frozen contracts** — the `F` entries are not goals and not negotiable by this skill. A shape frozen by an authority outside the scope stays as it is; a flip that changes one is out unless the user first lifts the freeze, which is a question for the checkpoint, never a cost to price silently. Two goals with the same subject and one frozen contract with the same subject are three separate constraints — hold all three.
- **Unlisted observable behavior is incidental** and flips are free to discard it — ordering nobody relies on, a wire field no consumer reads, a knob for a long-fixed problem. This holds over the **covered** files. Behavior that lives in an uncovered file is unbound instead: a flip that changes it raises the change at the checkpoint.

**Trust, but flag contradictions.** Check each goal's `Evidence:` paths exist — that is the one mechanical check the format allows, and a dead path is the usual way a stale goal shows itself. Read the `Need:` line beside it: the need is what binds a flip, and the evidence only says where today's behavior is checked. A primary goal whose need is `[test]` or `[maintenance]` and nothing else is a goal to raise, not one to hold: propose its demotion to secondary at the checkpoint, or its retirement when nothing observable depends on it. A secondary goal with such a need is already free to trade, so it is not raised. A `[consumer]` need names the party whose breakage the ledger must price. When the step 1 read contradicts the file — a pinning test is gone, the cited consumer was deleted, a goal states something the code does not do, new code serves an outcome the list doesn't mention, a secondary goal turns out to have a production consumer, a frozen contract's authority no longer exists — note it. Where a contradiction turns on a fact the code can settle in minutes — a counting test, a compile check, a one-query look at the data — settle it now, outside the working tree (step 5's prototype rules apply), so the checkpoint asks about a fact rather than a suspicion. The same holds for a flip that would need a freeze lift or a scope extension: if a cheap probe can show the flip is unsound anyway, run it before asking.

**An existing goal that states a mechanism** rather than an outcome — "retains the warmed capacity" where the outcome is "a warm request allocates nothing" — is raised as a contested reword when a live flip turns on the wording.

**An `F` entry frozen only by the scope's own code is not frozen** — DESIGN-FORMAT.md freezes only what an outside party depends on. Raise it as an evidence-forced change: retype it as a goal when a consumer needs the outcome, or leave its rationale to a decision record.

**Collect every proposed change** (promote, demote, add, retire, reword — a fix to a `Need:` or `Loss:` line included, and a fix to an `F` entry's text where it misdescribes the frozen shape, which changes the words and not the shape — unfreeze, every descendant goal a flip needs to drop, every unbound-file change) for the checkpoint in step 4. Do not ask yet; the checkpoint is one batch. Silent reclassification — quietly redefining or re-tiering a goal so a favored flip wins — is disqualifying.

**A goal written back states an outcome, never a mechanism.** A promotion or a demotion moves a line and touches no wording, but a new or reworded goal is new prose in a `DESIGN.md`, and the rule that binds `/design-doc` binds it too: write what a consumer can observe — "alert state survives a restart", not "alerts are stored in Postgres" — and never the absence of a mechanism. A goal that names a winning flip's mechanism writes this run's implementation into the one file that has to outlive it.

## Step 3 — Discover the decisions

Build the decision tree from four sources. Discovery reads the scope; it does not work from a checklist, and it stops at the scope's edge (a decision that belongs to a `[boundary]` owner is named as out of scope, not listed).

1. **Every mechanism the code commits to.** Each structural choice visible in the code is a decision: number of tiers, storage and its shape, serialization format, eviction or ordering algorithm, sync vs async, key shape, TTL and invalidation policy, who computes what (backend vs frontend, stored vs derived), push vs poll, one endpoint vs three.
2. **Every dependency and backing service.** Each library and each external system the scope leans on is a decision to flip: this codec vs that one, this store vs that one, this algorithm's library vs another's. The alternatives come from research in step 5.
3. **Every recorded decision and its reopen trigger.** `DESIGN.md` decision records and `COMPROMISE:` comments seed the list. A recorded decision whose reopen trigger the step 1 read does not meet is listed as **settled**: named, its record cited, not measured, unless the user pins it at the checkpoint. One whose trigger is met is a live decision, and the new fact that met the trigger is stated on it. A fact is new when the record's author could not have priced it: check its date against the record's commit (`git log -S`). A trigger met by an older fact is badly worded, not met; the decision stays settled and the trigger's wording goes to the open questions. **A record settles only what it priced.** A mechanism its Alternatives never considered — often one a newer fact made possible — is a live decision beside it, citing the record as context. A choice fixed by a commit message or a code comment but by no record is *not* settled: it is live, and its source is cited as prior art. A `COMPROMISE:` comment with no reopen trigger is settled unless a fact newer than the comment bears on its price. A flip that re-proposes a priced-out alternative with a fresh argument and no new fact is the record being re-litigated, not a decision.
4. **Capabilities the code never took.** What the scope could do and does not: negative caching, stampede protection, observability, batching. Each is a decision whose keep option is "absent". A capability a primary goal already requires is not a decision: the keep fails that goal, and it is a defect for **Found along the way**. Bounded by the non-goals: an `[excluded]` capability is listed only when its `Reopens when:` line is met.

**The run's decisions are `C<n>`; `D` belongs to records.** A `DESIGN.md` gives its decision records permanent `D` ids, so the tree uses its own prefix and the two never collide. Cite a record with its document, always — `devices:D12`. A bare `D<n>` in the report is a defect, and the step 7 check flags it.

**Each decision carries:** an id `C<n>`, with `C<n>.<m>` for its options where more than one flip exists (`C2.1 share Go values`, `C2.2 clone Go values`); the keep option, stated as what the code does today; a `Goals:` line naming every goal and frozen contract it touches by `<document>:<id>` — keeps judged by reasoning included, since the checkpoint's lists are built from these lines; a **`depends on`** link to any decision whose verdict changes its options or its price, and a **`moot if`** link naming the upstream option that removes it (`C3 moot if C2 = C2.1`); its **evidence plan** — the cheapest kind from step 5 that can settle it, and the instrument and settings it takes (which benchmark and `-count`, which query at which data size, how many simulation replicates); and its **leverage**.

**Where a flip's code lands outside the scope** — it rewrites a type another package owns, or needs new mechanism in a package the binding set does not cover — it is a scope extension. Two changes outside the scope are not extensions: a dependency flip's edit to `go.mod` and `go.sum`, and the call-site edits a change to the scope's own API forces on its consumers. Those callers are breakage, priced in the ledger. Load that package's `DESIGN.md` now, if one exists, and carry the extension to the checkpoint. Do not work around the gap by recommending the flip "under an assumption".

**Rank by leverage.** Leverage is the expected size of the flip's consequence on the axes of step 6, weighted by how much of the scope and how many consumers it touches. Order the list by it, no cap. Leverage orders the report and the dispatch; it does not decide which decisions get evidence. **Evidence goes where it can change a verdict.** For each live decision, name the result that would flip its verdict. If a plausible result would, plan the cheapest evidence that can produce it — low-leverage decisions are often settled by a five-minute probe, and past runs found much of their value there. If no plausible result would — the gate already closes every flip, or the read of the call sites already settles the question — the read decides it, and the decision is marked `reasoned` with the sentence that settles it. Evidence that can only confirm a verdict the read already fixed costs time and changes nothing: one past run spent twenty minutes running seven test trees to confirm a keep its own read had settled. Links make the tree: a decision is a root when nothing it depends on is live.

**No time estimates in hours.** Past estimates were 4 to 20 times too high and made cheap work look like a request for permission. State the instrument and its settings; that is the cost a reader can check.

## Step 4 — The checkpoint

One `AskUserQuestion` round, and the only one the run makes on its own. **It asks only what the run cannot decide itself:**

- **Goal changes** from step 2 — every promote, demote, add, retire, reword and descendant drop. Each one names the decisions whose verdict it decides ("while G5 stays primary, C1.1 is out and is not measured"), because that link is what the user is really choosing. Build that list from the tree, not from memory: search the `Goals:` lines for the id and list every decision that cites it, no more and no fewer, marking which of them the change would move. Write "none" when nothing cites it. Treat an `F` id the same way.
- **Freeze lifts** — each frozen contract a live flip needs to change, with the authority that owns it.
- **Scope extensions** — each flip whose code lands outside the scope, and each change to behavior in an uncovered file.
- **Reopened exclusions** — each `[excluded]` non-goal whose `Reopens when:` line is met.

Ask for a freeze lift, a scope extension or a reopened exclusion only when the flip it unlocks gains on some axis on the read; a flip that would gain nothing even with it loses at the tie-break without a question. None of the three is ever marked Recommended — the run's own flip gains from it — but each one states what it unlocks, with the figure where the read already has one and the evidence plan that will price it where it does not, so the user can grant it knowingly.

**A contested goal change is never marked Recommended.** A change is contested when its readings favor different options of a live decision — loosening a goal so a flip passes the gate, dropping a requirement the keep fails, dropping a descendant goal, or adding a goal that some live option would fail. **Ask a drop or a loosening only when it decides the flip**: gate the flip on every other binding goal first, and if one of them already closes it, the question is moot — name the goal that closes it in the tree instead. The run has a stake in that answer. State both readings plainly, say which option each leaves open, and let the user choose unprompted. One past run recommended loosening a goal to unblock its flip; the user accepted it in two minutes and reversed it as a recorded decision two hours later. A change is **evidence-forced** when the code or a probe settles it and no option gains from it — a dead `Evidence:` path, a deleted consumer, a goal line that misstates what the code does in a way no verdict turns on. An evidence-forced change may be marked Recommended, and it still names every decision in the tree that cites the goal, so the user can check the "no option gains" claim.

**Fit the checkpoint into the tool's four questions** in this order: the evidence-forced changes as one multi-select question (single-select with a "leave as written" option when there is only one); each contested change as its own question, highest leverage first; the freeze lifts, scope extensions and reopened exclusions as one multi-select question. A multi-select that would hold one item gets a second option, "leave as is", so it can be answered. A question takes at most four options: when they outnumber them, make each option one kind — freeze lifts, extensions into another package, changes to uncovered files, reopened exclusions — and name its members in the description. If the contested changes do not all fit, hold the rest as written, judge their decisions under both readings, and carry them to the open questions.

**The tree is shown, not asked about.** Put the ranked tree — each entry with its options, links and evidence plan, settled decisions at the end with their record cited — in the message that precedes the tool call. Say that the run will proceed with it as shown, and that the user may strike, add or pin in the free-text answer. Do not ask "which decisions should be struck" and do not ask for a measurement plan. **If nothing needs asking, do not call the tool**: show the tree and proceed.

**Read a delegated answer as a job for the evidence.** When the user answers "up to you", "gather data to decide", or leaves a question blank:

- For a goal change: hold the goal as written, judge the decisions it gates under both readings, and give the verdict on the goal change in the report as an open question with a recommended default. Write nothing to a `DESIGN.md` for it.
- For a freeze lift or a scope extension: it is not granted. Judge the option that stays inside the scope and the freeze. Report the extended option as a fork beside it, with its evidence, so the user can grant it in `/plan`. **One exception: a keep that fails a primary goal must be repaired.** If the sound repair needs the extension, recommend it as `flip, requires <extension>` rather than settling for a weaker in-scope workaround, and say so in the verdict.
- For the tree: proceed as shown.

A scope extension or freeze lift that evidence reveals after the checkpoint is treated like a delegated one: not granted, and reported as a fork with its evidence.

**Write back only the confirmed goal changes**, to the file that owns each goal, which is not always the anchor, so each file stays the single source of truth for what it governs. A new goal takes the next unused `G` number in its file; a retired one frees nothing; a promotion or a demotion moves the entry and changes no id. Nothing else reaches a `DESIGN.md` in this run — not a record's `Serves:` line, not a non-goal's trigger, and not a secondary goal that a recommended flip trades away. If a confirmed change forces a knock-on edit elsewhere, list the edit in the report for the follow-up. A traded secondary goal is the same: the ledger names the tier change — retire it, or move it to Non-goals as `[no guarantee]` — and the change that implements the flip makes it. Make the change on a branch off `dev`, never on `dev` itself (`git switch -c docs/<scope>-goals dev`), and commit it with a Conventional Commit message, so the next session finds it instead of an uncommitted diff. Everything after the checkpoint runs without a question.

## Step 5 — Gather evidence

**Where each decision is judged.** A decision that needs only reasoning, or evidence already in the notes, is judged in the main context — a sub-agent for it costs more than it saves. Each decision that needs research, a probe, a measurement or a prototype goes to its own sub-agent, which returns one **evidence record** per decision and nothing else — no files, no branch. Probes that share one instrument, such as one scratch database, may go to one sub-agent. The main context holds the tree, the verdicts, and the scratch notes.

**Dispatch.** Research and probe sub-agents run in parallel. **Measurement sub-agents run one at a time**: this box is shared and noisy, and two benchmarks on it at once measure each other. A probe whose figure CPU load does not move — allocated bytes, retained heap, a count — is not a measurement for this rule and may run in parallel. A decision that `depends on` another is not held back: when the upstream has few live options, brief it with each and have it judge under each, and choose the matching result once the upstream verdict is in. Where that is too costly, dispatch it after the upstream verdict and tell it the verdict.

**The sub-agent's brief** states: the decision and its options; the goals it touches and their tiers, with the `Loss:` and `Measure:` lines; the workload to use; the evidence rules below; and the record shape. Any count or figure the brief passes on from the main context is marked unverified, so the sub-agent checks it rather than builds on it. A sub-agent that builds a prototype or runs a benchmark against repo code is given `isolation: "worktree"` — a sub-agent may otherwise check out or modify the shared tree. **A sub-agent returns only when its record is complete.** It runs its benchmark in the foreground or waits on it; a sub-agent that starts a background run and reports "finished" hands the main context an empty record and a wait loop.

**Evidence kinds**, weakest last. Each record states its kind; a verdict may rest on `measured` or `probed`, and on `researched` or `reasoned` only where the question is one of fact or logic. A probed count that bounds a gain — "at most one Redis round trip per event" — settles a performance question without a measurement when the bound is far below anything an axis would notice.

- **`measured`** — a paired comparison of keep and flip at a workload the record states (sizes, cardinality, hit ratio, concurrency, and where the workload came from). The pairing depends on the instrument:
  - *Go code*: the keep and the flip run interleaved on the same benchmark, summarised with `benchstat`. `-count=N` does not interleave: it repeats one variant back to back. Build each variant's test binary once (`go test -c`), then loop N rounds that run each binary with `-test.count=1`, appending to one file per variant, pinned to the same cores (`taskset`). Prefer the scope's own benchmarks; add a scratch benchmark only where none covers the flip, and record its command.
  - *A database query*: run the keep and flip queries interleaved against the same scratch database, at least ten runs each, and report the median and spread of time, rows read and peak memory. Take the figures from the database's own counters (Postgres `EXPLAIN (ANALYZE, BUFFERS)`, `pg_stat_statements`), not from client wall-clock.
  - *A statistical or behavioral property* (detection power, false-alarm rate, error bounds): a seeded simulation or replay with the replicate count stated, both options on the same seeds, and a sweep over every tuning parameter the option introduces. A figure taken at one tuning value is `indicative`.

  Separate runs drift, so back-to-back single runs are not a pair. **Know the noise floor before a figure decides.** A change to inlining, field order or function layout moves code addresses, and a relink of unchanged code can move a hot loop by 5–10%. When a verdict turns on a figure within about 20% — twice the layout range — first build the keep twice with different padding and pair those two builds; a delta inside their spread is a tie. A larger delta needs no control. The padding must be code the binary keeps — the linker drops an unreferenced function. When the verdict does not turn on the figure, a delta of a few percent is a tie and goes to the tie-break — do not spend a control run on it. A production profile bundle, where one exists, sets the workload; otherwise the record says the workload was chosen and why.
- **`probed`** — an exact answer from the code itself: a compile check, a size or alignment probe, a counting test, a worked computation, a grep over callers, or a one-sided figure that CPU load does not move (the heap a crafted input leaves retained). It decides as firmly as `measured` when the question is exact ("does this compile", "how many callers pass a borrowed value", "what does the formula give at the boundary"). **A probe that claims "every" or "none" states what it covered**: which builds (hosted, `cloud`, the app-billing tree), which process roles (`serve`, `worker`, Kafka mode), which config switches (an unset key, a disabled feature), and for a schema claim, both migration histories (`cmd/hosted/migrations` and `cmd/cloud/migrations`). A subscriber, caller or code path that exists only when a key is unset is still a case. A probe that covered one variant is `probed` for that variant only.
- **`researched`** — the repo first (in-repo use of the alternative, `go.mod`, `vue/package.json`, earlier records and their evidence files, session memory), then the web and library docs (`WebSearch`, `context7`) for an algorithm's published properties, a library's guarantees, a service's limits. **Every external claim carries a URL or a doc reference**; a claim no source backs is stated as `unverified`, and the verdict says whether it turns on that claim. For a security mechanism, prefer the established construction and cite it (an OWASP cheat sheet, an RFC); a construction of the run's own invention adds bug surface and must win on another axis to be recommended.
- **`reasoned`** — the argument in three sentences.
- **`indicative`** — any figure that fails its instrument's pairing rule: an unpaired run, a single timing, a one-shot program, a single tuning value. It is reported, marked, and never carries a verdict alone.

**Use a production-shaped workload.** Before a test fixture, look on the host for real inputs — a full data file, a dev database with volume, a profile bundle — and say which one the record used. **Check a workload assumption before you rest a verdict on it.** "Counter-backed SLOs dominate" or "most values are short" is a fact about the data, and a query against a dev or production-shaped dataset often settles it in minutes. Where it cannot be checked, compute the break-even — the hit ratio, share or size at which the verdict flips — and state it beside the assumption, so the open question asks the user about one number they can check. A verdict that holds only on one side of that break-even is recommended under the assumption, never as a plain flip.

**Prototype rules.** A prototype exists to answer one named question, not to start the implementation, and it is built only where a probe, research and reasoning cannot settle the flip. Time-box it; build only the load-bearing slice. It lives outside the working tree — the session scratch directory for standalone programs, a throwaway git worktree when it needs the repo's code, and scratch databases on the local docker-compose Postgres/ClickHouse for storage and query experiments (never the shared dev or test databases). **The main checkout is never touched**, not even by a test that is written, run and deleted. **A shared database is never changed**: never run a migration, a `task …:migrate`, or DDL against the dev or test databases, which other sessions use at the same time. If a probe needs the repo's tests and the test database is behind the worktree's migrations, the probe is blocked: report it as blocked, or rerun it against a scratch database. A prototype is never committed and never left on a branch, and its code quality is irrelevant — only the answer it produces enters the record. Prefer a `clickhouse-client` experiment or a 50-line scratch program to anything larger. A fresh worktree holds empty submodules, and `go` cannot load the module until `pkg/tempo` is filled: run `git submodule update --init` in it, or link the main checkout's copy, before the first build.

**What evidence turns up that the tree did not hold.**

- *A new option or decision* — a flip nobody listed, or a choice the read missed — enters the tree with an id and is judged like the others. If its evidence cannot reach full strength before delivery (a simulation at one tuning value, a single seed), it is **provisional**: it is reported as a fork with its evidence, never as the headline verdict. One past run recommended an option invented late and tested at one setting; a sweep the next day found a fraction of the claimed power.
- *Two records that contradict each other* — resolve the contradiction by reading the code or rerunning the instrument, not by inventing an option that reconciles them.
- *A defect or risk outside any decision* — a correctness bug, a security exposure, a memory peak — goes to the report's **Found along the way** section with its evidence. The checkpoint is over; do not ask about it, and do not bury it in open questions.

**The evidence record**, one per option judged:

```
Decision: C3 — value codec
Option: C3.1 gob (keep: msgpack)
Evidence: measured | probed | researched | reasoned | indicative
Workload: <sizes, cardinality, hit ratio, concurrency; source of the workload>
Result: <raw benchstat table or query-log figures, or the probe's output, or citations, or the reasoning in three sentences>
Gate: primary <doc:id> held/dropped …; frozen <doc:id> held/needs lift; non-goal <doc:id> served/not served
Consequence: latency … | store load … | memory … | code +/− … | bug surface … | ops … | <Measure: axis> …
Breaks: <contract or persisted shape, affected party>
Requires: <a prerequisite change or release that must land first, or "none">
Assumes: <anything outside the step 1 stack, and each unchecked workload assumption>
Risk: <the main unknown>
```

**Clean up** before delivery: remove every worktree and scratch database the run created, and stop any benchmark still running. Check with `git worktree list`; a worktree a sub-agent left behind is listed in the report for the user to remove if the run cannot.

## Step 6 — Judge each decision, then assemble the assignment

**A better choice is defined, not felt.** For each decision, in dependency order:

1. **The gate.** An option is out when it drops a primary goal (a descendant's only with the user's word from the checkpoint), serves an `[excluded]` non-goal, takes in a `[boundary]` responsibility from outside the scope, changes a frozen contract the user did not lift, or lands outside the scope in an extension the user did not confirm. One option with an unconfirmed extension passes: the repair of a keep that fails a primary goal, where a delegated answer applies step 4's exception. It passes as `flip to C<n>.<m>, requires <extension>`, and the user grants the extension in `/plan`. The gate is pass or out, never "partly": an option whose reach is unclear is split into a narrow option that passes and an extended one that is judged on its own. The keep option can fail the gate too — a stale contradiction the checkpoint confirmed — and the output says so.
2. **The consequence.** Among options that pass, the evidenced consequence for the app decides, stated on the **axes**: latency, backing-store load, memory, code added or deleted, bug surface, security exposure, ops cost, plus any axis a binding goal's `Measure:` line names. A flip wins on the axes only when its evidence supports the claim at the record's stated strength: a `measured` or `probed` figure beats a `reasoned` one, and an `indicative` figure decides nothing on its own. Where a measured cost stands against a benefit on another axis — a slowdown against deleted code — and no `Measure:` line sets a budget, the trade is the user's: recommend under a stated assumption (below).
3. **The tie-break.** When the axes do not separate the options, rank lexicographically: **less mechanism** (fewer packages, tables, endpoints, background jobs, config knobs, contracts, and the code that upholds them), then **secondary-goal coverage** (compare the priced losses, not the count of goals retained), then **lower breakage cost** from the ledger.

**Where a traded secondary goal counts.** A secondary goal whose loss the run measured on an axis — a latency or memory figure — is part of step 2, budget or not; where no budget weighs that figure against the benefit, the trade is the user's, as step 2 says. A secondary goal whose loss has no figure — an ops signal, a convenience — counts at the tie-break, after less mechanism. So deleting the mechanism that serves such a goal wins a tie by design: deletion is the benefit this skill exists to find, and the ledger prices the loss. Do not move the loss onto an axis to avoid that result. A flip that trades a secondary goal wins without asking; it is not a fork for the user, and the goal is not raised at the checkpoint.

Each decision gets a **verdict**: `flip to C<n>.<m>`, `flip to C<n>.<m>, requires <prerequisite>` (a change or a release that must land first, named in the ledger, or the scope extension a primary-goal repair needs), `keep`, `moot` (with the upstream option that made it so), or `settled` (record cited, not judged). A verdict states the sentence or the figure that decided it and the first of the three steps that decided it, and, for each losing option, the sentence that killed it.

**Assemble the assignment.** The recommendation is one consistent set of verdicts across the tree: every `depends on` link was judged against its upstream's final verdict, every `moot if` is honored, and no two verdicts assume different states of the same mechanism. Walk the links once more after the last verdict; a downstream verdict judged under an upstream assumption that later changed is re-judged, not patched. **Where two or more flips each claim a measured gain on the same path**, measure the whole assignment against the keep once, on the same instrument — in the same round as the single flips when they share a function, so it costs no second round: per-flip gains do not add, and a headline gain must be one the run took. Flips that won as ties need no combined run — there is no gain to double-count.

**When a decision is the user's.** Where the options trade on a dimension only the user can weigh — how much migration pain is acceptable, whether a behavior change users would notice is tolerable, which future to build for — the run does not ask. It recommends under a stated assumption, carries the other assignment beside it in the output, and lists the fork under the closing open questions with its default.

## Step 7 — Deliver

The report is what the user carries into `/plan`. Lead with the answer, keep bookkeeping out of the way, and leave the next session nothing to re-ask. It holds:

- **The verdict** — three to six lines: what flips, what stays, what it buys on the axes with the figure that says so, and what it breaks. When a fork beats the recommended assignment on an axis — an option held back by an ungranted extension, a freeze, or a delegated goal — name it here with its figure and what the user must grant. The biggest finding of the run never sits only in a fork or an open question.
- **The decision list** — the live decisions in rank order, each with its options, evidence kind, pros and cons carrying the figure or the citation, the gate result, the verdict, and its links. Keeps judged by reasoning alone go in one table, one row each. Settled decisions get one line each, with their record cited.
- **The recommended assignment** — the verdicts as one set, how the app is better for it on the axes, and the combined figure where step 6 took one.
- **The breakage ledger** — one for the whole assignment: every contract it breaks and every secondary goal it trades, who is affected (from step 1's consumer classification), each prerequisite a `requires` verdict names, and a migration sketch in a sentence or two (backfill, tolerant-reader window, config deprecation via the pointer-field rule, or a clean cut). Name each traded goal by `<document>:<id>` and price it by its `Loss:` line. Persisted-data breaks say what happens to existing rows. A frozen contract the user lifted appears as its own line with the `F` id and the authority that has to agree.
- **What gets deleted** — mechanisms, tables, endpoints, dependencies and knobs the assignment makes unnecessary. Deletion is a first-class benefit; enumerate it.
- **Found along the way** — defects and risks, each with its `file:line`, its evidence and a recommended action. A keep that fails a primary goal is a defect too: list it here with a pointer to its decision, so this section alone names every defect. When one outranks every flip, it leads the verdict. Where the fix has real options, list them in a line each and mark the one recommended. A sentence of a `DESIGN.md` **Solution** that the code contradicts goes here too, with the corrected sentence for the follow-up to land.
- **Decision records (drafts)**, ready to paste. The change that implements the assignment lands them, per the repo convention; name the file each belongs in — the one whose scope the decision spans. `design-doc/DESIGN-FORMAT.md` holds the record and evidence shapes.
  - *Corrections.* When a probe refutes a sentence in an existing record — an Alternative priced on a wrong premise — draft the corrected paragraph beside the record's id. The change that implements the assignment lands it; this run edits no record.
  - *Which decisions get one.* Each recommended flip, and each keep whose flip passed the gate and lost on a `measured` or `probed` figure — a count that leaves the flip nothing to gain counts — when no record covers it yet — that keep is the one a later run would re-measure, and the record stops it. A keep judged by reasoning alone gets none, and neither does a settled decision or a keep whose every flip failed the gate: the goals already say why.
  - *Ids.* A draft writes ids as its target file will hold them: bare for its own document (`Serves: G1`), qualified for any other. `check_report.py` reads a fenced block holding **Decision.** as a draft and allows its bare ids.
  - *A flip that only deletes* — a call, a line, a knob — leaves no code for a `COMPROMISE:` comment. It gets a record only when a maintainer could plausibly add the thing back; otherwise it gets none.
  - *Where.* Draft where the blast radius is: a decision contained in one file or one function is drafted as the text of a `COMPROMISE:` comment on that code, not as a `DESIGN.md` record. Its deciding figure goes in the comment, and its raw output goes in the report, not in an evidence entry.
  - *Shape.* The five-part record: the title, which states the decision; `Serves:` and `Trades:` by goal id; **Decision** opening with the choice and its decisive reason; **Alternatives** naming each losing option with the one sentence or figure that killed it; **Consequences** naming what becomes harder; the *Upheld by* list; and **What would reopen it**. One decision per draft, within the 60-line budget, with no date and no commit hash. It takes no `D` id — the landing change claims the next unused one — so the draft and its evidence heading both write `D?`.
  - *Honesty.* A flip that makes anything measurably worse trades something, so `Trades: none` next to a measured cost is wrong. A draft claims no more than its evidence covers: a fix for one version, table or case is recorded for that case, not raised into a rule for every future one. A reopen trigger states a threshold only where a `Measure:` line or a run figure supplies it.
  - *Evidence.* Each `measured` record comes with its evidence entry: a dated `###` entry with the `Workload:` line and the **raw** benchstat or query-log output pasted in full, and the ledger of flips that lost, ready to go under the record's heading in the evidence file of the document that owns the record: `DESIGN-EVIDENCE.md` beside a `DESIGN.md`, and `{topic}.evidence.md` beside `docs/design/{topic}.md`. The scratch directory does not outlive the session. The record keeps only the figure that decided, with an `Evidence:` line.
- **Open questions** — each fork recommended under an assumption, each goal change the user delegated, and each follow-up the run could not settle. Every question states the recommended default and the evidence behind it, so the next session can accept it without re-deriving it.
- **The handoff prompt** — a short prompt the user can paste into `/plan` in a fresh session: the flips to implement in order, the prerequisites, the records to land with each, and the saved report's real absolute path. It holds no placeholder; a fresh session cannot resolve one.

**Save the full report** to a file in the session scratch directory as well as printing it, and give the path: the user clears the context before implementing, and the printed report is otherwise the only copy.

**Before you deliver, check the saved report mechanically**, because these slips recur in every long report:

```sh
python3 .agents/skills/redesign/scripts/check_report.py <saved report path>
```

It fails a bare `G`, `N`, `F` or `D` id, a qualifier that names no document (`invite:G1` where the document is `core/invite`), and a handoff path that does not exist or still holds a placeholder. Fix each failure and run it again. It also lists every "every", "all", "only", "none", "no caller" and "0 of N" line: read each, and make each claim about the code state the builds, repositories, roles and config switches it covered, not only the headline one.

**Implementation detail stays out of `DESIGN.md`.** The ledger, the deletions and the migration sketches are the *response to the user*. This run writes only the confirmed goal changes from the checkpoint; the follow-up that implements the assignment lands the records and rewrites the **Solution** sketch in the same change. A record names the mechanism it chose, because the choice is its subject, and names files in the *upheld by* list, because an editor needs them. It states the price, not the procedure. One test settles a borderline sentence: would a refactor that keeps this design intact make the sentence false? If it would, the sentence is implementation, and it belongs in `README.md`, in `doc.go`, or in the code itself.

## Self-check before writing the output

- The entry gate ran over the whole binding set found by `design-doc/SCOPE.md`, and the output names the uncovered part of the scope; where nothing was covered, the run stopped at `/design-doc`. An ambiguous topic was asked about, not narrowed.
- Every goal and record is named by `<document>:<id>` throughout, tree nodes are `C<n>`, and `check_report.py` passes. No list was renumbered.
- The run asked at most two times: the scope question `SCOPE.md` allows, and the checkpoint. The checkpoint asked only goal changes, freeze lifts, scope extensions and reopened exclusions; evidence effort was never asked; no contested goal change was marked Recommended; a delegated lift or extension was judged as not granted. Delegated answers wrote nothing to a `DESIGN.md`, and the confirmed changes are committed on a branch.
- Every decision came from one of the four sources, has a keep option, states its links, and carries its evidence plan and leverage. Settled decisions cite a record, not a commit.
- Every verdict rests on `measured` or `probed` evidence, or on research or reasoning for a question of fact or logic; every `measured` record meets its instrument's pairing rule; every external claim carries a citation or is marked `unverified`.
- Every option was gated pass or out before it was scored; every verdict names the figure or sentence that decided it and the step that decided it, and every losing option its cause of death.
- The assignment is consistent: dependent verdicts match their upstream's final verdict, `moot if` honored, and a combined figure taken where measured flips share a path.
- No production code changed. No prototype touched the main checkout, a commit or a branch; every worktree and scratch database is gone. The only files edited are binding `DESIGN.md` files, for confirmed goal changes, on a branch off `dev`.
- No frozen contract was traded as a priced cost; no goal was re-tiered or reworded to let a flip win; an `indicative` or `unverified` figure or a `provisional` option decided nothing alone; a flip that assumes beyond the step 1 stack carries that as a flagged cost.
- The breakage ledger covers API, wire, persisted data, config, dependencies, prerequisites and traded secondary goals, each with an affected party and a migration sketch.
- Drafted records exist only for flips and for measured or probed keeps without a record; a one-file decision is drafted as a `COMPROMISE:` comment; each claims no more than its evidence, its `Trades:` names every measured cost, and each `measured` draft carries its raw evidence entry.
- Every open question carries a recommended default; the report is saved to a file and ends with the handoff prompt.

## Output to the user

```
# Redesign: <scope>

## Verdict
<three to six lines: flips, keeps, the gain on the axes with its figure, what breaks>

## 1. Scope and goals
<the binding set marked anchor / ancestor / descendant / overlapping, with an overlapping document's intersecting files named; the covered and uncovered parts of the scope>
Goals a verdict turns on: <document>:G1 — <outcome> (primary) … <document>:G2 — <outcome> (secondary, priced against: <observable>) …
Frozen contracts in play: <document>:F1 — <shape>, frozen by <authority> …
Goal changes this run: <each user-confirmed, the file and branch it was written to; each delegated one, pointing to its open question; or "none">
Descendant goals dropped / scope extensions confirmed: <each, or "none">

## 2. Decisions (ranked by leverage)
### C3 — value codec   [measured]   depends on: C2   moot if: C2 = C2.1
Keep: msgpack via the scope's codec; every value crosses the codec on both tiers.
Goals: pkg/redcache:G1 (hit served under <budget>), pkg/redcache:G4 (values readable by the previous release).
Options:
  C3.1 gob — Pros: one codec fewer in go.mod; stdlib only. Cons: encode 2.1× slower, 1.6× larger payload (benchstat, 8 runs, 1 KiB structs, 90% hit; example figures).
  C3.2 JSON via pkg/json — Pros: shared with the wire layer. Cons: 1.4× slower than msgpack on the same pair; type loss on int64 keys (probed).
Gate: all options pass; no frozen contract touched.
Verdict: keep (decided by the consequence) — the flips buy no axis; gob's deletion of one dependency does not outweigh a measured 2.1× encode cost on the hot path.
Killed: C3.1 by the encode figure; C3.2 by the int64 loss.
Other reading: <only where a delegated goal change gates this decision — the verdict if the goal changes, and the open question that decides it>
### C1 — …
…
### Keeps judged by reasoning
| Decision | Keep | Why the flip loses |
### Settled
C7 — <title> (record: <document>:D<id>, trigger unmet)

## 3. Recommended assignment
<one line per decision: verdict and the figure or sentence behind it>
Better for the app: <per axis, with figures; the combined figure where one was taken>
Assumed where the fork is the user's: <assumption, and the alternative assignment beside it>

## 4. Breakage ledger — what breaks or is traded / who is affected / prerequisite / migration sketch
## 5. Deleted — what the assignment makes unnecessary
## 6. Found along the way — defect or risk / evidence / recommended action
## 7. Decision records (drafts) — each naming its DESIGN.md, with its raw evidence entry where measured
## 8. Open questions — each with its recommended default and the evidence behind it
## 9. Handoff — the prompt to paste into /plan; the saved report's path
```

If there are no open questions, say so.
