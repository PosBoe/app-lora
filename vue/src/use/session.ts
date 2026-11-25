import { isAxiosError } from 'axios'

import { api } from '@/lib/api'
import type { Language } from '@/lib/i18n'

// Me mirrors meResponse in core/auth_handler.go: the logged-in user.
export interface Me {
  id: number
  login: string
  is_admin: boolean
  password_temporary: boolean
  language: Language
  organization: { id: number; name: string } | null
}

// useSession is the one session state of the app. `me` is null when nobody
// is logged in. load() asks the server one time and caches the answer;
// login, logout and forget update the cache.
export const useSession = createGlobalState(() => {
  const me = shallowRef<Me | null>(null)
  let loaded: Promise<Me | null> | undefined

  function load(): Promise<Me | null> {
    loaded ??= fetchMe()
    return loaded
  }

  async function fetchMe(): Promise<Me | null> {
    try {
      const { data } = await api.get<Me>('/auth/me')
      me.value = data
    } catch (err) {
      me.value = null
      // Only a 401 is an answer. Ask again on the next load after any other
      // failure, such as a server that is down.
      if (!isAxiosError(err) || err.response?.status !== 401) {
        loaded = undefined
      }
    }
    return me.value
  }

  async function login(login: string, password: string): Promise<Me> {
    const { data } = await api.post<Me>('/auth/login', { login, password })
    remember(data)
    return data
  }

  async function logout() {
    try {
      await api.post('/auth/logout')
    } finally {
      forget()
    }
  }

  async function changePassword(currentPassword: string, newPassword: string) {
    await api.post('/auth/password', {
      current_password: currentPassword,
      new_password: newPassword,
    })
    if (me.value) {
      remember({ ...me.value, password_temporary: false })
    }
  }

  function remember(next: Me) {
    me.value = next
    loaded = Promise.resolve(next)
  }

  // forget drops the session state, for example after the server answered
  // that the session ended.
  function forget() {
    me.value = null
    loaded = Promise.resolve(null)
  }

  return { me, load, login, logout, changePassword, forget }
})
