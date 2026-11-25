import { api } from '@/lib/api'

// DeviceModel mirrors modelResponse in device/model_handler.go.
export interface DeviceModel {
  id: number
  vendor: string
  model: string
  device_count: number
  created_at: string
}

// modelLabel is how the UI names a device model.
export function modelLabel(m: { vendor: string; model: string }): string {
  return `${m.vendor} ${m.model}`
}

// useDeviceModels loads the device models of the organization when the
// component mounts, and holds the model actions. Each action reloads the
// list.
export function useDeviceModels() {
  const models = shallowRef<DeviceModel[]>([])
  const loading = shallowRef(false)
  const error = shallowRef<unknown>()

  async function reload() {
    loading.value = true
    error.value = undefined
    try {
      const { data } = await api.get<{ device_models: DeviceModel[] }>('/device-models')
      models.value = data.device_models
    } catch (err) {
      error.value = err
    } finally {
      loading.value = false
    }
  }

  async function create(vendor: string, model: string): Promise<DeviceModel> {
    const { data } = await api.post<DeviceModel>('/device-models', { vendor, model })
    await reload()
    return data
  }

  async function rename(id: number, vendor: string, model: string) {
    await api.put(`/device-models/${id}`, { vendor, model })
    await reload()
  }

  async function remove(id: number) {
    await api.delete(`/device-models/${id}`)
    await reload()
  }

  onMounted(reload)

  return { models, loading, error, reload, create, rename, remove }
}
