---
name: draft-followup-prompt
description: Turn the unresolved items of the current conversation — found-but-not-fixed defects, deferred decisions, skipped tests, follow-up work — into one self-contained prompt the user pastes into a fresh agent. Prints the prompt and changes no file. For compacting a whole conversation so a next session continues the same work, use `handoff` instead.
---

# draft-followup-prompt

Write one prompt that starts a fresh agent on the work this session left open. Print it. Do not edit a file, and do not start the work.

## Source

The current conversation is the only source. Use what this session found, decided, deferred, or reported — the Debrief items, the "found, not fixed" notes, the tests you skipped, the questions the user parked.

Do not invent an item. Do not add work the session did not surface. If the user gave an argument, keep only the open items that match it.

Read a file again when you must copy an exact path, symbol, or command into the prompt. Do not start a new investigation.

## What counts as open

Keep an item when all of these are true:

- The session found it, and the session did not fix it.
- A fresh agent can act on it.
- It is not already recorded where the next agent will read it anyway — an existing branch commit, a TODO comment in the diff, an open plan file. Name that record instead of restating its content.

Drop an item that the session fixed, refuted, or the user declined. State a refusal only when a reader would otherwise redo the refuted work.

## The prompt

The fresh agent has no memory of this session. Every fact it needs is in the prompt.

Give it, in this order:

1. **Where the work is** — the branch, and whether the tree is clean.
2. **One block per open item.** Each block states: what is wrong or missing, the file path and symbol, why the session left it, and what "done" is.
3. **What not to redo** — a refuted idea, a design fork the user already settled, an approach that failed.
4. **How to verify** — the exact command, with its flags, from `CLAUDE.md`.
5. **Which skill to call**, when one fits an item.

Rank the items by severity, worst first. Say when items are independent, so the agent can do them in any order.

Write the prompt in ASD-STE100 Simplified Technical English, as `CLAUDE.md` asks. Use the imperative mood. Keep one instruction per sentence.

## Delivery

Print the prompt in one fenced block, and nothing else inside that block. Add one line under the block that says how many items it holds.

If no item is open, say so in one line. Do not pad the prompt to fill it.
