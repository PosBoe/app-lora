# Frontend Development

Rules here apply to `vue/`. Backend rules live in `../AGENTS.md`.

`MODULE-BOUNDARIES.md` holds the placement rule, `lib` versus `use`, the dependency direction, and
the auto-import mechanics. `../docs/frontend-ui-direction.md` holds the design system and the
reasons behind the rules below.

## Core Rules

- Use Vue 3 + TypeScript + Vuetify. State lives in composables in `src/use/`; do not add Vuex or
  Pinia.
- Use `<script setup lang="ts">` in every component.
- JavaScript and TypeScript comments use `//` line comments, including comments for exported
  types, fields, and functions.
- Auto-import is enabled for the Vue, Vue Router, and VueUse APIs, for every composable in
  `src/use/`, and for every component in `src/components/`. Do not add manual imports for them.
  `packages.ts` lists the scanned directories; `vite.config.ts` and `scripts/component-tags.ts`
  both read it.
- Component auto-import uses `directoryAsNamespace: true` and `collapseSamePrefixes: true`: the
  directory is the namespace, so `src/components/page/NotFound.vue` is `<PageNotFound>`. When
  several components share a prefix, put them in a directory named after that prefix and drop the
  shared prefix from the file names.
- **A component's runtime name is its tag.** When the file name differs from the tag, add
  `defineOptions({ name: '<Tag>' })`. `pnpm component-tags` fails a component that breaks this.
- A component in a feature module (`src/system/`, …) is not auto-imported. Import it by its
  relative path.
- Commit `src/auto-imports.d.ts` and `src/components.d.ts`. The dev server and the build
  regenerate them; never edit them by hand.
- Call the Go server through `api` from `src/lib/api.ts`. Every path is relative to `/api/v1`.
  The Vite dev server proxies `/api` to the Go server (`task dev`, port 8080).
- A response type in TypeScript mirrors the Go struct that the handler encodes. Name the Go
  struct in a comment on the type, and keep the JSON keys equal.

## Styling

- Use Tailwind CSS utility classes via UnoCSS with the Wind4 preset for new markup.
  `$utilities: false` in `src/styles/settings.scss` turns off Vuetify's utility layer, so a
  Vuetify layout, spacing, or typography class produces no CSS. It fails silently: the class stays
  in the markup and styles nothing. Keep Vuetify components and props for component behavior.
- **A colour is a token.** `src/lib/tokens.ts` is the one place a value of the look is written.
  It feeds the Vuetify theme (`src/plugins/vuetify.ts`) and the UnoCSS theme and CSS variables
  (`uno.config.ts`). Never write a `dark:` pair in markup for a colour that changes with the theme.
- **Ink is one of these roles: `text-strong`, `text-body`, `text-muted`, `text-faint`**, from
  loudest to quietest. Do not write a palette step such as `text-slate-500` for text.
- Grounds are `bg-background`, `bg-surface`, `bg-surface-hover`; a quiet border is
  `border-subtle`. Intent colours are `primary`, `success`, `warning`, `error`, `info`.
- Do not use `v-card`, `v-card-title`, `v-card-text`, or `v-card-actions` for new shells. Use a
  `div` with the `app-card` shortcut. Title bar: `app-card-header`; body: `p-4`; actions:
  `app-card-actions`. The shortcuts live in `uno.config.ts`.
- Do not use `v-row`, `v-col`, or `v-spacer`. Use flex and grid utilities: a row is
  `flex flex-wrap gap-6`, a growing column `flex-1 min-w-0`, a grid `grid grid-cols-12 gap-6`.
  Never use fractional `w-*` utilities inside a flex row with a gap; use a grid and `col-span-*`.
- A repeated pattern with a name and props is a component in `src/components/`, not a new
  shortcut.

## Tests

- Specs sit beside their subject as `*.spec.ts` and run under Vitest with jsdom.
- Mount a component with `mountApp` from `src/lib/testing.ts`, which installs Vuetify and an
  in-memory router.
- Mock the network at `api` (`vi.spyOn(api, 'get')`), not at a composable.
- Assert on rendered text through `data-test` attributes, not on CSS classes.

## Commands

Run from `vue/`:

- `pnpm dev` — the dev server (`task vue` from the root).
- `pnpm type-check`, `pnpm lint`, `pnpm format:check` — the static gates.
- `pnpm test:run` — the unit tests once. Bare `pnpm test` starts watch mode.
- `pnpm component-tags` — print every component tag; it fails on a collision or a runtime name
  that is not its tag. Run it after you move or rename a component.
- `pnpm build-only` — build `dist/`, which `task build` embeds into the Go binary.
