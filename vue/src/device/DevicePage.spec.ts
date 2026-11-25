import { AxiosHeaders, type AxiosResponse } from 'axios'
import { flushPromises } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { mountApp } from '@/lib/testing'

import DevicePage from './DevicePage.vue'

function response(status: number, data: unknown): AxiosResponse {
  return { status, statusText: '', data, headers: {}, config: { headers: new AxiosHeaders() } }
}

describe('DevicePage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('shows the device and says why each data section is empty', async () => {
    vi.spyOn(api, 'get').mockImplementation(async (url: string) => {
      if (url === '/devices/7/uplinks') {
        return response(200, { uplinks: [], total: 0 })
      }
      return response(200, {
        id: 7,
        dev_eui: '70b3d57ed0001234',
        name: 'Bloc A',
        device_model: { id: 3, vendor: 'Milesight', model: 'EM300-DI' },
        meter_type: 'gas',
        address: null,
        note: null,
        archived_at: '2026-11-25T10:00:00Z',
        created_at: '',
        updated_at: '',
        groups: [{ id: 4, name: 'Botanica' }],
        last_uplink_at: null,
      })
    })
    const wrapper = mountApp(DevicePage)
    await wrapper.vm.$router.push('/devices/7')
    await flushPromises()

    expect(wrapper.get('[data-test="device-title"]').text()).toBe('Bloc A')
    expect(wrapper.get('[data-test="device-dev-eui"]').text()).toBe('70B3D57ED0001234')
    expect(wrapper.get('[data-test="device-model"]').text()).toBe('Milesight EM300-DI')
    expect(wrapper.get('[data-test="device-groups"]').text()).toBe('Botanica')
    expect(wrapper.find('[data-test="archived-banner"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="section-pulses"]').text()).toContain(
      'Apar când modelul acestui device are un decodor.',
    )
    expect(wrapper.get('[data-test="section-uplinks"]').text()).toContain('Niciun uplink încă.')
  })

  it('lists the uplinks with the payload in hex bytes', async () => {
    vi.spyOn(api, 'get').mockImplementation(async (url: string) => {
      if (url === '/devices/7/uplinks') {
        return response(200, {
          uplinks: [
            { id: 1, received_at: '2026-11-25T09:34:15Z', f_cnt: 42, payload_hex: '0175670201' },
          ],
          total: 1,
        })
      }
      return response(200, {
        id: 7,
        dev_eui: '70b3d57ed0001234',
        name: 'Bloc A',
        device_model: { id: 3, vendor: 'Milesight', model: 'EM300-DI' },
        meter_type: null,
        address: null,
        note: null,
        archived_at: null,
        created_at: '',
        updated_at: '',
        groups: [],
        last_uplink_at: '2026-11-25T09:34:15Z',
      })
    })
    const wrapper = mountApp(DevicePage)
    await wrapper.vm.$router.push('/devices/7')
    await flushPromises()

    expect(wrapper.get('[data-test="uplink-payload"]').text()).toBe('01 75 67 02 01')
    expect(wrapper.get('[data-test="device-last-uplink"]').text()).toContain('11:34')
  })
})
