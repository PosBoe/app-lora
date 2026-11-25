import { AxiosError, AxiosHeaders, type AxiosResponse } from 'axios'
import { flushPromises } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { mountApp } from '@/lib/testing'

import DeviceDialog from './DeviceDialog.vue'
import type { Device } from './use-devices'

function response(status: number, data: unknown): AxiosResponse {
  return { status, statusText: '', data, headers: {}, config: { headers: new AxiosHeaders() } }
}

function apiError(status: number, code: string) {
  const res = response(status, { code, message: code })
  return new AxiosError(String(status), 'ERR_BAD_REQUEST', undefined, undefined, res)
}

const device: Device = {
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
}

// byTest finds an element that the dialog teleported to the document body.
function byTest(name: string): HTMLElement | null {
  return document.body.querySelector(`[data-test="${name}"]`)
}

describe('DeviceDialog', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    document.body.innerHTML = ''
  })

  it('offers to restore an archived device that has the DevEUI', async () => {
    vi.spyOn(api, 'get').mockImplementation(async (url: string) => {
      if (url === '/device-groups') {
        return response(200, { device_groups: [] })
      }
      if (url === '/device-models') {
        return response(200, { device_models: [] })
      }
      return response(200, {
        devices: [{ ...device, archived_at: '2026-11-25T10:00:00Z' }],
        total: 1,
      })
    })
    const post = vi.spyOn(api, 'put').mockRejectedValue(apiError(409, 'dev_eui_archived'))
    const restore = vi.spyOn(api, 'post').mockResolvedValue(response(204, ''))

    const wrapper = mountApp(DeviceDialog, {
      props: { modelValue: true, device },
      attachTo: document.body,
    })
    await flushPromises()

    byTest('device-dialog')!.dispatchEvent(new Event('submit'))
    await flushPromises()

    expect(post).toHaveBeenCalled()
    expect(byTest('device-error')?.textContent).toBe(
      'Un device arhivat are acest DevEUI. Îl poți restaura.',
    )
    byTest('device-restore-archived')!.click()
    await flushPromises()

    expect(restore).toHaveBeenCalledWith('/devices/7/restore')
    expect(wrapper.emitted('saved')?.[0]?.[0]).toMatchObject({ id: 7 })
    wrapper.unmount()
  })
})
