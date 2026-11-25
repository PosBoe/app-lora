import { useLocale } from 'vuetify'

import { DEFAULT_LANGUAGE, isLanguage, LANGUAGE_STORAGE_KEY, type Language } from '@/lib/i18n'

// useLanguage reads and sets the UI language. A change sets the vue-i18n
// locale, the Vuetify locale, the lang attribute of the page, and the stored
// choice. Call it in a component.
export function useLanguage() {
  const { locale } = useI18n()
  const vuetifyLocale = useLocale()

  const language = computed<Language>(() =>
    isLanguage(locale.value) ? locale.value : DEFAULT_LANGUAGE,
  )

  function setLanguage(next: Language) {
    locale.value = next
    vuetifyLocale.current.value = next
    document.documentElement.lang = next
    try {
      localStorage.setItem(LANGUAGE_STORAGE_KEY, next)
    } catch {
      // Storage can be blocked; the choice then lasts for this page only.
    }
  }

  return { language, setLanguage }
}
