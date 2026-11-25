import { api } from '@/lib/api'

// DeviceGroup mirrors groupResponse in device/group_handler.go.
export interface DeviceGroup {
  id: number
  name: string
  device_count: number
  created_at: string
}

// useDeviceGroups loads the device groups of the organization when the
// component mounts, and holds the group actions. Each action reloads the
// list, so the device counts stay true.
export function useDeviceGroups() {
  const groups = shallowRef<DeviceGroup[]>([])
  const loading = shallowRef(false)
  const error = shallowRef<unknown>()

  async function reload() {
    loading.value = true
    error.value = undefined
    try {
      const { data } = await api.get<{ device_groups: DeviceGroup[] }>('/device-groups')
      groups.value = data.device_groups
    } catch (err) {
      error.value = err
    } finally {
      loading.value = false
    }
  }

  async function create(name: string): Promise<DeviceGroup> {
    const { data } = await api.post<DeviceGroup>('/device-groups', { name })
    await reload()
    return data
  }

  async function rename(id: number, name: string) {
    await api.put(`/device-groups/${id}`, { name })
    await reload()
  }

  async function remove(id: number) {
    await api.delete(`/device-groups/${id}`)
    await reload()
  }

  // addDevices and removeDevices return how many devices changed.
  async function addDevices(id: number, deviceIds: number[]): Promise<number> {
    const { data } = await api.post<{ changed: number }>(`/device-groups/${id}/add`, {
      ids: deviceIds,
    })
    await reload()
    return data.changed
  }

  async function removeDevices(id: number, deviceIds: number[]): Promise<number> {
    const { data } = await api.post<{ changed: number }>(`/device-groups/${id}/remove`, {
      ids: deviceIds,
    })
    await reload()
    return data.changed
  }

  onMounted(reload)

  return { groups, loading, error, reload, create, rename, remove, addDevices, removeDevices }
}
