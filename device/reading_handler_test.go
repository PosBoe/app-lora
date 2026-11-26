package device

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadingsAndDecoderAssignment(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	org := env.organization(t)
	m := env.model(t, org, "Milesight", "EM300-DI")
	cs := env.integration(t, org, kindChirpStack)
	w := newDecodeWorker(env.pool, slog.New(slog.NewTextHandler(io.Discard, nil)))
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
	eui := randomDevEUI()
	d := env.device(t, org, deviceRequest{DevEUI: eui, Name: "bloc", DeviceModelID: m.ID})
	// The test clock is 25 November 2026 12:00 UTC. Chisinau is UTC+2.
	sends := []struct {
		at      time.Time
		battery byte
		pulses  uint32
	}{
		{time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC), 60, 1000},
		{time.Date(2026, 11, 23, 21, 30, 0, 0, time.UTC), 60, 1100}, // 23:30 on the 23rd in Chisinau
		{time.Date(2026, 11, 23, 22, 30, 0, 0, time.UTC), 60, 1150}, // 00:30 on the 24th in Chisinau
		{time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC), 7, 1400},
	}
	for i, s := range sends {
		rec := env.ingest(t, chirpStackUpPath, cs.Secret, "application/json",
			chirpStackPayload(eui, i+1, s.at, em300Payload(s.battery, s.pulses)))
		require.Equal(t, http.StatusNoContent, rec.Code)
	}

	t.Run("a model without a decoder has no pulses", func(t *testing.T) {
		drain(t)
		got := decode[pulsesResponse](t, env.do(t, org, http.MethodGet, devicePath(d.ID, "/pulses?days=7"), nil), http.StatusOK)
		assert.Empty(t, got.Points)
		dev := decode[deviceResponse](t, env.do(t, org, http.MethodGet, devicePath(d.ID, ""), nil), http.StatusOK)
		assert.False(t, dev.DeviceModel.HasDecoder)
	})

	t.Run("the admin assigns a decoder; the history is decoded", func(t *testing.T) {
		rec := env.do(t, 0, http.MethodPut, fmt.Sprintf("/api/v1/admin/device-models/%d/decoder", m.ID),
			setDecoderRequest{Decoder: ptrString("unknown")})
		assert.Equal(t, "decoder_unknown", errorCode(t, rec))

		rec = env.do(t, 0, http.MethodPut, fmt.Sprintf("/api/v1/admin/device-models/%d/decoder", m.ID),
			setDecoderRequest{Decoder: ptrString("milesight_em300_di")})
		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())

		list := decode[adminModelListResponse](t, env.do(t, 0, http.MethodGet, "/api/v1/admin/device-models", nil), http.StatusOK)
		var found *adminModelResponse
		for i := range list.DeviceModels {
			if list.DeviceModels[i].ID == m.ID {
				found = &list.DeviceModels[i]
			}
		}
		require.NotNil(t, found)
		assert.Equal(t, org, found.OrganizationID)
		assert.Equal(t, 4, found.PendingUplinks, "decoding has not run yet")

		drain(t)
		list = decode[adminModelListResponse](t, env.do(t, 0, http.MethodGet, "/api/v1/admin/device-models", nil), http.StatusOK)
		for _, x := range list.DeviceModels {
			if x.ID == m.ID {
				assert.Zero(t, x.PendingUplinks)
			}
		}

		decoders := decode[decoderListResponse](t, env.do(t, 0, http.MethodGet, "/api/v1/admin/decoders", nil), http.StatusOK)
		assert.Contains(t, decoders.Decoders, decoderResponse{ID: "milesight_em300_di", Name: "Milesight EM300-DI"})
	})

	t.Run("pulses by point and by Chisinau day", func(t *testing.T) {
		got := decode[pulsesResponse](t, env.do(t, org, http.MethodGet, devicePath(d.ID, "/pulses?days=7"), nil), http.StatusOK)
		assert.Len(t, got.Points, 4)
		days := make([]string, len(got.Days))
		for i, x := range got.Days {
			days[i] = x.Day
		}
		assert.Equal(t, []string{"2026-11-22", "2026-11-23", "2026-11-24", "2026-11-25"}, days)
		assert.Nil(t, got.Days[0].Delta, "the first day has no day before it")
		require.NotNil(t, got.Days[1].Delta)
		assert.Equal(t, int64(100), *got.Days[1].Delta)
		assert.Equal(t, int64(50), *got.Days[2].Delta, "00:30 in Chisinau is the next day")
		assert.Equal(t, int64(250), *got.Days[3].Delta)
	})

	t.Run("an unknown range falls back to 30 days", func(t *testing.T) {
		got := decode[pulsesResponse](t, env.do(t, org, http.MethodGet, devicePath(d.ID, "/pulses?days=3"), nil), http.StatusOK)
		assert.Len(t, got.Days, 4)
	})

	t.Run("alarms and the active alarm of the device", func(t *testing.T) {
		got := decode[alarmListResponse](t, env.do(t, org, http.MethodGet, devicePath(d.ID, "/alarms"), nil), http.StatusOK)
		require.Equal(t, 1, got.Total)
		assert.Equal(t, "low_battery", got.Alarms[0].Kind)
		assert.Nil(t, got.Alarms[0].ClearedAt)

		dev := decode[deviceResponse](t, env.do(t, org, http.MethodGet, devicePath(d.ID, ""), nil), http.StatusOK)
		assert.True(t, dev.DeviceModel.HasDecoder)
		assert.Equal(t, []string{"low_battery"}, dev.ActiveAlarms)
		list := env.list(t, org, "q="+eui)
		assert.Equal(t, []string{"low_battery"}, list.Devices[0].ActiveAlarms)
	})

	t.Run("another organization reads nothing", func(t *testing.T) {
		other := env.organization(t)
		assert.Equal(t, http.StatusNotFound, env.do(t, other, http.MethodGet, devicePath(d.ID, "/pulses"), nil).Code)
		assert.Equal(t, http.StatusNotFound, env.do(t, other, http.MethodGet, devicePath(d.ID, "/alarms"), nil).Code)
	})

	t.Run("an unknown model is not found", func(t *testing.T) {
		rec := env.do(t, 0, http.MethodPut, "/api/v1/admin/device-models/999999999/decoder",
			setDecoderRequest{Decoder: nil})
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func ptrString(s string) *string { return &s }
