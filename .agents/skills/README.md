# Repo Skill Naming

This directory contains the repo-scoped agent skills. `.claude/skills` is a symlink to it, so
Claude Code and Codex read the same files. Skill names are part of the
agent's selection surface, so keep them stable, short, and easy to mention.

## Hard Rules

- Use lowercase letters, digits, and hyphens only.
- Keep names under 64 characters.
- Do not start or end with a hyphen.
- Do not use consecutive hyphens.
- Match the skill directory name to the `name` field in `SKILL.md`.

## Naming Convention

Use this shape for new skills:

```text
<scope>-<action>[-object]
```

Pick the narrowest durable scope:

- `go-` for skills that only work on Go source, packages, benchmarks, or tests.
- `vue-` for skills that only work on `vue/` or Vue components.
- A product or subsystem scope, such as `system-` or `billing-`, for
  skills tied to one bounded context.
- No scope prefix for repo-wide workflows whose trigger is not technology-bound.

Prefer verb-led action names after the scope:

- Good: `review-code`, `go-optimize-bench`, `go-test-coverage`,
  `vue-split-components`, `vue-organize-components`.
- Avoid: `cleanup-go`, `components-vue-split`, vague names like `cleanup`
  or `helper`.

When several skills form a family, keep the family immediately after the scope:

```text
go-test-cleanup
go-test-coverage
go-test-datadriven
go-test-decouple
go-test-groom
go-test-reshape
```

## Triggering Guidance

Do not rely on names alone for implicit invocation. The `description` in
`SKILL.md` must say what the skill does, when to use it, and when not to use it.
Front-load important trigger phrases because Codex may shorten descriptions
when many skills are installed.

## Catalog

Workflow:

- `grill-me`, `wait-what`, `handoff`, `draft-followup-prompt` — conversation control.
- `feature-design`, `domain-modeling`, `design-doc`, `redesign`, `plan-ideal-refactor` — design.
- `review-code`, `fix-or-refute` — review.
- `trim-agents-md`, `audit-terms` — documentation upkeep.
- `run-app` — start the local stack and drive the UI.

Go: `go-test-datadriven`, `go-test-to-datadriven`, `go-test-reshape`, `go-test-decouple`, `go-test-coverage`, `go-test-cleanup`,
`go-test-groom`, `go-test-lockin`, `go-test-determinism`, `go-test-speedup`,
`go-optimize-bench`, `go-bench-coverage`.

Vue: `vue-organize-components`, `vue-split-components`.

## Origin

The skills come from the reference repository this project copied its layout from. They were
adapted to this repository: one build (no hosted/cloud split), PostgreSQL only, and no shared
testbed. `pkg/datadriven` was copied with them. When a skill names a file this repository does not hold,
treat the name as an example of the shape, not as a path.
