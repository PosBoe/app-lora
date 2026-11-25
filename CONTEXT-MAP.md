# Context Map

This repo is split into bounded contexts. Domain glossaries are added per context, lazily, as
they get modeled — a context without a `CONTEXT.md` below simply has not been modeled yet. A
glossary can span the stack: a frontend module that models words of its own is listed under the
context it serves.

Use the `domain-modeling` skill to model a context and write its `CONTEXT.md`. Add the context to
this map in the same change.

## Contexts

- **system** (`system/`, `vue/src/system/`) — reports on the process itself: health and version.
  No glossary: it models no domain terms.
- [Core](./core/CONTEXT.md) — who uses the platform and what they own: organizations, users,
  the admin, and user profiles.
- [Device](./device/CONTEXT.md) — the LoRaWAN devices of an organization and what they report:
  device models, decoders, integrations, uplinks, pulse counts, alarms, and device groups.

## Relationships

Write one line per relationship: the upstream context, the downstream context, and what crosses
the boundary (a call, an event, a shared table).

- **Core → Device**: each device, device model, device group, and integration belongs to one
  organization.
  Device refers to the organization by its ID. Core does not know about devices.
