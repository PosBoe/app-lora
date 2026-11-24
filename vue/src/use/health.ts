import { isAxiosError } from 'axios'

import { api } from '@/lib/api'

export interface Health {
  status: 'ok' | 'degraded'
  db: 'ok' | 'error'
  version: string
}

// useHealth loads GET /api/v1/health when the component mounts. The server
// answers 503 with a body when the database is down, so a 503 still sets
// `health`; only a transport failure sets `error`.
export function useHealth() {
  const health = shallowRef<Health>()
  const loading = shallowRef(false)
  const error = shallowRef('')

  async function reload() {
    loading.value = true
    error.value = ''
    try {
      const { data } = await api.get<Health>('/health')
      health.value = data
    } catch (err) {
      if (isAxiosError<Health>(err) && err.response?.data?.status) {
        health.value = err.response.data
      } else {
        health.value = undefined
        error.value = err instanceof Error ? err.message : String(err)
      }
    } finally {
      loading.value = false
    }
  }

  onMounted(reload)

  return { health, loading, error, reload }
}
