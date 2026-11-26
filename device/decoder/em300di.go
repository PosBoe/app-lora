package decoder

import "encoding/binary"

// lowBatteryPercent is the battery level at or below which the EM300-DI
// raises a low battery alarm.
const lowBatteryPercent = 10

func init() {
	register(em300DI{})
}

// em300DI decodes the Milesight EM300-DI pulse counter. The payload is a
// list of channels: a channel id byte, a channel type byte, and a value whose
// length the pair sets. The layout follows the Milesight codec in the TTN
// device repository.
type em300DI struct{}

func (em300DI) ID() string   { return "milesight_em300_di" }
func (em300DI) Name() string { return "Milesight EM300-DI" }

// em300DIChannels gives the value length of each channel id and type.
var em300DIChannels = map[[2]byte]int{
	{0x01, 0x75}: 1,  // battery, %
	{0x03, 0x67}: 2,  // temperature, int16 LE / 10
	{0x04, 0x68}: 1,  // humidity, / 2
	{0x05, 0x00}: 1,  // GPIO state
	{0x05, 0xC8}: 4,  // pulse count, uint32 LE
	{0x20, 0xCE}: 13, // history record
}

// Decode reads the battery and the live pulse count. An unknown channel
// ends the walk, as the Milesight codec does, and keeps what came before it.
// A history record carries an old reading; the live channel is the reading
// of the uplink, so history records are skipped.
func (em300DI) Decode(payload []byte) (Result, error) {
	var r Result
	for i := 0; i+2 <= len(payload); {
		key := [2]byte{payload[i], payload[i+1]}
		n, ok := em300DIChannels[key]
		if !ok {
			break
		}
		value := payload[i+2:]
		if len(value) < n {
			return Result{}, ErrMalformed
		}
		switch key {
		case [2]byte{0x01, 0x75}:
			r.report(LowBattery, value[0] <= lowBatteryPercent)
		case [2]byte{0x05, 0xC8}:
			count := int64(binary.LittleEndian.Uint32(value))
			r.PulseCount = &count
		}
		i += 2 + n
	}
	return r, nil
}
