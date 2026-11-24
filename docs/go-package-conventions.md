# Go package conventions

The naming, export and wiring rules every Go package in this repo follows. A package's own
`AGENTS.md` states only what **differs** from this file, and names the reason.

The root `AGENTS.md` holds the language-level rules — which libraries to use, error handling,
build tags, migrations. This file holds the shape of a package. `system/` is the reference shape.

## Packages and layout

- A domain package lives at the repository root, named for its bounded context (`system/`, and
  later e.g. `devices/`, `billing/`). Shared, domain-free code lives in `pkg/`.
- A CLI command group lives in `cmd/<name>/`, in one package with sibling files. The parent file
  defines `Command()`.
- `bootstrap/` assembles a process. Only `cmd/*` imports it. A domain package gets its
  dependencies from the fx graph, not from `bootstrap`.
- Do not create `internal/` packages. This application has no external consumers, so `internal/`
  only adds import-path friction.

## Handlers, repos and providers

- A handler is `new<Name>Handler` plus `register<Name>Handler`. Both stay unexported. Both appear
  in the owning package's `module.go`: construction in `fx.Provide`, route binding in `fx.Invoke`.
- The handler **type and its methods** are unexported too. Nothing outside the package names
  them.
- A route is bound on the shared `*http.ServeMux` with a Go 1.22 pattern that names the method:
  `mux.HandleFunc("GET /api/v1/health", h.health)`. Every API route starts with `/api/v1/`.
- A repo is `new<Name>Repo`, constructor-injected through fx.
- **A constructor is exported only when the fx module that provides it lives in another
  package.** Where a package owns its `module.go`, the constructor stays unexported.
- Exporting the **type** is a separate decision, driven by whether another package injects it.
- A new domain package adds its `Module` to `Modules()` in `cmd/serve/serve.go`.
  `TestModulesGraph` in `cmd/serve` fails a graph that does not resolve.

## Dependencies are held, never republished

- A service, repo, handler or middleware holds its dependencies as **named unexported fields**,
  assigned one by one in the constructor.
- **Never embed an `fx.In` params struct.** Embedding republishes every dependency as a public
  field.
- **An fx dependency bundle is `<Name>Deps`, never `<Name>Params`.** `*Params` means domain data a
  caller configured.
- A dependency read only to build a shared value does not belong on a bundle. Build the shared
  value once, in `module.go`, and put *it* on the bundle.

## What is exported

- **A new type is unexported unless a caller outside its package needs to name it.** Before you
  export anything, name the outside caller. If there is none, keep it lowercase.
- Request and response DTOs stay unexported and live beside their handler. Serialize a struct;
  do not hand-build a `map[string]any` that repeats its fields.
- Inside a package, names drop the stuttering context prefix: `devices.Repo`, not
  `devices.DeviceRepo`. Names that the database, the wire or a `CONTEXT.md` pins keep their
  prefix.

## Seams between packages

- **A seam is defined by its consumer.** When two packages must cooperate across a layering, the
  *downstream* one declares an interface for what it needs. The composition root binds the
  implementation. The implementing package imports neither.
- **First check that the two packages are really two.** A seam that mediates a package with
  itself is a boundary to remove, not a seam to keep.
- Do not introduce an interface only to make a trivial unit test possible.

## Persistence

- Repos own the SQL for their tables. Handlers, middleware and services run none. If a handler
  needs a query, add a repo method.
- Repos persist rows and answer nothing more: no orchestration, no event publishing. A write that
  spans two repos belongs to the service one layer up.
- `pgx.ErrNoRows` is the not-found sentinel. A repo decides no policy, so it returns no HTTP
  error.
- A schema change is a new migration in `cmd/lora/migrations`. The root `AGENTS.md` states the
  rules.

## Errors that reach a client

**A rule the user can trip is a typed error raised in the service, not in the handler.** The
handler maps it to a status. A bare error answers **500**, and an expired link or a refused
policy is an ordinary ending, not a server fault.

## Persisted strings are contracts

Task names, lock-key prefixes, JSON `type` tags, column names, enum values and identity hashes
all have live counterparts: stored rows and in-flight jobs.

- A refactor may **move** them. It may not rewrite them. After you move a file, diff its string
  literals against the original.
- When one is renamed on purpose, ship the decode alias that accepts both spellings and the
  backfill migration in one change. The code ships first and the migration second.

## Function flow

When a function has separate phases, put a blank line between the coherent blocks. Keep the
statements of one phase together. Do not add placeholder comments such as "step 1".

## Files

- **A file holds one reason to change.** When two unrelated changes start editing one file, split
  the file.
- If a file holds more than one section of one concept, separate them with a bare `//` followed
  by 78 dashes, to column 80, with a blank line on each side.

## Where the reasoning goes

`AGENTS.md` holds operative rules — see [Maintaining AGENTS.md](maintaining-agents-md.md).

- The package tour, its file map and how the mechanism works go in `README.md` or `doc.go`.
- Why the design is what it is goes in `DESIGN.md`. The `design-doc` skill states its shape.
- A decision contained in one file or one function goes in a `COMPROMISE:` comment on that code.
- The words a context uses go in its `CONTEXT.md`; `CONTEXT-MAP.md` maps the contexts to each
  other.
