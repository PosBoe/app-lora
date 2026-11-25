# lora

A platform with a Go backend and a Vue 3 frontend. The product is not defined yet; `PRODUCT.md`
holds the brief.

## Quick start

### Prerequisites

- Go 1.26+
- Node.js 22.18+ with corepack (`task init` enables pnpm)
- [Task](https://taskfile.dev): `go install github.com/go-task/task/v3/cmd/task@latest`
- [watchexec](https://github.com/watchexec/watchexec) for live reload
- Docker with the compose plugin

### Setup

1. **Initialize the checkout** — creates every missing `config/*.yml` from its `.default.yml`
   (existing configs are never overwritten) and enables pnpm:

   ```bash
   task init
   task tools   # golangci-lint
   ```

2. **Start PostgreSQL:**

   ```bash
   task up
   ```

3. **Migrate the dev database** (docker compose creates `lora_dev`):

   ```bash
   task dev:migrate
   ```

4. **Run the backend** with live reload (http://localhost:8080):

   ```bash
   task dev
   ```

5. **Run the frontend** (http://localhost:5173, proxies `/api` to the backend):

   ```bash
   task vue
   ```

Each environment (`dev`, `test`) exposes the database commands, e.g. `task dev:migrate:status`,
`task dev:migrate:create -- add_users`, `task test:reset`. Run `task --list` to see everything.

## Running

The binary is `cmd/lora`:

| Command | What it does |
| --- | --- |
| `lora serve` | Runs the HTTP server: the API under `/api/v1/`, and the SPA when built with `-tags=vue`. |
| `lora migrate up\|down\|status\|create` | Manages the PostgreSQL migrations in `cmd/lora/migrations`. |
| `lora pg create_db\|drop_db\|ping` | Manages the configured database. |
| `lora admin create --login <login>` | Creates an admin and prints its temporary password one time. |
| `lora version` | Prints the version. |

`task build` builds `vue/dist` and then `./bin/lora` with the frontend embedded.

## Tests

- `task test` — `go vet`, the Go tests with `-short`, and the frontend unit tests. Run it before
  a PR.
- `task test:setup` — creates the migrated `lora_test` database for integration tests.
- `task test:full` — every Go test without `-short`, also the tests that use the test database.
- `task lint`, `task fmt` — linters and formatters for both languages.

## Directory map

| Path | Content |
| --- | --- |
| `cmd/lora/` | The binary's `main` and the migrations. |
| `cmd/serve/`, `cmd/migrate/`, `cmd/pg/`, `cmd/admin/` | CLI command groups. |
| `bootstrap/` | CLI root, config loading, logging, the fx application. |
| `system/` | Health endpoint; the reference shape of a domain package. |
| `core/` | Organizations, users, the admin, logins, and sessions. See `core/README.md`. |
| `device/` | The device registry: devices and device models. See `device/README.md`. |
| `pkg/` | Shared, domain-free packages, for example `appconf`, `pgdb`, `httpkit`, `clock`, `pgtest`. |
| `config/` | `*.default.yml` templates; `task init` copies them. |
| `vue/` | The frontend. See `vue/README.md`. |
| `docs/` | Shared guides for Go, the frontend, and agent workflows. |
| `.agents/skills/` | Agent skills; `.claude/skills` links to it. |

## Documentation roles

- `AGENTS.md` (and `CLAUDE.md`, a symlink to it) — operative rules for agents and people.
- `README.md` — a directory map and file tour.
- `DESIGN.md` — a scope's purpose, goals, and frozen contracts.
- `CONTEXT.md` — the domain terms of one bounded context; `CONTEXT-MAP.md` maps them.
- `PRODUCT.md` — users and product purpose.
