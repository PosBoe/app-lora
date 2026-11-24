# Territory: config

## Scope

The application configuration: `pkg/appconf/` and the shipped config files under `config/` (`*.default.yml`). Load `go` beside it for the Go code.

## Read

- `pkg/appconf/config.go` — the structure, `setDefaults`, and `validate`.
- The config rule in the root `AGENTS.md` (Repo Conventions).
- **Consumers**: grep the Go field path of each changed option (`conf.PG.DSN`, `conf.Listen.HTTP`) across the repository, and grep its YAML path in `*.md`, comments, and `config/*.default.yml`.

## Probes

```
go test -count=1 ./pkg/appconf/
```

Also load each shipped file through the binary: `LORA_CONFIG=config/dev.default.yml go run ./cmd/lora pg ping` fails on a parse error before it dials.

## Hunt

**Sections**

- A top-level section that is not one subsystem or one external dependency.
- An option in a section other than the one whose code reads it.
- A third key with one shared prefix in a section, not moved into a sub-block.

**Keys**

- A key that is not lowercase `snake_case`, repeats its section name, has an `is_`/`has_` prefix, or holds a Go type name.
- A time span that is not a `time.Duration`, or a duration key that names a unit (`_hours`, `_ms`) instead of a role (`_timeout`, `_interval`, `_ttl`).
- A byte size without `_bytes`. A bound without `max_`/`min_`.
- A section switch whose default is true.

**Descriptions**

- A YAML-tagged field without a doc comment, or a doc comment that does not state the default, the unit a type does not carry, or the meaning of zero.
- A stated default that differs from `setDefaults`. Check each stated default against the code.

**Loading**

- A renamed or moved key with no compatibility path. `Parse` decodes in strict mode, so an old key in an operator's file stops the process.
- A default set in `validate` instead of `setDefaults`, or an explicit zero that validation re-defaults instead of refusing.
- An error without the full YAML path of the option.
- Logging inside `pkg/appconf`. A problem is an error.
- A secret written into a `*.default.yml` instead of an `${ENV}` reference.

## Repo contracts

A breach of any of these is always reported:

- The environment expansion syntax (`$VAR`, `${VAR}`) does not change.
- The shipped config files parse.

## Mirrors

- The shipped files: `config/dev.default.yml`, `config/test.default.yml`.
- `docker-compose.yml`, whose credentials the DSNs in those files name.
- Prose and comments that cite the YAML path of a changed option.
