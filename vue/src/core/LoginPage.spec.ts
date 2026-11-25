import { AxiosError, AxiosHeaders, type AxiosResponse } from 'axios'
import { flushPromises } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { mountApp } from '@/lib/testing'

import LoginPage from './LoginPage.vue'

function response(status: number, data: unknown): AxiosResponse {
  return { status, statusText: '', data, headers: {}, config: { headers: new AxiosHeaders() } }
}

function apiError(status: number, code: string) {
  const res = response(status, { code, message: code })
  return new AxiosError(String(status), 'ERR_BAD_REQUEST', undefined, undefined, res)
}

async function submit(wrapper: ReturnType<typeof mountApp>) {
  await wrapper.get('[data-test="login-input"] input').setValue('apa')
  await wrapper.get('[data-test="password-input"] input').setValue('secret-password')
  await wrapper.get('[data-test="login-form"]').trigger('submit')
  await flushPromises()
}

describe('LoginPage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it.each([
    ['invalid_credentials', 400, 'Loginul sau parola este greșită.'],
    ['login_locked', 429, 'Prea multe încercări greșite. Încearcă din nou peste câteva minute.'],
    ['organization_deactivated', 403, 'Accesul este suspendat. Contactați administratorul.'],
  ])('shows the message of %s', async (code, status, message) => {
    vi.spyOn(api, 'post').mockRejectedValue(apiError(status, code))
    const wrapper = mountApp(LoginPage)

    await submit(wrapper)

    expect(wrapper.get('[data-test="login-error"]').text()).toBe(message)
  })

  it('shows the generic message for a transport failure', async () => {
    vi.spyOn(api, 'post').mockRejectedValue(new Error('Network Error'))
    const wrapper = mountApp(LoginPage)

    await submit(wrapper)

    expect(wrapper.get('[data-test="login-error"]').text()).toBe(
      'A apărut o eroare. Încearcă din nou.',
    )
  })

  it('sends the login and the password', async () => {
    const post = vi.spyOn(api, 'post').mockResolvedValue(
      response(200, {
        id: 1,
        login: 'apa',
        is_admin: false,
        password_temporary: true,
        language: 'ro',
        organization: { id: 1, name: 'Apă' },
      }),
    )
    const wrapper = mountApp(LoginPage)

    await submit(wrapper)

    expect(post).toHaveBeenCalledWith('/auth/login', {
      login: 'apa',
      password: 'secret-password',
    })
    expect(wrapper.find('[data-test="login-error"]').exists()).toBe(false)
  })
})
