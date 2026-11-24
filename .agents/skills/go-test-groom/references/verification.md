# Shared verification protocol for the `go-test-*` suite

Every skill in the suite baselines a package, edits its tests, and then proves it
did not lose anything. That protocol is written here once. A skill cites this
file instead of restating it, so the recipes cannot drift between skills.

`/go-test-groom` hosts this file because it is the suite's hub; the protocol is
not groom-specific and applies to every sibling run standalone.

## 1. Scratch paths

Baseline artifacts are namespaced per package, never bare `/tmp/cover.before.out`
— two packages in one session, or two sessions at once, would silently overwrite
each other's proof-of-no-loss.

```
/tmp/gotest-skills/<pkg-slug>/<artifact>
```

`<pkg-slug>` is the package path with `/` replaced by `-`: `./system`
→ `system`. Create the directory first (`mkdir -p`). Artifacts:

| Artifact | What it is |
|---|---|
| `cover.<phase>.<tag>.out` | coverage profile — `<phase>` is `before`/`after` (or the pipeline phase name), `<tag>` is `default`/`invariants` |
| `tests.<phase>.<tag>.txt` | the `--- PASS/FAIL` case inventory |
| `slow.<phase>.txt` | per-test timings (`/go-test-speedup`) |
| `time.<phase>.<tag>.txt` | the suite's elapsed time in both modes, with the run's own output and exit status — see §3 |

When `/go-test-groom` drives the pipeline it names phases (`baseline`,
`reshape`, `decouple`, `coverage`, `cleanup`, `determinism`, `speedup`) so each
phase's "after" is the next phase's "before" without re-running anything.

This table is the one register of artifact names. Add a name here before a
skill writes it, so no two skills spell the same artifact differently.

Two words in this file mean different things, and the timing rules turn on
both. A **lane** is a build tag: the default build or `-tags=invariants`, per §2. A **mode** is
whether the command passes `-short`, per §3. An artifact names its lane in the
file name and its modes inside the file.

## 2. Baseline both build tags when the package reads them

The repo has one deployment build. The `invariants` tag (`pkg/featflag`) flips
`featflag.Invariants`, which turns a broken invariant into a panic in place of
a graceful error. `task dev` sets it; `task test` does not. A package that has
build-tagged files or reads `featflag` therefore behaves differently in the two
lanes, and a default-only baseline:

- reports invariants-only branches as uncovered, which reads as "dead code" to
  `/go-test-coverage` and as "safe to delete" to `/go-test-speedup`;
- omits `//go:build invariants` test functions from the case inventory, so
  `/go-test-reshape`'s case accounting cannot see them move or vanish.

Decide once, at baseline time:

```
grep -rln --include='*.go' 'go:build'  <pkg>/  # tagged files?
grep -rn  --include='*.go' 'featflag\.' <pkg>/ # flag-dependent behavior?
```

Pass the **directory**, not `<pkg>/*.go`: that glob expands to the files directly
under `<pkg>` and leaves `-r` nothing to recurse into.

Either hit → baseline both lanes. No hits → the default lane alone is honest;
say so in the report rather than leaving it implicit.

Pass `-p 1` whenever the scope can hold more than one integration package; §4
says why, and `go test` does not add it for you.

```
mkdir -p /tmp/gotest-skills/<pkg-slug>

go test -p 1                   -coverprofile=/tmp/gotest-skills/<pkg-slug>/cover.before.default.out    ./<pkg>/...
go test -p 1 -tags=invariants  -coverprofile=/tmp/gotest-skills/<pkg-slug>/cover.before.invariants.out ./<pkg>/...

inventory() {  # inventory <output-file> [extra go test flags...]
  local out=$1; shift
  : > "$out.raw"
  : > "$out"
  local incomplete=0 pkg pkg_raw run_status
  # One package per run, and each one validated on its own. A package that fails
  # to build, or panics before it reports a case, is silent in the -v stream, so
  # a sibling's genuine --- FAIL would otherwise satisfy a run-wide check for it.
  # Not `status`: zsh holds that name read-only, so `local status=$?` is an error.
  for pkg in $(go list "$@" ./<pkg>/...); do
    pkg_raw="$out.$(echo "$pkg" | tr / _).raw"
    go test -v "$@" "$pkg" > "$pkg_raw" 2>&1
    run_status=$?
    cat "$pkg_raw" >> "$out.raw"
    grep -E '^--- (PASS|FAIL)' "$pkg_raw" >> "$out"
    if [ $run_status -ne 0 ] && ! grep -q '^--- FAIL' "$pkg_raw"; then
      echo "INCOMPLETE: $pkg exited $run_status with no failing case — see $pkg_raw" >&2
      incomplete=1
    fi
  done
  return $incomplete
}

inventory /tmp/gotest-skills/<pkg-slug>/tests.before.default.txt
inventory /tmp/gotest-skills/<pkg-slug>/tests.before.invariants.txt -tags=invariants
```

