import { api } from '@/lib/api'

// AdminModel mirrors adminModelResponse in device/admin_model_handler.go.
export interface AdminModel {
  id: number
  organization_id: number
  organization_name: string
  vendor: string
  model: string
  decoder: string | null
  device_count: number
  pending_uplinks: number
}

// Decoder mirrors decoderResponse in device/admin_model_handler.go.
export interface Decoder {
  id: string
  name: string
}

// useAdminModels loads the device models of all organizations and the
// decoders when the component mounts. setDecoder reloads the list.
export function useAdminModels() {
  const models = shallowRef<AdminModel[]>([])
  const decoders = shallowRef<Decoder[]>([])
  const loading = shallowRef(false)
  const error = shallowRef<unknown>()

  async function reload() {
    loading.value = true
    error.value = undefined
    try {
      const [m, d] = await Promise.all([
        api.get<{ device_models: AdminModel[] }>('/admin/device-models'),
        api.get<{ decoders: Decoder[] }>('/admin/decoders'),
      ])
      models.value = m.data.device_models
      decoders.value = d.data.decoders
    } catch (err) {
      error.value = err
    } finally {
      loading.value = false
    }
  }

  async function setDecoder(modelId: number, decoder: string | null) {
    await api.put(`/admin/device-models/${modelId}/decoder`, { decoder })
    await reload()
  }

  onMounted(reload)

  return { models, decoders, loading, error, reload, setDecoder }
}
