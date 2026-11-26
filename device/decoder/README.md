# decoder

The payload decoders of the device models. A decoder is Go code that all organizations share,
and the admin assigns one to a device model (`device/DESIGN.md` D3).

| File          | Content                                                               |
| ------------- | --------------------------------------------------------------------- |
| `decoder.go`  | `Decoder`, `Result`, the alarm kinds, and the registry.               |
| `em300di.go`  | The Milesight EM300-DI pulse counter.                                 |

## Rules

- `Decoder.ID` is persisted in `device_models.decoder` and `uplinks.decoder`. Never change an
  ID. A new payload format gets a new decoder with a new ID.
- A decoder is a pure function of the payload. It reads no database and no clock.
- `Result.Reported` lists the alarm kinds that a payload tells about, and `Result.Raised` the
  ones whose flag is on. An alarm of a kind that a payload does not report stays as it is.
- An `AlarmKind` value is persisted, and the frontend names it in `vue/src/lib/i18n`. A new kind
  needs a message in Romanian and Russian.

## Add a decoder

1. Write the decoder in a new file, with `register` in its `init`.
2. Test it with payloads built from the vendor's documentation, and name that source in a
   comment.
3. The admin then sees it in the decoder list of the device models page.

## EM300-DI

The layout follows the Milesight codec in the TTN device repository: channel id, channel type,
value. The decoder reads the battery (`0x01 0x75`, %) and the live pulse count (`0x05 0xC8`,
uint32 little-endian). It skips the other known channels, including history records
(`0x20 0xCE`), and stops at an unknown channel. A battery at 10 % or less raises low battery.
Milesight notes that a later firmware changes the pulse definition: check new firmware against
real uplinks.
