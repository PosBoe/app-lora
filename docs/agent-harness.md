# Agent harness

Tests, documentation, and domain language form the harness generated code runs inside. They tell the
agent what the code is supposed to do, what conventions matter, and which words mean something
specific in this codebase.

## Tests as constraints

Tests become especially important with agents because they constrain the generated code. If you have
good tests, an agent will write good code; add good architecture on top and it will optimize as
needed. Even with only tests in place, you can ask the agent to try different architectures and pick
the best one.

With agents, tests change less often than the code: the implementation underneath churns as the
agent rewrites and refactors, but the assertions stay put. That also flips the usual reading pattern
— tests get read more than the code, both by you when checking what an agent did and by the next
agent figuring out what the code is supposed to do. Optimize them for reading, not just for running.

See [writing-good-tests-ai-era.md](writing-good-tests-ai-era.md).

## Documentation as constraints

Tests are not the only way to constrain an agent. Documentation — `README.md`, `AGENTS.md`,
`CLAUDE.md`, and package-level docs — works alongside tests to tell the agent what the code is
supposed to do and which conventions to follow.

Tests pin down behavior; documentation pins down intent, conventions, and the shape of the
surrounding code. Together they keep the agent from drifting into plausible-looking but wrong
solutions.

Keep these files honest: outdated docs mislead agents the same way they mislead humans, except
agents will confidently produce code based on the stale claims.

See [maintaining-agents-md.md](maintaining-agents-md.md) for keeping `AGENTS.md` current.

## Domain language as constraints

This repo also uses domain glossaries to keep agents and humans aligned on the words the codebase
uses. The root [CONTEXT-MAP.md](../CONTEXT-MAP.md) lists the bounded contexts and links to each
modeled context's `CONTEXT.md`.

A `CONTEXT.md` file is a glossary, not a spec or implementation guide. Use it to define domain
terms, pick one canonical name when several words are possible, and list words to avoid when they
would blur an important distinction. Keep implementation details, planned behavior, and design
decisions out of it; those belong in code, specs, package docs, or a `DESIGN.md` decision record.

Use the local `domain-modeling` skill when the vocabulary itself is part of the work: naming a new
concept, resolving an overloaded term, recording a ubiquitous-language decision, or creating the
first `CONTEXT.md` for a context. The skill should challenge fuzzy language against the existing
glossary, cross-check claims against the code, and update the relevant `CONTEXT.md` as terms are
settled instead of leaving the decision buried in chat.
