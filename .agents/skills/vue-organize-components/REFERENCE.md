# Reference: resolution semantics, rewrite mechanics, edge cases

## How a path becomes a tag

`unplugin-vue-components` computes the tag as `pascalCase(folders under the component root + filename)`, with this repo's options `directoryAsNamespace: true` and `collapseSamePrefixes: true`. Rules that matter:

- Folder names are PascalCased per segment; kebab-case dirs become one word: `empty-state/` → namespace `EmptyState`. Non-alphanumeric characters other than `-` are stripped from folder names.
- A filename of `index.vue` contributes nothing: `nav/index.vue` → `Nav`.
- **Collapse rule**: each segment drops the longest *contiguous chain of nearest-parent* prefixes it repeats. Rows marked **real** are paths in the tree; rows marked **probe** are layouts the repo does not have, whose derivation was confirmed by feeding the move to `--plan`.

  | Path (under a scan root) | Tag | Why |
  |---|---|---|
  | real · `components/date/RangePicker.vue` | `DateRangePicker` | plain namespace join |
  | real · `components/empty-state/NoData.vue` | `EmptyStateNoData` | no shared prefix, plain join |
  | real · `alerting/alert-card/AlertCard.vue` | `AlertCard` | filename repeats the whole parent → collapses to nothing, tag = namespace |
  | real · `apps/main/nav/user-notification-list/UserNotificationList.vue` | `NavUserNotificationList` | collapse is against the **nearest** parent only: the filename eats `user-notification-list/`, and `nav/` still contributes |
  | probe · `components/date/DatePicker.vue` | `DatePicker` | the `Date` prefix collapses against the `date/` parent |
  | probe · `components/empty-state/State.vue` | `EmptyStateState` | **trap**: partial overlap does not collapse — the filename must start with the parent chain from its beginning |
  | probe · `transpipe/pipe-op/PipeOpsTable.vue` | `PipeOpSTable` | **trap**: stripping `PipeOp` strands the `s`, which then pascal-cases into its own word |

  The two trap rows are why plans are checked by the tool and never by eye. These rows illustrate resolution mechanics only — they are not names to emulate; naming rules live in NAMING.md.

- `excludeComponentNames` in `vue/packages.ts` (currently `[/^Views.+/]`) drops matching tags from registration. The tool imports it, so a change to it needs no edit here.

## Component roots and targets

Roots come from `componentDirs` in `vue/packages.ts`. The tool imports it, so it cannot drift from what the build scans. All roots share **one global tag namespace**. Each target generates its own dts (`src/components.d.ts`, `src/components.license.d.ts`).

## The tag tool

`vue/scripts/component-tags.ts` — the repo's own resolver, run from `vue/`, all paths relative to it. It is not a skill script: `pnpm component-tags` runs it, so it stays true whether or not this skill is used.

```sh
# list Tag<TAB>path; exit 2 on a collision, a drifted derivation, or an unpinned name
node scripts/component-tags.ts

# same with usage counts: Tag<TAB>count<TAB>path
node scripts/component-tags.ts --usages

# check a plan against the layout it would produce; exit 1 on mismatch/collision
node scripts/component-tags.ts --plan plan.tsv
```

Plan rows: `old_path<TAB>new_path<TAB>planned_tag`, relative to `vue/`; `#` comments and blank lines ignored. The `rename:` output lines are the authoritative worklist of tags needing usage rewrites — a move whose tag is unchanged needs **no template edits at all**. A row whose new path lands under a `views/` directory is rejected, because that tag is excluded from the namespace and means nothing.

`--usages` counts identifier tokens equal to the tag plus kebab-case template tags across `.vue`/`.ts`/`.tsx`/`.js` under `src/` (excluding `*.d.ts`). It is a deliberate *overcount* heuristic for rename budgeting — an unrelated identifier that happens to equal a tag also counts. A count of 0 is reliable in the other direction: the component is unused (dead) code.

