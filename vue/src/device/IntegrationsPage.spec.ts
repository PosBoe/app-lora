import { AxiosHeaders, type AxiosResponse } from 'axios'
import { flushPromises } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { mountApp } from '@/lib/testing'

import IntegrationsPage from './IntegrationsPage.vue'
import type { Integration } from './use-integrations'

function response(status: number, data: unknown): AxiosResponse {
  return { status, statusText: '', data, headers: {}, config: { headers: new AxiosHeaders() } }
}

const quiet: Integration = {
  id: 1,
  name: 'ChirpStack principal',
  kind: 'chirpstack',
  ingest_url: 'https://lora.md/api/v1/ingest/chirpstack',
  created_at: '2026-11-25T10:00:00Z',
  last_uplink_at: null,
  uplinks_24h: 0,
  last_error: {
    at: '2026-11-25T10:05:00Z',
    code: 'chirpstack_protobuf',
    message: 'set the payload encoding of the ChirpStack integration to JSON',
  },
}

// byTest finds an element that a dialog teleported to the document body.
function byTest(name: string): HTMLElement | null {
  return document.body.querySelector(`[data-test="${name}"]`)
}

describe('IntegrationsPage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    document.body.innerHTML = ''
  })

  it('shows the status and the last error in the UI language', async () => {
    vi.spyOn(api, 'get').mockResolvedValue(response(200, { integrations: [quiet] }))
    const wrapper = mountApp(IntegrationsPage)
    await flushPromises()

    expect(wrapper.get('[data-test="integration-title"]').text()).toBe('ChirpStack principal')
    expect(wrapper.get('[data-test="integration-last-uplink"]').text()).toContain(
      'Niciun uplink încă',
    )
    expect(wrapper.get('[data-test="integration-last-error"]').text()).toContain(
      'ChirpStack trimite Protobuf. Alege JSON la „Payload encoding”.',
    )
  })

  it('shows the secret and the setup steps after a create', async () => {
    vi.spyOn(api, 'get').mockResolvedValue(response(200, { integrations: [] }))
    const post = vi.spyOn(api, 'post').mockResolvedValue(
      response(201, {
        integration: {
          ...quiet,
          id: 2,
          kind: 'ttn',
          ingest_url: 'https://lora.md/api/v1/ingest/ttn',
        },
        secret: 'lora_abc',
      }),
    )
    const wrapper = mountApp(IntegrationsPage, { attachTo: document.body })
    await flushPromises()

    await wrapper.get('[data-test="integration-add"]').trigger('click')
    await flushPromises()
    const name = byTest('integration-name-input')!.querySelector('input')!
    name.value = 'TTN'
    name.dispatchEvent(new Event('input'))
    await flushPromises()
    byTest('integration-dialog')!.dispatchEvent(new Event('submit'))
    await flushPromises()

    expect(post).toHaveBeenCalledWith('/integrations', { name: 'TTN', kind: 'chirpstack' })
    expect(byTest('integration-url')?.textContent).toBe('https://lora.md/api/v1/ingest/ttn')
    expect(byTest('integration-secret')?.textContent?.trim()).toBe('Bearer lora_abc')
    expect(byTest('integration-steps')?.querySelectorAll('li')).toHaveLength(5)
    wrapper.unmount()
  })
})
