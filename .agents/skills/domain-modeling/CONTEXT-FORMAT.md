# CONTEXT.md Format

## Structure

```md
# {Context Name}

{One or two sentence description of what this context is and why it exists.}

## Language

**Order**:
{A one or two sentence description of the term}
_Avoid_: Purchase, transaction

**Invoice**:
A request for payment sent to a customer after delivery.
_Avoid_: Bill, payment request

**Customer**:
A person or organization that places orders.
_Avoid_: Client, buyer, account
```

## Rules

- **Be opinionated.** When several words fit one concept, pick one and list the rest under `_Avoid_`.
- **Pick the word the field already uses.** The best term is the one a competent engineer in this field already knows — from the standard, the textbook, the widely used library, the product's own UI, or this repo elsewhere. Find that word before you pick. If you cannot name a source for a term, you are coining it.
- **Do not coin, and do not redefine.** Coin only when the concept is new here and no field has named it. Then say so in the definition, and keep the word plain. Never take a word the industry already owns and give it a private meaning: a reader who knows `sentinel`, `ledger` or `broker` will not know they are lost.
- **A rare word is right when the field's own word is rare.** `transductive`, `contamination`, `burn rate` and `EVT` all stay. The test is not "is this word common in English". The test is "would an engineer in this field recognize it".
- **The definition test.** If the definition reaches for a plainer word to explain the headword, that plainer word is the better headword.
- **`_Avoid_` lists the less precise synonyms, not the recognized term.** If the field's own word must still be avoided here, the definition says why.
- **Keep definitions tight.** One or two sentences max. Define what it IS, not what it does.
- **Only include terms specific to this project's context.** General programming concepts (timeouts, error types, utility patterns) don't belong even if the project uses them extensively. Before adding a term, ask: is this a concept unique to this context, or a general programming concept? Only the former belongs.
- **Group terms under subheadings** when natural clusters emerge. If all terms belong to a single cohesive area, a flat list is fine.

## Single vs multi-context repos

**Single context (most repos):** One `CONTEXT.md` at the repo root.

**Multiple contexts:** A `CONTEXT-MAP.md` at the repo root lists the contexts, where they live, and how they relate to each other:

```md
# Context Map

## Contexts

- [Ordering](./src/ordering/CONTEXT.md) — receives and tracks customer orders
- [Billing](./src/billing/CONTEXT.md) — generates invoices and processes payments
- [Fulfillment](./src/fulfillment/CONTEXT.md) — manages warehouse picking and shipping

## Relationships

- **Ordering → Fulfillment**: Ordering emits `OrderPlaced` events; Fulfillment consumes them to start picking
- **Fulfillment → Billing**: Fulfillment emits `ShipmentDispatched` events; Billing consumes them to generate invoices
- **Ordering ↔ Billing**: Shared types for `CustomerId` and `Money`
```

The skill infers which structure applies:

- If `CONTEXT-MAP.md` exists, read it to find contexts
- If only a root `CONTEXT.md` exists, single context
- If neither exists, create a root `CONTEXT.md` lazily when the first term is resolved

When multiple contexts exist, infer which one the current topic relates to. If unclear, ask.
