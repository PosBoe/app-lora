---
name: feature-design
description: Design a product feature or run a gap analysis on an existing one — interview the user to pin down scope and intent, analyze the relevant existing code and adjacent features, then, acting as a product designer, deliver a goal-driven feature set (or a ranked gap list) where every feature composes with what exists today and leaves room to grow as the product does. Designs only — it never edits code; the approved design is the input to a follow-up planning or implementation prompt. For restructuring existing code without changing the product, use plan-ideal-refactor or its Go/Vue variants instead; for re-deriving how an existing purpose is served — goals kept, contracts free to break — use redesign. Invoke ONLY when the user explicitly runs `/feature-design` or explicitly names this skill — do not infer it from generic phrasing like "design a feature", "what's missing here", "improve this feature", or "what should we build next" alone; treat those as ordinary requests instead.
---

# feature-design

Design a feature (or a set of features) for this product, or audit an existing feature for gaps. The deliverable is a product design: explicit goals, a coherent feature set that serves them, and for each feature its user-facing behavior, how it composes with what already exists, and how it can grow later. This skill wears the product designer's hat, not the implementer's — it reasons from user problems and product coherence first, and treats the codebase as evidence of what the product *is* today, not as a constraint on what it should become.

The skill has two modes, chosen in step 0:

- **Design mode** — the user wants a new feature or capability designed.
- **Gap mode** — the user has an existing feature and wants its gaps identified: missing capabilities, unfinished edges, places where it falls short of what users would expect given the rest of the product.

Both modes share the same spine: understand the intent → learn the existing product → set goals → design against them. This skill designs only — it changes no code, and it does not produce an implementation plan (file layouts, migration steps); when the user approves the design, that is a follow-up task, typically `/plan-ideal-refactor` or a direct implementation prompt.

## Step 0 — Understand the ask (interview first)

Do not start reading code before this step is done. The single most common failure mode of feature design is solving a well-analyzed version of the wrong problem — front-load the questions that would invalidate the work.

First, resolve from the invocation argument what you can: which feature area, and whether this is design mode or gap mode. Then interview the user with `AskUserQuestion`. Always establish the following before proceeding — ask about whichever of these the argument didn't already settle:

- **The problem, not the solution** — what user pain, request, or opportunity motivates this? If the user leads with a solution ("add X"), ask what X is for; the design may land somewhere better than X.
- **Who it's for** — which kind of user (new evaluator, daily operator, admin, API/MCP consumer) and in which deployment or plan, when the product has more than one. A feature great for one audience is often noise for another.
- **Scope boundaries** — what is explicitly out of bounds, and how big the ambition is: a rounding-out of one screen, or a new capability area.
- **Success** — what the user would look at afterwards to say it worked (adoption of a screen, fewer support questions, parity with a competitor, an internal need served). This feeds the goals in step 2; goals invented without this anchor are decoration.
- **Constraints that shape the design** — deadlines, pricing/plan gating, compatibility promises, or "must work with feature Y" requirements the code can't reveal.

Batch these into one or two `AskUserQuestion` calls with concrete options and your recommended reading first — not an open-ended questionnaire. Skip only the questions the invocation already answered explicitly; when in doubt whether it did, ask. Later steps may surface new questions (a fork that turns on product intent, a gap whose severity only the user can judge) — ask those inline as they arise, same tool, rather than saving them for the end.

## Step 1 — Analyze the existing product

Now learn what exists, in two layers — the product layer first, the code layer to verify it:

1. **Map the feature terrain.** `CONTEXT-MAP.md` for which bounded contexts the area touches, and each relevant context's `CONTEXT.md` for the domain vocabulary — the design must speak the product's existing language, not invent parallel terms. `PRODUCT.md` states the users and the purpose; read it first. `git log` is the product's feature history: read the entries around this area to see what was recently shipped, what direction the product is already moving, and what has already been tried.
2. **Read the feature as a user meets it.** For UI-facing areas: the Vue routes, pages, and stores involved (`vue/src/`), and the endpoints they call. For API/config/MCP-facing areas: the handlers, the config surface, `mcp/` tools. Walk the main user journey end to end and note where it dead-ends, requires manual work, or silently lacks an affordance that sibling features have.
3. **Read the adjacent features**, not just the target. Composition is the point: the features this design must play well with (alerting on it? dashboards over it? project/org scoping? billing gating?) each get enough of a read to know their extension points and conventions. A feature that duplicates an adjacent feature's half-solved version of the same job is a design bug — prefer extending the existing mechanism.
4. **Note the variant dimensions** — build tags (`pkg/featflag/`), config switches, plans. A design that only makes sense in one variant must say so.
5. **In gap mode, also collect the comparison baseline**: what the rest of the product offers in analogous places (every list has filtering except this one; every entity has MCP tools except this one), since "gap" is always relative to an expectation the product itself sets.

Collect, don't design yet. Keep notes in a scratch file for large areas; record concrete evidence (`file:line`, route paths, endpoint names) — the design document will cite it.

## Step 2 — Define the goals