Keep the raw output and check `go test`'s own status, because a **package** can
fail without producing a single `--- FAIL` line. A build error prints
`FAIL <pkg> [build failed]`, and a panic in `TestMain` or an fx `OnStart`
prints a stack trace — neither matches the grep, so that package's whole case
set leaves the inventory in silence while the pipe still exits zero on some
other package's lines. §6 then diffs two lists that never held those cases and
reports nothing lost.

**Ask the question per package, not per run.** The status and the `--- FAIL`
lines belong to different packages once the scope holds more than one, so a run
where package A fails a case and package B fails to build satisfies any run-wide
pairing of the two — and B's whole case set is dropped in silence, which is the
shape the guard exists to catch. Running one package at a time also gives `-p 1`
for free.

A nonzero status **with** `--- FAIL` lines *from that same package* is fine and
expected — a failing case still records itself, and §6 diffs case lists rather
than reading a status. It is the silent shape the guard is for.

Coverage profiles from different tag sets are not mergeable — diff each lane
against its own baseline, never default-after against invariants-before.

## 3. Know what mode your baseline is in

A bare `go test` does **not** pass `-short`, but every default lane does
(`task test`, `task test:pkg`, `task test:go`). So a raw baseline includes `testing.Short()`-gated stress
and soak tests that CI never runs.

- **Coverage baselines**: run without `-short`, before and after alike. The extra
  coverage is real, and you want the widest picture before deciding a branch is
  untested. Because both profiles are taken this way, a test newly gated behind
  `testing.Short()` still runs for coverage, and the two numbers still compare.
- **Timing baselines** (`/go-test-speedup`): measure **both modes**. The
  unqualified number is what the nightly full suite pays; the `-short` number is
  what every developer and every push actually pays. Report both, and say which
  one the budget was judged against.

### The timing recipe

`go test` prints one elapsed figure per package and never a total, and a pipe
into `tail` reports the pipe's exit status rather than the run's. So a timing
artifact built that way can report a pass for a failed suite, and its last lines
are one package's time rather than the suite's. Measure the elapsed time around
the command, and keep the whole output:

```
run() {  # run <mode-label> <output-file> [extra go test flags...]
  local label=$1 out=$2; shift 2
  local start=$(date +%s)
  go test -p 1 -count=1 "$@" ./<pkg>/... >> "$out" 2>&1
  local status=$? end=$(date +%s)
  echo "## $label: exit $status, $((end - start))s elapsed" >> "$out"
}

: > /tmp/gotest-skills/<pkg-slug>/time.<phase>.default.txt
run unqualified /tmp/gotest-skills/<pkg-slug>/time.<phase>.default.txt
run -short      /tmp/gotest-skills/<pkg-slug>/time.<phase>.default.txt -short
```

Run the same two commands with `-tags=invariants`, writing
`time.<phase>.invariants.txt`, whenever §2 says the package needs that lane.

A non-zero exit means the artifact holds a failed run. Fix the failure before
you compare any number in it: a suite that stops early looks fast.

## 4. Integration packages need the database up

A package whose tests dial PostgreSQL is an integration package. Its tests — and
therefore every baseline and `-rewrite` run in this suite — need the live test
database:

```
task up          # PostgreSQL in docker compose
task test:setup  # creates config/test.yml and the migrated lora_test database
                 # (destructive to lora_test by design)
```

Two consequences for this suite:

- **Serialize.** Packages share the one test database, so `-p 1` is required
  for correctness. Put it in the command yourself whenever the scope can expand
  to more than one integration package:

  ```
  go test -p 1 ./<pkg>/...
  ```
- **A failure may not be yours.** Another process on the shared test database
  produces failures that look like your edit. Before you diagnose a baseline
  mismatch, confirm nothing else is running against it and re-run.

## 5. The gate list

Three gates. Use the cheapest one that covers the change; do not reach for the
last gate by reflex.

| Gate | Command | When |
|---|---|---|
| 1 | `go test -race -count=1 ./<pkg>/...` (add `-p 1` for an integration scope, per §4) | after every edit batch, and after each pipeline phase. `-count=1` forces a fresh test-binary build, which is what catches helpers only the deleted tests used — plain `go build` does not, since `*_test.go` is excluded from non-test builds |
| 2 | `task test:pkg -- ./<pkg>/...`, plus `go test -short -tags=invariants ./<pkg>/...` when §2 says the package needs that lane | the finalize gate for a single-package change |
| 3 | `task test` | pre-PR only, or when the change touched shared test helpers, fixtures, or anything outside the package |

Gate 2 is the default finalize for every skill in this suite. `/go-test-groom`
runs gate 3 once at the end of the whole pipeline, never per phase.

Compile-checking is `go vet ./...` (and `go vet -tags=invariants ./...`): it
compiles the test files, which `go build` does not. A skill that edits only
`*_test.go` does not need even this: gate 1 already compiles the test binaries.

## 6. Proving nothing was lost

### Coverage must not drop

```
diff <(go tool cover -func=/tmp/gotest-skills/<pkg-slug>/cover.before.default.out) \
     <(go tool cover -func=/tmp/gotest-skills/<pkg-slug>/cover.after.default.out)
```

...and the same for `invariants` when both lanes were baselined. A function whose
coverage dropped means a case was lost — grep the deleted tests for that
function and recreate the case in the new shape.

Sanctioned exceptions, each of which must be named in the report with its
reason: `/go-test-decouple` removing a test that asserted nothing observable,
and `/go-test-speedup` deleting a redundant test within its stated tolerance.

### Case accounting

For **splits and moves**, leaf case names survive and only the owning test
function changes:

```
sed 's|--- \(PASS\|FAIL\): Test[^/]*/|--- \1: |' <tests.before.<tag>.txt> | sort > /tmp/gotest-skills/<pkg-slug>/leaves.before.<tag>.txt
sed 's|--- \(PASS\|FAIL\): Test[^/]*/|--- \1: |' <tests.after.<tag>.txt>  | sort > /tmp/gotest-skills/<pkg-slug>/leaves.after.<tag>.txt
diff /tmp/gotest-skills/<pkg-slug>/leaves.before.<tag>.txt \
     /tmp/gotest-skills/<pkg-slug>/leaves.after.<tag>.txt
```

For **consolidations**, old test names legitimately become case names, so leaf
names change by design. Verify counts instead (cases in the new shape ≥ tests +
cases deleted) and lean on the coverage diff.

## 7. Regenerating expectations

`-rewrite` materializes *current behavior* as "expected".
They will happily encode a bug.

```
go test ./<pkg>/... -rewrite       # flag AFTER the package list;
                                   # `go test -rewrite ./...` silently tests nothing
```

- Only packages that use datadriven accept `-rewrite`.
- It regenerates by *running* the tests, so an integration package needs the
  databases up (§4) like any other run.
- Review every rewritten line as an assertion you are **authoring**, not output
  you are accepting. If a golden surprises you, the code, the case, or the
  renderer is wrong — never accept it to move on.
- Some corpora cannot be rewritten offline at all, because they need the live
  database. Check the package's `README.md` / `AGENTS.md` before assuming.

A skill that regenerates goldens has manufactured exactly the conditions
`/go-test-lockin` exists to audit. Say so in the report and name the goldens
worth a human read.
