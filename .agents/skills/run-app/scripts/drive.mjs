// drive.mjs opens the app that up.sh started in Playwright's Chromium.
// Import it from a plain Node .mjs script.
//
//   import { openApp, shot } from '<skill>/scripts/drive.mjs'
//   const app = await openApp()
//   await app.page.goto('/')
//   await shot(app.page, 'home')
//   await app.close()
//
// Playwright resolves from vue/node_modules. Screenshots go to RUN_APP_SHOTS,
// or to the current directory.
import { execFileSync } from 'node:child_process'
import { createRequire } from 'node:module'
import path from 'node:path'

const repo = execFileSync('git', ['rev-parse', '--show-toplevel'], {
  cwd: path.dirname(new URL(import.meta.url).pathname),
  encoding: 'utf8',
}).trim()

let chromium
try {
  ;({ chromium } = createRequire(path.join(repo, 'vue/package.json'))('playwright'))
} catch {
  throw new Error(
    'drive.mjs: playwright is not installed. From vue/, run: pnpm add -D playwright && pnpm exec playwright install chromium',
  )
}

export const BASE_URL = process.env.RUN_APP_URL ?? `http://localhost:${process.env.RUN_APP_VITE_PORT ?? 15173}`

// openApp launches Chromium. It records page errors and failed /api/ reads in
// app.failures, so a script can assert on them.
export async function openApp({ headless = true, viewport = { width: 1440, height: 900 } } = {}) {
  const browser = await chromium.launch({ headless })
  const context = await browser.newContext({ baseURL: BASE_URL, viewport })
  const page = await context.newPage()
  const failures = []
  page.on('pageerror', (err) => {
    failures.push(`pageerror: ${err.message}`)
    console.log('pageerror:', err.message)
  })
  page.on('response', (resp) => {
    if (resp.url().includes('/api/') && resp.status() >= 400) {
      const line = `${resp.status()} ${resp.request().method()} ${resp.url()}`
      failures.push(line)
      console.log('http:', line)
    }
  })
  return { browser, context, page, failures, close: () => browser.close() }
}

// shot writes a full-page screenshot and prints its path.
export async function shot(page, name) {
  const dir = process.env.RUN_APP_SHOTS ?? process.cwd()
  const file = path.join(dir, `${name}.png`)
  await page.screenshot({ path: file, fullPage: true })
  console.log('shot:', file)
  return file
}
