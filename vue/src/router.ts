import {
  createRouter,
  createWebHistory,
  type RouteLocationRaw,
  type Router,
  type RouteRecordRaw,
} from 'vue-router'

import { api, apiErrorCode } from '@/lib/api'
import type { Me } from '@/use/session'

declare module 'vue-router' {
  interface RouteMeta {
    // public routes need no session.
    public?: boolean
    // admin routes need an admin session.
    admin?: boolean
    // member routes need a member of an organization; an admin goes home.
    member?: boolean
  }
}

// A feature module adds its routes here. Load a page lazily.
export const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/core/LoginPage.vue'),
    meta: { public: true },
  },
  {
    path: '/set-password',
    name: 'SetPassword',
    component: () => import('@/core/SetPasswordPage.vue'),
  },
  {
    path: '/',
    name: 'Home',
    component: () => import('@/device/DevicesPage.vue'),
    meta: { member: true },
  },
  {
    path: '/devices/:id(\\d+)',
    name: 'Device',
    component: () => import('@/device/DevicePage.vue'),
    meta: { member: true },
  },
  {
    path: '/device-models',
    name: 'DeviceModels',
    component: () => import('@/device/DeviceModelsPage.vue'),
    meta: { member: true },
  },
  {
    path: '/admin/organizations',
    name: 'Organizations',
    component: () => import('@/core/OrganizationsPage.vue'),
    meta: { admin: true },
  },
  {
    path: '/health',
    name: 'Health',
    component: () => import('@/system/HealthPage.vue'),
    meta: { public: true },
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/components/page/NotFound.vue'),
    meta: { public: true },
  },
]

export function createAppRouter() {
  const router = createRouter({
    history: createWebHistory(import.meta.env.BASE_URL),
    routes,
  })
  installGuards(router)
  installSessionRedirects(router)
  return router
}

// homeFor is the first page of a user.
function homeFor(me: Me): RouteLocationRaw {
  return me.is_admin ? { name: 'Organizations' } : { name: 'Home' }
}

// installGuards sends each navigation to a page the session may open:
// nobody logged in goes to the login page, a temporary password goes to the
// set-password page, a member never opens an admin page, and an admin never
// opens a member page.
export function installGuards(router: Router) {
  router.beforeEach(async (to) => {
    const me = await useSession().load()

    if (to.meta.public) {
      return to.name === 'Login' && me ? homeFor(me) : true
    }
    if (!me) {
      return { name: 'Login', query: { redirect: to.fullPath } }
    }
    if (me.password_temporary) {
      return to.name === 'SetPassword' ? true : { name: 'SetPassword' }
    }
    if (to.name === 'SetPassword') {
      return homeFor(me)
    }
    if (to.meta.admin && !me.is_admin) {
      return { name: 'Home' }
    }
    if (to.meta.member && me.is_admin) {
      return homeFor(me)
    }
    return true
  })
}

// installSessionRedirects follows a session that ends while a page is open:
// an expired or ended session goes to the login page, and a temporary
// password goes to the set-password page. The /auth routes answer for
// themselves, so they are left alone.
function installSessionRedirects(router: Router) {
  api.interceptors.response.use(undefined, (err) => {
    const url = (err?.config?.url as string | undefined) ?? ''
    if (!url.startsWith('/auth/')) {
      const code = apiErrorCode(err)
      if (code === 'unauthenticated' || code === 'organization_deactivated') {
        useSession().forget()
        void router.push({
          name: 'Login',
          query: { redirect: router.currentRoute.value.fullPath },
        })
      } else if (code === 'password_change_required') {
        void router.push({ name: 'SetPassword' })
      }
    }
    return Promise.reject(err)
  })
}
