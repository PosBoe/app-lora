# Datadriven templates

Copy-paste templates matching this repo's conventions, plus exemplar testdata files. All are distilled from real tests in-tree; the cited originals are the source of truth.

## 1. Pure handler (no DB)

Modeled on `pkg/logparser/loglexer/tokenizer_test.go`.

```go
package foo

import (
    "errors"
    "fmt"
    "strings"
    "testing"

    "github.com/waneock/app-lora/pkg/datadriven"
)

// TestParse runs `> parse` cases through Parse and renders the normalized
// result, or the error message when parsing fails. Errors are part of the
// contract, so they are golden output, not test failures.
func TestParse(t *testing.T) {
    datadriven.Walk(t, "testdata/parse", func(t *testing.T, path string) {
        datadriven.RunTest(t, path, func(t *testing.T, tc *datadriven.TestCase) string {
            switch tc.Name {
            case "parse":
                return parseCase(t, tc)
            default:
                tc.Fatalf(t, "unsupported command: %s", tc.Name)
                return ""
            }
        })
    })
}

func parseCase(t *testing.T, tc *datadriven.TestCase) string {
    // Default first; Scan leaves it unchanged when the arg is absent.
    strict := false
    if err := tc.NamedArg("strict").Scan(&strict); err != nil {
        tc.Fatalf(t, "scan strict: %v", err)
    }

    got, err := Parse(tc.Input, strict)
    if err != nil {
        // Behavior-under-test error: golden it. Render typed fields
        // separately so a regression in any one shows up locally in the diff.
        var sb strings.Builder
        if pe, ok := errors.AsType[*ParseError](err); ok {
            fmt.Fprintf(&sb, "pos: column %d\n", pe.Column)
            fmt.Fprintf(&sb, "msg: %s\n", pe.Msg)
        }
        fmt.Fprintf(&sb, "error: %v", err)
        return sb.String()
    }
    return got.String()
}
```

Handler-enforced invariant (from `TestParse` in `tracing/spanquery/tql_parse_corpus_test.go`) — the golden pins the value, Go pins the fixed point:

```go
normalized := query.String()
_, reparsed, err := parseQueryError(normalized, cr)
if err != nil {
    tc.Fatalf(t, "round-trip parse failed for %q: %v", normalized, err)
}
if got := reparsed.String(); got != normalized {
    tc.Fatalf(t, "round-trip not idempotent: %q became %q", normalized, got)
}
return normalized
```

Testdata (`testdata/parse/errors`):

```
# a dangling operator reports the column after the last token.
> parse
1 +
----
pos: column 4
msg: unexpected end of input
error: parse error at column 4: unexpected end of input

# strict mode rejects what lenient mode normalizes.
> parse strict=true
1 ++ 2
----
pos: column 3
msg: duplicate operator
error: parse error at column 3: duplicate operator
```

## 2. Testbed runner with a custom command

Modeled on `tsmetric/metricread/query_handler_test.go`. `Run` boots the app and returns a `*testbed.App`; `app.Runner()` is a preseeded `*testbed.Runner` (built-ins `reset`, `seed`, `insert`/`update`/`delete`/`select`, `http`, `adjust_clock` already registered). Register only what's missing, then run the corpus.

```go
package foo_test

import (
    "testing"

    "go.uber.org/fx"

    "github.com/waneock/app-lora/core/coremodel"
    "github.com/waneock/app-lora/dbfixture"
    "github.com/waneock/app-lora/pkg/datadriven"
    "github.com/waneock/app-lora/pkg/ddtest"
    "github.com/waneock/app-lora/pkg/tabletest"
    "github.com/waneock/app-lora/tracing"
)

// TestEval runs `> eval` cases against the real engine under the testbed.
// Output is the result rendered by tabletest.RenderColumns.
func TestEval(t *testing.T) {
    // Run is the package's own entry point over a testbed.Bed (see its
    // testbed_test.go), which owns the module preamble.
    var engine *tracing.Engine // any application dependency
    app := Run(t, fx.Populate(&engine))
    runner := app.Runner()

    // Stateless command: CommandFunc. Look up fixture entities via f on every
    // Run — the fixture is reseeded per testdata file.
    runner.Register("eval", ddtest.CommandFunc(
        func(t *testing.T, f *dbfixture.Fixture, tc *datadriven.TestCase) string {
            project := dbfixture.MustGet[*coremodel.Project](f, "default")
            columns, rows, err := engine.Eval(t.Context(), project, tc.Input)
            if err != nil {
                return err.Error()
            }
            return tabletest.RenderColumns(columns, rows)
        }))

    app.RunDatadriven("testdata/eval")
}
```

