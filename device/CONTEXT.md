# Device

The LoRaWAN devices of an organization and the data they report: how an uplink arrives, how it
is decoded, and the pulse counts and alarms that it gives.

## Language

### Hardware

**Device**:
A LoRaWAN end device that an organization added to the platform. In this product it is a pulse
module on a meter. The LoRaWAN specification calls it an end device.
_Avoid_: counter, contor, module, sensor, node

**DevEUI**:
The 64-bit EUI that the LoRaWAN specification gives each end device. It identifies a device in
one organization, not on the whole platform: two organizations can each have a device with the
same DevEUI.
_Avoid_: device ID, serial number

**Archived device**:
A device that its users took out of the device list. It keeps its history and can be restored.
It gets no new uplinks: an uplink with its DevEUI is an unmatched uplink.
_Avoid_: deleted device, disabled device, inactive device

**Meter**:
The water or gas meter that a device is mounted on. The meter makes the pulses. The platform
does not keep a record of meters.
_Avoid_: contor, counter (the device is not the meter)

**Meter type**:
What a device measures: water or gas. A user sets it on the device; it is optional.
_Avoid_: medium, device type, kind

**Pulse**:
One electrical pulse that a meter sends to its device for a fixed volume of water or gas.
_Avoid_: impulse, tick

**Pulse count**:
The cumulative number of pulses that a device reports in one uplink. It is not a volume.
_Avoid_: impulses, reading, index, consumption (these name a volume in m³)

**Device model**:
The vendor and the model name of a device, as one organization names it. Each organization has
its own device models; no other organization sees them. Two organizations can each have a device
model for the same hardware.
_Avoid_: device type, device profile (in ChirpStack, a device profile holds radio settings)

**Vendor**:
The company that makes a device, as a device model names it, for example Milesight.
_Avoid_: manufacturer, brand, producer

**Device group**:
A named set of devices in one organization, chosen by its users. A device is in zero or more
device groups, whatever its device model.
_Avoid_: folder, category, tag

### Data path

**Network server**:
The LoRaWAN network server that receives uplinks from gateways and sends them on to the
platform. ChirpStack and TTN are network servers.
_Avoid_: gateway (a gateway only relays radio frames to the network server), backend

**Integration**:
The link through which one network server sends the uplinks of one organization to the
platform. Each uplink gets the organization of its integration. ChirpStack calls it an HTTP
integration; TTN calls it a webhook.
_Avoid_: webhook (one kind of integration), connector, endpoint

**Uplink**:
One message from a device, as the network server sends it on: the DevEUI, the payload, the
frame counter, and the receive time.
_Avoid_: message, packet, frame, raw data, reading

**Payload**:
The application bytes of an uplink before decoding. The LoRaWAN specification calls it
FRMPayload.
_Avoid_: raw data, data, frame

**Unmatched uplink**:
An uplink whose DevEUI matches no device of its integration's organization. The platform keeps
it for 30 days. When the organization adds a device with that DevEUI in that time, the uplink
becomes an uplink of that device. This term is coined here: ChirpStack and TTN do not keep such
uplinks.
_Avoid_: orphan uplink, unknown uplink

### Decoding

**Decoder**:
The rule that reads one vendor's payload format. All organizations share the decoders. The
admin assigns zero or one decoder to each device model, and many device models can use the
same decoder. TTN calls it a payload formatter; ChirpStack calls it a codec.
_Avoid_: parser, codec, formatter, payload format

**Decoded payload**:
The values that a decoder gets from one payload: the pulse count and the alarm flags. TTN uses
the same term.
_Avoid_: object, measurement, reading

### Alarms

**Alarm flag**:
A condition that a device reports in one payload, for example low battery or tamper.
_Avoid_: alert, status bit, event

**Alarm kind**:
The condition that an alarm flag names. All decoders map vendor flags to one shared list of
alarm kinds, so the UI can name each kind in Romanian and Russian.
_Avoid_: alarm type, alarm code

**Alarm**:
One period in which a device reports an alarm flag of one alarm kind. The first uplink with the
flag raises the alarm. The first decoded uplink without the flag clears it.
_Avoid_: alert, alarm event, incident
