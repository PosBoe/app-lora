# Product

A platform for companies in Moldova that run LoRaWAN pulse modules on water and gas meters.
The platform receives each uplink, decodes the payload by the device model, and keeps the
history: pulse counts, alarms, and uplinks.

The words in this file have the meanings that the glossaries give them:
[Core](core/CONTEXT.md) and [Device](device/CONTEXT.md).

## Users

- **Organization (client company).** A water or gas utility, or a building operator, in the
  Republic of Moldova. It has LoRaWAN pulse modules mounted on water or gas meters. It runs its
  own network server (ChirpStack or TTN). It knows LoRaWAN, but it cannot decode each vendor's
  payload format. Its users use the UI in Romanian or Russian.
- **Admin (platform operator).** Creates organizations and their users. Writes the decoders and
  assigns a decoder to each device model. Uses user profiles to collect feedback.

## Product purpose

- **Ingest.** Receive uplinks through integrations from ChirpStack and TTN. The integration
  gives the organization. Find the device by its DevEUI in that organization. Always store the
  payload. Keep an unmatched uplink for 30 days, so a device added later gets its earlier
  history. Show the users of the organization a warning that names each DevEUI with unmatched
  uplinks.
- **Decode.** Decode the payload with the decoder of the device model. Get the pulse count and
  the alarm flags. If the device model has no decoder, keep the payload only. When a decoder is
  added, decode the stored history again.
- **My devices.** A list of the devices of the organization. Filter by device group. A device
  can be in zero or more device groups, whatever its device model.
- **Device page.** Device model, DevEUI, name, last uplink time. A chart and a table of the
  reported pulse count. A table of alarms, each with the time it was raised and cleared. A
  table of uplinks with their payloads. Until the device model has a decoder, the page shows
  the uplinks only.
- **Add a device.** Give DevEUI, name, and device model. Pick a device model of the
  organization or add a new one. The DevEUI must be unique in the organization only.
- **Device models (admin).** See the device models of all organizations, and which of them have
  a decoder. Assign a decoder to a device model.
- **Organizations (admin).** Create an organization and its user with login and password.
- **Profile.** The user fills in and edits their profile: first name, last name, email, and
  phone number in the `+373` format.

## MVP scope

- One organization is one client company. It has users. In the MVP, the admin creates one user
  for each organization. No roles inside an organization.
- An organization sees only its own devices, device models, device groups, and uplinks. Only
  the admin sees all organizations.
- Login with login and password. The admin makes the credentials. No self-registration and no
  password reset by email.
- Decoders are Go code in the application. A new decoder ships with a new release.
- Alarms come only from alarm flags in the decoded payload. An alarm is raised by the first
  uplink with its flag and cleared by the first decoded uplink without it.
- The platform shows the raw pulse count. It does not convert pulses to m³ or litres.
- A pulse count does not reset and does not overflow. It only increases.
- Each integration has its own secret. The network server sends the secret in a request
  header. A request without a valid secret is refused.
- A device sends one to three uplinks per day.
- The platform keeps the uplinks of each device for at least 10 years. The MVP deletes no
  uplinks.
- The platform keeps only the DevEUI, the payload, the frame counter, and the receive time of an
  uplink. It does not keep the port or the radio metadata (RSSI, SNR).
- A device stays in the organization that added it. It cannot move to another organization.
- UI in Romanian and Russian. Times shown in Europe/Chisinau. Times stored in UTC.

## Non-goals

- The platform is not a LoRaWAN network server. It does not manage gateways or device keys.
- No consumption in m³, no billing, no tariffs.
- No alarms that the platform computes itself (for example "no data for N hours").
- No notifications (SMS, email, push).
- No decoders written by users or uploaded at runtime.
- No data export and no public API.
- No markets outside Moldova and no languages other than Romanian and Russian.

## Design register

Quiet and dense. The user reads data tables and charts, so data comes first. Few colors: use a
strong color only for alarms. `docs/frontend-ui-direction.md` holds the design system that
follows from it.

## Open questions

- None.
