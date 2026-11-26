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
      if (url.endsWith('/uplinks')) {
        return response(200, { uplinks: [], total: 0 })
      }
      return response(200, {
        id: 7,
        dev_eui: '70b3d57ed0001234',
        name: 'Bloc A',
        device_model: { id: 3, vendor: 'Milesight', model: 'EM300-DI', has_decoder: false },
        meter_type: 'gas',
        address: null,
        note: null,
        archived_at: '2026-11-25T10:00:00Z',
        created_at: '',
        updated_at: '',
        groups: [{ id: 4, name: 'Botanica' }],
        active_alarms: [],
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
      if (url.endsWith('/uplinks')) {
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
        device_model: { id: 3, vendor: 'Milesight', model: 'EM300-DI', has_decoder: false },
        meter_type: null,
        address: null,
        note: null,
        archived_at: null,
        created_at: '',
        updated_at: '',
        groups: [],
        active_alarms: [],
        last_uplink_at: '2026-11-25T09:34:15Z',
      })
    })
    const wrapper = mountApp(DevicePage)
    await wrapper.vm.$router.push('/devices/7')
    await flushPromises()

    expect(wrapper.get('[data-test="uplink-payload"]').text()).toBe('01 75 67 02 01')
    expect(wrapper.get('[data-test="device-last-uplink"]').text()).toContain('11:34')
  })

  it('shows the daily pulses and the alarms of a decoded device', async () => {
    vi.spyOn(api, 'get').mockImplementation(async (url: string) => {
      if (url.endsWith('/uplinks')) {
        return response(200, { uplinks: [], total: 0 })
      }
      if (url.endsWith('/pulses')) {
        return response(200, {
          points: [
            { at: '2026-11-24T09:00:00Z', pulse_count: 1000 },
            { at: '2026-11-25T09:00:00Z', pulse_count: 1250 },
          ],
          days: [
            { day: '2026-11-24', last: 1000, delta: null },
            { day: '2026-11-25', last: 1250, delta: 250 },
          ],
        })
      }
      if (url.endsWith('/alarms')) {
        return response(200, {
          alarms: [
            { id: 1, kind: 'low_battery', raised_at: '2026-11-25T09:00:00Z', cleared_at: null },
          ],
          total: 1,
        })
      }
      return response(200, {
        id: 7,
        dev_eui: '70b3d57ed0001234',
        name: 'Bloc A',
        device_model: { id: 3, vendor: 'Milesight', model: 'EM300-DI', has_decoder: true },
        meter_type: 'water',
        address: null,
        note: null,
        archived_at: null,
        created_at: '',
        updated_at: '',
        groups: [],
        active_alarms: ['low_battery'],
        last_uplink_at: '2026-11-25T09:00:00Z',
      })
    })
    // A canvas does not exist in jsdom; the chart components are stubbed.
    const wrapper = mountApp(DevicePage, {
      global: { stubs: { PulseLineChart: true, DailyPulseChart: true } },
    })
    await wrapper.vm.$router.push('/devices/7')
    await flushPromises()

    expect(wrapper.get('[data-test="device-active-alarm"]').text()).toBe('Baterie descărcată')
    const deltas = wrapper.findAll('[data-test="daily-delta"]').map((w) => w.text())
    expect(deltas).toEqual(['250', '—'])
    expect(wrapper.get('[data-test="alarm-kind"]').text()).toBe('Baterie descărcată')
    expect(wrapper.get('[data-test="alarm-active"]').text()).toBe('activă')
  })
})
