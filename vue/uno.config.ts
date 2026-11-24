// UnoCSS configuration. The colors come from src/lib/tokens.ts, which is the one
// place a value of the app's look is written. Keep the dark selector and the
// layer names aligned with src/plugins/vuetify.ts and public/layers.css.
import { defineConfig, presetWind4 } from 'unocss'

import { cssVariables, TOKEN_NAMES, type TokenName } from './src/lib/tokens'

// A token is a CSS variable per theme, so a utility follows the theme with no
// `dark:` pair in markup.
const colors = Object.fromEntries(
  TOKEN_NAMES.map((name: TokenName) => [name, `var(--app-${name})`]),
)

export default defineConfig({
  presets: [
    presetWind4({
      preflights: { reset: true, theme: 'on-demand' },
      dark: {
        dark: '.v-theme--dark',
        light: '.v-theme--light',
      },
    }),
  ],
  theme: { colors },
  preflights: [{ getCSS: () => cssVariables() }],
  shortcuts: {
    // A card or panel shell. Use it in place of v-card.
    'app-card': 'rounded-lg border border-subtle bg-surface',
    'app-card-header': 'px-4 py-3 text-base font-medium text-strong border-b border-subtle',
    'app-card-actions': 'flex justify-end gap-2 p-4',
    // An inline text link.
    'app-link': 'text-primary underline-offset-2 hover:underline',
  },
  outputToCssLayers: {
    cssLayerName: (layer) => (layer === 'properties' ? null : `uno-${layer}`),
  },
  content: {
    pipeline: {
      include: [/\/src\/.*\.(vue|[jt]sx?)($|\?)/],
    },
  },
})
