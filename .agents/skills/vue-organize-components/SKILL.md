---
name: vue-organize-components
description: Redesign Vue component names and folder hierarchy so auto-import namespacing reads naturally — discover each component's role from its usage sites, design the ideal taxonomy (including renames), group components into prefix folders, then execute the moves and usage rewrites safely. Operates on vue/; the naming plan is user-approved before any file moves. Invoke ONLY when the user explicitly runs `/vue-organize-components` or explicitly names this skill — do not infer it from generic phrasing like "organize components", "restructure the components directory", or "clean up flat components" alone; treat those as ordinary requests instead.
---

# vue-organize-components

Vue component auto-import here uses `unplugin-vue-components` with `directoryAsNamespace: true` and `collapseSamePrefixes: true`, so a folder path *is* a tag name: `components/date/RangePicker.vue` ⇔ `<DateRangePicker>`. The core principle of this skill follows from that:

> **Work in name-space, not file-space.** First design what each component should be *called* and which conceptual family it belongs to, as if the codebase were written from scratch. Then derive the folder layout as the physical encoding of those names. Files follow names, never the reverse.

Judgment (roles, taxonomy, renames) and mechanics (resolution rules, moves, rewrites) are kept strictly separate. The seam between them is the **naming plan**, a TSV artifact checked by `vue/scripts/component-tags.ts` — the repo's own resolver, which reads the scan roots from `vue/packages.ts` and verifies every tag it derives against the plugin.

Read [NAMING.md](NAMING.md) before designing a taxonomy. Consult [REFERENCE.md](REFERENCE.md) for resolution semantics, rewrite mechanics, and edge cases.

## Phase 0 — Scope and ground truth

1. Pin the scope. Default: the flat `.vue` files at the root of `vue/src/components`. The user may widen it to whole roots or narrow it to one family.
2. From `vue/`, capture the current name map and confirm it is collision-free:

   ```sh
   node scripts/component-tags.ts > /tmp/.../before.tsv
   ```

   The tool imports the component roots and `excludeComponentNames` from `vue/packages.ts`, the same manifest `vite.config.ts` reads, so it needs no arguments and cannot drift. It exits non-zero on a collision, on a derivation that no longer matches the plugin, and on a component whose runtime name is not its tag — fix those before designing anything.

## Phase 1 — Inventory: understand, don't list

For every in-scope component, produce one line: **current tag, usage count, role**. The role is a one-line statement of what the component does *for its callers* — derived from reading the SFC and its usage sites, not from its filename. Usage sites matter more than implementation: the best name describes the role at the call site.

- Get usage counts mechanically: `node scripts/component-tags.ts --usages` emits `tag<TAB>count<TAB>path` for every component (PascalCase tokens plus kebab-case template tags; a slight overcount heuristic — see REFERENCE.md). A count of 0 means the component is likely dead — flag it for removal instead of renaming it.
- Usage count is the **rename budget**: a component used in 40 templates needs a much stronger reason to rename than one used twice.
- Still read actual usage sites (`rg '<PascalTag\b'` / `<kebab-tag`) to derive the role — the count says how expensive a rename is, not what the component is for.
- For more than ~30 components, fan out parallel Explore agents, each taking a batch of files and returning `tag<TAB>role` lines (counts come from `--usages`, not the agents).
- **Inventory the folders too, not only the components.** One line per folder already in scope: its members, and the call sites that render each (`rg -l '<Tag\b'`). This is the input the growth test needs (NAMING.md rule 7), and collecting it here is what stops an inherited umbrella from reaching the winning candidate unexamined. Read it by subject, not by call-site count — members that are views of one subject are a unit even when two views render them; surfaces of different subjects sharing an owner's name are an umbrella however tidy the folder is named.

## Phase 2 — Taxonomy design: design it twice

Read [NAMING.md](NAMING.md) first — it holds the naming principles, the scoring rubric, and worked examples including the suffix-family tradeoff.

Produce two or three candidate taxonomies from genuinely different organizing principles — e.g. by domain concept (`date`, `query`, `facet`), by feature area, and a conservative leave-mostly-flat baseline. For each candidate, score against the rubric in NAMING.md (idiomatic fit, call-site readability, folder coherence, rename cost, resolver fit). Pick the winner and graft in the best ideas from the runners-up. The existing tree is evidence of roles, not a naming authority — names are justified by the style-guide rules in NAMING.md, never by "matches a neighboring component".

"Leave it flat" is a first-class outcome for any component; not every lexical family deserves a folder. Favor the flattest candidate that still groups genuinely tight couplings, and treat a broad domain umbrella as a smell rather than a win: `alerting/monitor/` (13 members) and `alerting/alert/` (7) are what one looks like once it has settled in. A folder earns itself by tight coupling — a feature unit, or a parent and its dedicated children — with bounded membership. See NAMING.md rules 7–9.

