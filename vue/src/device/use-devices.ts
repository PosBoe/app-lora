import { api } from '@/lib/api'

// MeterType mirrors the devices.meter_type values in device/fields.go.
export type MeterType = 'water' | 'gas'

export const METER_TYPES: MeterType[] = ['water', 'gas']

// Device mirrors deviceResponse in device/device_handler.go.
export interface Device {
  id: number
  dev_eui: string
  name: string
  device_model: { id: number; vendor: string; model: string }
  meter_type: MeterType | null
  address: string | null
  note: string | null
  archived_at: string | null
  created_at: string
  updated_at: string
}

// DeviceInput mirrors deviceRequest in device/device_handler.go. An empty
// optional field clears it.
export interface DeviceInput {
  dev_eui: string
  name: string
  device_model_id: number
  meter_type: MeterType | ''
  address: string
  note: string
}

// DeviceFilter is the query of the device list. The list page keeps it in
// the URL.
export interface DeviceFilter {
  q: string
  modelId: number | null
  meterType: MeterType | null
  archived: boolean
  page: number
  perPage: number
}

// displayDevEUI writes a DevEUI in upper case, as device labels show it.
export function displayDevEUI(devEUI: string): string {
  return devEUI.toUpperCase()
}

// useDeviceList loads pages of the device list. The page calls load each
// time its filter changes.
export function useDeviceList() {
  const devices = shallowRef<Device[]>([])
  const total = shallowRef(0)
  const loading = shallowRef(false)
  const error = shallowRef<unknown>()

  async function load(filter: DeviceFilter) {
    loading.value = true
    error.value = undefined
    try {
      const { data } = await api.get<{ devices: Device[]; total: number }>('/devices', {
        params: {
          q: filter.q || undefined,
          model_id: filter.modelId ?? undefined,
          meter_type: filter.meterType ?? undefined,
          archived: filter.archived ? 'true' : undefined,
          page: filter.page,
          per_page: filter.perPage,
        },
      })
      devices.value = data.devices
      total.value = data.total
    } catch (err) {
      error.value = err
    } finally {
      loading.value = false
    }
  }

  return { devices, total, loading, error, load }
}

export async function fetchDevice(id: number): Promise<Device> {
  const { data } = await api.get<Device>(`/devices/${id}`)
  return data
}

export async function createDevice(input: DeviceInput): Promise<Device> {
  const { data } = await api.post<Device>('/devices', input)
  return data
}

export async function updateDevice(id: number, input: DeviceInput): Promise<Device> {
  const { data } = await api.put<Device>(`/devices/${id}`, input)
  return data
}

export async function archiveDevice(id: number) {
  await api.post(`/devices/${id}/archive`)
}

export async function archiveDevices(ids: number[]): Promise<number> {
  const { data } = await api.post<{ archived: number }>('/devices/archive', { ids })
  return data.archived
}

export async function restoreDevice(id: number) {
  await api.post(`/devices/${id}/restore`)
}

// findArchived returns the archived device with a DevEUI, for the restore
// offer of a taken DevEUI.
export async function findArchived(devEUI: string): Promise<Device | undefined> {
  const { data } = await api.get<{ devices: Device[] }>('/devices', {
    params: { q: devEUI, archived: 'true', per_page: 10 },
  })
  const want = devEUI.toLowerCase().replace(/[:\-\s]/g, '')
  return data.devices.find((d) => d.dev_eui === want)
}
