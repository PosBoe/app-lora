# Terms in Vue component names

Read this file when the scope reaches `vue/`. It adds the frontend surface to steps 2 and 4 of the audit. The rename steps for a tag are in [renaming.md](renaming.md).

## A tag is a term

A component tag is a domain term with a shape word attached. The seven categories apply to the domain part exactly as they apply to a Go identifier. A hypothetical `<AlertSentinelPanel>` hijacks `sentinel` (category 2) whether it is a tag or a struct. The tag is the worse of the two: every template that renders it shows it, to a reader who has not opened the SFC and cannot see a godoc.

Read the tag in two parts before you judge it.

- **The shape word is not a domain term.** `vue/AGENTS.md` states that a shared component's first word is its shape, and that the word order matters because namespacing is prefix-based: `BtnCopy`, not `CopyBtn`. So `Btn`, `Dialog`, `Card`, `Panel`, `Field`, `Menu`, `Table`, `Picker` are the shape vocabulary, and they are never findings. Neither is the word order they force. A tag that reads oddly in English because the shape leads is the convention working.
- **The rest of the tag is the domain term.** Audit it against the same glossary as the Go side. `vue/` and the Go package that feeds it are two ends of one concept. A term settled at one end and not the other is category 6 — one concept, two names — and the wire key between them is the evidence.

Two frontend-only shapes to hunt. A Go-only sweep sees neither. (The names below are illustrations, not code in the tree.)

- **The tag disagrees with the wire key it renders.** The component reads `budget_burn` and calls itself `<SloExhaustionChart>`. A reader who greps the word they saw in the UI finds nothing on the backend.
- **A composable and the component that uses it name one thing differently** — `useSubjectHealth` rendered by `<SloStatusBadge>`. Both are in scope, and the fix is one word, not two.

## Sweep

Run these from `vue/`:

```sh
node scripts/component-tags.ts                                  # every tag<TAB>path — the names a template reader meets
node scripts/component-tags.ts --usages | sort -k2 -nr          # tag<TAB>count<TAB>path; the count is the rename budget
grep -rnE 'export (const|function) (use|create)[A-Z]' src       # composable names
grep -rnE 'const \w+ = (ref|computed|reactive)' src/<area>      # the words the templates bind to
```

`component-tags.ts` is the repo's own resolver. A tag derives from the path, so no grep for a filename gives you the name a reader actually types.

## Refute

**Does the new word change the prefix?** The auto-import namespace is flat and prefix-based. A tag whose first word changes belongs in a different folder, and that move is `/vue-organize-components`' work. Renaming the domain word inside an unchanged prefix is this skill's work. Report a rename that moves the component to another folder, and hand it off.

**Check the collision against the whole tag.** A duplicate tag does not fail the build. The plugin logs it, keeps the last one scanned, and exits 0. The loser then fails silently where it is called.

## Rename cost

Every template that renders the tag moves with it, and `--usages` prints that count. Under about ten usages the rename is cheap. It is also the rename a reader gains the most from, because a reader sees the tag at every call site and does not open the SFC.
