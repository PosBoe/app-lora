# Territory: ci

## Scope

CI workflows (`.forgejo/`, `.github/`), `Taskfile.yml`, `Taskfile.db.yml`, `docker-compose.yml`, `vue/scripts/`, and `go.mod`.

## Read

- `Taskfile.yml` — what each `task test:*` lane runs. `task test` is the pre-PR suite.
- The Commands section of the root `AGENTS.md`.

## Probes

- `task --list` parses every Taskfile.
- CI runs a full fallback for a change it cannot scope. Read its records; do not rerun the full suite.

## Hunt

- **Skipped is not passed**: a job whose `if:` skips it and whose dependents read the skip as success; a missing artifact read as an empty result.
- **Pins**: a service image in a workflow that differs from `docker-compose.yml`; a tool version (`GOLANGCI_LINT_VERSION`, node, Go) that differs between the Taskfile, CI and `go.mod`.
- **Secrets**: a secret that reaches a step or a container that does not need it; a token printed to a log or written to an artifact.
- **Taskfile environment**: an `env:` block at file level in an included Taskfile. Task merges it into the global environment, where an include var is undefined. Declare it per task, as `Taskfile.db.yml` does.
- **Scope claims**: text that says scoped tests cover everything. Name what the change leaves uncovered.

## Mirrors

- The same step in each job of a workflow, and the same command in the `Taskfile`.
- The image pins listed under **Pins**.
