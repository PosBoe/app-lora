import { AxiosHeaders, type AxiosResponse } from 'axios'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'

import { api } from '@/lib/api'
import { installGuards, routes } from '@/router'
import { useSession, type Me } from '@/use/session'

function response(status: number, data: unknown): AxiosResponse {
  return { status, statusText: '', data, headers: {}, config: { headers: new AxiosHeaders() } }
}

const member: Me = {
  id: 1,
  login: 'apa',
  is_admin: false,
  password_temporary: false,
  language: 'ro',
  organization: { id: 7, name: 'Apă Canal' },
}

const admin: Me = { ...member, id: 2, login: 'root', is_admin: true, organization: null }

// signIn puts me in the session state through the login call.
async function signIn(me: Me) {
  vi.spyOn(api, 'post').mockResolvedValueOnce(response(200, me))
  await useSession().login(me.login, 'password')
}

function newRouter() {
  const router = createRouter({ history: createMemoryHistory(), routes })
  installGuards(router)
  return router
}

describe('route guards', () => {
  beforeEach(() => {
    useSession().forget()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('sends a visitor to the login page and keeps the target', async () => {
    const router = newRouter()
    await router.push('/admin/organizations')
    expect(router.currentRoute.value.name).toBe('Login')
    expect(router.currentRoute.value.query.redirect).toBe('/admin/organizations')
  })

  it('lets a visitor open a public page', async () => {
    const router = newRouter()
    await router.push('/health')
    expect(router.currentRoute.value.name).toBe('Health')
  })

  it('keeps a temporary password on the set-password page', async () => {
    await signIn({ ...member, password_temporary: true })
    const router = newRouter()
    await router.push('/')
    expect(router.currentRoute.value.name).toBe('SetPassword')
  })

  it('keeps a member out of the admin pages', async () => {
    await signIn(member)
    const router = newRouter()
    await router.push('/admin/organizations')
    expect(router.currentRoute.value.name).toBe('Home')
  })

  it('keeps an admin out of the member pages', async () => {
    await signIn(admin)
    const router = newRouter()
    await router.push('/device-models')
    expect(router.currentRoute.value.name).toBe('Organizations')
    await router.push('/devices/5')
    expect(router.currentRoute.value.name).toBe('Organizations')
  })

  it('lets a member open a device page', async () => {
    await signIn(member)
    const router = newRouter()
    await router.push('/devices/5')
    expect(router.currentRoute.value.name).toBe('Device')
    expect(router.currentRoute.value.params.id).toBe('5')
  })

  it('sends an admin from the home page to the organizations', async () => {
    await signIn(admin)
    const router = newRouter()
    await router.push('/')
    expect(router.currentRoute.value.name).toBe('Organizations')
  })

  it('sends a signed-in user away from the login and set-password pages', async () => {
    await signIn(member)
    const router = newRouter()
    await router.push('/login')
    expect(router.currentRoute.value.name).toBe('Home')
    await router.push('/set-password')
    expect(router.currentRoute.value.name).toBe('Home')
  })
})
