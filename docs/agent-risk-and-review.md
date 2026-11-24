# Risk and review

Some work is riskier to delegate to agents. Most of that risk is manageable if the prompt gives the
agent something concrete to anchor on: a spec, a test, an invariant in code, a benchmark, logs, or a
domain glossary.

## Risk areas

- **Trade-off decisions.** Every solution comes with a trade-off, and agents lack the context to pick
  the right side — they produce a plausible answer but rarely the right one. _Mitigation:_ make the
  trade-off call yourself (use the agent as a sounding board, not the decider), then hand off the
  implementation.
- **Security-critical code.** Auth, crypto, input validation at trust boundaries — plausible code
  can be subtly wrong in ways tests won't catch. _Mitigation:_ write the negative test cases
  yourself (what an attacker would try), let the agent fill in the implementation, and review in a
  separate session with security as the explicit lens.
- **Subtle invariants.** Concurrency, ordering, lock discipline — anything that depends on a
  property the code doesn't make explicit. _Mitigation:_ make the invariant explicit. Encode it as a
  comment, an assertion, a type, or a property-based test. The agent will respect what it can see
  and quietly break what it can't.
- **Large cross-cutting refactors.** The agent loses the global picture and ends up making the same
  local edit inconsistently across the codebase. _Mitigation:_ break the refactor into incremental
  commits. Give the agent one worked example as a pattern, then have it repeat the pattern file by
  file, with type checks and tests gating each step.
- **Performance-sensitive code.** Agents reach for the conventional approach, not the fast one, and
  benchmarks rarely make it into the test suite. _Mitigation:_ add benchmarks. Pin a budget —
  latency, allocations, throughput. The agent can optimize against numbers; it can't optimize
  against your intuition.
- **Out-of-repo context.** Production behavior, ops knowledge, business constraints, what a
  stakeholder actually meant. _Mitigation:_ bring the context in. Paste the logs, the stakeholder
  message, the runbook into the prompt. The agent cannot infer what isn't in front of it.

If you can't mitigate — the invariant resists being made explicit, a benchmark would be too
expensive to write, the stakeholder's intent is genuinely ambiguous — that's the signal to take the
work back yourself.

## Context hygiene

Long sessions drift. The agent loses track of the original goal, accumulates contradictory decisions
from earlier in the conversation, or starts repeating itself. Sometimes it digs in on a bad approach
and argues with you about it.

When that happens, end the session. Start a new one with a tight prompt — what you want, the
relevant constraints, and a pointer to whatever the previous session got right (a commit, a file, a
test that already passes). A fresh context is almost always faster than fighting an anchored one.

The same dynamic makes the separate-session review below work: a clean context is effectively a
different agent, even with the same model underneath.

## Self-review

Once the task is done, ask the agent to self-assess the code it wrote. Agents are good at spotting
and reporting code smells in code they just produced — often catching things you missed in review.

For anything non-trivial, follow up with a review in a **separate session**. The agent that wrote
the code is anchored to the decisions it just made: it tends to defend its own structure, gloss over
shortcuts it took, and miss issues that contradict its earlier reasoning. A fresh session has no
investment in the existing approach and will more readily flag design problems, missing edge cases,
unused code, and places where the implementation drifted from the spec.

Useful prompts for the review session:

- "Review this change for correctness, edge cases, and code smell. Be critical."
- "What's missing from these tests? What behavior is asserted that shouldn't be? What should be
  asserted but isn't?"
- "If you were rewriting this from scratch, what would you do differently and why?"

Treat the review output as a second opinion, not a verdict — the reviewing agent is just as capable
of being confidently wrong as the one that wrote the code.
