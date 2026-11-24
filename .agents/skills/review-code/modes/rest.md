# Rest mode

The scope is code as it stands now: a package, a directory, a set of files, or a topic that crosses them. Every defect in scope counts, however old it is.

## What to read

Read the whole scope, its documents, its tests, and enough of its consumers to know how its surface is used (core step 1).

For a topic, first list the files that carry it (`grep -rln`), and state that list in the coverage. The topic is the scope, not the directories the files sit in.

## Large scopes

"Read everything" does not scale past a few thousand non-test lines, and a pretended full read is worse than an honest partial one. Count first (`wc -l`). Above roughly 5,000 non-test lines, read the documents, the public surface and its doc comments in full, then prioritize:

- the entry points,
- every validator and normalizer,
- every file that starts a goroutine, a timer or a listener, or owns a resource,
- every file touched by recent churn,
- code no test reaches.

Record what you did not read; the close-out states it.

## What counts

Every surviving finding in scope, ranked by severity (core step 6). A defect you find outside the scope while tracing a consumer is reported under its own heading after the findings, with its site, and is not ranked with them.
