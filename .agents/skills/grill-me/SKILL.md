---
name: grill-me
description: Grill the user relentlessly about a plan, decision, or idea, one round of AskUserQuestion at a time. Use when the user wants to stress-test their thinking, or uses any 'grill' trigger phrases.
---

Interview the user relentlessly until you reach a shared understanding. Map this as a **design tree**: every decision branches into the decisions that hang off it.

Work the tree in **rounds**. The **frontier** is every decision whose prerequisites are already settled — the questions you can ask _now_ without guessing at answers you haven't heard yet. Ask the frontier with the **AskUserQuestion** tool, then wait for the answers before the next round.

## How to ask a round

- Put the round's frontier questions in **one AskUserQuestion call** (the tool takes up to 4). If the frontier is wider than 4, ask the 4 that unblock the most branches, and carry the rest to the next round.
- Every question carries your **recommended answer as the first option**, with `(Recommended)` at the end of its label. The rest are the live alternatives you actually considered.
- Each option's `description` says what that choice commits the user to — the consequence, not a restatement of the label.
- Set `multiSelect: true` when the choices compose instead of competing.
- Use an option `preview` when the choice is a concrete artifact the user should compare by eye: a layout sketch, two code shapes, a config or schema variant. Previews are single-select only.
- The user can always answer "Other" with free text. Treat that answer as a new branch and reshape the tree around it.

Do not print the questions as prose as well — the tool call is the question.

## Between rounds

Each round the user answers reshapes the tree — settled decisions push the frontier outward and unblock questions that depended on them. Recompute the frontier and ask the next round.

A question whose answer depends on another question still open in this round belongs to a _later_ round, not this one.

Finding _facts_ is your job, never the user's. When a frontier question needs a fact from the environment (filesystem, tools, etc.), dispatch a sub-agent to find it — don't ask the user for anything you could look up yourself. Don't block on it: a running exploration is an unsettled prerequisite, so only the questions downstream of it wait for the sub-agent to report — ask the rest of the frontier now. The _decisions_ are the user's — put each to them and wait.

The session is done when the frontier is empty: every branch of the design tree visited, nothing left silently assumed. Then state the settled design back in plain text. Do not act on it until the user confirms you have reached a shared understanding.
