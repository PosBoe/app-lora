# vue

The web frontend: Vue 3, TypeScript, Vuetify, and UnoCSS, built with Vite.

| Path                        | Content                                                                             |
| --------------------------- | ----------------------------------------------------------------------------------- |
| `src/main.ts`               | Entry: installs Vuetify and the router, mounts `App.vue`.                           |
| `src/router.ts`             | Routes. A module adds its pages here.                                               |
| `src/plugins/vuetify.ts`    | The Vuetify instance and its themes, built from the tokens.                         |
| `src/lib/`                  | Plain TypeScript: `tokens.ts` (the look), `api.ts` (the HTTP client), `testing.ts`. |
| `src/use/`                  | Composables, auto-imported.                                                         |
| `src/components/`           | Shared components, auto-imported by path.                                           |
| `src/system/`               | The system module: the health page.                                                 |
| `src/types/`                | Global type declarations.                                                           |
| `packages.ts`               | The directories that auto-import scans.                                             |
| `scripts/component-tags.ts` | Prints the tag of every component and checks the names.                             |
| `uno.config.ts`             | UnoCSS: the token colours and the `app-*` shortcuts.                                |
| `public/layers.css`         | The CSS cascade layer order.                                                        |

`AGENTS.md` holds the rules, and `MODULE-BOUNDARIES.md` holds the placement rules.

## Run

From the repository root: `task dev` starts the Go server on :8080, and `task vue` starts the
dev server on :5173, which proxies `/api` to it.
