<!-- lora-design:v3 -->

# Device

Scope: `device/`

## Motivation

Companies in Moldova run LoRaWAN pulse modules on water and gas meters. Their own network server
delivers each uplink, but the payload is bytes in a vendor format that the company cannot read.
Each vendor encodes the pulse count and the alarm flags differently. The companies need one place
that receives the uplinks from their network server and decodes them. That place keeps the
history of each device for years. It also keeps the data of each company apart from the data of
every other company. One admin operates the platform and writes the decoders. So a new vendor
format must not need work from the companies.

## Solution

Each organization connects its ChirpStack or TTN network server through an integration. The
integration has its own secret, which the network server sends in a request header. The platform
refuses a request without a valid secret. It takes the organization of an uplink from the
integration, and finds the device by DevEUI only in that organization. If no device matches, the
uplink is kept as an unmatched uplink for 30 days. The users of the organization see a warning
that names its DevEUI. When the organization adds the device, the device gets the unmatched
uplinks.

Users add devices one at a time or from a CSV file, and sort them into device groups. A device
leaves the list by archive, never by delete, so its history stays. The DevEUI of a device is
fixed after its first uplink, because the DevEUI then identifies that history.

The platform stores the payload of every uplink as it arrives, with or without a decoder.
Decoders are Go code in the application, and all organizations share them. Device models are
private to each organization. The admin assigns a decoder to each device model. When a device
model gets a decoder, the platform decodes the stored payloads of its devices again. A decoded
payload gives a pulse count and alarm flags. The alarm flags of consecutive uplinks make alarms:
an alarm is raised by the first uplink with its flag and cleared by the first decoded uplink
without it.

## Primary goals

- **G1** — An uplink shows only on a device of the organization whose integration sent it. When
  two organizations each have a device with the same DevEUI, each sees only the uplinks of its
  own integration.
  Need: [safety] multi-tenancy: an uplink on the wrong device shows one company's meter data to
  another company. [user] confirmed.
  Evidence: none
- **G2** — A request without a valid integration secret adds no uplink to any organization.
  Need: [safety] the ingest address is public, so anyone who knows it can send false pulse counts
  or alarms. [user] confirmed.
  Evidence: none
- **G3** — The users of an organization see only the devices, device models, device groups,
  uplinks, and alarms of that organization.
  Need: [safety] multi-tenancy. [user] confirmed.
  Evidence: none
- **G4** — Two organizations can each add a device with the same DevEUI. In one organization, a
  DevEUI names at most one device.
  Need: [user] confirmed: a module can go to a second company, which adds it as its own device.
  Evidence: none
- **G5** — An organization connects a ChirpStack v4 or a TTN network server with only the
  integration settings of that network server, and with no proxy or script between them.
  Need: [user] confirmed: the client companies run ChirpStack or TTN.
  Evidence: none
- **G6** — The payload of each accepted uplink of a device stays readable on the device page for
  at least 10 years, with or without a decoder for its device model.
  Need: [user] confirmed the 10-year history.
  Evidence: none
- **G7** — After the admin assigns a decoder to a device model, the pages of its devices show
  pulse counts and alarms also for the uplinks received before the assignment.
  Need: [user] confirmed: decoders arrive after the devices.
  Evidence: none
- **G8** — An uplink received up to 30 days before its organization adds the device shows in the
  history of that device.
  Need: [user] confirmed: a company registers a device in its network server before it adds the
  device here.
  Evidence: none
- **G9** — While an organization has unmatched uplinks, its users see a warning that names each
  DevEUI that sent them.
  Need: [user] confirmed: the user must know to add the device before the 30 days end.
  Evidence: none
- **G10** — The device page shows the pulse count of each decoded uplink as a chart and as a
  table.
  Need: [user] confirmed: the pulse count is what the client company reads.
  Evidence: none
- **G11** — The alarm table of a device shows one row for each period in which the device
  reported an alarm flag of one alarm kind. The row shows the time the alarm was raised and,
  after it is cleared, the time it was cleared.
  Need: [user] confirmed: a flag repeated in many uplinks is one alarm.
  Evidence: none
- **G12** — A user reads each alarm kind in Romanian or Russian, whatever the vendor of the
  device.
  Need: [contract] `PRODUCT.md` promises the UI in Romanian and Russian.
  Evidence: none
