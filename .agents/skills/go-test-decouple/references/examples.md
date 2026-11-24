# Assertion fix examples

Concrete before/after rewrites, scoped to this repo's Go conventions. Examples 1–6 are the loud patterns a grep finds; the rest are the quiet ones that only surface when reading each test file — they are the reason most packages that "look clean" are not.

## 1. Mock call count → return value assertion

Before:

```go
func TestProcessBatch(t *testing.T) {
    store := new(MockStore)
    store.On("Put", mock.Anything).Return(nil)

    err := ProcessBatch(store, []Item{{ID: 1}, {ID: 2}, {ID: 3}})
    require.NoError(t, err)
    store.AssertNumberOfCalls(t, "Put", 3) // ← pins batching strategy
}
```

After:

```go
func TestProcessBatch(t *testing.T) {
    store := newMemStore() // a fake that keeps what it is given

    err := ProcessBatch(store, []Item{{ID: 1}, {ID: 2}, {ID: 3}})
    require.NoError(t, err)

    got, err := store.GetAll()
    require.NoError(t, err)
    require.ElementsMatch(t, []int{1, 2, 3}, idsOf(got))
}
```

Why: `ProcessBatch` might switch from per-item puts to a single bulk put. The call count changes; the result doesn't. A mock that only records calls cannot answer "what is stored", so swap it for a fake that keeps state and assert on what is in the store, not how it got there.

## 2. Error substring → typed error

Before:

```go
func TestParseInvalid(t *testing.T) {
    _, err := Parse("1 +")
    require.Error(t, err)
    require.Contains(t, err.Error(), "unexpected") // ← matches any "unexpected" error
}
```

After:

```go
func TestParseInvalid(t *testing.T) {
    _, err := Parse("1 +")
    parseErr, ok := errors.AsType[*ParseError](err)
    require.True(t, ok)
    require.Equal(t, 4, parseErr.Pos.Column())
}
```

Why: `Contains(err.Error(), "unexpected")` would pass for completely unrelated errors that happen to contain the word "unexpected." The typed assertion confirms both the error kind and the structured position.

## 3. Unexported field → public API

Before:

```go
func TestCacheWarm(t *testing.T) {
    c := NewCache(100)
    c.Get("key")
    c.Get("key")
    require.Equal(t, 1, c.missCount) // ← unexported field
}
```

After:

```go
func TestCacheWarm(t *testing.T) {
    c := NewCache(100)
    c.Put("key", "value")

    got, ok := c.Get("key")
    require.True(t, ok)
    require.Equal(t, "value", got)
}
```

Why: `missCount` is an internal bookkeeping field. The observable contract is "Get returns the value after Put." If `missCount` is renamed or removed, the old test breaks for no behavioral reason.

If miss/hit stats are part of the public API (e.g. exposed via `Stats()` method), assert on `c.Stats().Misses` instead — that's a contract, not an implementation detail.

## 4. Log output → return value / state assertion

Before:

```go
func TestRetryExhausted(t *testing.T) {
    var buf bytes.Buffer
    logger := slog.New(slog.NewTextHandler(&buf, nil))

    client := NewClient(WithLogger(logger), WithMaxRetries(2))
    err := client.Send(unreachableAddr, payload)

    require.Error(t, err)
    require.Contains(t, buf.String(), "retry attempt 2") // ← pins log wording
    require.Contains(t, buf.String(), "level=WARN")      // ← pins log level
}
```

After:

```go
func TestRetryExhausted(t *testing.T) {
    client := NewClient(WithMaxRetries(2))
    err := client.Send(unreachableAddr, payload)

    retryErr, ok := errors.AsType[*RetryError](err)
    require.True(t, ok)
    require.Equal(t, 2, retryErr.Attempts)
}
```

Why: The observable contract is "Send returns a `*RetryError` with the attempt count after exhausting retries." Log lines are a debugging aid — their wording and level can change freely. If retry metadata isn't available through the error or a public API, flag it as an API gap rather than asserting on logs.

## 5. Metric counter → behavioral outcome

Before:

```go
func TestIngestRecordsMetric(t *testing.T) {
    reg := prometheus.NewRegistry()
    srv := NewServer(WithRegistry(reg))

    srv.Ingest([]Record{{ID: 1}, {ID: 2}})

    counter := testutil.ToFloat64(
        reg.(*prometheus.Registry).MustFind("records_ingested_total"),
    )
    require.Equal(t, 2.0, counter) // ← pins metric name + instrumentation
}
```