Before any feature is proposed, write the goals the design serves — this is the contract everything after must answer to. Derive them from the step 0 interview (the user's problem and success definition) and the step 1 analysis (what the product evidently needs). For each goal:

- One sentence, stated as a user or product outcome — "an operator can go from alert to root cause without leaving the incident view" — never as a feature in disguise ("add a button").
- A way to tell it's been met: observable behavior, a measurable proxy, or an explicit judgment call the user signed off on.
- Numbered G1…Gn so features can cite them.

Also write the **non-goals**: the plausible adjacent ambitions this design deliberately does not serve. Non-goals are what keep the feature set coherent — without them every review reopens scope.

If the goals you derive diverge from what the user seemed to want in step 0, surface that with `AskUserQuestion` now — goals are the user's to own, and everything downstream hangs on them.

## Step 3 — Design the feature set / identify the gaps

Now put on the product designer's hat and design against the goals.

**In design mode**, produce a coherent feature set — usually 2–6 features that together serve the goals, not one monolith and not a brainstorm list. For each feature:

- **What the user gets** — the behavior described from the user's side: what they see, do, and can now accomplish. Concrete enough to sketch the screen or the API call; no implementation detail.
- **Goals served** — the G-numbers it advances. A feature citing no goal gets cut, however appealing — it is scope creep with good posture.
- **Composition** — named integration with existing features: which existing mechanism it extends, which conventions it follows (scoping, permissions, filtering idioms, MCP exposure), and what it deliberately reuses instead of duplicating. "Plays well with existing features" is demonstrated here per feature, not asserted globally.
- **Growth path** — one or two sentences on how it extends as the product grows: what dimension is expected to expand (more entity types, more channels, more granularity) and how the design absorbs that without redesign. Design the seam, not the speculative features behind it — extensibility means the shape doesn't box the future in, not that v1 ships half-built generality.
- **Slicing** — if the feature is big, the smallest version that already serves its goals, and what the follow-up increments would be.

Rank the set: which features are the core (the goals fail without them) and which are supporting. If the user must choose a subset, the ranking is what they choose from.

**In gap mode**, produce a ranked gap list. Each gap:

- **The gap** — what a user reasonably expects and doesn't get, with the step 1 evidence (the dead-end, the missing affordance, the baseline feature it lacks relative to siblings).
- **Who hits it and how often** — tied to the audience from step 0; a gap nobody hits is not a gap.
- **Goals served by closing it** — the G-numbers.
- **The fix, sketched** — the feature-level shape of the remedy (same fields as design mode, compressed), including its composition with what exists.
- **Severity** — ranked by user impact against the goals, not by implementation ease.

**Discipline for both modes:**

- Design from the user journey, not the data model. If a feature is easiest to state as "expose table X in the UI", restate it as what the user is trying to do; if it can't be restated, it fails.
- One product language: names come from the `CONTEXT.md` glossaries; a new concept gets a name that will survive being said aloud in the UI, the API, and the docs.
- Coherence beats coverage. A smaller set that composes is better than a longer list that serves every goal a little.
- When two designs genuinely fork on product intent — which audience wins, whether to extend an existing feature or introduce a sibling — ask the user inline (`AskUserQuestion`, recommended option first). Pure design-craft calls (naming, slicing, which existing mechanism to extend when one is clearly idiomatic) you resolve yourself and record.
- Note, without designing around it, anything discovered in step 1 that makes a feature disproportionately expensive (a persisted format, a missing seam) — the user weighs cost, but only if the design tells them it exists.

## Constraints

- Design only: no code edits, no migration plans, no file layouts. Interface sketches (a screen described in words, an example API shape) belong in the design; implementation structure does not.
- The design must be buildable in this product: it names the existing features and mechanisms it composes with, and it does not assume capabilities (new infra, new dependencies) the product lacks — or it flags that assumption explicitly as a cost.
- Deployment honesty: every feature states which deployment or plan it applies to, when the product has more than one.

## Self-check before writing the output

- Step 0 questions were actually asked — the design cites the user's answers, not assumptions about them.
- Every goal is an outcome with a way to tell it's met; non-goals exist.
- Every feature/gap cites at least one goal; anything citing none was cut.
- Every feature names the existing mechanism it composes with — no feature is an island, and none silently duplicates an adjacent feature's job.
- Every feature has a growth path that is a seam, not a speculative roadmap.
- The set is ranked, and sliced where big.
- Forks on product intent were asked inline, not guessed.

## Output to the user

Deliver the design as the response, using this structure (gap mode swaps section 3's body for the ranked gap list):

```
# Feature design: <area>            (or: # Gap analysis: <feature>)

## 0. Understanding
<the problem, audience, scope bounds, and success definition — as established with the user, citing their answers>

## 1. Existing product
<what exists today in this area and adjacent to it, with evidence; in gap mode, the baseline the rest of the product sets>

## 2. Goals
G1 — <outcome>. Met when: <observable check>.
…
Non-goals: <the ambitions deliberately not served>

## 3. Feature set                    (or: ## 3. Gaps, ranked)
### F1 <name> — core|supporting
What the user gets: …
Goals: G…
Composes with: <existing features/mechanisms, by name>
Grows by: <the seam>
Slices: <smallest useful version → increments>       (when big)
Deployment/plan: <which>  (when live)
…

## 4. Recommended cut
<which features (or gap fixes) to build first and why, in one short paragraph>
```

End with the open questions that remain the user's to decide. Anything that would change the goals or the core feature set should already have been asked inline — this list is for the low-stakes remainder. If there are none, say so.
