# Territory: pg

The PostgreSQL schema of a set of tables: normalization, keys and constraints, enum types, indexes, jsonb boundaries, and the relationships between tables. Held: API responses and what the frontend sees; column renames stay invisible at the wire unless the user asked otherwise. Fair game: table layout, constraints, and the shape of the SQL the app emits.

This territory is deliberately ambitious. The current tables are evidence about the domain, never the starting sketch of the answer. A plan whose "ideal" is the existing DDL plus two constraints is a failed run, unless the existing design survives the goals below in open competition against real alternatives. PostgreSQL gives transactional DDL and in-place `ALTER`, so the migration mechanics are rarely the hard part; the hard part is data — rows that violate the constraint you want, denormalized copies that must be merged, jsonb blobs whose keys become columns.

## Scope

Tables named in the argument (`pg: incidents incident_events incident_alerts`), each resolved to its full footprint before anything else is read:

- The `CREATE TABLE` and every later `ALTER` in the migrations under `cmd/lora/migrations`.
- The relational neighborhood: every table that references a scoped table by foreign key, and every table a scoped table references. A scope that cuts a foreign key in half widens to both ends, at least read-only.
- The enum types the tables use, and other tables that share them (the sharing is a design fact).
- The Go model structs (`pg.BaseModel` embeds, `pg:` tags, `inherit` compositions) and the repos that own them.

A topic ("how monitors persist state") maps to tables by grepping table names in migrations and model tags.

## Ideal end state

Six goals, in priority order. Every candidate is judged against them, and every element of the final schema serves at least one. When goals conflict, the higher-numbered goal yields.

1. **Invalid states are unrepresentable.** Every invariant the application relies on is a constraint the database enforces — and "constraint" means the whole arsenal: a CHECK that makes a status/timestamp pair coherent, a partial unique index for "at most one *open* incident per alert", an exclusion constraint for "no overlapping maintenance windows", a range type replacing a from/to pair so emptiness and overlap become algebra, a deferrable constraint for an invariant that holds only at commit. A rule enforced by SELECT-then-INSERT in Go is a race; the constraint is the fix and the Go code becomes the error handler. The test: for each way the domain can be wrong, name the constraint that rejects it — or admit the schema permits it. A race is fixed by the shape, not by a lock: prefer a schema where the contested write is one statement on one row, or where a constraint rejects the losing write. An advisory lock or `SELECT … FOR UPDATE` taken in Go is the last resort, and the verdict counts it as an invariant still held in Go.
2. **The schema teaches the domain.** The DDL alone — table, column, and enum names, foreign keys, CHECK clauses — teaches what the entities are, how they relate, what states they pass through, and which vocabularies are closed. One table, one entity, one lifecycle: a god table where half the columns apply to one kind of row splits along the entity boundary. A NULL means "optional", never a secret third state. A jsonb column means "keys the schema genuinely cannot know", never "we didn't decide".
3. **The database does the work it is best at.** Drift exists only where two copies of a truth live apart, so let Postgres own derived truths: a generated column, not app code that must remember to update it; a `DEFAULT` expression; `ON DELETE` actions, not cleanup jobs. A counter or latest-state cache the queries need is kept only where the plan names the mechanism that keeps it consistent — and a design that removes the copy beats one that synchronizes it.
4. **Hot queries read like their questions.** The query inventory is a design input: shape tables so the dominant queries become simple (a junction table instead of jsonb containment; a column that turns three joins into one). Then give every hot query its physical shape: indexes led by the tenant column (`project_id`/`org_id` — every query here filters on it), partial indexes for hot subsets, expression indexes for computed filters, covering (`INCLUDE`) indexes where a scan-free read is worth the write cost, GIN only where the inventory shows containment or search. Every index cites the inventory entries it serves; an index citing none is deleted — writes pay for it on every statement.
5. **Change is priced in.** Closed, code-owned vocabularies are native enums (snake_case, matching the Go enum); user-extensible vocabularies are lookup tables with FKs. Surrogate `bigint` identity keys with the natural identity as a unique constraint beside them keep foreign keys stable while the natural key evolves. No speculative generality.
6. **Everything earns its place.** Every table, column, constraint, and index cites an invariant, a domain fact, a derived truth, or a query. Fewer tables, fewer indexes, a jsonb column deleted because its keys became the schema — these are wins.

