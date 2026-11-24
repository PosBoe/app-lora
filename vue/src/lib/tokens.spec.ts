import { describe, expect, it } from 'vitest'

import { cssVariables, TOKEN_NAMES, TOKENS } from './tokens'

describe('tokens', () => {
  it('defines every token in every theme', () => {
    for (const theme of Object.values(TOKENS)) {
      expect(Object.keys(theme).sort()).toEqual([...TOKEN_NAMES].sort())
    }
  })

  it('writes one variable block per theme', () => {
    const css = cssVariables()
    expect(css).toContain('.v-theme--light{')
    expect(css).toContain('.v-theme--dark{')
    expect(css).toContain(`--app-primary:${TOKENS.light.primary};`)
  })
})
