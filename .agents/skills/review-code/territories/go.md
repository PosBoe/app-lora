# Territory: go

## Scope

`*.go` files, `go.mod` and `go.sum`.

## Read

- The root `AGENTS.md` (Core Rules and Repo Conventions), which states most of the repo contracts below.
- `docs/go-package-conventions.md` and `docs/go-test-conventions.md`.
- The `README.md` of each package in scope, and `pkg/pgdb/README.md` when code runs a query.
- **Surface**: `go doc -all ./<pkg-path>` is the checklist. Read each exported doc comment against the code it describes.
- **Consumers**: `grep -rl "github.com/waneock/app-lora/<pkg-path>" --include=*.go .` lists the importers.
- **Variants**: read build-tag variants together (`invariants`, `!invariants`, `vue`, `!vue`). Behavior that silently diverges between the builds is a finding.

## Probes

```
go vet ./<pkg-path>/...
timeout 600 go test -short -count=1 ./<pkg-path>/...
timeout 600 go test -race -short -count=1 ./<pkg-path>/...
```

- The race run is a rest-mode probe. Skip it when the invoking prompt forbids race suites.
- A package whose tests need PostgreSQL fails or skips without it. When the environment has no database, say so in the coverage.
- `-short` does not keep a test off a database. Before you run the tests of a package in an environment that the prompt says has no database, grep its `_test.go` files for a DSN, `LORA_CONFIG`, or `localhost`/`127.0.0.1`. For those packages, compile the tests only (`go test -short -run '^$'`), and run by name only the tests that you confirmed are pure.
- A scoped green run is not a full-suite pass. `task test` is the full suite.
- Compile-check both builds when a change touches tagged code: `go vet ./...` and `go vet -tags=invariants ./...`.

## Hunt

- **Resources on error paths**: when a constructor's call site does not show whether it returns a live resource together with an error, read the dependency's source under the module cache (`go env GOMODCACHE`).
- **Transactions**: a `pgx.Tx` passed to a goroutine, or used after its `Commit`/`Rollback`. A `pgx.Tx` holds one connection and is not concurrency-safe.
- **Tenant scoping**: a query that filters by one id and acts on a row selected by another, or a list query with no tenant filter at all.
- **Goroutines**: every `go` statement needs a proven stop path, and every channel an owner that closes it. Check `goleak` coverage in the package's `TestMain` against what it starts.
- **fx wiring**: a new `Module` that `cmd/serve/serve.go` does not list, or a provider that two modules both supply. `TestModulesGraph` catches the second; nothing catches the first.

## Repo contracts

A breach of any of these is always reported:

- A `(T, error)` method that can return `(nil, nil)`, or a nil map or slice where consumers expect "not found" as an error.
- `errors.As` instead of `errors.AsType`; a `yaml.v*` package instead of `github.com/goccy/go-yaml`; the default `slog` logger instead of the `*slog.Logger` from the graph.
- A panic for a broken invariant that `featflag.Invariants` does not gate, or a gated panic where a caller can reach the state with valid input.
- An exported constructor whose fx module lives in the same package.
- Enum string values and JSON keys that are not `snake_case`.
- A new `internal/` package.

## Mirrors

- Sibling kinds and files of the same package that restate a rule.
- Parallel `testdata/` corpora.
- The `vue/src/` form, rule or type that restates a Go validation rule, limit, or response shape. Grep for the constant, the JSON key, and the error text.
