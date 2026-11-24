---
name: vue-split-components
description: Find oversized Vue components in vue/ and split them into smaller idiomatic pieces — extract pure functions, composables, and child components along real template seams, following this repo's script setup auto-import conventions. Splitting plans are user-approved before any edit; operates on vue/. For renaming or re-foldering existing components without splitting them, use vue-organize-components instead. Invoke ONLY when the user explicitly runs `/vue-split-components` or explicitly names this skill — do not infer it from generic phrasing like "split a component", "break up a large component", or "this component is too big" alone; treat those as ordinary requests instead.
---

# vue-split-components

Splitting is a behavior-preserving refactor: the page must render and behave exactly as before, but the code becomes navigable — each piece has one nameable job. Two principles govern everything below:

> **Size screens, structure convicts.** Line count only nominates candidates. A component is split because it contains real seams — independent regions, separable state clusters, extractable logic — never because it crossed a number.

> **A seam is real when the interface is small.** If you can name the extracted piece by its role in a few words and its interface is narrow (a handful of props + emits, one `defineModel`, or one composable-instance prop), the seam is real. If the cut needs a raft of props mirrored back by a storm of emits, the seam is wrong — move it, extract the shared logic into a composable first, or leave the component whole.

## Phase 1 — Find candidates

Skip discovery when the user names a component. Otherwise, from `vue/src`:

```sh
find . -name '*.vue' | xargs wc -l | sort -rn | head -30
```

In this repo anything over ~300 lines is in the top 5% and worth a look; below that, only visit files the user flagged. Line count nominates — then read each nominee and look for the signals that actually justify a split:

- **Independent template regions** — panels, dialog/drawer contents, toolbar vs. body — that share little or no state. Inline dialog content is the classic case.
- **Mode branches**: top-level `v-if`/`v-else` arms that are really separate screens.
- **Repeated markup blocks** differing only in bound data (a `v-for` + child component in waiting).
- **State clusters**: refs/computeds/watchers consumed by only one template region.
- **Mixed altitudes in script**: data fetching + heavy transformation (chart-option building, tree walking) + presentation state in one `<script setup>`.

And the counter-signals that say *leave it whole*:

- Long but flat and homogeneous — e.g. a form of many `PanelSection`s all reading/writing one model object. Splitting scatters a screenful of uniform structure across files and adds nothing.
- Interwoven state: every candidate cut needs most of the component's refs on both sides.
- A large `<style>` or a big static options object inflating the count.

Produce a shortlist: component, line count, one-line split rationale (or "leave whole: <why>"). If the user asked only to *find* large components, present this and stop.

## Phase 2 — Design the split

Design each split top-down before touching code. Prefer extractions in this order — each level is cheaper and lower-risk than the next:

1. **Pure functions** (formatting, chart-option builders, data reshaping) → a plain `.ts` file next to the component, or `src/lib` only if it has no Vue dependency *and* two or more feature-module consumers.
2. **Composables** for stateful logic (fetching, derived state, watchers) → `use-<thing>.ts` at the feature-module root, matching the existing `use-*.ts` files; `src/use` only for genuinely app-wide state. Composables are auto-imported by export name, so no import edits anywhere. Accept refs or getters as arguments and return an object of refs/computeds — never destructure a reactive object into plain values on the way in or out.
3. **Child components** along the template seams found in Phase 1. Often extracting a composable first is what makes a clean child seam possible: the child takes the composable's return object as a single prop instead of ten scalars.

For each extracted child, decide:

- **Role and tag.** Name the child by what it does for its caller, never by position (`Part1`, `TopSection`). Naming principles live in [../vue-organize-components/NAMING.md](../vue-organize-components/NAMING.md) — read it when naming more than a couple of children.
- **Placement.** Component auto-import uses `directoryAsNamespace` + `collapseSamePrefixes`, so the folder is the namespace. Splitting a root-level `Foo.vue` means creating `foo/` and moving the parent in: `foo/Foo.vue` still resolves to tag `Foo` (see `tracing/span-card/SpanCard.vue`), and children drop the prefix from their filename: `foo/Toolbar.vue` → `FooToolbar`. A component already in a family folder just gains siblings. Children extracted from a `views/` page must *not* stay under `views/` (route targets only, and the dir would prefix their tags) — put them at the module root or in a prefix folder there.
- **Interface**, in house style:
  - Props: runtime object syntax with `PropType` and explicit `required`/`default` — not type-only generics.
  - Emits: array syntax, `defineEmits(['click'])`.
  - Two-way binding: `defineModel`.
  - For tightly-coupled families, passing a composable instance as a single prop (e.g. `tableQuery: Object as PropType<UseUql>`) is established idiom — prefer it over drilling its fields.
  - Slots when the child is a layout shell the parent fills.

The parent stays the orchestrator: it owns route/query state and data fetching; children receive data and emit intent.

## Phase 3 — Plan approval

Present one plan per component before editing: the extracted children (tag, role, interface sketch), extracted composables/functions, and rough before/after line counts. Check planned tags against the auto-import resolver before presenting, from `vue/`:

```sh
node scripts/component-tags.ts --plan plan.tsv
```

(plan TSV: `old_path<TAB>new_path<TAB>planned_tag` relative to `vue/`, one row per moved/new file; the check catches tag collisions across all component roots.) Do not edit files until the user approves the plan.

## Phase 4 — Execute, one extraction at a time

Work bottom-up — pure functions, then composables, then children — verifying between steps. For each child extraction:

1. `git mv` the parent into its prefix folder first if the plan calls for one (history-preserving).
2. Move the template region verbatim, then its state cluster — the refs/computeds/watchers/handlers only that region uses. Whatever the region shares with the rest becomes props/emits/model per the plan. No manual imports for auto-imported components and composables.
3. Move `<style scoped>` rules with their markup. Parent scoped styles no longer reach the child's internals: a parent rule that targeted the moved region moves to the child; a parent `:deep()` into the moved region usually becomes a plain scoped rule in the child.
4. Splitting moves markup **verbatim** — do not modernize Vuetify layout to UnoCSS, rename classes, or restyle while splitting. One concern per change; note modernization candidates for a follow-up instead.

Watch the seams the compiler won't flag:

- **Template refs** crossing the cut: the parent can no longer reach a DOM/component ref inside the child — pass it out via `defineExpose` or restructure so the owner of the ref owns the behavior.
- **`provide`/`inject`, `v-model` modifiers, `KeepAlive include` / `<component :is>` name strings** — string- and context-based couplings that survive type-check silently.
- **Watcher/lifecycle timing**: hooks moved into a child now run per-instance and on the child's mount timing; a watcher moved into a `v-if`-guarded child no longer fires while the region is hidden.

If type-check reports an unknown tag for a new component, regenerate the declarations: `rm src/components.d.ts && timeout 15 pnpm dev > /dev/null 2>&1`. The plugin only regenerates the dts when the file is missing.

## Phase 5 — Verify

All from `vue/`, cheap to authoritative:

1. The oracle still validates (`--plan` prints `plan ok`; no collisions in list mode).
2. `pnpm type-check`, `pnpm lint`, and `pnpm test:run` pass (`test:run`, not bare `test` — that starts watch mode).
3. Regenerated `components.d.ts` diff shows exactly the planned new/moved tags and nothing else.
4. Read the final parent top to bottom: it should now tell the story of the screen in one screenful. If it's still hard to follow, the split moved lines without moving concepts — say so rather than declaring victory.
