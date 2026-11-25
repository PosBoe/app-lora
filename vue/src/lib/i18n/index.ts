import { ro, type Messages } from './ro'
import { ru } from './ru'

export type { Messages }

// Language is a UI language. The values are persisted: the users.language
// column and localStorage hold them.
export type Language = 'ro' | 'ru'

export const LANGUAGES: Language[] = ['ro', 'ru']

export const DEFAULT_LANGUAGE: Language = 'ro'

// LANGUAGE_STORAGE_KEY is the localStorage key of the chosen language.
export const LANGUAGE_STORAGE_KEY = 'lora.language'

export const messages: Record<Language, Messages> = { ro, ru }

export function isLanguage(value: unknown): value is Language {
  return value === 'ro' || value === 'ru'
}

// pickLanguage returns the stored language, else the first browser language
// the app has, else Romanian.
export function pickLanguage(stored: string | null, browser: readonly string[]): Language {
  if (isLanguage(stored)) {
    return stored
  }
  for (const tag of browser) {
    const base = tag.toLowerCase().split('-')[0]
    if (isLanguage(base)) {
      return base
    }
  }
  return DEFAULT_LANGUAGE
}

// formatDateTime formats an ISO time in the Moldova time zone.
export function formatDateTime(iso: string, language: Language): string {
  return new Intl.DateTimeFormat(language === 'ru' ? 'ru-MD' : 'ro-MD', {
    timeZone: 'Europe/Chisinau',
    dateStyle: 'short',
    timeStyle: 'short',
  }).format(new Date(iso))
}
