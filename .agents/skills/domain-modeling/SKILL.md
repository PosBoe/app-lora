---
name: domain-modeling
description: Build and sharpen a project's domain model. Invoke ONLY when the user explicitly runs `/domain-modeling` or explicitly names this skill, or when another skill's instructions explicitly direct invoking it — do not infer it from generic phrasing like "pin down terminology" or "record an architectural decision" alone; treat those as ordinary requests instead.
---

# Domain Modeling

Actively build and sharpen the project's domain model as you design. This is the *active* discipline — challenging terms, inventing edge-case scenarios, and writing the glossary and decisions down the moment they crystallise. (Merely *reading* `CONTEXT.md` for vocabulary is not this skill — that's a one-line habit any skill can do. This skill is for when you're changing the model, not just consuming it.)

## File structure

Two kinds of file hold the model, and they never mix. `CONTEXT.md` holds the vocabulary. `DESIGN.md` holds the decisions, as records under its `## Decisions` heading. There is no `docs/adr/`.

Most repos have a single context:

```
/
├── CONTEXT.md                        ← the glossary
├── DESIGN.md                         ← Scope, goals, and the decision records
└── src/
```

If a `CONTEXT-MAP.md` exists at the root, the repo has multiple contexts. The map points to where each one lives:

```
/
├── CONTEXT-MAP.md
├── docs/design/
│   └── monitor-resolution.md         ← a decision whose scope crosses contexts
├── src/
│   ├── ordering/
│   │   ├── CONTEXT.md
│   │   └── DESIGN.md                 ← decisions scoped to ordering
│   └── billing/
│       ├── CONTEXT.md
│       └── DESIGN.md
```

A design document binds what its `Scope:` line names, never what its directory holds, so find the one that covers a decision with one read — `grep -rn '^Scope:' --include='*.md' .` — and not by walking directories.

Create files lazily — only when you have something to write. If no `CONTEXT.md` exists, create one when the first term is resolved. A `DESIGN.md` is different: a record needs a document that already carries the marker, the `Scope:` line and the goal tiers, and writing those is `/design-doc`'s work. Where no document covers the scope, run `/design-doc` on it first, then add the record. [`design-doc/DESIGN-FORMAT.md`](../design-doc/DESIGN-FORMAT.md) holds the file's format.

## During the session

### Challenge against the glossary

When the user uses a term that conflicts with the existing language in `CONTEXT.md`, call it out immediately. "Your glossary defines 'cancellation' as X, but you seem to mean Y — which is it?"

### Sharpen fuzzy language

When the user uses vague or overloaded terms, propose a precise canonical term. "You're saying 'account' — do you mean the Customer or the User? Those are different things." Sharpen an overloaded word by qualifying it — `BillingAccount` against `UserAccount` — not by reaching for an unfamiliar synonym. The most common words are the most overloaded ones, so this rule points at exactly the words a reader already knows. Keep them.

### Name with the field's own word

Before you propose a term, find what the field already calls the thing: the standard, the textbook, the widely used library, the product's UI, this repo elsewhere. Propose that word. Coin only when no field has named the concept, and say in the glossary that you coined it. [CONTEXT-FORMAT.md](./CONTEXT-FORMAT.md) holds the rules; this is when to apply them.

Two failures cost the most, and both look clever:

- **Coining** — you mint a rare or abstract word for a concept that already has a name. One repo glossary defines `Worthiness` as "the admissibility check" a detector runs before training. The definition reached for the plainer word, so the plainer word was the better headword: *eligibility*.
- **Redefining** — you take a word the industry already owns and give it a private meaning. The same glossary calls a control run an `axis sentinel`. Everywhere else `sentinel` means a marker value, so the reader does not know they are lost. *Control* was free and correct.

A rare word is right when the field's own word is rare — `transductive`, `burn rate`, `contamination`. Ask "would an engineer in this field recognize it", not "is this word common in English".

### Discuss concrete scenarios

When domain relationships are being discussed, stress-test them with specific scenarios. Invent scenarios that probe edge cases and force the user to be precise about the boundaries between concepts.

### Cross-reference with code

When the user states how something works, check whether the code agrees. If you find a contradiction, surface it: "Your code cancels entire Orders, but you just said partial cancellation is possible — which is right?"

### Update CONTEXT.md inline

When a term is resolved, update `CONTEXT.md` right there. Don't batch these up — capture them as they happen. Use the format in [CONTEXT-FORMAT.md](./CONTEXT-FORMAT.md).

`CONTEXT.md` should be totally devoid of implementation details. Do not treat `CONTEXT.md` as a spec, a scratch pad, or a repository for implementation decisions. It is a glossary and nothing else.

### Offer decision records sparingly

Only offer to record a decision when all three are true:

1. **Hard to reverse** — the cost of changing your mind later is meaningful
2. **Surprising without context** — a future reader will wonder "why did they do it this way?"
3. **The result of a real trade-off** — there were genuine alternatives and you picked one for specific reasons

If any of the three is missing, skip the record. The bar is a **real fork**: a decision a future maintainer might plausibly "fix" back. A decision whose blast radius is one file or one function gets a `COMPROMISE:` comment on that code instead, never a record.

What clears the bar: architectural shape ("the write model is event-sourced, the read model is projected into Postgres"); integration patterns between contexts ("Ordering and Billing communicate via domain events, not synchronous HTTP"); technology choices that carry lock-in — the ones that would take a quarter to swap out, not every library; boundary decisions, where the explicit no-s are as valuable as the yes-s; deliberate deviations from the obvious path, which stop the next engineer from "fixing" something that was deliberate; constraints not visible in the code ("response times must be under 200ms because of the partner API contract"); and rejected alternatives whose rejection is non-obvious, or someone will propose the same thing again in six months.

A record is a `###` section under the `## Decisions` heading of a `DESIGN.md`. It carries the `Serves:` and `Trades:` goal ids, and it owes five parts, one time each and no other: the Decision, the Alternatives, the Consequences, the *Upheld by* list, and the condition that would reopen it. Add its line to the index under `## Decisions` in the same edit.

Write every new record to the v3 rules — a new record must not add to what the trim has to cut:

- **One decision.** If two claims can reopen on two different observations, write two records.
- **The next unused `D` id** before the title. A deleted record frees no id. The title states the decision, not the topic.
- **Decision opens with the choice and its decisive reason**: "{X}, because {Y}."
- **Each alternative gets its name, what it buys, and the one reason it lost.** State the decisive factor only.
- **Consequences names one thing that becomes harder.** When you can name no cost, there was no fork: skip the record.
- **The trigger is an observation**, never "nothing" and never "if requirements change".
- **No date, no commit hash, no story of how the design came to be.** State the present design.
- **Proof goes to the evidence file.** A record that states a measured figure carries an `Evidence:` line, and the table, the benchstat output and the rejected experiments go in a dated entry of the scope's evidence file, under the record's id.
- **60 lines at most.** Operative detail — steps, specs, file maps — belongs to `README.md` or `doc.go`. [`design-doc/DESIGN-FORMAT.md`](../design-doc/DESIGN-FORMAT.md) holds the full format and the rules the repo test checks; this is when to apply them.
