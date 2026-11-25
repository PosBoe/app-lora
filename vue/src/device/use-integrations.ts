import { api } from '@/lib/api'

// IntegrationKind mirrors the integrations.kind values in
// device/integration_repo.go.
export type IntegrationKind = 'chirpstack' | 'ttn'

export const INTEGRATION_KINDS: IntegrationKind[] = ['chirpstack', 'ttn']

// Integration mirrors integrationResponse in device/integration_handler.go.
export interface Integration {
  id: number
  name: string
  kind: IntegrationKind
  ingest_url: string
  created_at: string
  last_uplink_at: string | null
  uplinks_24h: number
  last_error: { at: string; code: string; message: string } | null
}

// useIntegrations loads the integrations of the organization when the
// component mounts, and holds the integration actions. create and
// regenerate return the secret, which the server shows one time.
export function useIntegrations() {
  const integrations = shallowRef<Integration[]>([])
  const loading = shallowRef(false)
  const error = shallowRef<unknown>()

  async function reload() {
    loading.value = true
    error.value = undefined
    try {
      const { data } = await api.get<{ integrations: Integration[] }>('/integrations')
      integrations.value = data.integrations
    } catch (err) {
      error.value = err
    } finally {
      loading.value = false
    }
  }

  async function create(name: string, kind: IntegrationKind) {
    const { data } = await api.post<{ integration: Integration; secret: string }>('/integrations', {
      name,
      kind,
    })
    await reload()
    return data
  }

  async function rename(id: number, name: string) {
    await api.put(`/integrations/${id}`, { name })
    await reload()
  }

  async function regenerate(id: number): Promise<string> {
    const { data } = await api.post<{ secret: string }>(`/integrations/${id}/secret`)
    await reload()
    return data.secret
  }

  async function remove(id: number) {
    await api.delete(`/integrations/${id}`)
    await reload()
  }

  onMounted(reload)

  return { integrations, loading, error, reload, create, rename, regenerate, remove }
}
