import { createI18n } from 'vue-i18n'

import {
  DEFAULT_LANGUAGE,
  LANGUAGE_STORAGE_KEY,
  messages,
  pickLanguage,
  type Messages,
} from '@/lib/i18n'

// createAppI18n builds the vue-i18n instance. The first language is the one
// the user chose before, else the browser language, else Romanian.
export function createAppI18n(language = initialLanguage()) {
  return createI18n<[Messages], 'ro' | 'ru'>({
    legacy: false,
    locale: language,
    fallbackLocale: DEFAULT_LANGUAGE,
    messages,
  })
}

function initialLanguage() {
  let stored: string | null = null
  try {
    stored = localStorage.getItem(LANGUAGE_STORAGE_KEY)
  } catch {
    // Storage can be blocked; the browser language is then used.
  }
  return pickLanguage(stored, navigator.languages ?? [])
}
