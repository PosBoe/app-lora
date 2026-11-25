import { describe, expect, it } from 'vitest'

import { formatDateTime, messages, pickLanguage } from '@/lib/i18n'

// keys lists the dotted paths of every leaf message.
function keys(obj: object, prefix = ''): string[] {
  return Object.entries(obj).flatMap(([k, v]) =>
    typeof v === 'string' ? [prefix + k] : keys(v as object, `${prefix}${k}.`),
  )
}

describe('i18n', () => {
  it('has the same keys in every language', () => {
    expect(keys(messages.ru).sort()).toEqual(keys(messages.ro).sort())
  })

  it('picks the stored language first, then the browser, then Romanian', () => {
    expect(pickLanguage('ru', ['ro-RO'])).toBe('ru')
    expect(pickLanguage(null, ['en-US', 'ru-RU'])).toBe('ru')
    expect(pickLanguage('de', ['en-US'])).toBe('ro')
  })

  it('formats a time in the Chisinau time zone', () => {
    // 22:30 UTC is 00:30 the next day in Chisinau (UTC+2 in winter).
    expect(formatDateTime('2026-11-25T22:30:00Z', 'ro')).toContain('00:30')
  })
})
