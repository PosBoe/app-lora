# Reshape patterns

Concrete templates and worked examples for both directions of `/go-test-reshape` — consolidating and splitting — scoped to this repo's Go conventions.

## Datadriven template

File: `pkg/<area>/<name>_test.go`

```go
package <pkg>

import (
    "fmt"
    "strings"
    "testing"

    "github.com/waneock/app-lora/pkg/datadriven"
)

// TestFooDatadriven exercises Foo by rendering the observable result of each
// case to text and comparing against testdata/foo/<file>.
func TestFooDatadriven(t *testing.T) {
    datadriven.Walk(t, "testdata/foo", func(t *testing.T, path string) {
        datadriven.RunTest(t, path, func(t *testing.T, tc *datadriven.TestCase) string {
            switch tc.Name {
            case "eval":
                return evalCase(tc)
            case "parse":
                return parseCase(tc)
            default:
                tc.Fatalf(t, "unsupported command: %s", tc.Name)
                return ""
            }
        })
    })
}

// evalCase renders Foo.Eval output. Keep the renderer flat: one observable
// fact per line, "key: value" form, so diffs read clearly.
func evalCase(tc *datadriven.TestCase) string {
    var precision int
    if err := tc.NamedArg("precision").Scan(&precision); err != nil {
        return fmt.Sprintf("scan precision: %v", err)
    }

    got, err := Eval(tc.Input, precision)
    var sb strings.Builder
    if err != nil {
        // Render structured error fields so regressions show up locally in the diff.
        // See renderer rules below — prefer typed fields over raw err.Error().
        if pe, ok := errors.AsType[*ParseError](err); ok {
            fmt.Fprintf(&sb, "typed: true\n")
            fmt.Fprintf(&sb, "pos: column %d\n", pe.Column)
            fmt.Fprintf(&sb, "msg: %s\n", pe.Msg)
        }
        fmt.Fprintf(&sb, "error: %v", err)
        return sb.String()
    }
    fmt.Fprintf(&sb, "result: %s\n", got)
    return sb.String()
}
```

File: `pkg/<area>/testdata/foo/eval`

```
# basic arithmetic

> eval
1 + 1
----
result: 2

> eval precision=2
1 / 3
----
result: 0.33

# errors

> eval
1 /
----
error: unexpected end of input at column 4
```

Renderer rules:

- One observable fact per line; never embed Go struct dumps. Stable, ordered fields.
- For errors, render type + position + message separately so a regression in any one shows up locally in the diff:
  ```
  typed: true
  pos: column 4
  msg: unexpected end of input
  error: parse error at column 4: unexpected end of input
  ```
- If output may legitimately contain blank lines, use `---- BEGIN` / `---- END`.
- Never include timestamps, random IDs, or map iteration order in output. Sort, or render only deterministic facts.

`#` comment lines group related cases visually inside one file; use them generously — they never affect execution and are preserved by `-rewrite`. The file is the subtest unit (one `t.Run` per file via `Walk`); to re-run a single failing case, copy the `path:line` from the failure message into `go test -focus <path>:<line>`.

## Table-driven template (no snapshot)

```go
func TestExtractUint64s(t *testing.T) {
    // Scenario: alert filters must reject malformed numeric IDs instead of
    // silently dropping them and widening the result set.
    tests := []struct {
        name    string
        values  []any
        want    []uint64
        wantErr bool
    }{
        {name: "strings and numbers", values: []any{"1", float64(2)}, want: []uint64{1, 2}},
        {name: "malformed string",    values: []any{"not-a-number"},  wantErr: true},
        {name: "fractional number",   values: []any{float64(1.5)},     wantErr: true},
        {name: "unsupported type",    values: []any{true},             wantErr: true},
        {name: "nil value",           values: []any{nil},              wantErr: true},
        {name: "empty",               values: []any{},                 want: nil},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := extractUint64s(map[string][]any{
                attrkey.AlertAssignedUser: tt.values,
            }, attrkey.AlertAssignedUser)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            require.Equal(t, tt.want, got)
        })
    }
}
```

Field order: `name`, inputs, `want`, `wantErr` last. One case per line where it fits.

The `// Scenario:` comment is required by AGENTS.md for *standalone* plain tests covering corner cases or complex behavior. For tables it's optional but helpful when the table's purpose isn't obvious from the function name — the comment should name the behavior, not the API.

When the cases are independent (no shared mutable state between iterations), call `t.Parallel()` at the top of the `t.Run` body. Datadriven cases are NOT parallelizable — the framework runs them in order.

## Table-driven with snapshot

