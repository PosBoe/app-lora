import { AxiosHeaders, type AxiosResponse } from 'axios'
import { flushPromises } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { mountApp } from '@/lib/testing'

import AdminDeviceModelsPage from './AdminDeviceModelsPage.vue'

function response(status: number, data: unknown): AxiosResponse {
  return { status, statusText: '', data, headers: {}, config: { headers: new AxiosHeaders() } }
}

const model = {
  id: 3,
  organization_id: 7,
  organization_name: 'Apă Canal',
  vendor: 'Milesight',
  model: 'EM300-DI',
  decoder: null,
  device_count: 12,
  pending_uplinks: 0,
}

describe('AdminDeviceModelsPage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('assigns a decoder to a model and shows the decoding progress', async () => {
    let decoded = false
    vi.spyOn(api, 'get').mockImplementation(async (url: string) => {
      if (url === '/admin/decoders') {
        return response(200, {
          decoders: [{ id: 'milesight_em300_di', name: 'Milesight EM300-DI' }],
        })
      }
      const m = decoded ? { ...model, decoder: 'milesight_em300_di', pending_uplinks: 40 } : model
      return response(200, { device_models: [m] })
    })
    const put = vi.spyOn(api, 'put').mockImplementation(async () => {
      decoded = true
      return response(204, '')
    })
    const wrapper = mountApp(AdminDeviceModelsPage)
    await flushPromises()

    expect(wrapper.text()).toContain('Apă Canal')
    wrapper.findComponent({ name: 'VSelect' }).vm.$emit('update:modelValue', 'milesight_em300_di')
    await flushPromises()

    expect(put).toHaveBeenCalledWith('/admin/device-models/3/decoder', {
      decoder: 'milesight_em300_di',
    })
    expect(wrapper.get('[data-test="decoding-3"]').text()).toBe('se decodează 40 uplink-uri…')
  })
})
