# Territory: vue

The `vue/` frontend: components, composables, plain TypeScript modules, or a topic across them ("date-range handling", "how forms validate"). Held: behavior and rendered appearance, and persisted user-facing state (saved URLs, stored dashboard configs). Fair game: script shape, template idiom, state ownership, interfaces. Backend responses are fixed inputs here; when the wire itself should change, load the `api` territory.

## Scope

- **Named files** (`vue/src/system/HealthPage.vue`, `use/health.ts`, `lib/api.ts`) — those files plus their private collaborators: composables, `.ts` helpers, and child components used nowhere else.
- **A folder or feature area** (`system/`, "the settings forms") — list its files.
- **A topic** — follow the data flow view → composable → API call.

## Ideal end state

- **Correct reactivity.** Derived state is a `computed`, not state a `watch` writes. No deep watchers on large objects, no refs destructured into dead values, `ref` over `reactive` where it composes better.
- **One nameable job per piece.** Fetching, transformation, and presentation state do not interleave in one `<script setup>` or one composable.
- **State has one owner** — route/query, parent, this component, a composable, or a store. The component that owns fetching passes data down; children emit intent.
- **Composables** are named `use-<thing>.ts`, accept refs or getters, and return an object of refs and computeds. They do not return internal writable refs for callers to mutate, and they announce their side effects in their name.
- **Interfaces** — props in runtime object syntax with `PropType` and explicit `required`/`default`; emits in array syntax; `defineModel` for two-way bindings; for a tightly coupled component family, one composable instance passed as a single prop.
- **Script order** — interface declarations, then state and composable calls, then derived state, then handlers.
- **House markup** — `app-card` shells, `page-container`, UnoCSS row and grid utilities, `useRules()` for validation. Vuetify components stay for behavior (fields, menus, tables); no new `v-card`/`v-row`/`v-col`/`v-spacer`/`v-container` shells or Vuetify spacing utilities.
- **Placement** — single-consumer composables at the feature-module root; `src/components`, `src/use`, and `src/lib` only with two or more feature-module consumers. Pure logic lives in `.ts` files.
- **No derivation of what the backend already computes.** A formula spelled in Go and again in TypeScript is a defect, even when both lanes are green. Before planning a frontend derivation, survey what the backend already computes and serves. Best is a value in the openapi contract, generated into both languages; next, a field on `/internal/v1/config`. A hand-written mirror is the last resort, and a spec must then read the Go source it mirrors.
- **A form edits its own copy.** A form component takes its model as a prop and edits a copy it owns; the parent passes the object as it is and does not clone it. When two forms need the copy logic, it becomes one composable in `vue/src/use/`.
- **A write owns its error path and its refresh.** The composable that writes also refreshes every reader of what it changed. A form that emits "saved, please reload" for its parent to wire to a refresh is the coupling this replaces. Every awaited promise has an error path; a rejected save must not surface as an unhandled rejection.
- Auto-imports and the rest of `vue/AGENTS.md` are part of the ideal. Before you reject a house pattern as foreign, grep how many peers of the scope use it: the majority pattern among its peers is the default, and a departure needs a reason the peers do not share.

## Read

- `vue/AGENTS.md` first, then `vue/MODULE-BOUNDARIES.md` and `docs/frontend-ui-direction.md`.
- Every file in scope top to bottom. For an SFC: template, script, style — reactivity patterns, prop and emit declarations, lifecycle hooks, template refs, `provide`/`inject`. For a composable: what it accepts, returns, and registers. For a TS module: its exports and whether it is pure.
- Collaborators and parents: `grep -rn '<ComponentTag'` and `grep -rn 'useThing('` from `vue/src`. Auto-import means no import statement to grep; `vue/src/components.d.ts` and `vue/src/auto-imports.d.ts` list the auto-imported surface.
- Every usage site of the public interface: which props are passed, which emits are listened to, which slots are filled, which composable fields are consumed.
- `routes.ts` records if a view is in scope, query-param coupling (`useRoute`, query-sync composables), and the stores the scope reads or writes.

## Problems

