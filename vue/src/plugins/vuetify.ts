import '@mdi/font/css/materialdesignicons.css'
import 'vuetify/styles'

import { createVuetify, type ThemeDefinition } from 'vuetify'
import { ro, ru } from 'vuetify/locale'

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
  // useLanguage keeps the current locale equal to the vue-i18n locale.
  locale: {
    locale: 'ro',
    fallback: 'ro',
    messages: { ro, ru },
  },
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