After:

```go
func TestIngestRecordsMetric(t *testing.T) {
    srv := NewServer()

    err := srv.Ingest([]Record{{ID: 1}, {ID: 2}})
    require.NoError(t, err)

    got, err := srv.ListRecords()
    require.NoError(t, err)
    require.Len(t, got, 2)
}
```

Why: The contract is "Ingest stores records," not "Ingest increments a Prometheus counter by N." Renaming or restructuring metrics shouldn't break functional tests. Keep metric assertions only in tests for the metrics package itself or when metric emission is a documented requirement.

## 6. Map iteration order → order-insensitive comparison

Before:

```go
func TestListTags(t *testing.T) {
    m := map[string]string{"env": "prod", "app": "api", "region": "us"}
    got := TagKeys(m)
    require.Equal(t, []string{"app", "env", "region"}, got) // ← assumes sorted iteration
}
```

After:

```go
func TestListTags(t *testing.T) {
    m := map[string]string{"env": "prod", "app": "api", "region": "us"}
    got := TagKeys(m)
    require.ElementsMatch(t, []string{"app", "env", "region"}, got)
}
```

Why: Go randomizes map iteration order. `require.Equal` on an unsorted slice from a map produces tests that pass locally and fail in CI. Use `require.ElementsMatch` when the contract doesn't specify order. If the API documents sorted output, sort the result and use `require.Equal` — but verify the API actually guarantees it.

## 7. White-box test of an unexported function → exported entry point

Before:

```go
func TestNormalizeAttrs(t *testing.T) {
    got := normalizeAttrs(map[string]any{"HTTP.Method": "GET"}) // ← unexported function
    require.Equal(t, map[string]any{"http.method": "GET"}, got)
}
```

After:

```go
func TestProcessSpan_normalizesAttrKeys(t *testing.T) {
    span, err := ProcessSpan(RawSpan{Attrs: map[string]any{"HTTP.Method": "GET"}})
    require.NoError(t, err)
    require.Equal(t, "GET", span.Attrs["http.method"])
}
```

Why: the test pins the internal decomposition. Inlining `normalizeAttrs` into `ProcessSpan`, renaming it, or splitting it in two breaks the old test even though every caller of the package sees identical behavior. Retarget the same edge case at the exported entry point.

Exception: keep white-box tests for genuinely complex internal algorithms (a parser core, a rebalancing routine) where reaching every edge through the public API would be indirect and noisy — and note the exception in the report.

## 8. Overspecified struct equality → contract fields

Before:

```go
func TestPlanQuery(t *testing.T) {
    got, err := PlanQuery("dur > 100")
    require.NoError(t, err)
    require.Equal(t, &Plan{
        Table:         "spans",
        Filter:        "dur > 100",
        EstimatedRows: 12034,    // ← estimator output, not a contract
        Version:       3,        // ← internal bookkeeping
    }, got)
}
```

After:

```go
func TestPlanQuery(t *testing.T) {
    got, err := PlanQuery("dur > 100")
    require.NoError(t, err)
    require.Equal(t, "spans", got.Table)
    require.Equal(t, "dur > 100", got.Filter)
}
```

Why: whole-struct equality pins every field, including ones no caller depends on. Tuning the row estimator or bumping the internal version breaks the test with zero behavioral change. Assert only the fields callers rely on.

Exception: whole-value equality is right when fidelity IS the contract (round-trip serialization), or in a datadriven golden rendered from a purpose-built struct that already selects contract fields.

## 9. Ordered mock expectations → observable output

Before:

```go
func TestFlush(t *testing.T) {
    store := new(MockStore)
    first := store.On("Put", "a").Return(nil)
    store.On("Put", "b").Return(nil).NotBefore(first) // ← pins internal write order

    err := Flush(store, []Item{{Key: "a"}, {Key: "b"}})
    require.NoError(t, err)
    store.AssertExpectations(t)
}
```

After:

```go
func TestFlush(t *testing.T) {
    store := newMemStore()

    err := Flush(store, []Item{{Key: "a"}, {Key: "b"}})
    require.NoError(t, err)

    got, err := store.GetAll()
    require.NoError(t, err)
    require.ElementsMatch(t, []string{"a", "b"}, keysOf(got))
}
```

