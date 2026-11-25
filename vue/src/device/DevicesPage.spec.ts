import { AxiosHeaders, type AxiosResponse } from 'axios'
import { flushPromises } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { mountApp } from '@/lib/testing'

import DevicesPage from './DevicesPage.vue'
import type { Device } from './use-devices'

function response(status: number, data: unknown): AxiosResponse {
  return { status, statusText: '', data, headers: {}, config: { headers: new AxiosHeaders() } }
}

const device: Device = {
  id: 7,
  dev_eui: '70b3d57ed0001234',
  name: 'Bloc A',
  device_model: { id: 3, vendor: 'Milesight', model: 'EM300-DI' },
  meter_type: 'water',
  address: 'str. Mare 1',
  note: null,
  archived_at: null,
  created_at: '2026-11-25T10:00:00Z',
  updated_at: '2026-11-25T10:00:00Z',
  groups: [],
}

// mockGet answers the device list and the device model list.
function mockGet(devices: Device[], total = devices.length) {
  return vi.spyOn(api, 'get').mockImplementation(async (url: string) => {
    if (url === '/device-groups') {
      return response(200, { device_groups: [] })
    }
    if (url === '/device-models') {
      return response(200, {
        device_models: [
          { id: 3, vendor: 'Milesight', model: 'EM300-DI', device_count: 1, created_at: '' },
        ],
      })
    }
    return response(200, { devices, total })
  })
}

describe('DevicesPage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('lists the devices with the DevEUI in upper case', async () => {
    mockGet([device], 1)
    const wrapper = mountApp(DevicesPage)
    await flushPromises()

    expect(wrapper.get('[data-test="device-row-name"]').text()).toBe('Bloc A')
    expect(wrapper.text()).toContain('70B3D57ED0001234')
    expect(wrapper.text()).toContain('Milesight EM300-DI')
    expect(wrapper.text()).toContain('Apă')
    expect(wrapper.get('[data-test="devices-total"]').text()).toBe('(1)')
  })

  it('shows the empty state for an organization with no device', async () => {
    mockGet([], 0)
    const wrapper = mountApp(DevicesPage)
    await flushPromises()

    expect(wrapper.find('[data-test="devices-empty"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="devices-table"]').exists()).toBe(false)
  })

  it('reads the filter from the URL and sends it to the server', async () => {
    const get = mockGet([device], 1)
    const wrapper = mountApp(DevicesPage)
    const router = wrapper.vm.$router
    await router.push({
      path: '/',
      query: { q: 'bloc', model: '3', group: '4', type: 'water', page: '2' },
    })
    await flushPromises()

    const listCalls = get.mock.calls.filter(([url]) => url === '/devices')
    expect(listCalls.at(-1)?.[1]?.params).toEqual({
      q: 'bloc',
      model_id: 3,
      group_id: 4,
      meter_type: 'water',
      archived: undefined,
      page: 2,
      per_page: 50,
    })
  })
})
