package decoder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEM300DI(t *testing.T) {
	d, ok := Get("milesight_em300_di")
	require.True(t, ok)

	cases := []struct {
		name     string
		payload  []byte
		pulse    *int64
		reported []AlarmKind
		raised   []AlarmKind
	}{
		{
			name: "battery, temperature, humidity, pulse",
			payload: []byte{
				0x01, 0x75, 0x5C, // battery 92 %
				0x03, 0x67, 0x0E, 0x01, // 27.0 °C
				0x04, 0x68, 0x64, // 50 %RH
				0x05, 0xC8, 0x39, 0x30, 0x00, 0x00, // 12345 pulses
			},
			pulse:    ptr(12345),
			reported: []AlarmKind{LowBattery},
		},
		{
			name:     "battery at the threshold raises low battery",
			payload:  []byte{0x01, 0x75, 0x0A},
			reported: []AlarmKind{LowBattery},
			raised:   []AlarmKind{LowBattery},
		},
		{
			name:    "pulse only reports no alarm kind",
			payload: []byte{0x05, 0xC8, 0xFF, 0xFF, 0xFF, 0xFF},
			pulse:   ptr(4294967295),
		},
		{
			name: "a history record is skipped",
			payload: []byte{
				0x20, 0xCE, 0x10, 0x20, 0x30, 0x40, 0x0E, 0x01, 0x64, 0x02, 0x00, 0x01, 0x00, 0x00, 0x00,
				0x05, 0xC8, 0x02, 0x00, 0x00, 0x00,
			},
			pulse: ptr(2),
		},
		{
			name:     "an unknown channel ends the walk",
			payload:  []byte{0x01, 0x75, 0x50, 0xFF, 0x01, 0x05, 0xC8, 0x01, 0x00, 0x00, 0x00},
			reported: []AlarmKind{LowBattery},
		},
		{name: "empty", payload: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := d.Decode(tc.payload)
			require.NoError(t, err)
			assert.Equal(t, tc.pulse, got.PulseCount)
			assert.Equal(t, tc.reported, got.Reported)
			assert.Equal(t, tc.raised, got.Raised)
		})
	}

	t.Run("a truncated value is malformed", func(t *testing.T) {
		_, err := d.Decode([]byte{0x05, 0xC8, 0x01, 0x00})
		assert.ErrorIs(t, err, ErrMalformed)
	})
}

func TestRegistry(t *testing.T) {
	all := All()
	require.NotEmpty(t, all)
	for _, d := range all {
		got, ok := Get(d.ID())
		assert.True(t, ok)
		assert.Equal(t, d.Name(), got.Name())
	}
	_, ok := Get("unknown")
	assert.False(t, ok)
}

func ptr(n int64) *int64 { return &n }
