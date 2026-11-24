// The design tokens. This is the one place a value of the app's look is
// written. The Vuetify theme (src/plugins/vuetify.ts) and the UnoCSS theme
// (uno.config.ts) both read it, so a color changes in one place.
//
// Ink roles, from loudest to quietest: strong, body, muted, faint. Use them as
// `text-strong`, `text-muted`, and so on. Do not write a palette step such as
// `text-slate-500` for text.

export type ThemeName = 'light' | 'dark'

export const TOKEN_NAMES = [
  // grounds
  'background',
  'surface',
  'surface-hover',
  'subtle',
  // ink
  'strong',
  'body',
  'muted',
  'faint',
  // intent
  'primary',
  'success',
  'warning',
  'error',
  'info',
] as const

export type TokenName = (typeof TOKEN_NAMES)[number]

export const TOKENS: Record<ThemeName, Record<TokenName, string>> = {
  light: {
    background: '#f8fafc',
    surface: '#ffffff',
    'surface-hover': '#f1f5f9',
    subtle: '#e2e8f0',
    strong: '#0f172a',
    body: '#334155',
    muted: '#64748b',
    faint: '#94a3b8',
    primary: '#4f46e5',
    success: '#16a34a',
    warning: '#d97706',
    error: '#dc2626',
    info: '#2563eb',
  },
  dark: {
    background: '#0b0f17',
    surface: '#111827',
    'surface-hover': '#1f2937',
    subtle: '#263041',
    strong: '#f8fafc',
    body: '#cbd5e1',
    muted: '#94a3b8',
    faint: '#64748b',
    primary: '#818cf8',
    success: '#4ade80',
    warning: '#fbbf24',
    error: '#f87171',
    info: '#60a5fa',
  },
}

// cssVariables returns the tokens as custom properties, one block per Vuetify
// theme class.
export function cssVariables(): string {
  return (Object.keys(TOKENS) as ThemeName[])
    .map((theme) => {
      const vars = TOKEN_NAMES.map((name) => `--app-${name}:${TOKENS[theme][name]};`).join('')
      return `.v-theme--${theme}{${vars}}`
    })
    .join('\n')
}
