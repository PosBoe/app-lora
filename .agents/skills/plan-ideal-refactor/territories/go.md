# Territory: go

Go source: a package, a set of files, or a topic that cuts across packages ("error handling", "how spans are buffered"). Held: wire formats, config semantics, telemetry output, and anything the frontend or an outside observer can see. Fair game: package layout, exported API, internal structure.

## Scope

- **A package** (`system`) — every file in it.
- **A set of files** (`pkg/otlp/span.go pkg/otlp/logs.go`) — those files plus enough of their packages to understand them in place. The ideal design may conclude the files belong elsewhere.
- **A topic** — the files and packages the core's topic mapping finds; for a topic, the plan says which package each piece lands in and why that package owns it.

Exclude vendored trees such as `pkg/tempo/` from every grep.

## Ideal end state

- Interfaces are defined by the consumer and stay small. A one-implementation interface with no consumer-side seam is a problem, not a pattern.
- No stutter: the package name is part of every exported name (`devices.Repo`, not `devices.DeviceRepo`).
- Accept interfaces, return concrete types.
- Exported means "someone outside uses it"; unexport the rest. Do not export code just so tests can reach it.
- Each file and type has one reason to change; high-level entry points read before their helpers.
- Prefer plain functions over single-method objects, and data over indirection.
- Composition over inheritance. Embedding that exists to "inherit" methods and be overridden is a smell; embed only to compose, and prefer an explicit field when the inner type is an implementation detail.
- House idiom: fx modules, `featflag` gating, `errors.AsType`, `log/slog`, the rest of the root `AGENTS.md`, and [Go package conventions](../../../../docs/go-package-conventions.md).

**Package budget.** Creating packages is the move ideal-first design most overuses — in the reference repository, one refactor season took a context past twenty subpackages, and a later pass had to merge the vocabulary leaves back into one package. A package is the unit of *dependency*, not of *topic*: one responsibility per file buys the same readability at zero cost, so "this is conceptually its own thing" earns a file, never a package. Every package in the ideal layout — net-new or kept — is forced by one of exactly three needs, named next to it:

- **An import cycle** that files in one package cannot express (the `*model` leaf pattern). Name the two import paths that would collide. Never extract a leaf speculatively "so a cycle can never form".
- **A consumer that needs part without the rest** — a *named, existing* importer that must link one piece without the rest's dependency tree (a heavy driver, a test-only kit kept out of production binaries). A consumer that could exist someday forces nothing.
- **A deployment or lifecycle boundary** — build-tag divergence, or an fx module a binary assembles differently from its siblings.

Each package costs an import path and name every reader learns, an fx wiring decision, a doc surface, and forced exports — moving a helper across a package boundary makes it public API. State the package count before → after; same-or-fewer is the default, and a package under about 1,000 non-test lines gets its forcing need spelled out. Merging sibling packages is a first-class move. A new type lives in the package that owns the rule it carries, when every consumer already links that package; `go list -deps ./<importer>` (the `./` matters) shows what a new import edge adds. When the layout adds no package and keeps no subpackage, the Layout section says "packages: N → N, none added" and the budget needs nothing more.

## Read

- `go doc ./<pkg-path>` per package — the exported surface in one shot, the checklist for the public surface in step 2.
- Build-tag variants together (`//go:build invariants`, `!invariants`); note where they diverge.
- Consumers: `grep -rl "github.com/waneock/app-lora/<pkg-path>"` per package, or the exported symbols for a file or topic scope.
- `module.go` and how fx provides and consumes the scope, including config-dependent assembly in a `bootstrap.Assembly` that a command's `bootstrap.Plan` names.
- Performance evidence: `*_bench_test.go` and `Benchmark*` functions in scope, and whether the scope sits on a hot path (ingest decode, a per-row or per-attribute loop, a query kernel).

## Problems

Coupling, leaky abstraction, duplicated logic, unclear ownership, god type, primitive obsession, inverted dependency (domain importing transport or storage), exported-but-unused symbol, logic every caller writes around the API.

## Evidence lab

- **Inlining cost** — `go build -gcflags=-m=2 ./<pkg>` for each hot function the plan splits or moves. Splitting a small function can push it past the inlining budget; one past run lost 29% on a per-attribute lookup this way. The number is cheap; read it while planning.
- **Benchmarks** — for a claim that a shape is faster, or to refute a "duplicate" that exists for speed: the package's benchmarks under `benchstat`, `-count=10`, base and candidate interleaved.

## Design spec

- **API** — Go signatures (types, fields, interfaces, functions) as `go doc` would print them, with a doc-comment line each.
- **Layout** — packages and files, with the package count before → after and each package's forcing need.
- **Lifecycle** — who owns goroutines, what shuts them down, which errors are sentinels and which are wrapped, what is safe to call concurrently.

## Collisions

- **fx wiring** — providers, invokes, config-dependent module assembly.
- **DB migrations** — the pg and ch schema the scope reads or writes. When the schema itself must change, load the `pg` or `ch` territory.
- **Hot paths and benchmarks** — which benchmark guards each changed function, and what the change does to it. Keep benchmark names stable, since a rename breaks `benchstat` comparison with earlier runs; rename one only when the plan says why.
- **`app-billing`** — each exported signature change (core step 5, *Other repositories*).

## Migration

- Each step compiles under both builds. The binaries do not compile test files, so a step that changes a signature carries the edits to every test caller and helper; `go vet` catches one left behind.
- Prefer a bridge inside one package over a temporary export.
- **Test visibility** for new tests: external package tests (`package foo_test`) for exported contracts; internal tests only for private invariants, unexported pure logic, build-tag variants, or setup that would make a contract test unreadable. When the plan unexports a symbol, its direct tests move into the package under their current names (a `DESIGN.md` may cite them as evidence). When the plan merges a function into another, its tests follow the surviving function and take its name, and every evidence line that cites the old test name is a document edit.
- **Benchmark check** flag on a step that touches hot code: the benchmarks to run on base and step under `benchstat` (`-count=10`, interleaved), and the threshold that sends the step back to the user. A restructure should be neutral, so the default threshold is a significant `sec/op` slowdown above 2%, or any new allocation on a path that allocated nothing. A threshold the package's `DESIGN.md` states takes precedence.

## Verification

- Per step: `go vet ./...` and `go vet -tags=invariants ./...`; the step's benchmark check.
- Final: `task test:pkg -- <refactored packages and their users>`; `task test` when the plan touched `go.mod`, a Taskfile, a CI workflow, or more than a couple of bounded contexts. See [Go test conventions](../../../../docs/go-test-conventions.md).

## Forks

The core's list is complete for Go; there are no Go-only forks.

## Plan sections

- `## 2. Ideal design` → `### Layout` states the package count before → after with each forcing need; `### Surface` is titled `### API`, go-doc style; add `### Lifecycle` after `### Data flow`.
- `## 3. Reconciliation` → add `### Performance`: the benchmarks that guard hot code and the expected effect, or "none in scope".
- Migration step line adds `Bench: <benchmarks|none>`.

## Self-check

- The Layout section states the package count before → after, and every net-new or kept subpackage names its forcing need.
- Hot code has a named benchmark check, or the plan states the scope has none.
- Every exported signature change has an `app-billing` follow-up, or the reason none applies.

## Constraints

- No new third-party dependencies; `go.mod` as it stands is the toolbox.
