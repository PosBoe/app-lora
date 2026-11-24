# Renaming after approval

Read this file only after the user approves findings. Rename only the approved items.

## Order

1. Code identifiers — unexported first, then exported with their callers.
2. Vue component tags and composables — see [Component tags](#component-tags).
3. Wire keys and persisted values — only for items the user approved one by one. Never carry them along with a code rename.
4. The glossary headword last. **Move the old word to that entry's `_Avoid_` line, with the reason it lost.** That line stops the term coming back, and a reason is what lets a later audit trust it (an `_Avoid_` line with no reason is only a claim, as `SKILL.md` says under the exceptions to the bar). The line is also the handoff to `/domain-modeling`.

Make one conventional commit per term: `refactor(<pkg>): rename <old> to <new>`.

## Verify

- Compile-check both builds with the `go list -export -deps` form from `CLAUDE.md`.
- Run the renamed packages and their users with `task test:pkg`.
- For a wire-key rename, grep `vue/` for the old key, and move it in the same commit.
- A persisted enum value or column needs a migration. Follow the migration rules in the root `AGENTS.md` (`cmd/lora/migrations`). If the user did not approve that item, leave the term, and say so in the close-out.
- Run the census again for the old word: `termsites sites <context root> <old>`. The declarations section must be empty, or list only the other senses you meant to keep.

## Bulk renames with sed

A term with hundreds of occurrences is renamed with `sed`. Four hazards come with it.

- **Bound the file list to the scope, never to `.`** `grep -rl <term> .` reaches every context in the repo, and the audit left the same word standing in most of them on purpose (a *quantile* ladder in `pkg/floatpack`, a *precedence* ladder in `pkg/ast`). Sweeping those is scope creep that compiles, so it survives every check. After the pass, run `git diff --name-only | grep -v <scope>`, and revert what it lists.
- **The shell is zsh, which does not word-split an unquoted `$files`.** `sed -i … $files` passes the whole list as one filename, edits nothing, and reports success. Pipe through `xargs` instead. Confirm with a residue grep, not with the exit status.
- **Build the file list per identifier, never per family.** `grep -rl SloStatus` misses a file whose only reference is the auto-imported `sloStatusAxis`, and an auto-imported symbol has no import line to catch it. Grep each spelling you are about to change, and union the lists. Let `pnpm type-check` be the proof, after you delete and regenerate `src/auto-imports.d.ts` and `src/components.d.ts`. A family grep can report the sweep complete while two call sites still name the old symbol.
- **`wirepin.ConstStrings(t, dir, typeName)` takes the type name, not a name prefix.** It matches constants whose declared type is that string. A type named `SLOSubjectStatus` whose members are prefixed `SLOSubject` makes the third argument `"SLOSubjectStatus"`. It fails loudly (`no X constants found`), but only when someone runs the package's tests. Run them; do not trust the diff.

Then reread the prose. Do not trust the substitution. A term swap leaves tautologies and broken sentences behind — "the ascending width widths", "a width is a width". Grep for the doubled word and for the possessive form. Fix each by writing the sentence again, not by patching the phrase.

## Component tags

A tag rename is three edits, not one.

1. Rename the `.vue` file. The tag comes from the path, so the file *is* the name.
2. Bring the runtime name back into agreement with the tag. Where the filename is not already the tag, `vue/AGENTS.md` requires `defineOptions({ name: '<Tag>' })` as the first line of `<script setup>`. A spec's stub keys and `findComponent({ name })` are tags too.
3. Rewrite every usage in both spellings a template may use — `<PascalTag` and `<kebab-tag` — and in the specs.

Verify with the checks that see what `vue-tsc` cannot:

1. Capture `pnpm component-tags` before the rename. After it, run it again and diff. The diff proves that only the intended tags moved. The command itself fails on a duplicate tag, on a runtime name that is not its tag, and on a derivation that drifted.
2. `rg` for every old tag, because an unresolved tag type-checks and renders nothing.
3. Delete `src/auto-imports.d.ts` and `src/components.d.ts`, regenerate them with a brief `pnpm dev` run, then `pnpm type-check`.

Do not read `components.d.ts` for any of this. It is gitignored, absent on a fresh checkout, and keeps stale entries for deleted files.
