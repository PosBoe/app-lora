import { apiErrorCode } from '@/lib/api'

// useErrorMessage returns a function that turns an API error into a message
// in the UI language. A code without a message, or no code, gives the
// generic message. Call it in a component.
export function useErrorMessage() {
  const { t, te } = useI18n()

  return function errorMessage(err: unknown): string {
    const key = `errors.${apiErrorCode(err)}`
    return te(key) ? t(key) : t('errors.unknown')
  }
}
