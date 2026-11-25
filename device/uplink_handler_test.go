package device

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waneock/app-lora/pkg/clock"
)

type unmatchedOf = unmatchedListResponse

// dayAt returns a time on 25 November 2026, the day of the test clock.
func dayAt(hour, minute int) string {
	return time.Date(2026, 11, 25, hour, minute, 0, 0, time.UTC).Format(time.RFC3339)
}

func TestUnmatchedAndAttach(t *testing.T) {
	env := newTestEnv(t)
	org := env.organization(t)
	m := env.model(t, org, "Milesight", "EM300-DI")
	cs := env.integration(t, org, kindChirpStack)
	send := func(t *testing.T, eui string, fCnt, minute int) {
		t.Helper()
		rec := env.ingest(t, chirpStackUpPath, cs.Secret, "application/json", chirpStackBody(eui, fCnt, dayAt(8, minute)))
		require.Equal(t, 204, rec.Code, rec.Body.String())
	}
	uplinks := func(t *testing.T, id int64, query string) uplinkListResponse {
		t.Helper()
		return decode[uplinkListResponse](t, env.do(t, org, http.MethodGet, devicePath(id, "/uplinks"+query), nil), http.StatusOK)
	}

	t.Run("the warning names each unmatched DevEUI", func(t *testing.T) {
		eui := randomDevEUI()
		send(t, eui, 1, 0)
		send(t, eui, 2, 1)
		got := decode[unmatchedOf](t, env.do(t, org, http.MethodGet, "/api/v1/unmatched", nil), http.StatusOK)
		var found *unmatchedResponse
		for i := range got.DevEUIs {
			if got.DevEUIs[i].DevEUI == eui {
				found = &got.DevEUIs[i]
			}
		}
		require.NotNil(t, found)
		assert.Equal(t, 2, found.Count)
		require.NotNil(t, found.IntegrationName)
		assert.Equal(t, "NS chirpstack", *found.IntegrationName)

		other := env.organization(t)
		none := decode[unmatchedOf](t, env.do(t, other, http.MethodGet, "/api/v1/unmatched", nil), http.StatusOK)
		assert.Empty(t, none.DevEUIs, "another organization sees none of them")
	})

	t.Run("a new device takes its unmatched uplinks and locks its DevEUI", func(t *testing.T) {
		eui := randomDevEUI()
		send(t, eui, 1, 10)
		send(t, eui, 2, 11)
		d := env.device(t, org, deviceRequest{DevEUI: eui, Name: "late", DeviceModelID: m.ID})

		got := uplinks(t, d.ID, "")
		assert.Equal(t, 2, got.Total)
		assert.Equal(t, int64(2), got.Uplinks[0].FCnt, "the newest first")
		assert.Equal(t, "0102", got.Uplinks[0].PayloadHex)
		assert.NotNil(t, decode[deviceResponse](t, env.do(t, org, http.MethodGet, devicePath(d.ID, ""), nil), http.StatusOK).LastUplinkAt)

		rec := env.do(t, org, http.MethodPut, devicePath(d.ID, ""),
			deviceRequest{DevEUI: randomDevEUI(), Name: "late", DeviceModelID: m.ID})
		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Equal(t, "dev_eui_locked", errorCode(t, rec))
	})

	t.Run("a DevEUI correction takes the uplinks of the right DevEUI", func(t *testing.T) {
		right := randomDevEUI()
		send(t, right, 1, 20)
		d := env.device(t, org, deviceRequest{DevEUI: randomDevEUI(), Name: "typo", DeviceModelID: m.ID})
		rec := env.do(t, org, http.MethodPut, devicePath(d.ID, ""), deviceRequest{DevEUI: right, Name: "typo", DeviceModelID: m.ID})
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, 1, uplinks(t, d.ID, "").Total)
	})

	t.Run("a restored device takes the uplinks it missed", func(t *testing.T) {
		eui := randomDevEUI()
		d := env.device(t, org, deviceRequest{DevEUI: eui, Name: "away", DeviceModelID: m.ID})
		env.do(t, org, http.MethodPost, devicePath(d.ID, "/archive"), nil)
		send(t, eui, 1, 30)
		assert.Zero(t, uplinks(t, d.ID, "").Total)
		require.Equal(t, http.StatusNoContent, env.do(t, org, http.MethodPost, devicePath(d.ID, "/restore"), nil).Code)
		assert.Equal(t, 1, uplinks(t, d.ID, "").Total)
	})

	t.Run("an imported device takes its unmatched uplinks", func(t *testing.T) {
		eui := randomDevEUI()
		send(t, eui, 1, 40)
		env.doCSV(t, org, "/api/v1/devices/import", "dev_eui,name,vendor,model\n"+eui+",imported,Milesight,EM300-DI\n")
		r := env.list(t, org, "q="+eui)
		require.Len(t, r.Devices, 1)
		assert.Equal(t, 1, uplinks(t, r.Devices[0].ID, "").Total)
		assert.NotNil(t, r.Devices[0].LastUplinkAt)
	})

	t.Run("pages", func(t *testing.T) {
		eui := randomDevEUI()
		d := env.device(t, org, deviceRequest{DevEUI: eui, Name: "busy", DeviceModelID: m.ID})
		for i := range defaultPerPage + 1 {
			rec := env.ingest(t, chirpStackUpPath, cs.Secret, "application/json",
				chirpStackBody(eui, i, time.Date(2026, 11, 20, 0, i, 0, 0, time.UTC).Format(time.RFC3339)))
			require.Equal(t, 204, rec.Code)
		}
		first := uplinks(t, d.ID, "")
		assert.Len(t, first.Uplinks, defaultPerPage)
		assert.Equal(t, defaultPerPage+1, first.Total)
		assert.Len(t, uplinks(t, d.ID, "?page=2").Uplinks, 1)
	})

	t.Run("stats give the last uplink of the organization", func(t *testing.T) {
		last, err := newStats(env.pool).LastUplinkByOrganization(context.Background())
		require.NoError(t, err)
		assert.Equal(t, env.clock.Now(), last[org].UTC(), "the arrival time of the newest uplink")
	})

	t.Run("a device of another organization is not found", func(t *testing.T) {
		d := env.device(t, org, deviceRequest{DevEUI: randomDevEUI(), Name: "mine", DeviceModelID: m.ID})
		other := env.organization(t)
		assert.Equal(t, http.StatusNotFound, env.do(t, other, http.MethodGet, devicePath(d.ID, "/uplinks"), nil).Code)
	})
}

