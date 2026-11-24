---
name: audit-terms
description: Audit existing domain vocabulary and price sourced renames before editing. Use only when the user explicitly invokes `$audit-terms` or names this skill.
---

# audit-terms

> The worked examples below come from the reference repository this project copied its
> conventions from. Read them as the shape of a case, not as paths in this tree.

Audit the words a scope uses for its own concepts. The deliverable is a short ranked list of terms a competent engineer in this field would not recognize — each with the concept it names, the recognized term that replaces it, the source that makes it recognized, and what the rename costs. Not a naming-style review, not a refactor, not a tour of the vocabulary.

The cost model behind every finding: a recognized term costs a reader nothing, because they arrive already knowing it. Every other term is a tax charged to everyone who reads the code after you, forever, and it is charged again each time a new person joins. A term is worth that tax only when it names something the field has never named.

**The audit runs in plan mode; renames follow approval.** Enter plan mode with `EnterPlanMode` before step 0 reads anything, unless the session is in it already. Step 0 through step 4 are read-only: no edits, no commits, no branch. Present the findings with `ExitPlanMode` and wait. The user's approval starts the renames, and it approves them item by item. If the harness has no plan mode, still make no edit until the user approves.

## Step 0 — Resolve the scope

The scope comes from the invocation argument and takes one of four shapes:

- **A Go package** (e.g. `/audit-terms ./system`) — scope = every file in the package.
- **A directory or a bounded context** (e.g. `/audit-terms cmd/migrate`) — scope = the directory tree and its `CONTEXT.md`.
- **A frontend area** (e.g. `/audit-terms vue/src/use`) — scope = those files, plus the backend fields they read, plus the tag of every `.vue` file in it. A component tag is an identifier like any other, and it is the one a reader meets first.
- **The whole repo** — expensive. Confirm the user means it, then work context by context off `CONTEXT-MAP.md` and report per context.

Resolve the glossary anchor first: read `CONTEXT-MAP.md` to find which context owns the scope, then that context's `CONTEXT.md`. A scope inside a context inherits its glossary; a term the parent glossary already settles is not this audit's finding.

If no `CONTEXT.md` owns the scope — common for a `pkg/` utility — the bounded context of step 1 is the package plus every package that imports it, because that is where its words are read. Pass `-importers` to the census tool (see Procedure) rather than building the list in the shell. Run the definition test on the package's godoc and `README.md` instead of a glossary, and say in the report that no glossary exists.

If no argument was given, or two readings of the scope give different file sets, ask the user before reading anything.

## When to apply

Apply when the user wants the vocabulary already in the tree audited: do the names in this scope tell a reader what the things are, or do they teach a private language first?

Do NOT apply when:

- The concept is being named *now*, while the model is built — `/domain-modeling`. That skill picks terms; this one audits terms already picked.
- The subject is the folder hierarchy that encodes component names, the taxonomy behind it, or the file moves that carry it — `/vue-organize-components`. **The words are still this skill's**: a component whose tag names a concept by a coined or hijacked word is a finding here, whatever folder it sits in. The seam is what the change is *about* — a wrong word, or a wrong place.
- The structure is wrong rather than the words — `/plan-ideal-refactor`. A badly placed responsibility does not get better under a better name.
- What the scope is *for* is unstated — `/design-doc` first. You cannot judge a term against a concept nobody has written down.
- The names are correct but the code they name is wrong — `/review-code`.

## The bar for "non-idiomatic"

"I would have called it something else" is not a finding. A term is a finding only when **both** hold:

1. It falls in a catalog category below, **and**
2. a recognized term for the **same** concept exists, and you can name it **with its source** — a standard, a paper or textbook, a widely used library's API, a neighbouring product's UI, or this repo's own vocabulary elsewhere.

No source, no finding. A term you dislike but cannot replace with a sourced word is not non-idiomatic; it is a coinage the field left room for, and the report records it as sound.

One kind of finding is exempt from rule 2, because it needs no replacement to be
real: **one word naming two concepts** (category 6). The fix may be to qualify one
of them, or to split them into two words, and either way the defect is proved by
the two concepts alone. Source a replacement if you propose one; do not withhold
the finding for lack of one.

Three standing exceptions kill or downgrade a finding:

