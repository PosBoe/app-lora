# Frontend UI direction

This document records the design system for `vue/`. Where the app and this document disagree,
this document wins and the app changes. `vue/AGENTS.md` holds the operative rules that follow
from it.

## Target

- The look is dense and quiet: one accent color, low-contrast borders, subtle motion,
  keyboard-first.
- Values come from Tailwind's scale, palette, and idioms, through UnoCSS.
- Light and dark are both first-class themes. Every token has a value in both from day one.
- The consumers are AI agents and human developers in this repository. The system is not an npm
  package.

## Stack

| Layer                | Owner                       | Note                                             |
| -------------------- | --------------------------- | ------------------------------------------------ |
| Utilities            | UnoCSS with the Wind4 preset | Tailwind-compatible classes.                     |
| Framework components | Vuetify                     | Overlays, tables, tabs, feedback, form controls. |
| Icons                | Material Design Icons       | Through Vuetify's `mdi-*` aliases.               |
| State                | Composables in `src/use/`   | No Pinia, no Vuex.                               |
| HTTP                 | axios, in `src/lib/api.ts`  | One client, base path `/api/v1`.                 |

## Tokens

- `vue/src/lib/tokens.ts` is the only place a value of the look is written. It feeds the Vuetify
  theme (`vue/src/plugins/vuetify.ts`), the UnoCSS theme, and the CSS variables for both themes
  (`vue/uno.config.ts`). `vue/src/lib/tokens.spec.ts` checks that every theme defines every
  token.
- Colors are semantic roles only: grounds (`background`, `surface`, `surface-hover`, `subtle`),
  ink (`strong`, `body`, `muted`, `faint`), and intent (`primary`, `success`, `warning`, `error`,
  `info`). Markup never names a raw palette step such as `text-slate-500`.
- A color that changes with the theme is a token, never a `dark:` pair in markup.
- Dark mode keys on Vuetify's theme class (`.v-theme--dark`). The Vuetify theme follows the
  system preference.

## Components

- Vuetify owns the concepts with the most behavior: dialog, menu, tooltip, data table, tabs,
  progress, snackbar, and the form controls.
- A card or panel shell is a `div` with the `app-card` shortcut, not `v-card`.
- Layout is UnoCSS flex and grid utilities, not `v-row`, `v-col` or `v-spacer`.
- A repeated pattern with a name and props is a component in `vue/src/components/`. A shortcut
  is only for a shell that has no props.

## Why

- **Semantic colors, not raw steps.** Two themes come free, an agent learns a short list of role
  names instead of the whole palette, and a review can tell a mistake from a choice.
- **One tokens module, not CSS variables as the source.** Vuetify's theme needs literal colors
  for its variation math. A `var()` string breaks it.
- **Components, not shortcuts.** A component has typed props, shows up in auto-import, and
  `vue-tsc` checks its use. A shortcut is a string that drifts.
- **Vuetify keeps the behavior-heavy concepts.** They carry the most behavior and the least look.
