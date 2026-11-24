// Where vue/ is, for the scripts that scan it. Every script reads ROOT from
// here instead of deriving it from its own location.
//
// The scripts run under node's type stripping, where the `@/` alias does not
// resolve and an import needs a literal `.ts` extension.

import { existsSync } from 'node:fs'
import { dirname, join, parse, resolve } from 'node:path'

// MARKER is what says "the repository root". package.json would stop at vue/
// itself, which is the directory below the one that holds the Go tree these
// scripts never read but sit inside.
const MARKER = 'go.mod'

// repoRoot is the directory holding go.mod, found by walking up from this file.
function repoRoot(): string {
  const start = import.meta.dirname

  let dir = start
  for (;;) {
    if (existsSync(join(dir, MARKER))) {
      return dir
    }
    const up = dirname(dir)
    if (up === dir || dir === parse(dir).root) {
      throw new Error(`no ${MARKER} above ${start}: cannot find the repository root`)
    }
    dir = up
  }
}

// ROOT is vue/, which is what every glob in these scripts is relative to and
// what buildTagTable and its siblings take as a parameter.
export const ROOT = resolve(repoRoot(), 'vue')

// SRC is vue/src, the tree they scan.
export const SRC = resolve(ROOT, 'src')
