# Backend Development

Rules here apply to the Go backend. A directory with its own `AGENTS.md` adds rules for that
subtree; `vue/AGENTS.md` holds the frontend rules.

A `README.md` holds a directory map and file tour. A `DESIGN.md` records a scope's current
purpose, goals, frozen contracts, and design rationale; use the
[`design-doc` skill](.agents/skills/design-doc/SKILL.md) for its format. A `CONTEXT.md` defines
domain terms. `AGENTS.md` holds only operative rules — [Maintaining AGENTS.md](docs/maintaining-agents-md.md)
states what belongs in one and what does not. `PRODUCT.md` states who the users are and what the
product is for.

Two shared guides hold the rules that would otherwise be restated in every subtree. A subtree
`AGENTS.md` states only what **differs** from them, and names the reason:

- [Go package conventions](docs/go-package-conventions.md) — handler/repo naming, what is
  exported, fx dependency bundles, seams, persistence, persisted strings.
- [Go test conventions](docs/go-test-conventions.md) — lanes, layout, goleak, the test database,
  determinism.

## Core Rules

- Recommend the best long-term solution within the task's scope. Do not reject it only because
  it requires more work.
- `featflag.Invariants` is a compile-time constant enabled by the `invariants` build tag (on in
  `task dev`, off in `task build`). Use it to gate panics for broken invariants; return a
  graceful error otherwise.
- Go 1.26+. Use `errors.AsType`, not `errors.As`.
- Use `github.com/goccy/go-yaml`, not `gopkg.in/yaml.v2/v3`.
- Use `github.com/jackc/pgx/v5` for PostgreSQL. A long-lived process gets `*pgxpool.Pool` from
  `pkg/pgdb`; a CLI command that boots no application opens its own connection.
- Use `log/slog` for logging. Take the `*slog.Logger` from the fx graph; do not use the default
  logger.
- A `(T, error)` method must never return `(nil, nil)`: signal "not found" with a sentinel error
  (e.g. `pgx.ErrNoRows`) the caller can match with `errors.Is`. Reserve `(nil, nil)` only for
  `*T` results where absence is genuinely not an error and is documented as such.
- Enum string values, JSON keys, and span or metric names that our code chooses use `snake_case`.
- Prefer early returns. Keep code ordered top-down: high-level entry points before
  helpers/callees.
- Avoid unnecessary copying/allocations; document mutation when a function mutates inputs.
- Do not introduce an interface solely to make a trivial unit test possible; prefer testing
  observable behavior or using the concrete dependency unless the abstraction has a production
  need.

## Repo Conventions

- Write documentation and code comments in **ASD-STE100 Simplified Technical English**. Keep
  sentences short — a maximum of 20 words for an instruction, 25 for a description — and write
  one instruction per sentence. Use the active voice, the imperative mood for instructions, and
  only the simple tenses. Use one word for one meaning, and prefer short common words (`use`, not
  `utilize`; `start`, not `initiate`). Put a condition before its instruction. Do not use vague
  words such as "leverage", "robust", or "seamless". Technical names — identifiers, file names,
  flags — keep their exact spelling from the code. This governs `AGENTS.md`, `README.md`,
  `DESIGN.md`, `CONTEXT.md`, `doc.go`, comments, and commit messages; it does not govern
  user-facing UI copy.
- **A document cites a file this repository holds.** `docs_policy_test.go` in the repository root
  reads every `.md` for backticked repository paths and fails a path the tree no longer holds.
  Prose that names a path on purpose that does not exist goes in its allowance list with the
  reason.
- **Prose states an invariant, not a count of what the tree holds.** Do not write a number a
  reader must recount to trust ("and its 21 siblings"). Write the rule the number stands for, or
  name the exceptions.
- A package with a non-obvious mechanism has a `README.md`; read it before you change an
  unfamiliar package.
- The skills live in `.agents/skills/`; `.claude/skills` is a symlink to it. `.agents/skills/README.md`
  states the naming rules for a new skill.
- `CONTEXT-MAP.md` maps the repo's bounded contexts and their relationships; a modeled context
  has a `CONTEXT.md` domain glossary. **A new `CONTEXT.md` is added to the map in the same
  change.**
- Dependency injection uses `go.uber.org/fx`. A package exposes its providers in `module.go`, and
  `cmd/serve/serve.go` lists the modules of the serve process.
- CLI command groups under `cmd/` stay in one package with sibling files; the parent file
  defines `Command()`.
- Don't create new `internal/` packages — this application has no external consumers, so
  `internal/` only adds import-path friction.
- A migration in `cmd/lora/migrations` is named `<version>_<name>.sql`, where the **version** is
  the 14 digits `YYYYMMDDhhmmss` of the actual creation time **in UTC** — `task dev:migrate:create -- <name>`
  writes it. A local-time version sorts out of order against a UTC one. The version is the migration's persisted identity and may be claimed exactly once.
- Do not expand an existing migration's scope, because a deployed database might have applied it.
  Create a new migration for additional work. Modify an existing migration only to make its
  original work more robust for corner cases.
- `dev` is the integration branch once it exists: branch off it and do the work there
  (`git switch -c <type>/<topic> dev`), then open a PR. Never commit or push directly to `dev` or
  `main`.
- Use Conventional Commits for commit messages, for example `fix: avoid duplicate notifications`.
- Config files are `config/<env>.yml`, copied from `config/<env>.default.yml` by `task init` and
  ignored by git. A new config field gets a default in `pkg/appconf` and a line in every
  `.default.yml` that needs a non-default value.

## Debrief

After a non-trivial task, end the final response with a short factual report of anything you
know from doing the work that the summary alone wouldn't show — events from this session, not
opinions about the code:

- **Deviations**: where the work diverged from the stated or approved plan, and why.
- **Found, not fixed**: pre-existing bugs, smells, or risks you noticed and deliberately left
  alone (including workarounds you applied and tests you skipped).
- **Follow-ups**: actionable work outside the task's scope that this session surfaced.

Report only what actually happened — never invent an item to fill a category. Omit empty
categories; if all three are empty, say "Debrief: nothing to report" in one line so it's clear the
check ran. Skip entirely for trivial edits (typos, renames, one-line fixes).

## Commands

- Build: `go build ./...`. Build the release binary with the frontend embedded: `task build`.
- Compile-check without a binary: `go vet ./...`, which also compiles the test files.
- **While the task runs, compile-check only.** Do not run a test lane after each commit.
- **When the task's code is complete, run the packages the change can affect one time**:
  `task test:pkg -- ./system/... ./cmd/serve/...`. Name the packages you changed and the packages
  that use them.
- **Before you open a PR, run `task test`** and `task lint`. Run them also when you cannot bound
  what the change affects — a change to `go.mod`, a `Taskfile`, or a CI workflow reaches
  everything.
- Gate stress, soak, and large-sweep tests with `testing.Short()`. Every lane except `test:full`
  passes `-short`.
- Run the app locally with the `run-app` skill: `task up`, `task dev:migrate`, `task dev`,
  `task vue`.
