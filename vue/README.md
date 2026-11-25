# vue

The web frontend: Vue 3, TypeScript, Vuetify, and UnoCSS, built with Vite.

| Path                        | Content                                                                             |
| --------------------------- | ----------------------------------------------------------------------------------- |
| `src/main.ts`               | Entry: installs vue-i18n, Vuetify and the router, mounts `App.vue`.                 |
| `src/App.vue`               | The app bar: the menu, the language switch, and the user menu.                      |
| `src/router.ts`             | Routes and the session guards. A module adds its pages here.                        |
| `src/plugins/vuetify.ts`    | The Vuetify instance, its themes built from the tokens, and its locales.            |
| `src/plugins/i18n.ts`       | The vue-i18n instance and the first UI language.                                    |
| `src/lib/`                  | Plain TypeScript: `tokens.ts` (the look), `api.ts` (the HTTP client), `testing.ts`. |
| `src/lib/i18n/`             | The Romanian and Russian messages; `ro.ts` is the schema `ru.ts` must match.        |
| `src/use/`                  | Composables, auto-imported.                                                         |
| `src/components/`           | Shared components, auto-imported by path.                                           |
| `src/core/`                 | The core module: login, set password, and the admin organizations page.             |
| `src/device/`               | The device module: the device list and page, models, groups, and the CSV import.    |
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
