import { useTheme } from 'vuetify'

import { api } from '@/lib/api'
import { TOKENS } from '@/lib/tokens'

// PulseData mirrors pulsesResponse in device/reading_handler.go.
export interface PulseData {
  points: { at: string; pulse_count: number }[]
  days: { day: string; last: number; delta: number | null }[]
}

// Alarm mirrors alarmResponse in device/reading_handler.go.
export interface Alarm {
  id: number
  kind: string
  raised_at: string
  cleared_at: string | null
}

// PULSE_RANGES are the day ranges that the pulse route takes.
export const PULSE_RANGES = [7, 30, 90, 365] as const

export type PulseRange = (typeof PULSE_RANGES)[number]

export async function fetchPulses(deviceId: number, days: PulseRange): Promise<PulseData> {
  const { data } = await api.get<PulseData>(`/devices/${deviceId}/pulses`, { params: { days } })
  return data
}

export async function fetchAlarms(deviceId: number, page: number) {
  const { data } = await api.get<{ alarms: Alarm[]; total: number }>(
    `/devices/${deviceId}/alarms`,
    {
      params: { page },
    },
  )
  return data
}

// useChartTokens returns the colors of the current theme for a chart. A
// chart draws on a canvas, which reads no CSS variable, so it takes the
// token values.
export function useChartTokens() {
  const theme = useTheme()
  return computed(() => TOKENS[theme.current.value.dark ? 'dark' : 'light'])
}

// formatDuration writes the time between two instants in days, hours, or
// minutes, the largest unit that fits.
export function formatDuration(fromIso: string, toIso: string | null, now = Date.now()): string {
  const ms = (toIso ? new Date(toIso).getTime() : now) - new Date(fromIso).getTime()
  const minutes = Math.max(0, Math.round(ms / 60_000))
  if (minutes < 60) {
    return `${minutes} min`
  }
  const hours = Math.round(minutes / 60)
  if (hours < 48) {
    return `${hours} h`
  }
  return `${Math.round(hours / 24)} d`
}
