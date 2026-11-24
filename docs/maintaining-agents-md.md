# Keeping AGENTS.md up to date

`AGENTS.md` files are the single biggest lever for agent code quality in this repo. They tell every
agent — yours, your teammate's, CI — what conventions to follow, what mistakes to avoid, and how the
surrounding code is supposed to work. Stale or missing instructions produce the same drift you get
from stale tests, except the agent won't even know it's drifting.

**Update AGENTS.md as part of the work, not after it.** If you change a convention, add a
dependency, introduce a new pattern, or discover a gotcha that cost you time — capture it in the
relevant `AGENTS.md` before you move on. The next person (or agent) to touch that code will hit the
same issue otherwise.

## What belongs in AGENTS.md

- **Conventions and constraints** — which libraries to use (and which to avoid), naming rules, error
  handling patterns, build tags.
- **Gotchas** — non-obvious behavior, footguns, things that look wrong but are intentional.
- **Build and test commands** — how to verify the code in this area.
- **Scope boundaries** — what this directory owns and where its responsibilities end.

## What does NOT belong

- Architecture explanations or design docs — those go in `README.md` or `docs/`.
- Ephemeral notes — things that will be irrelevant after a single PR.
- Duplicated rules — if the root `AGENTS.md` already says it, don't repeat it in subdirectories.

## Using a skill to maintain AGENTS.md

Use the `trim-agents-md` skill (`.agents/skills/trim-agents-md/SKILL.md`) to audit the
`AGENTS.md` files of a scope. It moves explanation out to `README.md`, `DESIGN.md` or
`CONTEXT.md`, and it keeps only operative rules.

At the end of a work session, ask the agent which conventions the session followed or introduced,
and to propose targeted edits to the relevant `AGENTS.md`. Review the edits. Accept what is right,
and reject what is noise.

## Agents copy what they see

Agents learn conventions from the existing code as much as from `AGENTS.md`. If a bad pattern exists
in the repo, agents will reproduce it in new code — no matter what the docs say. Telling the agent
"don't do X" in `AGENTS.md` while the codebase is full of X doesn't work: the agent sees more
examples of X than the one rule against it, and pattern-matches accordingly.

When you spot a bad practice, fix the existing code first (or at least the files the agent is likely
to read as context), then update `AGENTS.md`. The rule and the code need to agree, or the code wins.

## Guidelines

- One `AGENTS.md` per area that has its own conventions (e.g. `vue/`, or a domain package
  with its own rules). Don't create one for a subdirectory
  unless it genuinely diverges from its parent.
- Keep entries short — one bullet per rule. If it takes a paragraph, it probably belongs in a
  `README.md` instead.
- Delete rules that no longer apply. A wrong rule is worse than no rule.
- When you see an agent make the same mistake twice, that's the signal to add a rule.
