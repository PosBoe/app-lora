# Territory: sql

## Scope

`*.sql` files; the migrations under `cmd/lora/migrations`; `pkg/pgdb/`; Go models of a table; and Go code that builds or runs a query.

## Read

- The migration rules in the root `AGENTS.md` (Repo Conventions).
- `pkg/pgdb/README.md` — the pool and the maintenance connection.
- `cmd/migrate/migrate.go` — the runner (goose) and the file format.
- `docs/go-package-conventions.md` § Persistence.

## Probes

- List the migrations in order: `ls cmd/lora/migrations`.
- `go test -short ./cmd/migrate/` checks the file names.
- Migrate a scratch database from empty when the environment has PostgreSQL: `task test:setup`. Integration tests need PostgreSQL. When the environment has none, say so in the coverage.

## Hunt

- **Upgrade, not reset**: a fresh `reset` proves nothing about an upgrade from deployed data. Walk each migration against a table that already holds rows: a `NOT NULL` column with no default, a unique index over duplicate rows, a type change that truncates, a backfill that does not scale, a lock held on a large table.
- **Version format**: a version that is not 14 digits `YYYYMMDDhhmmss` in UTC, a placeholder `120000` time, or a version that another migration already claims.
- **Edited history**: a change to a migration that deployed systems may have applied. It may only make the original work more robust; new work goes in a new migration.
- **Down migrations**: a down that does not undo its up, or that drops data the up did not create.
- **goose annotations**: a file without `-- +goose Up`, or a statement with a `;` inside a function body that lacks `-- +goose StatementBegin`/`StatementEnd`.
- **Queries**: tenant filters missing from a query (see [Go](go.md)); a query built by string concatenation from user input instead of `$1` parameters; a `pgx.Tx` passed to a goroutine; `rows` not closed or `rows.Err()` not checked.

## Repo contracts

- The migration version is the persisted identity and is claimed once. The `_name` slug is not persisted.
- Tests create their own rows and scope every read and delete to them.
- Say *migration version*, not *id*.

## Mirrors

- The Go model of the table, and every query that selects its columns.
- Test setup code that inserts into the table.
