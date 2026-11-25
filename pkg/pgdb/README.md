# pgdb

Opens the PostgreSQL connection pool (`pgxpool.Pool`) from `pg.dsn` and
provides it to the fx graph.

- `Module` provides `*pgxpool.Pool` and closes it on stop. The pool dials on
  first use, so `serve` starts when the database is down; `/api/v1/health`
  reports the state.
- `Open` creates a pool outside fx. A CLI command that boots no application
  uses it.
- `Maintenance` returns a connection config for the `postgres` database of the
  same server. `cmd/pg` uses it to create and drop the configured database.
- `Querier` is the part of `*pgxpool.Pool` and `pgx.Tx` that a repo uses. A
  repo that takes a `Querier` runs in a transaction or outside one.