- **Lazy content** — Vuetify mounts the content of expansion panels, dialogs, tabs, windows, and menus only while open, unless `eager` is set (its `useLazy` composable, under `vue/node_modules/vuetify`). Fields in closed content are absent from `v-form` validation, and their watchers and refs do not exist. Check this before any claim that a guard, a filter, or a rule makes other code unreachable.

- **Reactivity misuse** — a `watch` writing what a `computed` could derive; deep watchers; dead destructured refs; missed `computed` caching in hot templates.
- **Prop drilling / emit storms** — wide mirrored interfaces where a composable instance prop or `defineModel` would do.
- **Mixed altitudes** — fetching, heavy transformation, and presentation state in one script.
- **Leaky composable** — returned writable internals, hidden coupling to the route or a store, unannounced side effects.
- **Dead or legacy idiom** — Options API remnants, manual imports of auto-imported symbols, type-only prop generics where the repo uses runtime `PropType`, a `v-model:foo` hand-rolled via prop+emit.
- **Legacy markup** — Vuetify layout shells and spacing utilities.
- **Misplaced code** — pure logic in a component, app-wide state inline, shared UI living inside one feature module, one feature module reaching into another.

## Design spec

- **Interface** — props, emits, models, slots, composable signatures, and TS exports, as they would appear in the final code.
- **State ownership** — where each piece of state lives.
- **Naming** — new or renamed components follow [NAMING.md](../../vue-organize-components/NAMING.md). Check the resulting tags before presenting the plan, from `vue/`: `node scripts/component-tags.ts --plan plan.tsv`. The `plan.tsv` is scratch input; write it to the scratchpad, not the repo.

## Collisions

- **Callers** — prop, emit, and slot changes at every parent; `defineModel` renames changing `v-model` bindings; composable call sites consuming a changed return shape; import sites of moved TS exports.
- **Auto-import surface** — new or renamed components and composables change `components.d.ts` and `auto-imports.d.ts`; moving a component into a prefix folder changes its tag (`directoryAsNamespace` + `collapseSamePrefixes`).
- **String couplings** the compiler won't flag — `provide`/`inject` keys, `KeepAlive include`, `<component :is>` name strings, template refs reached through `defineExpose`.
- **Route and query state** — `routes.ts` records, query-param names other views or saved links depend on.
- **Stores** — state shape other views read.
- **Backend contracts** — response shapes and `/internal/v1/config` fields are fixed inputs; adapt in one place, don't reshape the design around the raw wire.

## Migration

- Cheapest risk first: pure-function extraction, then composables, then script restructuring, then markup modernization, then interface changes with their caller updates.
- **Declaration regeneration** flag when a step adds, renames, or moves an auto-imported component or composable: the dev server and the build rewrite `src/components.d.ts` and `src/auto-imports.d.ts`; delete them and run `pnpm dev` briefly to force the rewrite when a result looks stale. Commit the regenerated files.
- **Timing seams** flag — watcher and lifecycle timing when logic moves into a composable or a `v-if`-guarded child; template refs crossing a cut; `provide`/`inject` reach; `<style scoped>` rules moving with their markup (a parent `:deep()` into an extracted region usually becomes a plain scoped rule in the child).

## Verification

- Per step, from `vue/`: `pnpm type-check` and `pnpm lint`.
- Final: `pnpm type-check`, `pnpm lint`, `pnpm test:run` (not bare `test` — that starts watch mode), and `pnpm component-tags` when a component moved.

## Forks

- Whether a quirky current behavior is intended, or a bug to preserve.
- Whether two similar-looking UI states are meant to be one.

## Plan sections

- `## 1. Current state` → `### Public surface and its users` is titled `### Public interface and its users`.
- `## 2. Ideal design` → `### Surface` is titled `### Interface`; add `### State ownership` after `### Data flow`.
- `## 3. Reconciliation` → `### Consumer and wiring changes` is titled `### Caller and coupling changes`.

## Constraints

- No new npm dependencies; `vue/package.json` as it stands is the toolbox.
- Behavior-visible changes need an explicit instruction; a new look is design work, not refactoring.
