---
name: run-app
description: Launch the lora app (Go API + Vite dev server) on a scratch PostgreSQL database, then drive the web UI with Playwright's Chromium to take screenshots or check a UI change in the real app. Use when asked to run, start, open, or screenshot the app, or to see a page working. Do not use to run tests; use `task test:pkg` or `pnpm test:run` for that.
---

# run-app

Start the app on a new scratch database, drive the UI in Chromium, and look at
screenshots. Then stop everything and drop the database.

The scripts are in `scripts/` next to this file. Run them from any directory.

## Rules

- Do not edit `config/dev.yml`. `up.sh` writes a scratch copy.
- Do not migrate `lora_dev` or `lora_test`. Each run uses its own `lora_scratch_<HHMMSS>` database.
- Always run `down.sh` at the end, also after a failure.
- Take a screenshot after each navigation, and look at it with the Read tool. A green script does not prove that the page shows what you expect.

## Environment

- PostgreSQL runs from `docker-compose.yml`. Start it with `task up`. The user is `lora` and the password is `lora`.
- `pnpm` comes from corepack (`task init` enables it).
- Playwright is not a dependency of `vue/`. Before the first drive, install it once from `vue/`: `pnpm add -D playwright && pnpm exec playwright install chromium`. Ask the user before you add the dependency.

## 1. Start

```sh
.agents/skills/run-app/scripts/up.sh
```

The script does these steps:

1. It copies `config/dev.yml` (or `config/dev.default.yml`) to a work directory, `${TMPDIR:-/tmp}/run-app-<db>/`. It points `pg.dsn` at the scratch database and `listen.http` at port 18080.
2. It builds `./cmd/lora` with `-tags=invariants`. Then it runs `pg create_db` and `migrate up` with `LORA_CONFIG` set to the copy.
3. It starts `serve` (:18080) and the Vite dev server (:15173), which proxies `/api` to :18080. Each runs in its own process group.
4. It returns when `localhost:18080/api/v1/health` and `localhost:15173/` both answer 200.

Set `RUN_APP_HTTP_PORT` and `RUN_APP_VITE_PORT` to use other ports. If a port already has a listener, the script stops. Find the owner with `ss -ltnp "sport = :15173"`. If it is an earlier run, run `down.sh` for it.

The last line of the output is the state file. Keep it for `down.sh`. The logs `setup.log`, `serve.log` and `vite.log` are in the same directory.

## 2. Drive

Write a Node `.mjs` script that imports `scripts/drive.mjs`:

```js
import { openApp, shot } from '<repo>/.agents/skills/run-app/scripts/drive.mjs'

const app = await openApp()
try {
  await app.page.goto('/')
  await app.page.getByText('System health').waitFor()
  await shot(app.page, 'health')
} finally {
  console.log('failures:', app.failures)
  await app.close()
}
```

- `openApp()` records page errors and every `/api/` response with a status of 400 or more in `app.failures`.
- `shot(page, name)` writes `<name>.png` to `RUN_APP_SHOTS`, or to the work directory of the run, and prints the path. Read the file to see it.
- Wait for an element of the page, not for a timeout.

## 3. Stop

```sh
.agents/skills/run-app/scripts/down.sh <state file> [--keep-logs]
```

It stops both process groups, drops the scratch database, and removes the work directory. With `--keep-logs`, it keeps the logs and the screenshots.

## Report

Say which pages you opened, show the screenshots you read, and list `app.failures`. If a step failed, give the log lines from the work directory.