- **The field's own word is rare.** `transductive`, `contamination`, `burn rate`, `EVT`, `cardinality` — rare in English, standard in their field. Flagging these is the audit's main false positive.
- **The name is pinned by an external contract.** A JSON key an SDK sends, a column an existing migration created, a third-party API being emulated, an OpenTelemetry attribute name. The term may still be poor; the finding becomes "poor and frozen" and its rename cost says why. A column is frozen only when a reader outside this repository reads it. A column that this repository creates and upgrades itself is a rename cost, not a freeze: `migration_step_records.form` looked frozen, and `migrate.Init` renamed it in one statement.
- **The idiom was rejected on purpose.** A `DESIGN.md` entry, an `_Avoid_` line, or a `COMPROMISE:` comment that names the recognized word **and states why it lost**. That is an answer, not an oversight. Downgrade to "confirm the reason still holds", or drop it. The exception needs both halves in the repo's own text. An `_Avoid_` line that lists the recognized word with no reason is a claim under audit, like a citation (below). **Never write the missing reason yourself.** An earlier audit dropped `form` because `type` was on its `_Avoid_` line, and it gave its own reason: "`type` is a Go keyword". The reason was false for `StepType`, and goose uses `TypeGo` and `TypeSQL` for the same values.

**A stale citation of the term can pass `docs_policy_test.go`.** That check
records struct fields alongside types and funcs, so a comment writing
`slo.Status` survives when no such type exists but some struct in the package has
a `Status` field. A term audit therefore cannot lean on the check to have found
the term's dead citations: grep them yourself, and confirm the **kind** of the
symbol the citation resolves to.

**A citation already in the repo is not a fourth exception.** A glossary that
says "this is the field's own word, see [paper]" is a *claim under audit*, not an
answer that closes one. Audit it with the borrowed-term test in step 3. The most
expensive miss this skill has made was accepting one: `rung`, cited to
successive halving, sourced correctly, and still wrong — the citation was true
about the paper and false about this code.

**A split between two words is a claim too.** A glossary that gives one word to
the unit and a second word to the whole — "a step is *pending*, a migration is
*unapplied*" — reads settled. Check how the field uses **both** words before
you align code with the split. goose (`HasPending`), Flyway and Rails all say
*pending* for the migration. So the split gave the field's migration word to
the step, and the code already used *pending* the field's way in a dozen places.
An earlier audit aligned the former upgrade package with the split and made the drift worse.

## Catalog

What to hunt for, and why each is a red flag rather than proof.

### 1. Coinage

An invented word, or a rare abstract noun, for a concept that already has a name. Marks: the word appears in this repo and almost nowhere else; it is an `-ness`, `-ity` or `-age` noun built from a verb; the glossary entry is longer than the concept deserves because the headword carries no meaning on its own. `Worthiness`, once defined in `alerting/CONTEXT.md` as "the admissibility check" a detector runs before training, is the shape: the definition reached for *admissibility*, so *eligibility* was the headword all along.

### 2. Hijacked term

A word the industry already owns, given a private meaning here. The worst class, because the reader does not know they are lost — they read `sentinel` and confidently think "marker value", and nothing corrects them for an hour. `cmd/benchmark/CONTEXT.md` once called a control run an `axis sentinel` and a static lookup table an `axis ledger`; *control* and *registry* were both free, and both are now the headwords, with the old words on their `_Avoid_` lines. Check the industry meaning of every borrowed word: `sentinel`, `ledger`, `broker`, `journal`, `witness`, `oracle`, `arena`, `epoch`, `tenant`, `shard`, `barrier`, `store`, `pending` all carry one already. Check the common-English ones too. No word-level signal in step 2 fires on them, so only the definition test, the pairing test, and the full read catch them. `StepStore` named which database a step changes, while a Go reader, and goose's `database.Store`, read a store as the place that saves state.

### 3. Undecoded metaphor

A figure of speech the reader must translate before they can think. `cmd/benchmark` once named a broad exploratory pass `Scout`, and it sat in the same glossary as `Screen`, so the reader had to hold two metaphors and the difference between them; it became `Pilot`, the statistical word for a preliminary study. A metaphor is fine when it is the *field's* metaphor — `fleet` for a set of machines, `pipeline`, `sink`. It is a finding when it is private.

### 4. Private abbreviation

