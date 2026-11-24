// Test helpers. Only specs import this file.
import { mount, type ComponentMountingOptions } from '@vue/test-utils'
import type { Component } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { createVuetify } from 'vuetify'

import { routes } from '@/router'

// mountApp mounts a component with the plugins the app installs: Vuetify and
// an in-memory router.
export function mountApp<C extends Component>(
  component: C,
  options: ComponentMountingOptions<C> = {},
) {
  const vuetify = createVuetify()
  const router = createRouter({ history: createMemoryHistory(), routes })
  return mount(component, {
    ...options,
    global: { ...options.global, plugins: [vuetify, router] },
  } as ComponentMountingOptions<C>)
}
