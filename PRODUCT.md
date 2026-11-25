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
  assigns a decoder to each device model. Uses user profiles to collect feedback. Reads the
  data of each organization to help its users, and changes none of it.

## Product purpose

- **Ingest.** Receive uplinks through integrations from ChirpStack and TTN. The integration
  gives the organization. Find the device by its DevEUI in that organization. Always store the
  payload. Keep an unmatched uplink for 30 days, so a device added later gets its earlier
  history. Show the users of the organization a warning that names each DevEUI with unmatched
  uplinks.
- **Decode.** Decode the payload with the decoder of the device model. Get the pulse count and
  the alarm flags. If the device model has no decoder, keep the payload only. When a decoder is
  added, decode the stored history again.
- **My devices.** A list of the devices of the organization, paged on the server for thousands
  of devices. Search by name, DevEUI, or address. Filter by device group, device model, meter
  type, and archived devices. Select many devices to add them to a group, remove them from a
  group, or archive them. A device can be in zero or more device groups, whatever its device
  model.
- **Device page.** Device model, DevEUI, name, meter type, address, note, device groups, last
  uplink time. A chart and a table of the
  reported pulse count. A table of alarms, each with the time it was raised and cleared. A
  table of uplinks with their payloads. Until the device model has a decoder, the page shows
  the uplinks only. Each empty section says why it is empty.
- **Add a device.** Give DevEUI, name, and device model. Meter type (water or gas), address, and
  note are optional. Pick a device model of the organization or add a new one. The DevEUI must be
  unique in the organization only. The DevEUI can change only while the device has no uplink.
- **Import devices.** Upload a CSV file with up to 5,000 rows. A preview shows each row as new,
  already present, or wrong, with the reason. The import creates the missing device models and
  device groups. It adds devices only; it does not change present devices.
- **Archive a device.** An archived device leaves the list and keeps its history. A user can
  restore it. Its new uplinks are unmatched uplinks.
- **Device models.** A user sees the device models of the organization with their device count,
  renames a device model, and deletes a device model that no device uses. A device model is a
  vendor and a model name.
- **Device groups.** A user creates, renames, and deletes device groups. A deleted group leaves
  its devices in place.
- **Device models (admin).** See the device models of all organizations, with the organization,
  the device count, and the decoder. Assign a decoder to a device model.
- **Integrations.** A user adds any number of integrations, each for ChirpStack or TTN, with no
  help from the admin. The platform shows the ingest URL and the secret one time, with the setup
  steps for that network server. The integration page shows the last uplink, the uplinks of the
  last 24 hours, and the last format error.
- **Organizations (admin).** Create an organization and its user with a login. The platform
  shows a temporary password one time. Reset the password of a user. Deactivate and reactivate
  an organization. See each organization with its user profile, last login, device count,
  integration count, and last uplink.
- **Admin view.** Open an organization and see its pages as its users see them, read-only.
- **Profile.** The user fills in and edits their profile: first name, last name, email, phone
  number in the `+373` format, and UI language. The user changes their own password there.
  Until the profile is complete, a banner asks the user to fill it in.

## MVP scope

- One organization is one client company. It has users. In the MVP, the admin creates one user
  for each organization. No roles inside an organization.
- An organization sees only its own devices, device models, device groups, and uplinks. Only
  the admin sees all organizations, and only to read.
- Login with login and password. The admin makes the credentials. No self-registration and no
  password reset by email.
- A user must replace a temporary password at the next login, before doing anything else.
- The users of a deactivated organization cannot log in. Its integrations still deliver
  uplinks.
- The first admin is created with a command on the server. The UI creates no admins.
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
- No single sign-on, no two-factor login, and no roles inside an organization.
- No admin changes to the data of an organization, and no deletion of an organization in the UI.
- No markets outside Moldova and no languages other than Romanian and Russian.

## Design register

Quiet and dense. The user reads data tables and charts, so data comes first. Few colors: use a
strong color only for alarms. `docs/frontend-ui-direction.md` holds the design system that
follows from it.

## Open questions

- None.