- **G13** — A user puts a device in zero or more device groups, whatever its device model, and
  filters the device list by device group.
  Need: [user] confirmed.
  Evidence: none
- **G14** — The admin sees the device models of all organizations, sees which of them have a
  decoder, and assigns a decoder to a device model.
  Need: [user] confirmed: the admin decides which decoder to write next.
  Evidence: none
- **G15** — A user adds a new device model while adding a device, with no wait for the admin.
  Need: [user] confirmed.
  Evidence: none
- **G17** — A user adds up to 5,000 devices from one CSV file, sees the result of each row
  before anything is created, and then creates them in one step.
  Need: [user] confirmed: an organization has hundreds to thousands of devices.
  Evidence: none
- **G18** — A user finds one device among thousands by its name, DevEUI, or address, and
  filters the device list by device group, device model, meter type, and archived state.
  Need: [user] confirmed the fleet size.
  Evidence: none
- **G19** — An archived device leaves the device list, keeps its history, and shows that history
  again when a user restores it.
  Need: [user] confirmed archive in place of delete.
  Evidence: none
- **G20** — A CSV row gives the meter type in English, Romanian, or Russian: `water`, `apă`, and
  `вода` give the same meter type, and so do `gas`, `gaz`, and `газ`.
  Need: [user] confirmed.
  Evidence: none
- **G21** — A user corrects the DevEUI of a device that has no uplink yet.
  Need: [user] confirmed: a DevEUI typed wrong is a common error at setup.
  Evidence: none

## Secondary goals

- **G16** — When a network server sends the same uplink two times, with the same DevEUI and
  frame counter, the history of the device holds it one time.
  Need: [user] confirmed as wanted, not as required.
  Loss: the uplink table shows a row two times, and the pulse chart shows a repeated point.
  Measure: none; a duplicate is a row that is present or not, not a magnitude.
  Evidence: none

## Non-goals

- **N1** [boundary] — Radio network work: gateways, device keys, joins, and the removal of the
  copies of one uplink that several gateways receive.
  Owner: the network server of the organization (ChirpStack or TTN).
- **N2** [boundary] — Organizations, users, logins, and profiles.
  Owner: the core context (`core/CONTEXT.md`).
- **N3** [excluded] — A device that moves to another organization with its history.
  Reopens when: an organization asks to give its devices, with their history, to another
  organization.
- **N4** [excluded] — Alarms that the platform computes itself, for example "no uplink for N
  hours".
  Reopens when: users ask to know that a device stopped sending uplinks.
- **N5** [excluded] — The conversion of a pulse count to a volume in m³ or litres.
  Reopens when: an organization needs consumption figures, not pulse counts.
- **N6** [excluded] — Decoders that users write or upload at run time.
  Reopens when: device models wait for a decoder longer than the time between two releases.
- **N7** [excluded] — Notifications (SMS, email, push) about alarms.
  Reopens when: users ask to know about an alarm without opening the platform.
- **N8** [no guarantee] — A correct chart when a pulse count resets or overflows.
- **N9** [no guarantee] — The port and the radio metadata (RSSI, SNR) of an uplink.
- **N10** [no guarantee] — Unmatched uplinks older than 30 days.
- **N11** [excluded] — The deletion of a device with its history.
  Reopens when: an organization must remove the data of a device, for example on a legal
  request.
- **N12** [no guarantee] — A CSV import that changes the devices already present.

## Frozen contracts

- **F1** — The ChirpStack v4 HTTP integration uplink event in JSON encoding: a POST with the
  query `event=up`, whose body holds `deviceInfo.devEui`, `fCnt`, `data` (the payload in base64),
  and `time`.
  Frozen by: ChirpStack, which owns the format; the platform only reads it.
  A break costs: the uplinks of each organization on ChirpStack stop arriving.
  Evidence: none
- **F2** — The TTN (The Things Stack v3) webhook uplink message: `end_device_ids.dev_eui`,
  `uplink_message.f_cnt`, `uplink_message.frm_payload` (the payload in base64), and
  `received_at`.
  Frozen by: The Things Industries, which owns the format; the platform only reads it.
  A break costs: the uplinks of each organization on TTN stop arriving.
  Evidence: none