**Every folder the scope already has gets an explicit verdict in every candidate — keep or dissolve — naming the test it passed or failed.** Silence is not a verdict: keeping an inherited folder writes no plan row, so an umbrella that predates the run survives by default and is never priced. "Dissolve" is as ordinary an outcome for a folder as "leave it flat" is for a component, and it is frequently free, because the members usually already resolve to the tags they would carry flat. Never merge a new member into a folder that has not passed its own verdict first — NAMING.md rule 8.

## Phase 3 — The naming plan: the seam

1. Write the plan TSV to the scratchpad — one row per component that moves, `old_path<TAB>new_path<TAB>planned_tag` (paths relative to `vue/`). Components that stay put are simply absent. A member of a dissolved folder is *not* staying put: it needs a row even when the move leaves its tag unchanged, and a tag-neutral row is the normal shape for a dissolve. The tool counts those as moves and not as renames, which is the reading you want — the `iam/org/` dissolve reported `3 moves, 0 renames`, meaning no template is touched.
2. Check it before anything moves:

   ```sh
   node scripts/component-tags.ts --plan plan.tsv
   ```

   Iterate until it prints `plan ok`. The check catches planned tags the layout won't actually produce and future collisions across all roots (e.g. a new `facet-value/Item.vue` colliding with the existing `facet/ValueItem.vue`). Its `rename:` lines are the definitive list of tags whose usage sites must be rewritten.
3. Present the plan for approval as a per-group table: members with `old tag → new tag` (or "unchanged"), usage count, and a one-line rationale for every rename. Renames are outward-visible to every template — do not touch files before the user approves.

## Phase 4 — Execute, one group at a time

For each approved group:

1. `git mv` per plan row (history-preserving).
2. For each renamed tag, rewrite all usage forms — PascalCase and kebab-case, open and close tags, `<KeepAlive include>`, `<component :is>` strings, name strings in tests. The complete checklist is in [REFERENCE.md](REFERENCE.md).
3. Where the move makes a filename stop being its tag, add `defineOptions({ name: '<Tag>' })` **in this group's own step**, not in a sweep at the end. The pin is what keeps the runtime name equal to the tag; deferring it leaves every commit in between failing `component-tags`, so a bisect through the middle of the work reports a break that was never real.
4. Fix relative imports *inside* moved files (their depth changed), and rewrite explicit path imports elsewhere (`rg` the old path).
5. Quick gate before the next group: `rg` for each old tag returns zero hits outside generated files, and `node scripts/component-tags.ts` still exits 0.

Commit per group with a Conventional Commit (`refactor(vue): group date components`) only if the user asked for commits; otherwise leave the working tree staged group-by-group. Either way each group is self-contained — step 3 is what makes that true.

## Phase 5 — Verify: the pyramid

Cheap to authoritative, all mandatory:

1. **Tag tool**: default mode exits 0 — no collision, no drifted derivation, no unpinned name — and its table, diffed against the Phase 0 capture, holds exactly the planned moves and renames and nothing else. Do **not** re-run `--plan` here. It is a before-the-move gate and nothing else: once the moves land, every old path is gone and every new path is taken, so it reports `plan error: old path does not exist` and `new path already occupied` for every row. That is the check working, not the tree failing.
2. **Old-tag zero**: for every renamed tag, `rg '\bOldTag\b'` and the kebab form return nothing outside `components*.d.ts` and lockfiles. String-based usages (`:is`, `include`, tests) do not fail type-check — this grep is the only thing that catches them.
3. `pnpm type-check`, `pnpm lint`, and `pnpm test:run` pass (`test:run`, not bare `test` — that starts vitest watch mode). Tests can mount components by name or import them by path; type-check alone does not cover them.
4. **Ground truth**: every step above reads `component-tags.ts`, which *replicates* the plugin. This step asks the plugin itself. Regenerate the dts file, then diff the plugin's own table against the resolver's:

   ```sh
   rm src/components.d.ts && timeout 15 pnpm dev > /dev/null 2>&1
   sed -nE "s/^ +([A-Za-z0-9_]+): typeof import\('\.\/(.+)'\)\['default'\]\$/\1\t\2/p" \
     src/components.d.ts | sort > /tmp/.../dts.tsv
   node scripts/component-tags.ts | sort | diff /tmp/.../dts.tsv -
   ```

   An empty diff is the property. The `sed` drops the file's header and the two resolver-provided entries (`RouterLink`, `RouterView`), which name no path under `src/`.

   `git diff` on `src/components.d.ts` shows the same change, because the file is committed. The resolver diff above is still the proof: it compares the two derivations, not two versions of one.

   The `rm` is required: at startup the plugin regenerates the dts only when the file is *missing*; an existing stale one refreshes only on page traffic or watch events. (If a dev server is already running during the reorganization, its watcher keeps the dts fresh automatically.) Never hand-edit these files.
