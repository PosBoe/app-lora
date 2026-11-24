# Module boundaries

The rule set for where frontend code goes. `../docs/frontend-module-boundaries.md` holds the
reasoning.

## Layers

| Layer      | Directory                                                     | Holds                                                                                          | May import                 |
| ---------- | ------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- | -------------------------- |
| lib        | `src/lib/`                                                    | Plain TypeScript: tokens, the HTTP client, formatters. No reactive state that outlives a call. | nothing else in `src/`     |
| use        | `src/use/`                                                    | Composables: reactive state and lifecycle. Auto-imported.                                      | `lib`                      |
| components | `src/components/`                                             | Shared components with no domain knowledge. Auto-imported.                                     | `lib`, `use`               |
| module     | `src/<context>/`                                              | One bounded context, e.g. `src/system/`.                                                       | `lib`, `use`, `components` |
| shell      | `src/App.vue`, `src/main.ts`, `src/router.ts`, `src/plugins/` | Assembles the app.                                                                             | everything                 |

## Rules

- A module never imports another module. Move the shared code down to `lib`, `use`, or
  `components`.
- Put code in the lowest layer that can hold it. If it calls a Vue reactivity or lifecycle API, it
  is `use`; otherwise it is `lib`.
- A module keeps composables that only it needs in the module, until a second module needs them.
- A module adds its pages to `src/router.ts`, loaded lazily with `() => import(...)`.
- Use `@/` for an import from another layer; use a relative path inside one module or one layer.
