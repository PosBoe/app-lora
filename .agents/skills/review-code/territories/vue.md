# Territory: vue

## Scope

Everything under `vue/`: single-file components, composables, TypeScript modules, specs, and the configs and scripts at the root of `vue/`.

## Read

- `vue/AGENTS.md` — the core rules, the wire contract, the module boundaries and the commands. Most repo contracts below come from it.
- `vue/MODULE-BOUNDARIES.md` when a change moves code or adds an import between modules.
- `docs/frontend-ui-direction.md` when a change touches the look: tokens, shortcuts, components.
- `vue/src/lib/api.ts` and the Go handler that serves the route, when code reads an API response.
- **Surface**: a component's props, emits, models, slots and exposed methods; a composable's arguments and returned refs; a module's exports.
- **Consumers**: grep the component's **tag** and its file name, not only `from '`. Auto-import means most uses have no import statement. `pnpm component-tags` prints the tag of every component.

## Probes

Run from `vue/`, one after another — never two that write the generated declarations at the same time:

```
pnpm install --frozen-lockfile
pnpm type-check
pnpm lint
pnpm component-tags
pnpm test:run <spec>...
```

- Run the specs of the files in scope and of their consumers, not the whole suite.
- For a structural move, run `pnpm build-only` and then `pnpm type-check`, in that order.
- Native popovers, focus, pointer sequencing and layout need a real browser. A jsdom pass proves none of them. Check them in the running app with the `run-app` skill.
- A stale `auto-imports.d.ts` or `components.d.ts` is not a defect of the code. Delete both files, regenerate them with a brief `pnpm dev` run, and type-check again.

## Hunt

- **Reactivity lost**: a prop or a reactive object destructured outside `toRefs`/`toRef`, a ref's `.value` read once into a plain variable, a `computed` that reads a non-reactive source, a `reactive()` replaced instead of mutated.
- **Stale effects**: a `watch` with a source that misses what the callback reads, a missing `immediate` where the first value matters, a `deep` watch on a large structure in a hot path.
- **Effects with no stop**: `addEventListener`, `setInterval`, `setTimeout`, a `ResizeObserver` or a subscription with no cleanup in `onUnmounted`/`onScopeDispose` or the watcher's `onCleanup`.
- **Async races**: two requests in flight where the older response lands last and overwrites the newer one; a result written after the component unmounts; a debounce that drops the final value.
- **Loading and error states**: a reader whose `error` is never shown, so a failed request leaves a spinner for ever; a gate that waits on a value an errored request never sets.
- **Wire shapes**: an `as` cast or a hand-written type that disagrees with the Go response struct it mirrors (compare the JSON tags); an optional field read as required.
- **Forms**: a client rule that disagrees with the Go validator it restates.
- **Templates**: a `v-for` without a stable `:key`, or keyed by index over a list that reorders; `v-if` and `v-for` on one element; `v-html` on text a user or an API wrote.
- **Accessibility contracts**: a control whose role element gets no name.

## Repo contracts

A breach of any of these is always reported:

- A hand edit of `vue/src/auto-imports.d.ts` or `vue/src/components.d.ts`.
- A parent-relative `../` import that crosses a layer under `src/`, or a `vi.mock()` path that no longer resolves (it fails silently: the real module runs).
- A feature module that imports another feature module, or a shared layer (`lib`, `use`, `components`) that imports a feature module (`vue/MODULE-BOUNDARIES.md`).
- A component whose runtime name is not its tag, or a duplicate auto-imported export name (the build keeps one and exits 0).
- A theme colour written as a `dark:` pair instead of a token; ink written as a palette step instead of an ink role.
- Vuetify layout, spacing or typography utility classes, which produce no CSS here.

## Mirrors

- The Go validator, limit or enum a form or rule restates. Grep for the constant and the message.
- Sibling components of one family (one directory of `src/components/`) that implement the same prop or contract.
- Specs that name a component by its tag as a string, which a rename leaves green.
- `vue/packages.ts`, the one scan list that `vite.config.ts` and `scripts/component-tags.ts` share.
