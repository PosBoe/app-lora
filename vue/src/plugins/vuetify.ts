import '@mdi/font/css/materialdesignicons.css'
import 'vuetify/styles'

import { createVuetify, type ThemeDefinition } from 'vuetify'

import { TOKENS, type ThemeName } from '@/lib/tokens'

function theme(name: ThemeName): ThemeDefinition {
  const t = TOKENS[name]
  return {
    dark: name === 'dark',
    colors: {
      background: t.background,
      surface: t.surface,
      primary: t.primary,
      success: t.success,
      warning: t.warning,
      error: t.error,
      info: t.info,
    },
  }
}

export default createVuetify({
  theme: {
    defaultTheme: 'system',
    themes: {
      light: theme('light'),
      dark: theme('dark'),
    },
  },
  defaults: {
    VBtn: { rounded: 'md' },
  },
})
