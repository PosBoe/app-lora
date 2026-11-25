# device

The device registry of an organization: devices and device models. `CONTEXT.md` defines the
terms, and `DESIGN.md` states the goals and the decisions.

| File                | Content                                                                 |
| ------------------- | ----------------------------------------------------------------------- |
| `module.go`         | `Module`: provides the services, the handlers, and `Stats`.             |
| `guard.go`          | The `guard` seam, `member`, and the organization of the caller.         |
| `fields.go`         | The DevEUI rule, the meter types, the field rules, the search pattern.  |
| `device_service.go` | Create, update, list, archive, and restore devices.                     |
| `model_service.go`  | Create, list, rename, and delete device models.                         |
| `group_service.go`  | Create, list, rename, and delete device groups; add and remove devices. |
| `import.go`         | The CSV parser: delimiter, BOM, header, and the rules of each row.      |
| `import_service.go` | The import plan, the preview, and the commit in one transaction.        |
| `device_handler.go` | `/api/v1/devices*`.                                                     |
| `model_handler.go`  | `/api/v1/device-models*`.                                               |
| `group_handler.go`  | `/api/v1/device-groups*`.                                               |
| `import_handler.go` | `/api/v1/devices/import*` and the template file.                        |
| `integration_*.go`  | Integrations: secrets, status, `/api/v1/integrations*`.                 |
| `ingest_*.go`       | `/api/v1/ingest/*`: the public routes, the ChirpStack and TTN parsers.  |
| `uplink_*.go`       | Uplinks: store, attach, the device table, `/api/v1/unmatched`.          |
| `purge.go`          | The hourly delete of unmatched uplinks older than 30 days.              |
| `stats.go`          | `Stats`: the device count of each organization, for the admin.          |
| `errors.go`         | The rule errors and their HTTP status.                                  |
| `*_repo.go`         | The SQL of the device, device model, and device group tables.           |

## The organization of a request

Every route goes through `guard.Member`. In production the guard is `*core.Guard`: it lets in
only a member of an organization and puts a `core.Caller` in the request context. A route reads
the organization from that caller, never from the request. Every query of a repo takes the
organization, so a device or a device model of another organization answers 404.

A test uses a guard that puts a caller in the context with `core.ContextWithCaller`.

## DevEUI

- A DevEUI is stored as 16 lower-case hex characters. A request can write it in upper case, and
  with `:`, `-`, or spaces between the bytes.
- A DevEUI is unique in one organization. An archived device keeps its DevEUI, so a new device
  with it answers `dev_eui_archived`, and the user can restore the archived device.
- An update changes the DevEUI only while the device has no uplink (`hasUplinks`). The ingestion
  of uplinks makes this check real.

## Device models

A device uses a device model of its own organization. The foreign key
`(organization_id, device_model_id)` holds this rule in the database. A device model that a
device uses, also an archived device, cannot be deleted.

## Device groups

A device is in zero or more device groups, whatever its device model. The membership table has
the keys `(organization_id, group_id)` and `(organization_id, device_id)`, so a membership never
joins a device and a group of two organizations. A create or an update of a device sets its
groups in the same transaction. A bulk add or remove ignores the ids of other organizations,
but a group of another organization answers 404. A deleted group leaves its devices.

## CSV import

The CSV format is a contract with the files that users keep (`DESIGN.md` F3). Do not rename a
column or remove an accepted value.

- The header row names the columns, in any order and any case: `dev_eui`, `name`, `vendor`,
  `model` (required), and `type`, `address`, `note`, `groups` (optional). An unknown or repeated
  column refuses the file.
- `type` is `water` or `gas`, also written `apă`, `apa`, `gaz`, `вода`, or `газ`, in any case.
  An empty cell is no type.
- `groups` holds group names split by `;`. Quote the cell when the delimiter is `;`.
- The delimiter is `,` or `;`; the header row decides. A UTF-8 BOM is allowed. A file that is
  not UTF-8 is refused: Excel must save it as "CSV UTF-8".
- At most 5,000 data rows and 2 MiB. Blank rows are skipped.
- Each row has the rules of a single device. A DevEUI that an earlier row has is an error.
- A row whose DevEUI the organization has, archived devices too, is skipped with its groups.
  The import adds devices; it never changes one.

`POST /api/v1/devices/import/preview` plans and changes nothing. `POST /api/v1/devices/import`
plans again and creates the new rows, the missing device models, and the missing groups in one
transaction. Both take the file as a text/csv body, which an HTML form cannot send.

## Ingestion

- A network server posts to `/api/v1/ingest/chirpstack` or `/api/v1/ingest/ttn` with
  `Authorization: Bearer <secret>`. The secret names the integration, and the integration names
  the organization (`DESIGN.md` D1, D2). The routes take no session.
- A secret is `lora_` and 32 random bytes in base64url. The table holds only its SHA-256, so the
  secret shows one time. A new secret ends the old one at once.
- An unknown secret answers 401 and is not recorded: it names no organization. A request that
  ingestion cannot read (the wrong route, Protobuf from ChirpStack, bad JSON, a bad DevEUI or
  payload) answers 400 and becomes the last error of the integration, which its page shows. A
  message that is not an uplink answers 204, so the network server does not send it again. A
  failed store answers 500, so the network server can send it again.
- An uplink lands on the live device with its DevEUI in the organization. With no such device,
  or with an archived one, it is unmatched. The unique key `(organization_id, dev_eui, f_cnt,
  received_at)` keeps one copy of an uplink that a network server sends again.
- A create, an import, a restore, and a DevEUI change give the device the unmatched uplinks
  with its DevEUI, in the same transaction. The purge deletes unmatched uplinks 30 days after
  they arrive. The uplinks of a device are never deleted.
- The DevEUI of a device with an uplink cannot change (`hasUplinks`).
- `testdata/` holds sample bodies of both network servers. They are the evidence of the frozen
  formats F1 and F2.

## Search

`devices.search` is a stored lower-case join of the name, the DevEUI, and the address, with a
`pg_trgm` index. A search text matches any part of it. A search text that is a DevEUI with
separators loses the separators first.

## Tests

The tests use the test database through `pkg/pgtest`, so they skip under `-short`. Run them
with `task test:full`, or with `LORA_CONFIG` set to the absolute path of `config/test.yml`.