Why: `Flush` may batch, parallelize, or reorder writes without changing what callers can observe. If the caller-visible result is ordered (the store documents insertion order), assert on the output's order with `require.Equal`; if not, use `require.ElementsMatch`.

## 10. Wrapped third-party error substring → sentinel

Before:

```go
func TestLoadConfigMissing(t *testing.T) {
    _, err := LoadConfig("missing.yaml")
    require.ErrorContains(t, err, "no such file or directory") // ← OS wording; differs on Windows
}
```

After:

```go
func TestLoadConfigMissing(t *testing.T) {
    _, err := LoadConfig("missing.yaml")
    require.ErrorIs(t, err, fs.ErrNotExist)
}
```

Why: the substring belongs to the operating system, not to this package — it varies across platforms and library versions. The wrap chain is the contract: `errors.Is` against the sentinel survives any rewording. For errors this package itself defines, prefer `errors.AsType[*SpecificError](err)` and assert the typed fields.

## 11. Database server text → the server's code

Before:

```go
func TestInsertRejectsUnknownProject(t *testing.T) {
    err := repo.InsertAlert(ctx, &Alert{ProjectID: 999})
    require.ErrorContains(t, err, "violates foreign key constraint") // ← PostgreSQL's wording
}
```

After:

```go
func TestInsertRejectsUnknownProject(t *testing.T) {
    err := repo.InsertAlert(ctx, &Alert{ProjectID: 999})
    pgErr, ok := errors.AsType[pgconn.Error](err)
    require.True(t, ok, "the insert did not fail in PostgreSQL: %v", err)
    require.True(t, pgErr.IntegrityViolation())
}
```

Why: the package never wrote that sentence — the server did, and a server upgrade or a locale can reword it. The server's code is stable. The same holds for ClickHouse: `errors.AsType[*chproto.Exception]` and its `Code`. Before you judge a substring, grep the package's production files for it; no hit means the wording is not the package's.

## 12. Raw read of the package's own table → its exported read

Before:

```go
func TestRunMarksJobDone(t *testing.T) {
    require.NoError(t, queue.Run(ctx, id))

    n, err := db.NewSelect().
        Model((*jobRow)(nil)). // ← unexported model, private table
        Where("id = ? AND status = ?", id, "done").
        Count(ctx)
    require.NoError(t, err)
    require.Equal(t, 1, n)
}
```

After:

```go
func TestRunMarksJobDone(t *testing.T) {
    require.NoError(t, queue.Run(ctx, id))

    job, err := queue.Job(ctx, id)
    require.NoError(t, err)
    require.Equal(t, JobDone, job.Status)
}
```

Why: the table name, its columns, and `jobRow` are the package's private storage. Split the table or rename a column and the old test fails while `queue.Job` answers the same. Keep the raw query only when the row itself is a contract another package or release reads — and say so in a comment.

## 13. Arrange through an unexported field → the exported path

Before:

```go
func TestEvaluateUsesPlannedWindow(t *testing.T) {
    q := NewQuery(spec)
    q.window = Range{Start: 0, End: 10 * time.Minute} // ← a window the planner never chose
    got, err := q.Evaluate(ctx, points)
    require.NoError(t, err)
    require.Equal(t, 3, got.Count)
}
```

After:

```go
func TestEvaluateUsesPlannedWindow(t *testing.T) {
    q, err := Plan(spec, Range{Start: 0, End: 10 * time.Minute})
    require.NoError(t, err)
    got, err := q.Evaluate(ctx, points)
    require.NoError(t, err)
    require.Equal(t, 3, got.Count)
}
```

Why: the planner may round, align, or widen the window. Writing the field by hand tests a query no caller can hold, and it breaks when the field is renamed. Reach the state through the path production uses.

## 14. A test of a test-local copy → the production entry point

Before:

```go
// sortedKeys mirrors the ordering Encode uses.
func sortedKeys(m map[string]string) []string { /* ... */ }

func TestSortedKeys(t *testing.T) {
    require.Equal(t, []string{"a", "b"}, sortedKeys(map[string]string{"b": "", "a": ""}))
}
```

After:

```go
func TestEncodeOrdersKeys(t *testing.T) {
    got := Encode(map[string]string{"b": "2", "a": "1"})
    require.Equal(t, "a=1&b=2", got)
}
```

Why: the old test checks a function the package does not ship. `Encode` can stop sorting and the test stays green. Test the property through the code that has it; if another test already holds it, delete the copy and its test.
