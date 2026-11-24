# Territory: config

The application configuration: the sections and keys an operator writes, how they are named and described, and how a config file becomes a validated `Config`. Load `go` beside it for the Go code that reads the options.

Held: the meaning of every option an operator can write today, the environment expansion syntax, and the DSN shape of `pg.dsn`. Fair game: where an option sits, its key name, its Go field path, and the code that loads it — each at the price of a compatibility path, because `appconf.Parse` decodes in strict mode and refuses an unknown key.

## Scope

- **The package** (`pkg/appconf`) — every section.
- **A section or a set of sections** (`pg`, `logging`) — the section types, their files, the shipped files that write them, and every Go reader of their fields.
- **A topic** ("how timeouts are named") — the keys the topic names, across sections.

## Ideal end state

**Sections**

- A top-level section is one subsystem or one external dependency.
- An option lives in the section of the code that reads it.
- Sibling blocks of one kind sit under one parent.
- A repeated shape uses one shared block type.
- Three keys with one shared prefix are a sub-block. A key that copies a third-party setting name keeps that name.

**Keys**

- Lowercase `snake_case`. No section name repeated, no `is_`/`has_` prefix, no Go type name.
- A time span is a `time.Duration`, and its key names its role, never a unit.
- A byte size ends in `_bytes`. A bound starts with `max_` or `min_`.
- A section switch has a false default (`enabled` or `disabled`). A switch inside a section is named for its feature.
- At most four words.

**Descriptions**

- Every option has a doc comment that states what it controls, its unit when the type does not carry one, its default, and the meaning of zero when that is not "use the default".
- Every section type has a comment that states what the section configures.

**Loading**

- Defaults are set in `setDefaults` and checked in `validate`; validation never re-defaults an explicit zero.
- Nothing in the package logs.

## Read

- `pkg/appconf/config.go` and its tests.
- Every Go reader of the options in scope: grep the Go field paths.
- The shipped files: `config/*.default.yml`.

## Current state

- **Section map** — each top-level section and what it holds, marked where it breaks a section rule.
- **Key ledger** — each key in scope that breaks a key rule, with the rule.
- **Description gaps** — each YAML-tagged field without a doc comment.

## Problems

Misplaced option, sibling sections at the top level, a prefix family that is not a sub-block, a unit in a duration key, a Go-style key, an undocumented option, a stated default the code contradicts, an alias that can lose a value.

## Design spec

- **Layout** — the section tree after the change, as YAML with one line for each key.
- **Moves and renames** — one row for each old key: the new key, and how an old file keeps loading (a deprecated field that `setDefaults` copies over, with a warning the caller logs).
- **Go paths** — each changed Go field path, with its reader count.

## Collisions

- **Operators' files** — strict decoding refuses an old key. A rename without a compatibility path stops every deployed process at start.
- **Migrations** — a shipped migration does not change.

## Migration

1. Write the rule change in the root `AGENTS.md` first, when it is a rule.
2. Move or rename the keys with their compatibility path. Change the shipped files and the prose that cites the old paths in the same commit.
3. Each step passes `go vet ./...`.

## Verification

- `go test ./pkg/appconf/`, with a test for each moved key (old key alone, both keys written).
- Load each shipped file: `LORA_CONFIG=config/dev.default.yml go run ./cmd/lora pg ping` fails on a parse error before it dials.
- `task test` when the change moves Go field paths, because the readers span many packages.

## Forks

- Whether to move existing keys now, or to write the rules for new keys only. A move costs one compatibility path for each key.

## Self-check

- Every moved or renamed key has a compatibility path and a test.
- The shipped files parse.