## Palette

Vanilla columns, btree indexes, and FKs are the floor, not the ceiling. Consider whether the schema gets simpler or stronger with: range types (`tstzrange`, `int8range`) with GiST exclusion constraints; partial and expression indexes; covering indexes; generated columns; domains for value rules reused across columns; composite types for a value cluster that travels together; arrays of honest scalars with no identity; deferrable constraints; trigger-maintained invariants where nothing weaker suffices (with the trigger's contract spelled out, since triggers hide logic); `LISTEN`/`NOTIFY` for change propagation the app now polls for; text-search or trigram indexes where the inventory shows search; partitioning only at row counts where pruning demonstrably pays.

A feature is chosen because it serves a goal, never for novelty — "the app serializes these writes in Go to prevent overlap, and the constraint makes that race impossible" is a reason. Any feature this repo does not already use is proved in the lab before the plan leans on it.

**Not on the palette by default: the composite foreign key that carries a tenant column.** Keying a child to `(id, project_id)` on each parent looks like free structural rigour. It is not: PostgreSQL requires a foreign key's referenced columns to match a unique constraint or index **exactly**, and answers `there is no unique constraint matching given keys for referenced table` otherwise. Each parent pays a redundant unique index whose only reader is the referential-integrity check. Measured on a 2M-row alert table: 60 MB, **larger than the 43 MB primary key it duplicates**, plus an index write on every insert. If the tenant invariant is worth raising, state that price first, at this table's real row count, and let the user decide. Weigh the cheaper alternatives in the same breath: a trigger comparing the two parents (two indexed lookups per insert, no new index), or the query predicates that already uphold it. A declined invariant is recorded as a numbered problem and stays in Go — a legitimate outcome.

## Read

1. `pkg/pgdb/README.md` and the migration rules in the root `AGENTS.md`, plus the owning package's `README.md` and `CONTEXT.md`; `CONTEXT-MAP.md` says which bounded context owns each table.
2. The full DDL history in `cmd/lora/migrations`: the original `CREATE TABLE`, every `ALTER`, index, and enum type. The accreted history often *is* the diagnosis.
3. The Go models, tag by tag (`pk`, `nullzero,notnull,default:…`, `alias`, `inherit`), including test-local variants. A tag contradicting the DDL is a finding; so is a column absent from every model.
4. The write path: every `INSERT`/`UPDATE`/`DELETE` and the `pg.RunInTx` boundaries around them.
5. The read path: every query that touches the tables, found by grepping table names and aliases in `Model(...)`, `TableExpr(...)`, and raw SQL. This inventory must be complete, not sampled — indexes are designed from it.
6. Application-enforced invariants: SELECT-before-INSERT uniqueness checks, orphan cleanup jobs, validation paths, writes serialized in Go.
7. `LISTEN`/`NOTIFY` channels, triggers, and sequences the tables take part in.

## Current state

- **DDL facts** per table: columns with types, defaults, nullability; keys; FKs with `ON DELETE`; unique constraints; CHECKs; indexes; enums.
- **Relationships** — which tables reference which, including relationships that exist in the data but not as a foreign key, and "relationships" held as jsonb arrays of ids.
- **Query inventory** — each distinct shape with `file:line`, the columns it filters, joins, and orders on, and the index that serves it, if any.
- **Invariants** — each rule the domain relies on and where it lives: a constraint, Go only, or nowhere.

## Problems

Each names the goal it violates. Missing key, unenforced invariant, race behind an app-side check, denormalized copy that can drift, jsonb hiding structure, nullable-that-means-something, enum-as-text, unindexed hot filter, unused index, EAV, god table, schema shape forcing query contortions. The consequence is a drift or race that can happen, a scan the planner is forced into, or a query three joins more complex than its question.

## Candidates

Three candidates, always: **the repair** (constraints added, indexes corrected, jsonb keys promoted in place), **the redesign** (entity boundaries redrawn), and **the radical**. Radical moves here: state-snapshot rows become an event/transition table, or the reverse; a from/to pair becomes a range with an exclusion constraint; a polymorphic table becomes per-kind tables, or the reverse; a synchronized copy is deleted and its readers repointed; a jsonb payload becomes a child schema; an app-side scheduler queue becomes constraint-guarded rows. Dirty data that makes a constraint hard to add is reconciliation work, not a design argument. Each candidate carries a DDL sketch, and the verdict lists what each one still leaves to Go code — every invariant, race, and order rule the database does not hold. The repair wins only when it leaves no more there than the redesign does, or when the difference is priced and small.

## Evidence lab

The local Postgres from `docker-compose.yml` (`task up`). Find the running container with `docker ps` (`app-lora-postgres-1` by default); `docker exec -i <container> psql -U lora -d postgres` is the reliable entry. Constraints and normalization are correctness arguments and need no experiment. Run one when the plan rests on:

- **Planner behavior** — whether the planner picks the proposed partial, expression, covering, GIN, or GiST index for the inventory's query shape. Build the candidate in a scratch database (`CREATE DATABASE scratch`), fill it with `generate_series` data at realistic row counts and cardinalities, `ANALYZE`, and read `EXPLAIN (ANALYZE, BUFFERS)`. At 1k rows the planner rightly seq-scans everything.
- **Unfamiliar features** — exclusion constraints, range types, generated columns, a jsonb operator: confirm the behavior and the error shape on the server version `docker-compose.yml` pins.
- **Races** — a suspected race in a multi-statement write (step 2 *Concurrency*): run the two writers in two sessions with interleaved `BEGIN`/statements, and show the end state neither caller asked for. Then show the candidate schema refuses it or makes it impossible.
- **Migration-step cost** — lock duration and wall time of a rewriting `ALTER` or a backfill at production-like row counts, when a table is large enough that "cheap" or "lock-heavy" needs proving.

Laptop timings are optimistic lower bounds for a production server. Never touch `lora_dev` or `lora_test` in place.

## Design spec

- **Complete DDL** of every table, type, and index — `CREATE TYPE`, `CREATE TABLE` with every constraint inline, `CREATE INDEX` — in the repo's migration dialect. Keys stay boring: surrogate `bigint` identity primary keys with natural identity as a unique constraint beside them; composite primary keys only for pure junction tables; timestamps as `unixtime.Nano` with `default:now()`.
- **Models** — the Go struct signatures with `pg:` tags.
- **Scenario walk** — the dominant writes with their transaction boundary and the constraint that catches each race; the hot reads with the index each uses.
- Every current column mapped to its new home or to deletion; every app-enforced invariant mapped to the constraint that now owns it, or stated to stay in Go with the reason.

## Collisions

- **Models and repos** — every struct, tag, and repo method. Constraint errors become part of the contract: code that pre-checked uniqueness now handles the violation error, matched through the repo's error conventions, not string comparison.
- **Queries** — every site in the inventory: what its SQL becomes, per `file:line`.
- **Transaction boundaries** — `RunInTx` blocks whose shape changes, and new multi-table writes that need one.
- **Existing data** — the heart of it. For every new constraint: do current rows satisfy it? For every split or merge: the exact backfill (`INSERT … SELECT`, `UPDATE … FROM`). Dirty data follows the user's decision. Large backfills that need batching or app logic go in Go migrations (`MustRegister`, or the fx-invoked pattern with `MigrationParams`).
- **Both migration histories** — what runs in each deployment, since their DDL may differ.
- **`LISTEN`/`NOTIFY`, triggers, seeds** — payloads that mention changed columns; test setup code that creates rows in the old shape.
- **Tests** — datadriven corpora asserting responses or SQL; integration tests on the live test databases.

A compromise that exists only to avoid writing a backfill is not legitimate — backfills are exactly the work this plan specifies.

## Migration

- **One migration per logical change.** A migration is one goose file, `<version>_<name>.sql`, with an `-- +goose Up` and an `-- +goose Down` section. goose runs each file in one transaction; a statement that cannot run in a transaction (`CREATE INDEX CONCURRENTLY`) needs `-- +goose NO TRANSACTION` and a file of its own.
- **Repair, then constrain.** When data may be dirty, the repair or backfill is its own migration, and the constraint follows it. The plan states per constraint why current rows satisfy it, or which earlier step made them.
- **Locks are the cost model.** `ADD COLUMN` (with or without a default) is cheap; table-rewriting type changes and long backfills inside one transaction hold locks for their duration. Flag each step whose lock time scales with table size, and prefer a batched backfill in its own migration for large tables.
- **Enum evolution** — `ALTER TYPE … ADD VALUE` for additions; removal or rename is a real migration (new type, column cast, drop old).
- **Order** — additive schema, dual-write or backfill, reader cutover, constraint tightening, destructive drops last.
- **Write the migration file names into the plan, and check each name** before presenting. A file is `<version>_<name>.sql`, where the version is the 14 digits `YYYYMMDDhhmmss` of the actual current time in UTC; `task dev:migrate:create -- <name>` writes it. `TestMigrationNames` in `cmd/migrate` refuses any other name.
- Step kinds are `[schema-only|behavior]`; each step names `[migrations: <files>]` and `[code: <files>]`.

## Verification

- Per step: both builds compile (the `go` territory's per-step commands).
- Final: `task up`, then `task test:setup` so the new migrations run against fresh databases, then `task test`. A schema change is not verified by a compile.

## Forks

- **Domain identity** — whether two similar tables or columns are one concept; what a row *is* (one per user? per user per project?); whether a 1:1 relationship is essential or incidental. Keys hang on these answers.
- **Dirty data** — when rows violate a wanted constraint (orphans, duplicates, NULLs): repair, quarantine, or drop is a business decision.
- **History** — whether an append-only event table's past rows may be reshaped, or are an immutable record.
- **Vocabulary ownership** — closed and code-owned (native enum) or user-extensible (lookup table), when the code doesn't settle it.
- **The tenant composite FK**, when raised — with its price stated first.

## Plan sections

Title the plan `# PostgreSQL schema plan: <tables>`.

- `## 1. Current state` → replace responsibilities and surface with `### DDL facts`, `### Relationships`, `### Query inventory`, `### Invariants`.
- `## 2. Ideal design` → `### Candidates` gives the repair, the redesign, and the radical (with the assumption it challenged), each with a DDL sketch and goals won and lost, then `### Verdict`. `### Layout` and `### Surface` become `### DDL` and `### Models`; `### Data flow` becomes `### Scenario walk`.
- `## 3. Reconciliation` → add `### Data`: per new constraint, why rows satisfy it or the backfill that makes them; per reshape, the move SQL.

## Self-check

- The scope names every table, enum type, and FK neighbor.
- The query inventory is complete: every reader and writer with `file:line`.
- For each goal-1 invariant, the strongest constraint kind that can express it was considered, not only NOT NULL and UNIQUE.
- Every constraint maps to an invariant, every index to a query, every jsonb column to a genuinely open vocabulary.
- Every app-enforced invariant is now a constraint or explicitly left in Go with a reason.
- No tenant composite FK was proposed as free.
- Every backfill has its SQL or Go-migration shape; steps follow repair-then-constrain and flag lock-heavy operations.
- Every multi-statement write in scope was checked for concurrent callers, and a suspected race was reproduced or explicitly left untested.

## Constraints

- No new third-party dependencies; `pgx` and goose are the toolbox.

## Closing

Report the migration versions added per deployment.
