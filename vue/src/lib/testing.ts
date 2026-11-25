// Test helpers. Only specs import this file.
import { mount, type ComponentMountingOptions } from '@vue/test-utils'
import type { Component } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { createVuetify } from 'vuetify'

import { createAppI18n } from '@/plugins/i18n'
import { routes } from '@/router'

// mountApp mounts a component with the plugins the app installs: vue-i18n in
// Romanian, Vuetify, and an in-memory router without the guards.
export function mountApp<C extends Component>(
  component: C,
  options: ComponentMountingOptions<C> = {},
) {
  const i18n = createAppI18n('ro')
  const vuetify = createVuetify()
  const router = createRouter({ history: createMemoryHistory(), routes })
  return mount(component, {
    ...options,
    global: { ...options.global, plugins: [i18n, vuetify, router] },
  } as ComponentMountingOptions<C>)
}
