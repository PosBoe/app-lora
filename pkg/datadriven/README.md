# datadriven

Data-driven test framework for Go. Test cases live in plain text files — each case declares a
command, optional input, and expected output. Your handler returns the actual output as a string;
the framework diffs it against expected and fails on mismatch. Pass `-rewrite` to regenerate
expected output in-place instead of failing.

Internally the package is a pipeline over an explicit document model: `Parse` turns a file into a
`*File` (cases plus verbatim comment trivia), the runner executes the cases, and `Format` prints
the document back. `Format(Parse(x))` is byte-identical to `x`, so `-rewrite` only ever changes
the expected output your handler actually produced — comments, spacing, and untouched cases never
churn.

Forked from [cockroachdb/datadriven](https://github.com/cockroachdb/datadriven).

## Test file format

```
> eval
1 + 1
----
2
```

A single file holds many cases separated by blank lines:

```
> eval
1 + 1
----
2

> eval
6 * 7
----
42
```

**Comments.** Lines starting with `#` between cases are comments — always, whether it's a single
line, a block, a bare `#`, or a `# ====` divider. They never affect execution and are preserved
byte-for-byte by `-rewrite`. Use them generously to group and explain cases:

```
# ============================================================
# Aggregation over sparse series.
# ============================================================

# returns totals per service, falls back to existing series
> eval
...
```

There is no in-file subtest mechanism: the file is the unit of grouping (each file is its own
`t.Run` when using `Walk`), and a single case is targeted with `-focus` (see **Flags**). Keep each
file scoped to one behavior instead of sectioning a large file.

**Arguments** follow the command on the same line:

```
> eval precision=2
1 / 3
----
0.33
```

Arguments are parsed into `tc.Args` — they do not appear in `tc.Input`. Multiple values are
supported via square brackets:

```
> eval cols=[x, y, z]
x + y + z
----
x(1) + y(2) + z(3) = 6
```

A value may also be a **flow mapping**, grouping several named values under one argument. Its
inner spaces do not split the argument, and `Arg.Fields()` returns the keys in file order:

```
> http GET ./alerts query={search: "alert 1", limit: 10} format=status
```

`Fields` decodes each value by `Arg.Text` rules — the raw scalar text — so `30000` and
`2025-01-01T00:00:00Z` reach the command verbatim. A value carrying a YAML flow indicator
(`{ } [ ] , :` and friends) must be quoted, which is what a JSONPath, a `{entity:key.field}`
placeholder or an MQL expression needs:

```
> http GET ./alerts extract={first: '$.alerts[0].name', total: $.total}
```

Scalar values may be quoted when they contain spaces:

```txt
> cmd search="alert 1" note='it''s ok'
```

Quoted values are preserved as YAML scalars, so `Arg.Scan(&dest)` decodes them exactly like other
datadriven argument values.

Repeating an argument name is a parse error — pass multiple values as a bracket list instead
(`> http GET /x query={id: [1, 2]}`, which the ddtest http command encodes as `id=1&id=2`).

**Long directive lines** continue on the next line when that line starts with `>` too. Wrap
mid-arguments, or give the command a line of its own and put every argument below it:

```
> http GET ./alerts
>   query={status: open, limit: 10}
>   format=status
```

Continuation lines join with a single space, so a flow mapping may be split across them
(`query={status: open,` / `>   limit: 10}`). The wrapped layout is preserved by `-rewrite`.

Because the `>` marker alone decides, **a case's input can never start with a `>` line** — such a
line is read as a continuation of the directive. Input that must begin with `>` starts with a
blank line instead (blank lines inside input are content, not separators):

```
> parse

> xx a=b
----
"xx" [a=b]
```

**Input is optional** — omit it when the command needs no body.

**Blank lines in expected output** — use `---- BEGIN` / `---- END` delimiters when the command
returns multi-paragraph text where blank lines are meaningful (e.g. formatted query plans, diff
output):

```
> cmd
---- BEGIN
first paragraph

second paragraph
---- END
```

`-rewrite` picks the delimiter from the actual output alone: `---- BEGIN` / `---- END` when it
contains a blank line, plain `----` otherwise. It therefore *reformats* a section you wrote in the
explicit form when the output turns out not to need it — that is a normalization, not a failure.
Empty output is a bare `----`.

## Recommended format

Use YAML for both input and expected output when the data is structured (objects, lists, key-value
pairs). YAML is unambiguous about types, handles special characters via quoting, and uses standard
list syntax (`- `) instead of ad-hoc repeated keys. Use `github.com/goccy/go-yaml` to marshal the
result in your handler:

```
> parse_js
TypeError: boom
    at init (https://example.com/app.js:10:2)
----
format: chrome
type: TypeError
message: boom
frames:
- func: init
  file: https://example.com/app.js
  line: 10
  col: 2
```

Plain text is fine when the output is inherently unstructured (e.g. formatted strings, error
messages, query plans).

## Recommended structure

Organize tests as a directory of files, and use `Walk` so each file becomes its own `t.Run`
subtest. Inside a file, group related cases visually with `#` comments.

Two repo-wide layout rules:

- **Behavior files are extensionless.** A corpus lives at `testdata/<topic>/<behavior>`
  (e.g. `testdata/analyze/basic`), and the filename names the behavior — no `.txt`.
- **dbfixture seeds live under `testdata/fixtures/`.** Keeping seed `.yml` files
  in their own subdirectory separates them at a glance from datadriven corpora
  (which are directories).

Keep each file scoped to a single behavior, typically driven by a single command. Parametrize the
command with arguments rather than mixing unrelated commands:

```
> parse 1 + 2
----
...

> parse 3 * 4 hint=filter
----
...
```

Exceptions where multiple commands in one file are appropriate:

- Support commands that prepare the test environment or seed data (e.g. `> load_fixtures`,
  `> set_clock`) before exercising the case under test.
- HTTP API tests, where a file drives a sequence of `> http GET …` / `> http POST …` calls
  against the same endpoint or workflow.

## Writing a test

### Single file

```go
// handle is passed to RunTest and called once per test case.
func handle(t *testing.T, tc *datadriven.TestCase) string {
    switch tc.Name {
    case "eval":
        return evaluate(tc.Input)
    default:
        tc.Fatalf(t, "unsupported command: %s", tc.Name)
        return ""
    }
}

func TestEval(t *testing.T) {
    datadriven.RunTest(t, "testdata/eval", handle)
}
```

### Directory of files

`Walk` recurses into subdirectories and creates a `t.Run` subtest per file. Temp/hidden files and
markdown documentation (`*.md`) are skipped automatically:

```go
func TestEval(t *testing.T) {
    datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
        datadriven.RunTest(t, path, handle)
    })
}
```

## TestCase reference

```go
type TestCase struct {
    Leading  []string // verbatim comment/blank lines above the directive
    Pos      Pos      // Pos{Path: "testdata/eval", Line: 12}; prints as "testdata/eval:12"
    Name     string   // command name
    Args     Args     // parsed arguments
    Input    string   // text between the command line and ----
    Expected string   // text after ----
}
```

`tc.Fatalf` is preferred over `t.Fatalf` — it prepends the `file:line` position of the failing
case to the error message, making failures much easier to locate.

**Reading arguments:**

```go
// Positional argument (bare word, no value): "cmd foo"
name := tc.Arg(0).Name

// Named argument, scalar: "cmd limit=10"
// Scan returns nil and leaves dest unchanged when the argument is absent.
var limit int
if err := tc.NamedArg("limit").Scan(&limit); err != nil {
    tc.Fatalf(t, "scan limit: %v", err)
}

// Named argument, slice: "cmd cols=[x, y, z]"
var cols []string
if err := tc.NamedArg("cols").Scan(&cols); err != nil {
    tc.Fatalf(t, "scan cols: %v", err)
}

// Named argument, JSON/YAML map: "cmd agg={"foo":"sum"}"
agg := make(map[string]string)
if err := tc.NamedArg("agg").Scan(&agg); err != nil {
    tc.Fatalf(t, "scan agg: %v", err)
}

// Check presence before scanning
if arg := tc.NamedArg("group"); !arg.IsZero() {
    var group string
    if err := arg.Scan(&group); err != nil {
        tc.Fatalf(t, "scan group: %v", err)
    }
}
```

Note that a bare argument is both positional and named: `NamedArg("verbose")` finds the bare word
in `> eval verbose`, and `Arg(0).Name` reads `GET` in `> http GET /path`. The syntax cannot
distinguish the two roles.

## Flags

| Flag | Description |
|---|---|
| `-rewrite` | Overwrite expected output with actual results instead of failing |
| `-focus <path>[:<line>]` | Run only one testdata file, or one case within it |
| `-datadriven-quiet` | Suppress per-case log output |

### `-rewrite`

Regenerate all expected output after a broad behaviour change:

```bash
go test ./tsmetric -rewrite
```

The flag must come after the package list — `go test -rewrite ./...` makes `go test` stop parsing
at the unknown flag and silently test nothing. Only packages that use datadriven accept the flag.

Always review the diff before committing — `-rewrite` will happily encode bugs as "expected".

### `-focus`

Failure messages print each case's position as `path:line`. Copy that into `-focus` to iterate on
exactly the failing case:

```bash
# The failure said "testdata/analyze/basic:42". Re-run just that case:
go test ./pkg/sqldigest -run TestAnalyze -focus testdata/analyze/basic:42

# Or run just that file, all cases compared:
go test ./pkg/sqldigest -run TestAnalyze -focus testdata/analyze/basic
```

(As with `-rewrite`, keep `-focus` after the package list.)

With a `:line`, cases above the target still execute — stateful files need their seed/setup
prefix — but their expected output is not compared, so stale earlier cases don't get in the way
while you iterate. Cases below the target are skipped. Any line inside a case (input, `----`, or
expected output) selects it. Combined with `-rewrite`, only the focused case is rewritten.

Other files — in every `Walk` and `RunTest` in the test run — are skipped. Paths match by suffix,
so both the package-relative form (`testdata/foo`) and the repo-relative form
(`tsmetric/testdata/foo`) work.

## ClearResults

Strips expected output from a test file while keeping commands, inputs, and comments intact.

Use this when `-rewrite` would produce garbage rather than fix things — for example, if a
serialization bug causes every result to be malformed, running `-rewrite` would commit broken output
for every case. `ClearResults` lets you wipe expected output first, fix the bug, then rewrite
cleanly:

```go
err := datadriven.ClearResults("testdata/eval")
```
