# Plans, models, and effort

Use plans to constrain the shape of a change, effort levels to tune how widely the agent thinks, and
model choice to match the kind of work.

## Specs and plans

A spec describes what the change should do and why; a plan breaks it into concrete implementation
steps.

You can use [superpowers](https://github.com/obra/superpowers) to write a spec and a plan. Or you
can use the built-in `/plan` command, which is a quicker and lighter variant: no spec, and the plan
is smaller.

With superpowers, you spend more time reviewing the spec and plan, saving time on follow-up prompts
to fix issues in the generated code.

Without superpowers — and with lighter plans in general — you use more prompts to reach the same
result, but writing those prompts and fixing the issues along the way forces you to read the code
and learn it. You also end up with a better grasp of the trade-offs and the reasons behind the
chosen architecture.

End-to-end time is roughly the same — the difference is where you spend it: up front reviewing a
spec, or along the way reading the code. When in doubt, plain `/plan` is the best default: quick to
set up, keeps you close to the code, and works well for the majority of tasks.

## Effort levels

Many agents and LLM tools expose an effort level, usually shaped as low / medium / high, and
sometimes max. Effort is a budget for how much internal search and checking the model does before it
answers or edits. Higher effort does not make the model more intelligent, expand its context window,
or give it more facts. It makes the agent slower and more deliberate: more alternatives considered,
more edge cases checked, and a better chance of noticing contradictions.

Think of model choice as depth and effort as breadth. A stronger model can represent harder
problems and subtler trade-offs. Effort controls how widely that model searches before committing:
how many designs, failure modes, objections, and verification paths it considers.

Use **high** by default. It is the right setting for normal feature work, docs, tests, and
well-scoped refactors where the repo already gives the agent enough constraints.

Use **low** for bounded mechanical work: renames, formatting, small copy edits, obvious one-file
fixes, command output, or changes where the expected result is already clear. Low effort is also
useful when you want fast iteration and expect to steer the agent after each small step.

Use **high** when the expensive part is thinking, not typing: debugging a subtle failure, reviewing
for correctness, designing an API, planning a cross-cutting refactor, security-sensitive changes,
concurrency, data migrations, or anything where the first plausible solution is likely to be wrong.
High effort helps most when paired with good constraints: failing tests, logs, invariants,
benchmarks, a spec, or the relevant `CONTEXT.md`.

Use **max** only when the cost of missing one edge case is high and the output is mostly analysis:
architecture reviews, incident postmortems, security reviews, deep bug diagnosis, or auditing a
package for correctness. Avoid max effort for routine implementation. Agents at max effort tend to
write extra code for imagined corner cases, which is usually overkill and can make the result harder
to review.

Higher effort is not always better. For design work, high effort can produce an overly complex
architecture because the agent spends more time accounting for possible futures; good designs are
usually simple and grounded in current constraints. For implementation, high effort often writes
more verbose code and takes longer. When the task is exploratory, multiple lower-effort passes can
produce a better end result, because the first pass surfaces hidden facts and constraints that make
the next pass better informed.

## Claude vs Codex

Observations based on Claude Opus 4.8 and Codex 5.5 — model behavior shifts every few months, so
re-evaluate when a new release lands.

Claude (Opus 4.8) trades correctness for readability: it produces clean, idiomatic code but makes
more mistakes and cuts corners. It is better at design and frontend work, and a bit better at
guessing what you want.

Codex (5.5) trades readability for correctness: its code is verbose but right more often. It is more
thorough and more conservative about changing existing behavior, so it is less likely to introduce
subtle bugs.

Use Claude for design, frontend, and exploratory work. Reach for Codex when correctness matters more
than elegance — backend logic, refactors with invariants to preserve, anything safety-sensitive.

## Essential plugins

- [Context7](https://context7.com/) — fetches current library and framework docs on demand, so the
  agent stops inventing API signatures and flags that don't exist.

## Not recommended: rtk

This page used to recommend [rtk](https://github.com/rtk-ai/rtk), a CLI proxy that filters the
output of common commands (`git status`, `git diff`, `ls`) before it reaches the context, on the
strength of its advertised 60-90% token savings. Don't install it — and if you followed the earlier
advice, remove it.

A [JetBrains benchmark](https://blog.jetbrains.com/ai/2026/07/rtk-claude-code-token-savings/)
measured the opposite of the claim: Claude Code with rtk cost **7.6% more** at low reasoning effort
(p=0.004) and broke even at high effort. Task quality was unchanged. Two reasons the savings don't
materialize: Claude Code's built-in file-reading tools bypass the bash hook entirely, so rtk only
ever sees about 20% of tool-result characters, and the agent already truncates long results, so most
of what rtk compresses would never have been billed.

The general lesson is worth keeping past this particular tool: a tool's self-reported savings are a
claim about its counterfactual, not about your bill. rtk's reported "96.2 million tokens saved"
counts uncompressed output as the baseline, which is not the baseline you would actually pay.
