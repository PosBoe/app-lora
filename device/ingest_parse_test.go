package device

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return data
}

func TestParseChirpStack(t *testing.T) {
	t.Run("up event", func(t *testing.T) {
		msg, skip, err := parseChirpStack("up", "application/json", readTestdata(t, "chirpstack_up.json"))
		require.Nil(t, err)
		require.False(t, skip)
		assert.Equal(t, "70b3d57ed0001234", msg.DevEUI)
		assert.Equal(t, int64(42), msg.FCnt)
		assert.Equal(t, []byte{0x01, 0x75, 0x67, 0x02, 0x01, 0x00, 0x05}, msg.Payload)
		require.NotNil(t, msg.ReceivedAt)
		assert.Equal(t, time.Date(2026, 11, 25, 9, 34, 15, 775023242, time.UTC), msg.ReceivedAt.UTC())
	})

	t.Run("first uplink: no fCnt, no time, empty payload", func(t *testing.T) {
		msg, _, err := parseChirpStack("up", "application/json", readTestdata(t, "chirpstack_up_first.json"))
		require.Nil(t, err)
		assert.Equal(t, "70b3d57ed0001235", msg.DevEUI)
		assert.Zero(t, msg.FCnt)
		assert.Empty(t, msg.Payload)
		assert.Nil(t, msg.ReceivedAt)
	})

	t.Run("other events are skipped", func(t *testing.T) {
		_, skip, err := parseChirpStack("join", "application/json", readTestdata(t, "chirpstack_join.json"))
		assert.Nil(t, err)
		assert.True(t, skip)
	})

	t.Run("errors", func(t *testing.T) {
		_, _, err := parseChirpStack("up", "application/octet-stream", []byte{0x0a, 0x24})
		assert.Equal(t, errChirpStackProtobuf, err)
		_, _, err = parseChirpStack("up", "application/json", []byte("{"))
		assert.Equal(t, errIngestInvalidJSON, err)
		_, _, err = parseChirpStack("up", "application/json", []byte(`{"deviceInfo":{"devEui":"12"},"data":""}`))
		assert.Equal(t, errIngestDevEUI, err)
		_, _, err = parseChirpStack("up", "application/json", []byte(`{"deviceInfo":{"devEui":"70b3d57ed0001234"},"data":"!!"}`))
		assert.Equal(t, errIngestPayload, err)
	})
}

func TestParseTTN(t *testing.T) {
	t.Run("uplink message", func(t *testing.T) {
		msg, skip, err := parseTTN(readTestdata(t, "ttn_uplink.json"))
		require.Nil(t, err)
		require.False(t, skip)
		assert.Equal(t, "70b3d57ed0001234", msg.DevEUI)
		assert.Equal(t, int64(43), msg.FCnt)
		assert.Equal(t, []byte{0x01, 0x75, 0x67, 0x02, 0x01, 0x00, 0x05}, msg.Payload)
		require.NotNil(t, msg.ReceivedAt)
		assert.Equal(t, time.Date(2026, 11, 25, 9, 34, 15, 987654321, time.UTC), msg.ReceivedAt.UTC(),
			"the uplink_message time, not the top-level one")
	})

	t.Run("a join accept is skipped", func(t *testing.T) {
		_, skip, err := parseTTN(readTestdata(t, "ttn_join_accept.json"))
		assert.Nil(t, err)
		assert.True(t, skip)
	})

	t.Run("an uplink without its own time takes the message time", func(t *testing.T) {
		msg, _, err := parseTTN([]byte(`{"received_at":"2026-11-25T10:00:00Z",
			"end_device_ids":{"dev_eui":"70B3D57ED0001234"},"uplink_message":{"frm_payload":"AQ=="}}`))
		require.Nil(t, err)
		assert.Equal(t, time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC), msg.ReceivedAt.UTC())
	})
}
