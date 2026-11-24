import { fileURLToPath } from 'node:url'

import { configDefaults, defineConfig, mergeConfig } from 'vitest/config'

import viteConfig from './vite.config.ts'

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      environment: 'jsdom',
      root: fileURLToPath(new URL('./', import.meta.url)),
      setupFiles: ['./vitest.setup.ts'],
      exclude: [...configDefaults.exclude, 'e2e/**'],
      server: {
        deps: {
          // Vuetify ships CSS imports that only Vite can transform.
          inline: ['vuetify'],
        },
      },
    },
  }),
)
