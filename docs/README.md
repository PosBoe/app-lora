# Project docs

Shared documentation for agent workflows and for the Go and frontend conventions.

The agent notes cover what works, and what does not, when you use coding agents such as Claude
Code and Codex on this codebase. In short: agents write code well, but they need tight
constraints, tests, documentation, and a plan.

Agents are reliably bad at two things:

- Trade-offs — every solution has one, and agents lack the context to pick the right side.
- Scoping tests — an agent will write a test that locks in an undecided behavior.

## Guides

- [Go package conventions](go-package-conventions.md) — the naming, export, and wiring rules
  every Go package follows, in one place instead of in every `AGENTS.md`.
- [Go test conventions](go-test-conventions.md) — test layout, goleak, the test database, and
  determinism.
- [Agent harness](agent-harness.md) — tests, documentation, and domain language that keep agents
  aligned with the codebase.
- [Frontend UI direction](frontend-ui-direction.md) — the design system for `vue/`: tokens,
  UnoCSS, and which component owns which concept.
- [Frontend module boundaries](frontend-module-boundaries.md) — where frontend code goes and which
  direction its imports go.
- [Plans, models, and effort](agent-plans-and-models.md) — when to use specs or plans, how to pick
  an effort level, and when to choose Claude or Codex.
- [Risk and review](agent-risk-and-review.md) — high-risk delegation areas, context hygiene, and
  review of agent-written code.
- [Writing good tests in the AI era](writing-good-tests-ai-era.md) — tests that constrain agents
  without overfitting implementation details.
- [Maintaining AGENTS.md](maintaining-agents-md.md) — keeping agent-facing project instructions
  current.

## Design documents

A design document that spans more than one package goes in [`docs/design/`](design/README.md). A
design document for one scope goes in that scope's `DESIGN.md`.
