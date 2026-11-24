# Go test conventions

The rules every Go test package in this repo follows. A package's own `AGENTS.md` states only
what **differs** from this file.

The root `AGENTS.md` says **when** to verify. This file holds what a test package must look like.
[Writing good tests in the AI era](writing-good-tests-ai-era.md) holds what a test asserts on.

## Lanes

- `task test` runs `go vet`, every Go test with `-short`, and the frontend unit tests. Run it
  before you open a PR.
- `task test:pkg -- ./system/...` runs the named packages only.
- `task test:full` runs without `-short`. Gate stress, soak and large-sweep tests with
  `testing.Short()`.
- `task test:race` runs the race detector. Run it when you change concurrent code.
- `go build ./...` does **not** compile test files. Run `go vet ./...` after you move a symbol
  between packages.

## Layout

- The tests for `foo.go` go in `foo_test.go`.
- Prefer an external test package (`package foo_test`) for the exported API. Use an internal test
  package (`package foo`) only for an unexported handler or helper that has no exported surface.
- Prefer a datadriven test (`pkg/datadriven`) when the output is text or renders as text. Use
  a table-driven test with `t.Run` subtests when the expected output is a small Go value.
- Use `github.com/stretchr/testify`: `require` for a precondition that ends the test, `assert`
  for the checks after it.
- **A subtest reports only on its own `t`.** A closure that a subtest calls takes the subtest's
  `t` as its first parameter.
- An HTTP handler is tested through `httptest.NewRecorder` on a real `*http.ServeMux` that its
  `register<Name>Handler` bound, so the route pattern is under test too.

## Regenerating expectations

A datadriven corpus (`pkg/datadriven`) regenerates by **running** the tests:

```sh
go test ./<packages> -rewrite
```

- The `-rewrite` flag must come **after** the package list. `go test -rewrite ./...` silently
  tests nothing.
- Only a package that uses `pkg/datadriven` accepts `-rewrite`.
- `-rewrite` rewrites only the corpora the run executed, so it is safe under a `-run` filter.
- Review every rewritten expectation as an assertion you author. `pkg/datadriven/README.md`
  holds the format and the runner.
- **A datadriven command with no `----` swallows the commands after it.** Always terminate a
  no-output command with `----`.

## goleak

`main_test.go` enables `goleak.VerifyTestMain` in a package that starts goroutines, servers or
pools (`cmd/serve/main_test.go`).

- **Start bare.** Add an ignore only when a failure names the goroutine. Comment every ignore
  with why that goroutine is benign.
- **Never copy a sibling's ignore list.** An ignore added ahead of a failure silences the check
  instead of explaining one.

## The test database

Integration tests dial a live PostgreSQL. Bring it up with `task up`, then create the migrated
`lora_test` database with `task test:setup`. The task drops the database first, so it is
rerunnable.

- A test that needs the database reads `config/test.yml` (the `LORA_CONFIG` the test tasks set)
  and skips under `testing.Short()` when the database is not required for the change.
- A test isolates itself by the rows it creates, never by a global teardown. Scope every query
  and every delete to the test's own entities. Never write a bare `DELETE FROM <table>`.
- **Never assert that your own row comes back from a globally scoped, page-limited read.** Such a
  test passes until the table outgrows the page.
- A test that creates or drops a table uses its own schema, not `public`.

## Determinism

- Do not read `time.Now()` in code under test that a test must pin. Inject a clock.
- Seed randomness explicitly, and never assert on a literal random value.
- Let Postgres fill generated ids. Read them back with `RETURNING`.
- **Pin a float expectation with a tolerance, never with bits.**
- Never `time.Sleep` and hope. Use `require.Eventually` on the observable result.

## Config overrides

`config/dev.yml` and `config/test.yml` are gitignored copies of their `.default.yml` templates.
`task init` does **not** overwrite them. After you pull a change to a template, delete the copy
and run `task init` again.
