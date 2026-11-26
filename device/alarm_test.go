package device

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyAlarms(t *testing.T) {
	at := func(h int) time.Time { return time.Date(2026, 11, 25, h, 0, 0, 0, time.UTC) }
	battery := []string{"low_battery"}
	row := func(id int64, h int, reported, raised []string) decodedRow {
		return decodedRow{UplinkID: id, ReceivedAt: at(h), Reported: reported, Raised: raised}
	}

	t.Run("a flag repeated in many uplinks is one alarm, cleared by the first report without it", func(t *testing.T) {
		got := applyAlarms(nil, []decodedRow{
			row(1, 1, battery, nil),
			row(2, 2, battery, battery),
			row(3, 3, battery, battery),
			row(4, 4, nil, nil),
			row(5, 5, battery, nil),
			row(6, 6, battery, battery),
		})
		require.Len(t, got, 2)
		assert.Equal(t, at(2), got[0].RaisedAt)
		assert.Equal(t, int64(2), got[0].RaisedUplinkID)
		require.NotNil(t, got[0].ClearedAt)
		assert.Equal(t, at(5), *got[0].ClearedAt, "the uplink with no battery byte did not clear it")
		assert.Equal(t, at(6), got[1].RaisedAt)
		assert.Nil(t, got[1].ClearedAt, "still active")
	})

	t.Run("an active stored alarm is cleared and marked changed", func(t *testing.T) {
		stored := &alarmPeriod{ID: 9, Kind: "low_battery", RaisedAt: at(1)}
		got := applyAlarms([]*alarmPeriod{stored}, []decodedRow{
			row(7, 2, battery, battery),
			row(8, 3, battery, nil),
		})
		require.Len(t, got, 1)
		assert.Same(t, stored, got[0])
		assert.True(t, got[0].changed)
		assert.Equal(t, int64(8), got[0].ClearedUplinkID)
	})

	t.Run("rows that change nothing touch nothing", func(t *testing.T) {
		stored := &alarmPeriod{ID: 9, Kind: "low_battery", RaisedAt: at(1)}
		got := applyAlarms([]*alarmPeriod{stored}, []decodedRow{row(7, 2, battery, battery), row(8, 3, nil, nil)})
		assert.Empty(t, got)
		assert.Nil(t, stored.ClearedAt)
	})
}