A **stateful** command implements the full interface; `Reset` undoes what `Run` accumulated (called after every file and on `> reset`):

```go
type loadThingsCommand struct{ loaded []int64 }

func (c *loadThingsCommand) Run(t *testing.T, f *dbfixture.Fixture, tc *datadriven.TestCase) string {
    // parse tc.Input, insert rows pinned to f.Clock().Now(), record IDs in c.loaded
    return ""
}

func (c *loadThingsCommand) Reset(t *testing.T, f *dbfixture.Fixture) {
    // delete c.loaded rows; c.loaded = nil
}
```

Existing load commands — reuse before writing your own: `tsmetricddtest.LoadMetricsCommand`, `tracingddtest.LoadTracesCommand`, `tracingddtest.LoadEdgesCommand` (see `tracing/tracingddtest/` and `tsmetric/tsmetricddtest/`).

Testdata with setup + primary command (`testdata/eval/grouping`):

```
> load_metrics
  counter test{host="host1"} 1 2 3
  counter test{host="host2"} 4 5 6
----

# grouping by host yields one row per series.
> eval
  sum($test) by (host)
----
+-----------+------------------+
| host [G]  | sum($test) [A,N] |
+-----------+------------------+
| host1     | 6                |
+-----------+------------------+
| host2     | 15               |
+-----------+------------------+
```

## 3. HTTP handler test with a format

Modeled on `core/tenancy/org_user_handler_test.go`. Register a **named format** and set
it as the default when every case needs the same trimming; select a non-default
view per directive with `format=<name>`, or use per-request `extract={…}` /
`format=status` when only some cases differ. A format receives an immutable
`*ddhttp.Response` (status, headers, body bytes). Error responses
(status >= 400) auto-render as the standard envelope — no error branch needed.

```go
app := Run(t)
httpCmd := app.Runner().HTTP()

httpCmd.RegisterFormat("alert", func(t *testing.T, tc *datadriven.TestCase, r *ddhttp.Response) string {
    // Unmarshal the response into a purpose-built struct: explicit yaml
    // tags fix the field order, omitempty keeps goldens minimal, and
    // volatile fields (ids, timestamps) simply have no field to land in.
    var resp struct {
        Alert *struct {
            Name   string `json:"name" yaml:"name"`
            Status string `json:"status" yaml:"status"`
        } `json:"alert" yaml:"alert,omitempty"`
    }
    if err := json.Unmarshal(r.Body, &resp); err != nil {
        tc.Fatalf(t, "formatter: %v", err)
        return ""
    }
    buf, err := yaml.Marshal(resp)
    if err != nil {
        tc.Fatalf(t, "formatter: marshal: %v", err)
        return ""
    }
    return string(buf)
})
httpCmd.SetDefaultFormat("alert")

app.RunDatadriven("testdata/alert_handler")
```

Testdata — a workflow file (`testdata/alert_handler/create`). Base + bearer once at the top, relative paths, `store_key` to capture IDs, `select` to assert DB state:

```
> http base /internal/v1/orgs/{org:default.id}
----
/internal/v1/orgs/1

> http bearer {user_token:default.token}
----

# create with fake defaults for fields the case doesn't care about.
> insert user fake_data=true
key: alice
----

> http POST ./alerts
{"name": "cpu high", "userId": {user:alice.id}}
----
200 OK application/json
alert:
  name: cpu high
  status: open

> http store_key alert:cpu-high path=$.alert.id
----

# the row landed with the creating user attached.
> select alert key=cpu-high
UserID
----
2

# body irrelevant here — only the rejection matters.
> http POST ./alerts format=status
{"name": ""}
----
400 Bad Request application/json
```

## 4. Renderer rules (all tiers)

- One observable fact per line, `key: value`; never `%+v` struct dumps.
- YAML for structured output: marshal a purpose-built struct (fixed field order, `omitempty`), not the domain model.
- Errors under test: render typed fields (position, code, message) plus the full `error:` line.
- Computed floats at `%.12g`, never `%g` / `'g', -1` — the last digits differ between CPU architectures.
- A command with no output still ends with `----`; without it, the directives after it become its input and never run.
- Deterministic only — fake clock for time, sorted slices, no map iteration order, `store_key`/interpolation instead of literal generated IDs.
- Blank lines in output require `---- BEGIN` / `---- END` (`-rewrite` picks the delimiter automatically).
