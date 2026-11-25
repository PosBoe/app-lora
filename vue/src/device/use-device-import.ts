import { api } from '@/lib/api'

export type ImportStatus = 'new' | 'exists' | 'error'

// ImportRow mirrors importRowResponse in device/import_handler.go.
export interface ImportRow {
  line: number
  status: ImportStatus
  code?: string
  dev_eui: string
  name: string
  vendor?: string
  model?: string
  groups: string[]
  new_model: boolean
  new_groups: string[]
}

// ImportTotals mirrors importTotalsResponse in device/import_handler.go.
export interface ImportTotals {
  new: number
  exists: number
  error: number
  new_models: number
  new_groups: number
}

// TEMPLATE_URL gives the template file. The browser sends the session
// cookie, so a plain link works.
export const TEMPLATE_URL = '/api/v1/devices/import/template'

// The server reads the file as a text/csv body, whatever the browser thinks
// the file type is.
const csv = { headers: { 'Content-Type': 'text/csv' } }

export async function previewImport(file: Blob) {
  const { data } = await api.post<{ rows: ImportRow[]; totals: ImportTotals }>(
    '/devices/import/preview',
    file,
    csv,
  )
  return data
}

export async function commitImport(file: Blob): Promise<ImportTotals> {
  const { data } = await api.post<ImportTotals>('/devices/import', file, csv)
  return data
}