An initialism or clipping expanded nowhere in the scope. The repo's mandated short forms are not this (see the `attr` rule in `CLAUDE.md`), and neither is an abbreviation the field itself uses (`TTL`, `SLO`, `p99`). A finding is one this codebase invented and never spelled out.

### 5. Fiat narrowing

A common word given a narrower meaning here than it has everywhere else, with nothing marking the narrowing — "check" that means only one kind of check, "event" that excludes most events. The fix is to **qualify** the word, never to replace it: `CheckReport`, `AlertEvent`. Replacing a common word with an uncommon one to escape overloading is how category 1 is born.

### 6. Term drift, and its twin

**One concept, two names.** The glossary headword and the identifier that carries it disagree; two packages in one context name the same thing differently; or an `_Avoid_` line bans the word the field actually uses. Drift costs the reader more than either name would alone, because they must first work out that the two are one thing.

**One name, two concepts** — the same defect inverted, and the worse half. Here the reader has no signal at all: nothing looks inconsistent, so they carry the meaning they learned at the first site into the second and are silently wrong. Its tell is not a word, it is a **type**: two declaration sites where the same term names values of different types (`WidestRung` returning a `time.Duration`, `rungs` holding `[]builtRung`). Prose then inherits the ambiguity, and a glossary entry ends up describing one sense while the function it cites returns the other. Step 1's backwards walk is what catches it; no grep for the word itself will.

### 7. House name for a general concept

A universal thing renamed locally: a cache, a retry, a queue, a lock, a validator given a private label. `CONTEXT-FORMAT.md` already rules these out of the glossary; this catches them where they escaped into identifiers.

### Where a Vue component name carries a term

A component tag is a domain term with a shape word attached, and the seven categories apply to the domain part. When the scope reaches `vue/`, read [references/vue.md](references/vue.md): it says how to read a tag, the two frontend-only shapes to hunt, the frontend sweep, and the extra refutation for a tag.

## Procedure

The census tool in this skill does the mechanical parts. It reads Go syntax, not types, so it runs on a tree that does not build. Run it from the repository root:

```sh
T=./.agents/skills/audit-terms/scripts/termsites
go run $T sites <root>... <term>             # every declaration and parameter of the term, grouped by the type it names
go run $T -importers sites <dir> <term>      # the same, over the package and every package that imports it
go run $T surface <dir>                      # exported names, enum values, json keys, and the candidate checks of step 2
go run $T glossary <CONTEXT.md> <root>...    # headword pairing, the ambiguity index, the _Avoid_ lines
```

Add `-tests` before the subcommand to include `_test.go` files. The tool skips generated files. Every line it prints is a candidate for step 3, not a finding. Quote every glob you pass to `grep --include`: zsh fails on an unquoted `*.go` with "no matches found".

### Step 1 — Build the concept inventory before you read the names

The central failure mode of this audit is **acceptance**: read the name first and you will decide the concept *is* what the name says, and the name will then look correct. So invert the order.

For each concept in the scope, write one line in a scratch note: **what it is**, in plain language the field would use, derived from the code and the definition body, not from the headword. Cover the headword while you do it. Only then write down what the scope calls it. A concept you can state in five ordinary words that carries a rare name is the audit's whole subject.

Sources for the concept, in this order: the type and its fields, the function that produces it, the definition body in `CONTEXT.md`, the godoc, the call sites.

**Then walk the map backwards, term to concepts.** The inventory above finds a
concept wearing a bad name. It cannot find *one name worn by two concepts*,
because you wrote one line per concept and the second concept never got a line.
That is category 6, it is the most expensive kind to leave in, and it is invisible
to every grep in step 2.

The check is mechanical, so run it on every candidate rather than trusting your
reading. **Run it over the whole bounded context, not the scope.** The second
concept usually lives in a sibling package, which is exactly why the first one's
author never saw it; a walk bounded to the audited directory finds nothing and
reports the vocabulary sound. **List the term's declaration sites with the type of
the value each one names**, and compare:

```sh
go run $T sites <context root> <term>
```

It lists every type, func, method, field, const, var and function parameter whose name contains the term — including those in `type (…)` and `const (…)` blocks — grouped by the type of the value each one names. Two groups with different types are the signal. A type and the constructor that returns it are one concept; read each pair and decide.

Read the string literals it lists as well as the identifiers. A heading such as `"Pending migrations"` and a flag usage such as "list every pending migration" are the uses an operator reads. If you find such a use and leave it, report it as a finding. The tool reads Go only; for prose, SQL, config and Vue, grep without case:

