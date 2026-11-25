import { AxiosHeaders, type AxiosResponse } from 'axios'
import { flushPromises } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { mountApp } from '@/lib/testing'

import OrganizationsPage from './OrganizationsPage.vue'
import type { Organization } from './use-organizations'

function response(status: number, data: unknown): AxiosResponse {
  return { status, statusText: '', data, headers: {}, config: { headers: new AxiosHeaders() } }
}

const active: Organization = {
  id: 1,
  name: 'Apă Canal',
  deactivated_at: null,
  created_at: '2026-11-25T10:00:00Z',
  device_count: 12,
  users: [
    {
      id: 10,
      login: 'apa',
      password_temporary: false,
      first_name: 'Ion',
      last_name: 'Popescu',
      email: 'ion@example.md',
      phone: '+37369123456',
      last_login_at: '2026-11-25T11:00:00Z',
    },
  ],
}

const deactivated: Organization = {
  id: 2,
  name: 'Gaz Nord',
  deactivated_at: '2026-11-25T12:00:00Z',
  created_at: '2026-11-24T10:00:00Z',
  device_count: 0,
  users: [
    {
      id: 20,
      login: 'gaz',
      password_temporary: true,
      first_name: null,
      last_name: null,
      email: null,
      phone: null,
      last_login_at: null,
    },
  ],
}

// byTest finds an element that a dialog teleported to the document body.
function byTest(name: string): HTMLElement | null {
  return document.body.querySelector(`[data-test="${name}"]`)
}

describe('OrganizationsPage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    document.body.innerHTML = ''
  })

  it('lists the organizations with their status and profile', async () => {
    vi.spyOn(api, 'get').mockResolvedValue(response(200, { organizations: [active, deactivated] }))
    const wrapper = mountApp(OrganizationsPage)
    await flushPromises()

    const names = wrapper.findAll('[data-test="organization-row-name"]').map((w) => w.text())
    expect(names).toEqual(['Apă Canal', 'Gaz Nord'])
    const statuses = wrapper.findAll('[data-test="organization-status"]').map((w) => w.text())
    expect(statuses).toEqual(['Activă', 'Dezactivată'])
    expect(wrapper.text()).toContain('Ion Popescu · ion@example.md · +37369123456')
    expect(wrapper.get('[data-test="profile-missing"]').text()).toBe('Profil necompletat')
  })

  it('creates an organization and shows its temporary password', async () => {
    const get = vi.spyOn(api, 'get').mockResolvedValue(response(200, { organizations: [] }))
    const post = vi
      .spyOn(api, 'post')
      .mockResolvedValue(
        response(201, { organization: active, temporary_password: 'Kp7mWx3qRt9vZb2n' }),
      )
    const wrapper = mountApp(OrganizationsPage, { attachTo: document.body })
    await flushPromises()

    await wrapper.get('[data-test="organization-add"]').trigger('click')
    await flushPromises()
    const nameInput = byTest('organization-name-input')?.querySelector('input')
    const loginInput = byTest('organization-login-input')?.querySelector('input')
    nameInput!.value = 'Apă Canal'
    nameInput!.dispatchEvent(new Event('input'))
    loginInput!.value = 'apa'
    loginInput!.dispatchEvent(new Event('input'))
    await flushPromises()
    byTest('organization-create-dialog')!.dispatchEvent(new Event('submit'))
    await flushPromises()

    expect(post).toHaveBeenCalledWith('/admin/organizations', { name: 'Apă Canal', login: 'apa' })
    expect(get).toHaveBeenCalledTimes(2)
    expect(byTest('temporary-login')?.textContent).toBe('apa')
    expect(byTest('temporary-password')?.textContent).toBe('Kp7mWx3qRt9vZb2n')
    wrapper.unmount()
  })
})
