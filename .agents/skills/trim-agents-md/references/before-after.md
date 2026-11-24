# Worked trims

> The worked examples below come from the reference repository this project copied its
> conventions from. Read them as the shape of a case, not as paths in this tree.

Three calibration examples for [SKILL.md](../SKILL.md)'s step 3. Each shows the
verdict, the split, and where every part lands. The examples are modeled on real
content in this tree but are self-contained, so they do not go stale with it.

## 1. A fused bullet: payload + mechanism + rationale

The tree's common failure shape. One bullet welds the rule, the mechanism behind
it, and the design story that produced it.

**Before** (in `example/widgets/AGENTS.md`):

> - **A page declares its width cap; it never writes one.** The cap goes to the
>   page registry — `usePage({ width: 'page-4xl' })` — and the page binds the
>   answer to its own root with `:class="page.width"`. The app bar reads the same
>   declaration, so the row above and the body below share one edge. A page that
>   takes the app default declares nothing and binds the same value; a component
>   that owns no declaration reads `useActivePage()`. The registry settles each
>   field on its own, so something inside a page may declare only the width and
>   keep the name above it. We chose a registry over per-page CSS because twenty
>   pages had drifted apart, and a declaration the bar can read was the only way
>   to keep both edges aligned. A spec fails a static width class anywhere in the
>   tree.

**The split.** Apply the payload test: which sentences, if deleted, change what
an agent writes? The declaration rule and the spec that enforces it. The middle
sentences describe how the registry works — mechanism. The "we chose … because"
sentence is the design's story — rationale.

**After** — the AGENTS.md keeps one bullet:

> - A page declares its width cap in the page registry (`usePage({ width: … })`)
>   and binds `:class="page.width"`; never write a static width class. A spec
>   fails one anywhere in the tree.

The mechanism paragraph (how the registry resolves, who reads it, the
no-declaration default) moves to the module's `README.md` under a "Page
registry" section. The rationale sentence is parked verbatim in the run report
for `/design-doc`. Nothing was deleted: three parts, three homes.

## 2. A guide restatement, hiding one divergence

A subtree AGENTS.md that restates a shared guide at length. The restatement is a
DELETE — but diff before you delete, because one clause may differ from the
guide, and the difference is the file's only real content.

**Before** (in `example/store/AGENTS.md`):

> - A test must be deterministic in everything it asserts on. Take times from
>   the fake clock and randomness from the seeded source, and let the database
>   fill generated ids. Never assert on a literal fake value.
> - Tests never delete rows they wrote; a test is isolated by its fixture
>   minting a fresh project. **In this package the sweeper tests are the
>   exception: they truncate their own staging table, because the sweep's
>   correctness claim is "the table is empty after a pass".** Say so in a
>   comment.

**The split.** The first bullet restates the shared test guide's Determinism
section word for word — DELETE, citing the guide's line as the survivor. The
second bullet's first sentence restates the guide's no-teardown rule — but the
bolded clause *diverges* from it, with a reason. That clause is inviolable.

**After** — the file keeps the stock pointer plus one divergence bullet:

> Repo-wide test rules are not repeated here; see
> [Go test conventions](../../docs/go-test-conventions.md).
>
> - The sweeper tests truncate their own staging table — an exception to the
>   guide's no-teardown rule, because the sweep's correctness claim is "the
>   table is empty after a pass". Comment each truncate with this reason.

The wrong trim here is deleting the second bullet as "duplicate": a grep match
on "delete rows" is not a duplicate, the *diff* decides. The other wrong trim is
keeping both bullets because the second one is "mostly different".

## 3. A level move, with its allowance migration

A rule that constrains two sibling subpackages sits in one of them. It moves up
to the parent — and a citation it carries has an `allowedCitations` entry keyed
to the old file.

**Before** (in `example/alerting/notify/AGENTS.md`):

> - Channel payloads are versioned. Bump `payloadVersion` when a field changes
>   meaning, and keep the old decoder until every stored row is rewritten — the
>   incident path in `../incidents/` decodes the same rows. The old shape is the
>   one `notify_v1.go` held before it was deleted.

And in `docs_policy_test.go`:

```go
{"alerting/notify/AGENTS.md", "notify_v1.go"}: "a file named as deleted",
```

**The verdicts.** The rule binds `notify/` *and* `incidents/` — two siblings —
so it moves up to `alerting/AGENTS.md`. Two consequences travel with it:

1. Every backticked path re-resolves from the destination's directory. The
   relative `../incidents/` was written from `notify/`; from `alerting/` it
   must become `incidents/`.
2. The `allowedCitations` map is keyed by the pair `{doc, cited}`. The sentence
   naming the deleted `notify_v1.go` now lives in a different doc, so the entry
   migrates in the same commit:

```go
{"alerting/AGENTS.md", "notify_v1.go"}: "a file named as deleted",
```

**After** — in `example/alerting/AGENTS.md`:

> - Channel payloads are versioned. Bump `payloadVersion` when a field changes
>   meaning, and keep the old decoder until every stored row is rewritten —
>   `notify/` writes the rows and `incidents/` decodes them. The old shape is
>   the one `notify_v1.go` held before it was deleted.

The copy in `notify/AGENTS.md` is deleted, citing the new parent line as the
survivor. `go test -run TestDocCitationsResolve .` proves both consequences
were handled; forgetting either one is exactly the failure that test exists to
catch.