```sh
grep -rniw '<term>' <context root> --include='*.md' --include='*.sql' --include='*.yaml' --include='*.vue' --include='*.ts'
```

Two sites disagreeing on the type is the finding, and it does not need a
replacement to be reported. `WidestRung` returned a `time.Duration` while
`AnomalyDetector.rungs` was `[]builtRung` — a quantity and a configuration, one
import hop apart, both called a rung. The prose then inherited the ambiguity: a
sentence about one read as a sentence about the other, and the glossary entry
described the configuration while the function returned the quantity.

Where the two senses are real, the fix is two words, not one better word. Name
each for what it *is* — the quantity for what it measures, the configuration for
what it configures.

### Step 2 — Mechanical sweep

```sh
go run $T glossary <CONTEXT.md> <context root>
go run $T surface <scope>
git log -S'<term>' --oneline -- <scope>      # who introduced it, and whether a reason came with it
```

`glossary` pairs each headword with the identifier beside it and prints the words the identifier drops; it lists, for a headword with no identifier beside it, the declarations that spell it. It sweeps the ambiguity index both ways (below). It lists each `_Avoid_` word with its stated reason, or `NO REASON`, and the identifiers that still spell it.

`surface` lists what a reader of the package meets — exported names with their types, methods, string enum values, json keys — and the first doc sentence of every type, which is the definition test's input for code. It then runs four candidate checks; each prints `none` when it finds nothing:

- **Type names reused for a value of another type** — a parameter, field or var named like a type in the package that does not hold that type (`type table struct` next to `table string` parameters). This is category 6 on names the exported surface never shows.
- **Clipped words whose full form the package also writes** — `sig` where the package writes `signal`. Category 4.
- **Look-alike pairs** — names one letter apart, or one word inflected, with a different kind or type.
- **Names that carry no concept** — a one-letter exported type, or a name built only from `Data`, `Info`, `Manager` and the like.

Each is a candidate: it becomes a finding only when it meets the bar above — a catalog category and a sourced replacement.

For a scope that reaches `vue/`, add the frontend sweep in [references/vue.md](references/vue.md).

The **definition test** is the highest-yield line: for each headword, and for each type's first doc sentence that `surface` prints, read the definition and look for a plainer word doing the explaining. Two of this repo's four known cases were caught by it alone.

The **headword-pairing test** is the cheapest, and it is the one a careful reader
skips because the answer is written down. `glossary` runs it. This repo's glossary names the
identifier beside the headword — **Subject hash** paired with `slomodel.Hash` —
so the defect is on one line, and comparing the two costs nothing:

- Read only the words. `Subject hash` against `Hash`: the identifier is missing
  *subject*. `Budget projection` against `Projection`: missing *budget*.
- A missing word is a finding when the word it drops is one the context's
  **ambiguity index** lists (below). It is taste when the word is merely absent.
- The package qualifier does not supply the missing word unless the package
  identifier actually contains it. `slomodel` does not contain *subject*.

Run it on every scope, before anything else. On one glossary it found four
findings where careful reading had found one.

Two more cheap signals: a word that appears in this repo and in no dependency, and a word whose plural or verb form nobody ever writes (people do not conjugate a word they do not think in).

Every hit is a candidate for step 3. A hit is a flag, not a finding. Then read the scope's `CONTEXT.md` and its exported surface once end to end anyway — the worst terms are grep-invisible, because they look like ordinary words.

### Step 3 — Find the recognized term

The core work, and the step that decides whether a candidate survives. For each candidate, find what the field calls this concept, **and name the source**:

- a standard or specification (OpenTelemetry, Prometheus, RFCs, SQL);
- a paper or textbook, where the concept is a research one;
- a widely used library's public API (scikit-learn, Grafana, Kafka clients, the Go standard library);
- a neighbouring product's UI, where the concept is user-facing;
- the schema and migration DDL, which often carries the domain's true word and is
  free evidence: `slo_subject_status_enum` typed the column while the Go type and
  the wire schema both said a bare `Status`. Read the migration before you
  conclude the code is the only authority;
- this repo's own vocabulary elsewhere, which is the cheapest source and the most binding.

