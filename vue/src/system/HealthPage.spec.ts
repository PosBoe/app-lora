import { AxiosError, AxiosHeaders, type AxiosResponse } from 'axios'
import { flushPromises } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { mountApp } from '@/lib/testing'

import HealthPage from './HealthPage.vue'

function response(status: number, data: unknown): AxiosResponse {
  return { status, statusText: '', data, headers: {}, config: { headers: new AxiosHeaders() } }
}

describe('HealthPage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('shows a healthy server', async () => {
    vi.spyOn(api, 'get').mockResolvedValue(
      response(200, { status: 'ok', db: 'ok', version: 'dev' }),
    )

    const wrapper = mountApp(HealthPage)
    await flushPromises()

    expect(wrapper.get('[data-test="status"]').text()).toBe('ok')
    expect(wrapper.get('[data-test="db"]').text()).toBe('ok')
  })

  it('shows a 503 body as a degraded server', async () => {
    const body = { status: 'degraded', db: 'error', version: 'dev' }
    const err = new AxiosError('503', 'ERR_BAD_RESPONSE', undefined, undefined, response(503, body))
    vi.spyOn(api, 'get').mockRejectedValue(err)

    const wrapper = mountApp(HealthPage)
    await flushPromises()

    expect(wrapper.get('[data-test="status"]').text()).toBe('degraded')
    expect(wrapper.get('[data-test="db"]').text()).toBe('error')
  })

  it('shows a transport failure', async () => {
    vi.spyOn(api, 'get').mockRejectedValue(new Error('Network Error'))

    const wrapper = mountApp(HealthPage)
    await flushPromises()

    expect(wrapper.get('[data-test="error"]').text()).toBe('Network Error')
  })
})
