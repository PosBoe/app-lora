# Cleanup examples

Worked rewrites for the catalog in `SKILL.md`. Each "after" checks exactly what its "before" checked.

## 1. Small shared type → a copy in each test (§1)

Before — `Pair` sits at package level and two tests share it:

```go
type Pair struct {
    Key   string `msgp:"key"`
    Value string `msgp:"value"`
}

func TestEncodePair(t *testing.T) {
    b, err := Encode(Pair{Key: "host", Value: "localhost"})
    require.NoError(t, err)
    require.NotEmpty(t, b)
}

func TestDecodePair(t *testing.T) {
    p := Pair{Key: "host", Value: "localhost"}
    b, err := Encode(p)
    require.NoError(t, err)

    var got Pair
    require.NoError(t, Decode(b, &got))
    require.Equal(t, p, got)
}
```

After — each test declares its own two-field copy and reads as a complete story:

```go
func TestEncodePair(t *testing.T) {
    type Pair struct {
        Key   string `msgp:"key"`
        Value string `msgp:"value"`
    }

    b, err := Encode(Pair{Key: "host", Value: "localhost"})
    require.NoError(t, err)
    require.NotEmpty(t, b)
}
```

(`TestDecodePair` gets the same declaration.) A nested or many-field type stays shared.

## 2. Distant `want` → next to its assertion, still whole (§5)

Before:

```go
func TestParseDuration(t *testing.T) {
    expected := Duration{Value: 5, Unit: "m"}

    parser := NewParser()
    parser.SetStrict(true)
    parser.SetDefaultUnit("s")

    got, err := parser.Parse("5m")
    require.NoError(t, err)
    require.Equal(t, expected, got)
}
```

After:

```go
func TestParseDuration(t *testing.T) {
    parser := NewParser()
    parser.SetStrict(true)
    parser.SetDefaultUnit("s")

    got, err := parser.Parse("5m")
    require.NoError(t, err)
    require.Equal(t, Duration{Value: 5, Unit: "m"}, got)
}
```

Wrong "after" — it stops checking every field of `Duration` except these two:

```go
require.Equal(t, 5, got.Value)
require.Equal(t, "m", got.Unit)
```

It is correct only if `Duration` has exactly those two fields. If it has more and they look incidental, report the test for `/go-test-decouple`.

## 3. Raw error boilerplate → testify (§5, §7)

Before:

```go
res, err := Validate(input)
if err != nil {
    t.Fatalf("unexpected error: %v", err)
}
if len(res.Items) != 3 {
    t.Fatalf("expected 3 items, got %d", len(res.Items))
}

_, err = Validate(bad)
if err == nil {
    t.Fatal("expected error")
}
if !strings.Contains(err.Error(), "timeout") {
    t.Fatalf("wrong error: %v", err)
}
```

After:

```go
res, err := Validate(input)
require.NoError(t, err)
require.Len(t, res.Items, 3)

_, err = Validate(bad)
require.ErrorContains(t, err, "timeout")
```

## 4. Setup that looks dead but is not (§3)

```go
func TestListActive_excludesArchived(t *testing.T) {
    repo.Insert(Project{Name: "live"})
    repo.Insert(Project{Name: "old", ArchivedAt: epoch})

    got := repo.ListActive()
    require.Len(t, got, 1)
    require.Equal(t, "live", got[0].Name)
}
```

No assertion names `"old"` or `ArchivedAt`, and deleting the second `Insert` leaves the test green. It is still the whole point of the case: without it, the test no longer proves that archived projects are excluded. Keep it.

## 5. Renames (§6)

| Before | After |
|---|---|
| `TestParse2` | `TestParse_emptyInputReturnsNil` |
| `TestEdgeCase` | `TestDecode_truncatedInputReturnsError` |
| `{name: "case1"}` | `{name: "negative duration returns error"}` |

Before each one: grep the repository for the old name, check whether the test boots a bed (the seed changes) or matches a snapshot (the entry is orphaned), and follow the §6 steps.
