package device

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// em300Payload builds an EM300-DI payload with a battery level and a pulse
// count.
func em300Payload(battery byte, pulses uint32) []byte {
	p := []byte{0x01, 0x75, battery, 0x05, 0xC8, 0, 0, 0, 0}
	binary.LittleEndian.PutUint32(p[5:], pulses)
	return p
}

// chirpStackPayload returns a ChirpStack up event with a payload.
func chirpStackPayload(devEUI string, fCnt int, at time.Time, payload []byte) []byte {
	return []byte(`{"time":"` + at.Format(time.RFC3339) + `","deviceInfo":{"devEui":"` + devEUI +
		`"},"fCnt":` + itoa(int64(fCnt)) + `,"data":"` + base64.StdEncoding.EncodeToString(payload) + `"}`)
}

type decodedUplink struct {
	FCnt       int64
	PulseCount *int64
	Decoder    *string
}

func TestDecodeWorker(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	org := env.organization(t)
	m := env.model(t, org, "Milesight", "EM300-DI")
	cs := env.integration(t, org, kindChirpStack)
	w := newDecodeWorker(env.pool, slog.New(slog.NewTextHandler(io.Discard, nil)))
	decoderID := "milesight_em300_di"

	drain := func(t *testing.T) {
		t.Helper()
		for {
			n, err := w.runOnce(ctx)
			require.NoError(t, err)
			if n < decodeBatch {
				return
			}
		}
	}
	setDecoder := func(t *testing.T, modelID int64, id *string) {
		t.Helper()
		require.NoError(t, newModelRepo(env.pool).setDecoder(ctx, modelID, id))
		require.NoError(t, newDecodeRepo(env.pool).markModel(ctx, modelID))
	}
	hour := func(h int) time.Time { return time.Date(2026, 11, 20, h, 0, 0, 0, time.UTC) }
	send := func(t *testing.T, eui string, fCnt, h int, battery byte, pulses uint32) {
		t.Helper()
		rec := env.ingest(t, chirpStackUpPath, cs.Secret, "application/json",
			chirpStackPayload(eui, fCnt, hour(h), em300Payload(battery, pulses)))
		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	}
	uplinksOf := func(t *testing.T, deviceID int64) []decodedUplink {
		t.Helper()
		rows, err := env.pool.Query(ctx, `
			SELECT f_cnt, pulse_count, decoder FROM uplinks
			WHERE organization_id = $1 AND device_id = $2 ORDER BY received_at`, org, deviceID)
		require.NoError(t, err)
		defer rows.Close()
		var out []decodedUplink
		for rows.Next() {
			var u decodedUplink
			require.NoError(t, rows.Scan(&u.FCnt, &u.PulseCount, &u.Decoder))
			out = append(out, u)
		}
		return out
	}
	alarmsOf := func(t *testing.T, deviceID int64) []alarmPeriod {
		t.Helper()
		rows, err := env.pool.Query(ctx, `
			SELECT kind, raised_at, cleared_at FROM alarms
			WHERE organization_id = $1 AND device_id = $2 ORDER BY raised_at`, org, deviceID)
		require.NoError(t, err)
		defer rows.Close()
		var out []alarmPeriod
		for rows.Next() {
			var a alarmPeriod
			require.NoError(t, rows.Scan(&a.Kind, &a.RaisedAt, &a.ClearedAt))
			out = append(out, a)
		}
		return out
	}

	eui := randomDevEUI()
	d := env.device(t, org, deviceRequest{DevEUI: eui, Name: "bloc", DeviceModelID: m.ID})
	send(t, eui, 1, 1, 50, 100)
	send(t, eui, 2, 2, 8, 150)
	send(t, eui, 3, 3, 8, 210)
	send(t, eui, 4, 4, 40, 300)

	t.Run("without a decoder the uplinks stay raw", func(t *testing.T) {
		drain(t)
		for _, u := range uplinksOf(t, d.ID) {
			assert.Nil(t, u.PulseCount)
			assert.Nil(t, u.Decoder)
		}
		assert.Empty(t, alarmsOf(t, d.ID))
	})

	t.Run("assigning the decoder decodes the history and builds the alarms", func(t *testing.T) {
		setDecoder(t, m.ID, &decoderID)
		drain(t)
		var pulses []int64
		for _, u := range uplinksOf(t, d.ID) {
			require.NotNil(t, u.PulseCount)
			pulses = append(pulses, *u.PulseCount)
		}
		assert.Equal(t, []int64{100, 150, 210, 300}, pulses)

		alarms := alarmsOf(t, d.ID)
		require.Len(t, alarms, 1)
		assert.Equal(t, "low_battery", alarms[0].Kind)
		assert.Equal(t, hour(2), alarms[0].RaisedAt.UTC())
		require.NotNil(t, alarms[0].ClearedAt)
		assert.Equal(t, hour(4), alarms[0].ClearedAt.UTC())
	})

	t.Run("a new uplink is decoded and raises an alarm", func(t *testing.T) {
		send(t, eui, 5, 5, 5, 320)
		drain(t)
		alarms := alarmsOf(t, d.ID)
		require.Len(t, alarms, 2)
		assert.Equal(t, hour(5), alarms[1].RaisedAt.UTC())
		assert.Nil(t, alarms[1].ClearedAt)
	})

	t.Run("a late uplink replays the alarms", func(t *testing.T) {
		// Between hour 1 and hour 2 the battery was already low: the first
		// alarm starts earlier.
		rec := env.ingest(t, chirpStackUpPath, cs.Secret, "application/json",
			chirpStackPayload(eui, 9, hour(1).Add(30*time.Minute), em300Payload(9, 120)))
		require.Equal(t, http.StatusNoContent, rec.Code)
		drain(t)
		alarms := alarmsOf(t, d.ID)
		require.Len(t, alarms, 2)
		assert.Equal(t, hour(1).Add(30*time.Minute), alarms[0].RaisedAt.UTC())
	})

	t.Run("an uplink that arrived before the device is decoded when it is attached", func(t *testing.T) {
		late := randomDevEUI()
		send(t, late, 1, 6, 90, 7)
		drain(t)
		d2 := env.device(t, org, deviceRequest{DevEUI: late, Name: "late", DeviceModelID: m.ID})
		drain(t)
		ups := uplinksOf(t, d2.ID)
		require.Len(t, ups, 1)
		require.NotNil(t, ups[0].PulseCount)
		assert.Equal(t, int64(7), *ups[0].PulseCount)
	})

	t.Run("a device moved to a model without a decoder loses its decoded values", func(t *testing.T) {
		bare := env.model(t, org, "Other", "Module")
		rec := env.do(t, org, http.MethodPut, devicePath(d.ID, ""),
			deviceRequest{DevEUI: d.DevEUI, Name: "bloc", DeviceModelID: bare.ID})
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		drain(t)
		for _, u := range uplinksOf(t, d.ID) {
			assert.Nil(t, u.PulseCount)
		}
		assert.Empty(t, alarmsOf(t, d.ID))
	})

	t.Run("removing the decoder of a model clears its devices", func(t *testing.T) {
		eui := randomDevEUI()
		d3 := env.device(t, org, deviceRequest{DevEUI: eui, Name: "third", DeviceModelID: m.ID})
		send(t, eui, 1, 7, 5, 1)
		drain(t)
		require.Len(t, alarmsOf(t, d3.ID), 1)

		setDecoder(t, m.ID, nil)
		drain(t)
		assert.Nil(t, uplinksOf(t, d3.ID)[0].PulseCount)
		assert.Empty(t, alarmsOf(t, d3.ID))
	})

	t.Run("uplinks split across batches still raise their alarm", func(t *testing.T) {
		eui := randomDevEUI()
		d := env.device(t, org, deviceRequest{DevEUI: eui, Name: "split", DeviceModelID: m.ID})
		setDecoder(t, m.ID, &decoderID)
		send(t, eui, 1, 8, 50, 1)
		drain(t)

		send(t, eui, 2, 9, 5, 2)  // low battery
		send(t, eui, 3, 10, 5, 3) // still low
		small := newDecodeWorker(env.pool, w.logger)
		small.batch = 1
		for {
			n, err := small.runOnce(ctx)
			require.NoError(t, err)
			if n == 0 {
				break
			}
		}
		alarms := alarmsOf(t, d.ID)
		require.Len(t, alarms, 1)
		assert.Equal(t, hour(9), alarms[0].RaisedAt.UTC())
	})

	t.Run("new uplinks are applied in time order, not arrival order", func(t *testing.T) {
		eui := randomDevEUI()
		d := env.device(t, org, deviceRequest{DevEUI: eui, Name: "order", DeviceModelID: m.ID})
		setDecoder(t, m.ID, &decoderID)
		send(t, eui, 1, 11, 50, 1)
		drain(t)

		send(t, eui, 3, 14, 5, 3)  // later and low: arrives first
		send(t, eui, 2, 13, 50, 2) // earlier and fine: arrives second
		drain(t)
		alarms := alarmsOf(t, d.ID)
		require.Len(t, alarms, 1)
		assert.Equal(t, hour(14), alarms[0].RaisedAt.UTC())
		assert.Nil(t, alarms[0].ClearedAt, "the earlier uplink does not clear a later alarm")
	})

	t.Run("the loop starts and stops", func(t *testing.T) {
		w.start()
		require.NoError(t, w.close(ctx))
	})
}