func TestPurgeUnmatched(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	org := env.organization(t)
	m := env.model(t, org, "V", "M")
	in := env.integration(t, org, kindChirpStack)
	ref, err := newIngestService(env.pool, env.clock).authenticate(ctx, in.Secret)
	require.NoError(t, err)

	now := env.clock.Now()
	storeAt := func(t *testing.T, eui string, at time.Time) {
		t.Helper()
		svc := newIngestService(env.pool, clock.NewFixed(at))
		require.NoError(t, svc.store(ctx, ref, &uplinkMessage{DevEUI: eui, Payload: []byte{1}}))
	}
	oldEUI, freshEUI, matchedEUI := randomDevEUI(), randomDevEUI(), randomDevEUI()
	storeAt(t, oldEUI, now.Add(-unmatchedKeep-time.Hour))
	storeAt(t, freshEUI, now.Add(-unmatchedKeep+time.Hour))
	d := env.device(t, org, deviceRequest{DevEUI: matchedEUI, Name: "kept", DeviceModelID: m.ID})
	storeAt(t, matchedEUI, now.Add(-unmatchedKeep-time.Hour))

	p := newPurger(env.pool, env.clock, slog.New(slog.NewTextHandler(io.Discard, nil)))
	p.purge()

	un, err := newUplinkRepo(env.pool).unmatched(ctx, org)
	require.NoError(t, err)
	euis := make([]string, len(un))
	for i, x := range un {
		euis[i] = x.DevEUI
	}
	assert.NotContains(t, euis, oldEUI, "older than 30 days")
	assert.Contains(t, euis, freshEUI)
	has, err := newUplinkRepo(env.pool).hasAny(ctx, org, d.ID)
	require.NoError(t, err)
	assert.True(t, has, "an old uplink of a device stays")

	t.Run("the loop starts and stops", func(t *testing.T) {
		p.start()
		require.NoError(t, p.close(ctx))
	})
}
