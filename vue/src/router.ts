import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

// A feature module adds its routes here. Load a page lazily.
export const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'Health',
    component: () => import('@/system/HealthPage.vue'),
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/components/page/NotFound.vue'),
  },
]

export function createAppRouter() {
  return createRouter({
    history: createWebHistory(import.meta.env.BASE_URL),
    routes,
  })
}