**A context's ambiguity index is a source, and it binds code.** Where a
`CONTEXT.md` opens with a table of words that "bare, mean nothing", every row is
a rule with the repo's own authority behind it — `alerting/CONTEXT.md` states
that such a word is "never written bare where two of its senses are in scope —
**in code or in prose**". An exported identifier spelling one of those words bare
is a sourced finding, and the source is one grep away.

Sweep the index against the code both ways. A row with a bare identifier is a
finding. A word that is **missing** from the index while two packages in the
context declare it is also one: `projection` was on no row, and `alerting` held
both `slo.Projection` (a budget burn-rate forecast) and `detection.Projection` (a
metric point forecast). The index is a claim about the vocabulary, not a census
of it.

Search when the field is not one you can cite from memory — `WebSearch`, or `context7` for a library's API. Do not accept a replacement you merely find plausible; the report must survive the user asking "says who?".

If nothing names the concept, the candidate dies. Record it in the close-out as a confirmed coinage: the field left room, and the word stands.

#### The borrowed-term test

Run this on any term the scope **already claims** is the field's own — a glossary
entry with a citation, a comment saying "this is what X calls it". A citation
proves the word exists in the field. It does not prove the field's meaning is
this code's meaning, and that gap is invisible unless you look for it.

Decompose the source definition into its **defining attributes**, then check each
one against the local concept, in a table:

| Attribute of the term in its field | Here? |
|---|---|
| … | yes / no |

Every "no" is meaning the reader imports and the code does not honour. For `rung`
in successive halving: an ascending set of resource levels (yes), geometric η
spacing (no), a population of candidates at each level (no), a promotion decision
that prunes the losers (no). Two of four. The reader who knew the algorithm was
promised a selection mechanism this engine never runs.

Then ask the question that decides it: **strip the attributes that do not hold —
is the remainder still what the term names, or is it a different thing with a
name of its own?** If the remainder has its own name, the borrowing has failed
and the term is a category-2 finding however good the citation is. `rung` minus
the population and the pruning is a *resource level*, which multi-fidelity
optimization calls a **fidelity level** or a **budget** — so `rung` was naming
more than this code has.

Beware the near-miss the other way. Do not treat the remainder's name as a drop-in
replacement without checking it is the same *kind* of thing: a budget is a
quantity, a rung is a stage, and swapping one for the other silently changes what
the word denotes. Test a proposed replacement by substitution in a real sentence
from the scope. "The widest rung" → "the widest budget" does not survive it.

### Step 4 — Refute before reporting

For each survivor, try to kill it:

- Is the term the field's own rare word? Check the field, not English.
- Is it pinned by an external contract — a wire key, a persisted column, an emulated API?
- **Is the replacement already taken in this repo for a different concept? Test the compound, never the bare word.** A rename that creates a collision is worse than the coinage it fixes — but a generic word is not "taken" by being common. Grep the phrase the replacement would actually be spelled as. `budget` returns 500+ hits and looks fatal; `training budget` returns zero, and the 500 are `error budget`, `breach budget`, `promotion budget`, `byte budget` — every one already qualified, which is evidence the compound namespace is *open*, not closed. Qualifying a common word is the house rule the glossary states in its own voice ("**Always say which resolution**"; "name either one outside its own package … and qualify it"), so a bare-word grep contradicts the convention it is supposed to be enforcing. A real collision is the *same compound*, or a bare word this scope has reserved and says so.
- Was the idiom rejected on purpose? Check `DESIGN.md`, the `_Avoid_` lines, and `COMPROMISE:` comments.
- **"The qualified name would stutter" is the refutation most often got wrong.**
  Stutter is a string test on the **package identifier**, not a judgment about
  the domain: `http.HTTPClient` stutters because `http` is in `HTTPClient`.
  `alertvocab.SLOSubjectStatus` repeats nothing, because `alertvocab` does not
  contain *SLO*, *subject* or *status*. Spell both out and compare them before you
  accept the objection.
  Two corollaries:
  - **A field name is independent of its type name.** Renaming `Status` to
    `SubjectStatus` leaves `Status SubjectStatus`, so `Subject.Status` reads
    exactly as before. A rename never forces a stuttering field, so "the field
    would stutter" is never a reason to keep a bare type name.
  - **Renaming the package is usually the worse fix.** It moves the qualifier
    onto every export, including the ones already correct, and it collides:
    a package named `slosubject` gives `slosubject.Subject`, which stutters for
    real.
