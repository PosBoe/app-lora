import { api } from '@/lib/api'

// Uplink mirrors uplinkResponse in device/uplink_handler.go.
export interface Uplink {
  id: number
  received_at: string
  f_cnt: number
  payload_hex: string
}

// Unmatched mirrors unmatchedResponse in device/uplink_handler.go.
export interface Unmatched {
  dev_eui: string
  count: number
  last_received_at: string
  integration_name: string | null
  deletes_at: string
}

// UPLINKS_PER_PAGE mirrors defaultPerPage in device/device_handler.go.
export const UPLINKS_PER_PAGE = 50

// displayPayload writes a hex payload in upper case with a space between the
// bytes, as LoRaWAN tools show it.
export function displayPayload(hex: string): string {
  return (hex.toUpperCase().match(/../g) ?? []).join(' ')
}

// daysLeft is the number of whole days until iso, never below 0.
export function daysLeft(iso: string, now = Date.now()): number {
  return Math.max(0, Math.ceil((new Date(iso).getTime() - now) / 86_400_000))
}

// useUnmatched loads the unmatched DevEUIs of the organization when the
// component mounts.
export function useUnmatched() {
  const unmatched = shallowRef<Unmatched[]>([])

  async function reload() {
    try {
      const { data } = await api.get<{ dev_euis: Unmatched[] }>('/unmatched')
      unmatched.value = data.dev_euis
    } catch {
      // The warning is a hint; the device list works without it.
      unmatched.value = []
    }
  }

  onMounted(reload)

  return { unmatched, reload }
}

// useDeviceUplinks loads pages of the uplinks of one device.
export function useDeviceUplinks() {
  const uplinks = shallowRef<Uplink[]>([])
  const total = shallowRef(0)
  const loading = shallowRef(false)
  const error = shallowRef<unknown>()

  async function load(deviceId: number, page: number) {
    loading.value = true
    error.value = undefined
    try {
      const { data } = await api.get<{ uplinks: Uplink[]; total: number }>(
        `/devices/${deviceId}/uplinks`,
        { params: { page } },
      )
      uplinks.value = data.uplinks
      total.value = data.total
    } catch (err) {
      error.value = err
    } finally {
      loading.value = false
    }
  }

  return { uplinks, total, loading, error, load }
}
