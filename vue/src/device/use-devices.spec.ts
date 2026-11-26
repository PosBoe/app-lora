import { AxiosHeaders, type AxiosResponse } from 'axios'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { useDeviceList, type DeviceFilter } from './use-devices'

function response(data: unknown): AxiosResponse {
  return { status: 200, statusText: '', data, headers: {}, config: { headers: new AxiosHeaders() } }
}

const filter = (q: string): DeviceFilter => ({
  q,
  modelId: null,
  groupId: null,
  meterType: null,
  archived: false,
  page: 1,
  perPage: 50,
})

describe('useDeviceList', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('keeps the answer to the newest filter when an older one comes later', async () => {
    let answerOld!: (r: AxiosResponse) => void
    vi.spyOn(api, 'get')
      .mockImplementationOnce(() => new Promise((resolve) => (answerOld = resolve)))
      .mockResolvedValueOnce(response({ devices: [{ id: 2 }], total: 1 }))
    const list = useDeviceList()

    const old = list.load(filter('bl'))
    await list.load(filter('bloc'))
    answerOld(response({ devices: [{ id: 1 }, { id: 3 }], total: 2 }))
    await old

    expect(list.total.value).toBe(1)
    expect(list.devices.value.map((d) => d.id)).toEqual([2])
    expect(list.loading.value).toBe(false)
  })
})