Default mode verifies every derived tag against the plugin's own resolver on each run, so the derivation cannot silently drift. Plan mode does not: the resolver cannot answer for files that do not exist yet, and a tree mid-move holds names that are not yet pinned, which would abort the run before the plan was read. Capture the table in Phase 0 and that verification is already done for the tree the plan builds on.

## Rewrite checklist for a renamed tag

Type-check does not catch every usage form; work through all of them per rename:

1. Templates, PascalCase: `<OldTag ...>`, `</OldTag>`, self-closing `<OldTag ... />`. `\b` anchors are safe here — tags are single words.
2. Templates, kebab-case: `<old-tag ...>`, `</old-tag>` (both forms resolve to the same component; either may appear). **Anchor with a trailing delimiter** (`[\s/>]`): `-` sits on a `\b` word boundary, so an unanchored `<date-range` also matches and corrupts `<date-range-picker`.
3. String usages — **invisible to type-check**, only the final grep catches them:
   - `<component :is="'OldTag'">` and `:is` bound to variables holding the tag name;
   - `<KeepAlive include="OldTag">` / `exclude` (comma lists too);
   - test files referencing the name (`findComponent({ name: ... })`, stub keys, snapshots);
   - the **rendered stub element**, `<old-tag-stub>`. `shallowMount` and `global.stubs` render a
     stubbed child as its kebab tag plus `-stub`, so a spec that reaches for one by
     `querySelector('old-tag-stub')`, `find('old-tag-stub')` or in asserted HTML keeps compiling
     after the rename and stops testing anything. The PascalCase stub *key* is caught by grep 1;
     this element is caught by neither of the first two greps, because the kebab anchor below
     rejects the `-` that starts `-stub`. A rename left `'dashboard-new-menu-stub'` in
     `vue/src/dashboard/views/Layout.spec.ts` exactly this way.
4. Explicit imports of the file path (`@/components/OldName.vue`, `defineAsyncComponent(() => import(...))`) — rare here, but grep the old path.
5. Relative imports *inside* every moved file — moving into a folder changes `./` and `../` depth.

Verification for each rename: `rg '\bOldTag\b'`, `rg 'old-tag[\s/>]'` and `rg 'old-tag-stub\b'` return
nothing outside generated `components*.d.ts`. The third grep is separate on purpose: widening the
kebab anchor to accept `-` reopens the `<date-range` / `<date-range-picker` trap it exists to close,
while `old-tag-stub\b` is precise on its own — it cannot match a longer tag's stub.
No lint catches a stale stub yet, so run the grep every time.

## Colocated non-component files

Component folders may hold `.ts` files — composables (`apps/main/nav/use-nav.ts`), a vocabulary module (`alerting/alert-timeline/kinds.ts`), types, and `.spec.ts` files. They are not components and never appear in the plan TSV, so decide each one separately:

- Move one along only when it serves the group exclusively. `alert-timeline/kinds.ts` belongs to its folder; a composable that three modules call does not.
- Never move a `.ts` file as a side effect of grouping `.vue` files it does not belong to.
- A dissolved folder's `.ts` files go up to the module root with its components. Check whether an explicit importer exists first: most of these sit under an auto-import glob in `packages.ts` (`src/iam/**/*.ts` and friends recurse), and one that does needs no import rewrite at all — but `grep` before relying on that.
- A `.spec.ts` follows the component it tests, and the spec's own import path needs fixing.

## Never touch

- **Vuetify components** (`V*` tags) — they come from `vite-plugin-vuetify`, not these dirs.
- `RouterLink` / `RouterView` and anything else resolver-provided rather than dir-scanned.
- Generated files: `src/components.d.ts`, `src/auto-imports.d.ts` — ground truth, not things to edit. To force regeneration use the delete-plus-brief-dev-run recipe in SKILL.md Phase 5 (a *stale* dts is not refreshed at server startup; a *missing* one is).
