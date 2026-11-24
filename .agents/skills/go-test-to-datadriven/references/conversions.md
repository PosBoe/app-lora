# Worked conversions

Concrete before/after translations for `/go-test-to-datadriven`, scoped to this repo's conventions. A standalone → datadriven consolidation is worked in `.agents/skills/go-test-reshape/references/patterns.md`; this file covers the other two starting shapes and the renderer-design questions conversion raises.

## Table-driven → datadriven

The mechanical mapping: the `name` field → `#` comment; the primary string input → the case input block; scalar option fields → named args; `want` → the golden; `wantErr` → an error golden. Case order is preserved.

Before:

```go
func TestFormatSpanName(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        maxLen   int
        want     string
        wantErr  bool
    }{
        {name: "short name unchanged", input: "GET /users", maxLen: 32, want: "GET /users"},
        {name: "truncated with ellipsis", input: "GET /users/123/orders/456", maxLen: 16, want: "GET /users/123/…"},
        {name: "zero max is invalid", input: "GET /users", maxLen: 0, wantErr: true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FormatSpanName(tt.input, tt.maxLen)
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

After — `format_test.go`:

```go
// TestFormatSpanNameDatadriven exercises FormatSpanName via testdata/format_span_name.
// Command: format max_len=<n> — renders the formatted name, or the error message
// when formatting is rejected.
func TestFormatSpanNameDatadriven(t *testing.T) {
    datadriven.Walk(t, "testdata/format_span_name", func(t *testing.T, path string) {
        datadriven.RunTest(t, path, func(t *testing.T, tc *datadriven.TestCase) string {
            switch tc.Name {
            case "format":
                maxLen := 32 // default first; Scan is a no-op when the arg is absent
                if err := tc.NamedArg("max_len").Scan(&maxLen); err != nil {
                    tc.Fatalf(t, "scan max_len: %v", err)
                }
                got, err := FormatSpanName(tc.Input, maxLen)
                if err != nil {
                    return fmt.Sprintf("error: %v", err)
                }
                return got
            default:
                tc.Fatalf(t, "unsupported command: %s", tc.Name)
                return ""
            }
        })
    })
}
```

`testdata/format_span_name/truncation`:

```
# short name unchanged
> format
GET /users
----
GET /users

# truncated with ellipsis
> format max_len=16
GET /users/123/orders/456
----
GET /users/123/…

# zero max is invalid
> format max_len=0
GET /users
----
error: max length must be positive
```

Notes:

- The bare `require.Error` became a *stronger* assertion: the golden now pins the message. Read that golden as an assertion you are authoring — if the message is embarrassing, fix the message, not the golden. Call this out in the summary: it's new contract surface.
- A table field used by only a few cases still becomes a named arg with the common value as the declared default — do not make every case restate it.
- The input block is only for the *primary textual input* — the string the code under test parses or transforms. A short scalar (an identifier, a key, a mode) is a named arg even when it is the table's "main" field: `> describe metric=http_requests attr=_fingerprint`, not `> describe metric=…` with the attr as input. Splitting sibling scalars between the input block and args reads as an asymmetry the corpus never explains; a case whose fields are all scalars simply has no input block.
- If a table has two independent *textual* inputs, one goes in the input block and the other becomes a named arg; if both are multi-line, datadriven is likely the wrong shape — flag it as a skip candidate instead of inventing a delimiter syntax.

## Table + snapshot → datadriven

Only convert when the snapshotted value renders cleanly as one YAML document or text block per case. `pkg/ch/query_test.go:TestQueryLocal` (multiple parallel values, queries built in Go closures with no textual input) is the canonical *non*-candidate — leave that shape alone.

Before:

```go
func TestBuildPlan(t *testing.T) {
    for _, tt := range []struct{ name, query string }{
        {name: "simple filter", query: `where _status_code = "error"`},
        {name: "group by", query: `group by host | count()`},
    } {
        t.Run(tt.name, func(t *testing.T) {
            plan, err := BuildPlan(tt.query)
            require.NoError(t, err)
            snaps.MatchSnapshot(t, plan)
        })
    }
}
```

After — the handler YAML-renders a purpose-built struct, not the domain model:

```go
// TestBuildPlanDatadriven exercises BuildPlan via testdata/build_plan.
// Command: plan — renders the plan's stages and output columns as YAML.
func TestBuildPlanDatadriven(t *testing.T) {
    datadriven.Walk(t, "testdata/build_plan", func(t *testing.T, path string) {
        datadriven.RunTest(t, path, func(t *testing.T, tc *datadriven.TestCase) string {
            switch tc.Name {
            case "plan":
                plan, err := BuildPlan(tc.Input)
                if err != nil {
                    return fmt.Sprintf("error: %v", err)
                }
                return renderPlan(tc, plan)
            default:
                tc.Fatalf(t, "unsupported command: %s", tc.Name)
                return ""
            }
        })
    })
}

// renderPlan defines the golden's field order and keeps irrelevant fields out.
func renderPlan(tc *datadriven.TestCase, plan *Plan) string {
    type stageView struct {
        Kind string   `yaml:"kind"`
        Expr string   `yaml:"expr,omitempty"`
    }
    view := struct {
        Stages  []stageView `yaml:"stages"`
        Columns []string    `yaml:"columns"`
    }{ /* fill from plan */ }
    out, err := yaml.Marshal(view) // github.com/goccy/go-yaml
    if err != nil {
        tc.Fatalf(nil, "marshal plan: %v", err) // harness error, not behavior
    }
    return string(out)
}
```

Delete the `__snapshots__` entries for the converted test in the same edit; run the package once after deleting to confirm no obsolete-snapshot warnings remain.

Notes:

- The snapshot dumped the whole domain model; the golden should not. Design the view struct from what the *original test existed to check* plus whatever else makes the golden reviewable — and strip anything nondeterministic (timestamps, generated IDs, map order) rather than regenerating churn forever.
- Snapshot tests hide their expectations in a sidecar file; the win of conversion is that input and expectation now sit adjacent in one reviewable file. Preserve that: keep the input in the directive, never load it from another fixture file.

## Renderer design for lossy originals

Originals often assert *less* than the full output (`require.Len(got, 3)`, `require.Contains`, checking two fields of a ten-field struct). Two traps:

- **Don't reproduce the lossiness.** Rendering `len: 3` as the golden preserves the weak assertion in a shape that's now harder to strengthen. Render the actual (small) value; the golden naturally pins it.
- **Don't launder nondeterminism into the golden.** If the original checked two fields *because the rest are nondeterministic*, the fix is a view struct containing the deterministic contract fields — not sorting hacks bolted on after `-rewrite` churns.

When the original's weakness was deliberate (documented "we only guarantee X"), keep the golden to X via the view struct and say so in a `#` comment on the case.

## What stays in Go

Not every assertion moves into the golden:

- **Cross-case invariants** — round-trips (`parse(format(x)) == x`), idempotence, "output re-parses" — stay as `tc.Fatalf` checks inside the handler, documented in the test's doc comment. The golden pins the value; Go pins the invariant. Model: the re-parse check in `tracing/spanquery/tql_parse_test.go`.
- **Harness failures** — scan errors, fixture load failures — are `tc.Fatalf`, never golden output. Only errors that are themselves the behavior under test are output.
