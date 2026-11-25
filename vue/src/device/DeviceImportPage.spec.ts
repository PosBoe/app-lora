import { AxiosHeaders, type AxiosResponse } from 'axios'
import { flushPromises } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { mountApp } from '@/lib/testing'

import DeviceImportPage from './DeviceImportPage.vue'

function response(status: number, data: unknown): AxiosResponse {
  return { status, statusText: '', data, headers: {}, config: { headers: new AxiosHeaders() } }
}

const preview = {
  rows: [
    {
      line: 2,
      status: 'new',
      dev_eui: '70b3d57ed0001234',
      name: 'Bloc A',
      vendor: 'Milesight',
      model: 'EM300-DI',
      groups: ['Botanica'],
      new_model: true,
      new_groups: ['Botanica'],
    },
    {
      line: 3,
      status: 'error',
      code: 'dev_eui_invalid',
      dev_eui: '12',
      name: 'x',
      groups: [],
      new_model: false,
      new_groups: [],
    },
  ],
  totals: { new: 1, exists: 0, error: 1, new_models: 1, new_groups: 1 },
}

describe('DeviceImportPage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('previews a file, shows each row, and imports it', async () => {
    const post = vi.spyOn(api, 'post').mockImplementation(async (url: string) => {
      if (url === '/devices/import/preview') {
        return response(200, preview)
      }
      return response(200, preview.totals)
    })
    const wrapper = mountApp(DeviceImportPage)
    const file = new File(['dev_eui,name,vendor,model\n'], 'devices.csv', {
      type: 'application/vnd.ms-excel',
    })

    const input = wrapper.get('[data-test="import-file"] input')
    Object.defineProperty(input.element, 'files', { value: [file] })
    await input.trigger('change')
    await flushPromises()

    expect(post).toHaveBeenCalledWith('/devices/import/preview', file, {
      headers: { 'Content-Type': 'text/csv' },
    })
    expect(wrapper.get('[data-test="import-totals"]').text()).toContain('1 noi')
    const statuses = wrapper.findAll('[data-test="import-row-status"]').map((w) => w.text())
    expect(statuses).toEqual(['Nou', 'Eroare'])
    expect(wrapper.text()).toContain('DevEUI are 16 caractere hex (0–9, A–F).')

    await wrapper.get('[data-test="import-submit"]').trigger('click')
    await flushPromises()

    expect(post).toHaveBeenLastCalledWith('/devices/import', file, {
      headers: { 'Content-Type': 'text/csv' },
    })
    expect(wrapper.get('[data-test="import-result-text"]').text()).toBe(
      'Am adăugat 1 device-uri, 1 modele noi și 1 grupuri noi.',
    )
  })
})