- Would the rename split one concept's name across a boundary — renamed in Go, unchanged in `vue/` or in the database? Then the finding's real cost is the whole set, and partial renaming makes term drift (category 6) where none existed.
- **For a component tag**, run the extra refutation in [references/vue.md](references/vue.md).

Label every survivor **CONFIRMED** (the concept is stated, the replacement is sourced, the collision check is clean) or **PLAUSIBLE** (the term reads private but you could not source a replacement well enough to defend it).

Do not report: spelling or capitalization, names that are merely long, the repo's mandated abbreviations (`attr`), generated code, or a term you would have chosen differently on taste with no reader cost behind it.

## Rename cost

Every finding states its blast radius, because that decides whether the rename is worth buying. Rank the report by reader cost, but state the rename cost so the user can decline the expensive ones on the spot:

1. **Glossary and prose only** — free.
2. **Unexported identifiers** — mechanical, one package.
3. **Exported identifiers** — name the packages that import them.
4. **A Vue component tag** — every template that renders it; see [references/vue.md](references/vue.md).
5. **JSON keys read by `vue/`** — both sides move together, in one commit.
6. **Enum values or columns persisted in Postgres** — needs a migration and a backfill. Usually the point where a poor term stays.

## Output to the user

```
## Term findings (highest reader cost first)

1. `<term>` — <glossary path:line; the identifiers that carry it> [CONFIRMED|PLAUSIBLE] (<catalog category>)
   Means: <the concept in one plain sentence, written before the name was read>
   Why it costs: <coinage / hijack / metaphor — and what a reader guesses instead>
   Recognized term: <replacement> — <source>
   Rename cost: <glossary only | N unexported identifiers | exported, M packages | wire key read by vue/ | persisted enum, needs a migration>

2. …

## Keep the name, fix the meaning
<names whose word is right but whose code or doc breaks the word's promise; the cheaper fix is
 the code or the doc, not a rename — say what to change, and hand a code change to its skill>

## Poor and frozen
<terms that fail the bar but an external contract pins; say which contract>

## Checked and sound
<each headword and candidate that survived, with its source, or "coinage the field left room for">

## Handed off
<items that belong to another skill — a structural defect, a re-foldered component — and which skill>
```

Every headword on the "sound" list must pass step 3 like a flagged candidate: name the source that makes it the field's word, or name it a coinage the field left room for. A headword judged sound by reading alone is not checked. That is how `Store` and `Form` passed an earlier audit.

Close with what was checked and found sound, naming the terms confirmed as the field's own ("`contamination`, `burn rate` and `EVT` are the literature's words and stay; 31 of 38 headwords sourced to OpenTelemetry or Prometheus"). A report may count what it read: the count travels with the audit that took it, and no file stores it, so the rule against counting what the tree holds does not reach it. A short list must read as evidence, not as omission. If the vocabulary is sound, say so plainly. Do not manufacture findings.

## Renaming after approval

The user approves item by item. Rename only the approved items. Two rules decide which name moves:

**Qualify the type; leave the member alone.** A struct field, a map key and a
wire key are already qualified by what holds them, so `Subject.Status`,
`Subject.Hash` and `measurement.projection` do not move when their types gain a
qualifier. Move a member only when the member's **word** is wrong rather than
under-qualified — `band` had to become `status` on the wire because a
band is an interval and the value was a label; `projection` inside a
`measurement` was the right word all along.

**The inverse costs more: a container that borrows its member's word.** Renaming
`slo.BudgetStatus` to `BudgetMeasurement` freed `MeasuredStatus` to be a plain
`Status`, because the container then said which status it was. While both were
called status, three doc paragraphs existed only to hold them apart, and readers
still asked which was which. When a type and one of its fields want the same
word, the field keeps it.

Then read [references/renaming.md](references/renaming.md) before you edit anything. It gives the order of the edits, the verification, the hazards of a bulk `sed` rename, and the three edits of a component tag rename.

## Relationship to /domain-modeling

`/domain-modeling` picks a term while the model is being built. This skill is the retrospective pass over terms already in the tree, and it is the one that prices the fix. Both obey the same rules, and the rules live in one place: `.agents/skills/domain-modeling/CONTEXT-FORMAT.md`. Read them there — this file says what to hunt and what it costs, not what a good term is.
