import { fileURLToPath, URL } from 'node:url'

import Vue from '@vitejs/plugin-vue'
import UnoCSS from 'unocss/vite'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { defineConfig } from 'vite'
import Vuetify, { transformAssetUrls } from 'vite-plugin-vuetify'

import { autoImportDirs, componentDirs, excludeComponentNames } from './packages.ts'

export default defineConfig({
  plugins: [
    // Vue APIs, the router, VueUse and every composable in src/use are globals.
    // Do not import them by hand.
    AutoImport({
      imports: ['vue', 'vue-router', '@vueuse/core'],
      dirs: autoImportDirs,
      dts: 'src/auto-imports.d.ts',
      vueTemplate: true,
    }),
    // Shared components are globals. packages.ts states the directories and the
    // naming rule.
    Components({
      dirs: componentDirs,
      excludeNames: excludeComponentNames,
      directoryAsNamespace: true,
      collapseSamePrefixes: true,
      dts: 'src/components.d.ts',
    }),
    Vue({ template: { transformAssetUrls } }),
    Vuetify({
      autoImport: true,
      styles: { configFile: 'src/styles/settings.scss' },
    }),
    UnoCSS(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    proxy: {
      // `task dev` runs the Go server on :8080.
      '/api': {
        target: process.env.VITE_DEV_PROXY_API ?? 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