- **F3** — The device CSV file: the header names `dev_eui`, `name`, `vendor`, `model`, `type`,
  `address`, `note`, `groups`; the `type` values `water`, `gas`, `apă`, `apa`, `gaz`, `вода`,
  `газ`; `;` between group names; `,` or `;` between cells.
  Frozen by: the files that client companies keep and fill in again for each import.
  A break costs: a kept file is refused, or its rows lose a field, with no change on the
  client side.
  Evidence: `device/import_test.go:TestParseCSV`

## Decisions

- **D1 — Take the organization from the integration, not from the DevEUI** — reopens when one
  integration must deliver the uplinks of more than one organization.
- **D2 — Send the integration secret in a request header, not in the URL** — reopens when an
  organization must use a network server that cannot send a custom header.
- **D3 — Share decoders, and let the admin assign one to each private device model** — reopens
  when device models wait days for a decoder that already exists.
- **D4 — Keep unmatched uplinks for 30 days** — reopens when organizations often add a device
  more than 30 days after its first uplink.

### D1 — Take the organization from the integration, not from the DevEUI

Serves: G1, G4
Trades: none

**Decision.** The platform takes the organization of an uplink from the integration that sent
it, and finds the device by DevEUI only in that organization, because a DevEUI is unique in one
organization only.

**Alternatives.** *One DevEUI across the platform, found without the integration.* It buys one
shared ingest address and a lookup by DevEUI alone. It lost because a DevEUI that one
organization adds then blocks each other organization with the same module.

**Consequences.** Each organization configures its own integration in its network server. A
network server that serves several organizations through one integration cannot feed the
platform: it needs one integration for each organization.

*Upheld by* `device/CONTEXT.md` (an integration gives each uplink its organization; a DevEUI
identifies a device in one organization).

**What would reopen it.** A network server that the platform must support delivers the uplinks
of more than one organization through one integration.

### D2 — Send the integration secret in a request header, not in the URL

Serves: G2, G5
Trades: none

**Decision.** The network server sends the integration secret in a request header, because a
secret in the URL is written in plain text to the access logs of each proxy and server on the
path.

**Alternatives.** *The secret in the ingest URL.* It buys a setup with one field, and it works
with each network server. It lost because the URL, and the secret in it, goes to access logs
that the platform does not control.

**Consequences.** A setup has one more field to fill in. A network server that cannot send a
custom header cannot connect; ChirpStack and TTN both can.

*Upheld by* `PRODUCT.md` (the MVP scope states the header secret).

**What would reopen it.** An organization must use a network server that cannot send a custom
request header.

### D3 — Share decoders, and let the admin assign one to each private device model

Serves: G3, G7, G14, G15
Trades: none

**Decision.** Decoders are Go code that all organizations share, and the admin assigns one to
each device model, because device models are private to each organization but one vendor format
serves many organizations.

**Alternatives.** *One shared list of device models, each with its decoder.* It buys no
assignment step: a device model gets its decoder one time for all organizations. It lost because
each organization then sees the device models of the others. *One decoder for each device model
of each organization.* It buys decoders with no sharing. It lost because the same vendor format
is then written again for each organization.

**Consequences.** Each new device model of each organization waits for the admin to assign a
decoder, also when the decoder already exists. Until then, its devices show only uplinks.

*Upheld by* `device/CONTEXT.md` (a device model belongs to one organization; many device models
can use one decoder).

**What would reopen it.** Device models wait days for a decoder that already exists, because
organizations add device models faster than the admin assigns decoders.

### D4 — Keep unmatched uplinks for 30 days

Serves: G8, G9
Trades: none

**Decision.** The platform keeps an unmatched uplink for 30 days, because a company often adds a
device in its network server before it adds the device here, and the warning gives its users time
to add it.

**Alternatives.** *Drop unmatched uplinks.* It buys no storage and no warning for unknown
devices. It lost because a device added after its first uplinks then has no early history. *Keep
them for 10 years, as the uplinks of a device.* It buys the full history for a device added at
any time. It lost because test devices that are never added then hold storage for 10 years.

**Consequences.** A device added more than 30 days after its first uplink loses the older
uplinks.

*Upheld by* `device/CONTEXT.md` (an unmatched uplink is kept for 30 days).

**What would reopen it.** Organizations often add a device more than 30 days after its first
uplink, even with the warning.
