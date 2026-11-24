# Resolve a scope and its binding set

`/design-doc` and `/redesign` both start here. This file holds the procedure. What each
relation in the binding set means is in [DESIGN-FORMAT.md](./DESIGN-FORMAT.md) § Scope. A run
that binds fewer documents than this procedure finds has silently freed goals.

## Resolve the scope to files

The invocation argument takes one of three shapes:

- **Named files or a directory** (`system/`, `pkg/appconf/`). The scope is those
  files, plus enough surroundings to understand them in place.
- **A subsystem or feature area** ("the notification pipeline", "saved views"). Map it to
  concrete files first, and state the file list at the top of the output.
- **A broad topic**, possibly cross-stack ("how annotations get from ingest to the chart").
  Grep the key terms, check `CONTEXT-MAP.md` for the bounded contexts it touches, and follow
  the data from producer through storage or wire to consumer. Write down the file list as the
  scope.

**Ask when the scope has two readings.** If no argument was given, or the argument leads to
two different file sets, ask the user before you read code. A one-word topic is the usual
case: "slo" matches a package, its model package, and the whole feature across evaluation
and monitors. Narrowing silently to the package with the matching name leaves unloaded the
documents that bind the real work.

- To state the readings, list file and directory names and grep the `Scope:` lines. Read
  nothing else first.
- Name the file set each reading reaches and the documents that bind it.
- Mark no reading Recommended unless the user's words point at one.

The read that follows can find more packages that the chosen reading reaches. Add them
without asking again, and list them in the output as added. A package the reading names joins
the scope. A package that only consumes it does not join.

## Find the binding documents

A `DESIGN.md` binds what its `Scope:` line names, wherever the file sits. A walk up or down
the directories misses a document that binds across the tree. One read finds every candidate:

```sh
grep -rn '^Scope:' --include='*.md' .
```

Keep each document whose `Scope:` paths touch a file of the scope. Give each one its relation
as DESIGN-FORMAT.md § Scope defines it: anchor, ancestor, descendant or overlapping.

**Pick the anchor by fit.** The anchor is the document whose paths hold the whole scope and
fit it most closely. When a scope is larger than every document, there is no anchor: each
document it reaches is a descendant, the best fit included. Say so in the output. Do not
promote the best fit to anchor, because that binds absolutely the goals of the mechanism the
user asked to change.

**Record the coverage.** Split the scope into the **covered** files, which some `Scope:` line
names, and the **uncovered** remainder. Both skills print that split.

**A file without the marker is a defect.** A `DESIGN.md` that does not open with
`<!-- lora-design:v3 -->` is not a legal older shape. Report it, and read it as evidence
only. Its goals bind nothing until `/design-doc` brings it to the shape.