Use only when the output is a large nested value AND datadriven is a bad fit (multiple parallel values per case, lifecycle-bound state). Pattern from `pkg/ch/query_test.go`:

```go
func TestQueryLocal(t *testing.T) {
    db := queryTestDB()
    defer db.Close()

    for i, fn := range testQueries() {
        t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
            q := fn(db)
            query, err := q.AppendQuery(db.QueryGen(), nil)
            snaps.MatchSnapshot(t, map[string]any{
                "query": string(query),
                "err":   err,
            })
        })
    }
}
```

Refresh with `UPDATE_SNAPS=true go test ./...`. Inspect every changed snapshot before committing.

Prefer datadriven over snapshot when the output is text — snapshots are an opaque blob; datadriven files are reviewable.

## What to delete in the same edit

When consolidating, delete in the same edit:

- The original `TestX_a`, `TestX_b`, ... functions.
- Any helper that existed only to support those tests (verify with grep before removing).
- `// removed: ...` placeholders. Just delete.

Do not leave the originals next to the consolidated test for "easier diff review" — that defeats the consolidation. The git diff is the review surface.

## Worked consolidation example (compressed)

Before — three separate tests in `parser_test.go`:

```go
func TestParseInt(t *testing.T) {
    q, err := Parse("42")
    require.NoError(t, err)
    require.Equal(t, &IntLit{Value: 42}, q)
}

func TestParseAdd(t *testing.T) {
    q, err := Parse("1 + 2")
    require.NoError(t, err)
    require.IsType(t, &BinaryOp{}, q)
}

func TestParseError(t *testing.T) {
    _, err := Parse("1 +")
    require.Error(t, err)
}
```

After — one datadriven test plus `testdata/parser/basic`:

```go
func TestParseDatadriven(t *testing.T) {
    datadriven.Walk(t, "testdata/parser", func(t *testing.T, path string) {
        datadriven.RunTest(t, path, func(t *testing.T, tc *datadriven.TestCase) string {
            q, err := Parse(tc.Input)
            if err != nil {
                var sb strings.Builder
                if pe, ok := errors.AsType[*ParseError](err); ok {
                    fmt.Fprintf(&sb, "typed: true\n")
                    fmt.Fprintf(&sb, "pos: column %d\n", pe.Column)
                    fmt.Fprintf(&sb, "msg: %s\n", pe.Msg)
                }
                fmt.Fprintf(&sb, "error: %v", err)
                return sb.String()
            }
            return ast.String(q)
        })
    })
}
```

```
# literals

> parse
42
----
42

# operators

> parse
1 + 2
----
(1 + 2)

# errors

> parse
1 +
----
typed: true
pos: column 4
msg: unexpected end of input
error: parse error at column 4: unexpected end of input
```

Changes worth calling out in the summary to the user:

- Three standalone tests collapsed into one datadriven test with three cases.
- `require.IsType` replaced with full AST rendering — the datadriven output shows the actual structure, not just the type.
- Use `/go-test-decouple` afterward to audit the assertions, and `/go-test-coverage` to add missing cases (e.g. precedence, round-trip).

## Worked split example (compressed)

Before — one datadriven test whose handler became a switchboard:

```go
func TestParseDatadriven(t *testing.T) {
    datadriven.Walk(t, "testdata/parse", func(t *testing.T, path string) {
        datadriven.RunTest(t, path, func(t *testing.T, tc *datadriven.TestCase) string {
            switch tc.Name {
            case "parse":     // renders the AST
            case "format":    // renders formatted output
            case "roundtrip": // parse → format → parse, asserts fixed point
            case "errors":    // renders typed error fields
            case "complete":  // renders completion suggestions
            }
        })
    })
}
```

After — one test per behavior cluster, each with its own testdata directory and a one-or-two-command handler:

```go
func TestParseDatadriven(t *testing.T)    { /* walks testdata/parse;    commands: parse, errors */ }
func TestFormatDatadriven(t *testing.T)   { /* walks testdata/format;   commands: format, roundtrip */ }
func TestCompleteDatadriven(t *testing.T) { /* walks testdata/complete; command:  complete */ }
```

Testdata moves with the commands: cases from `testdata/parse/*` that used `> format` or `> roundtrip` relocate under `testdata/format/`, keeping their `#` comments. No case is edited in the move — goldens must survive byte-for-byte (verify with the leaf-case diff from the SKILL's finalize step).

The table-driven equivalent: case-name prefixes (`fmt_*`, `rt_*`) inside one table become `TestFormat_*` functions with their own scoped table structs.
