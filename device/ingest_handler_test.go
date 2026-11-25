package device

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const chirpStackUpPath = "/api/v1/ingest/chirpstack?event=up"

// chirpStackBody returns a ChirpStack up event for devEUI.
func chirpStackBody(devEUI string, fCnt int, at string) []byte {
	return []byte(`{"time":"` + at + `","deviceInfo":{"devEui":"` + devEUI + `"},"fCnt":` +
		itoa(int64(fCnt)) + `,"data":"AQI="}`)
}

func TestIngest(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	org := env.organization(t)
	m := env.model(t, org, "Milesight", "EM300-DI")
	cs := env.integration(t, org, kindChirpStack)
	ttn := env.integration(t, org, kindTTN)
	repo := newUplinkRepo(env.pool)

	uplinksOf := func(t *testing.T, deviceID int64) []uplink {
		t.Helper()
		ups, _, err := repo.listForDevice(ctx, org, deviceID, 100, 0)
		require.NoError(t, err)
		return ups
	}
	status := func(t *testing.T, id int64) integrationResponse {
		t.Helper()
		list := decode[integrationListResponse](t, env.do(t, org, http.MethodGet, "/api/v1/integrations", nil), http.StatusOK)
		for _, in := range list.Integrations {
			if in.ID == id {
				return in
			}
		}
		require.FailNow(t, "no integration", id)
		return integrationResponse{}
	}

	t.Run("a ChirpStack uplink lands on its device", func(t *testing.T) {
		eui := randomDevEUI()
		d := env.device(t, org, deviceRequest{DevEUI: eui, Name: "d", DeviceModelID: m.ID})
		rec := env.ingest(t, chirpStackUpPath, cs.Secret, "application/json", chirpStackBody(eui, 7, "2026-11-25T09:00:00Z"))
		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())

		ups := uplinksOf(t, d.ID)
		require.Len(t, ups, 1)
		assert.Equal(t, int64(7), ups[0].FCnt)
		assert.Equal(t, []byte{1, 2}, ups[0].Payload)
		got := status(t, cs.Integration.ID)
		assert.NotNil(t, got.LastUplinkAt)
		assert.GreaterOrEqual(t, got.RecentUplinks, 1)
	})

	t.Run("a TTN sample body lands on its device", func(t *testing.T) {
		org := env.organization(t)
		m := env.model(t, org, "V", "M")
		d := env.device(t, org, deviceRequest{DevEUI: "70b3d57ed0001234", Name: "d", DeviceModelID: m.ID})
		ttn := env.integration(t, org, kindTTN)
		rec := env.ingest(t, "/api/v1/ingest/ttn", ttn.Secret, "application/json", readTestdata(t, "ttn_uplink.json"))
		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		ups, _, err := repo.listForDevice(ctx, org, d.ID, 10, 0)
		require.NoError(t, err)
		require.Len(t, ups, 1)
		assert.Equal(t, int64(43), ups[0].FCnt)
	})

	t.Run("a redelivery is stored once", func(t *testing.T) {
		eui := randomDevEUI()
		d := env.device(t, org, deviceRequest{DevEUI: eui, Name: "d", DeviceModelID: m.ID})
		body := chirpStackBody(eui, 9, "2026-11-25T09:05:00Z")
		for range 2 {
			require.Equal(t, http.StatusNoContent, env.ingest(t, chirpStackUpPath, cs.Secret, "application/json", body).Code)
		}
		assert.Len(t, uplinksOf(t, d.ID), 1)
	})

	t.Run("an unknown DevEUI and an archived device give unmatched uplinks", func(t *testing.T) {
		unknown := randomDevEUI()
		archived := env.device(t, org, deviceRequest{DevEUI: randomDevEUI(), Name: "old", DeviceModelID: m.ID})
		env.do(t, org, http.MethodPost, devicePath(archived.ID, "/archive"), nil)

		env.ingest(t, chirpStackUpPath, cs.Secret, "application/json", chirpStackBody(unknown, 1, "2026-11-25T09:10:00Z"))
		env.ingest(t, chirpStackUpPath, cs.Secret, "application/json", chirpStackBody(archived.DevEUI, 1, "2026-11-25T09:10:00Z"))

		assert.Empty(t, uplinksOf(t, archived.ID))
		un, err := repo.unmatched(ctx, org)
		require.NoError(t, err)
		euis := make([]string, len(un))
		for i, x := range un {
			euis[i] = x.DevEUI
		}
		assert.Contains(t, euis, unknown)
		assert.Contains(t, euis, archived.DevEUI)
	})

	t.Run("two organizations with one DevEUI each get only their integration's uplinks", func(t *testing.T) {
		eui := randomDevEUI()
		mine := env.device(t, org, deviceRequest{DevEUI: eui, Name: "mine", DeviceModelID: m.ID})
		other := env.organization(t)
		om := env.model(t, other, "V", "M")
		theirs := env.device(t, other, deviceRequest{DevEUI: eui, Name: "theirs", DeviceModelID: om.ID})
		theirNS := env.integration(t, other, kindChirpStack)

		env.ingest(t, chirpStackUpPath, theirNS.Secret, "application/json", chirpStackBody(eui, 1, "2026-11-25T09:20:00Z"))

		assert.Empty(t, uplinksOf(t, mine.ID))
		ups, _, err := repo.listForDevice(ctx, other, theirs.ID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, ups, 1)
	})

	t.Run("no secret or a wrong secret stores nothing and records nothing", func(t *testing.T) {
		eui := randomDevEUI()
		d := env.device(t, org, deviceRequest{DevEUI: eui, Name: "d", DeviceModelID: m.ID})
		for _, secret := range []string{"", "lora_wrong", cs.Secret + "x"} {
			rec := env.ingest(t, chirpStackUpPath, secret, "application/json", chirpStackBody(eui, 1, "2026-11-25T09:30:00Z"))
			assert.Equal(t, http.StatusUnauthorized, rec.Code)
			assert.Equal(t, "invalid_secret", errorCode(t, rec))
		}
		assert.Empty(t, uplinksOf(t, d.ID))
	})

	t.Run("a request it cannot read is recorded on the integration", func(t *testing.T) {
		cases := []struct {
			name, path, secret, contentType, body, code string
			id                                          int64
		}{
			{"wrong endpoint", "/api/v1/ingest/ttn", cs.Secret, "application/json", "{}", "wrong_endpoint", cs.Integration.ID},
			{"protobuf", chirpStackUpPath, cs.Secret, "application/octet-stream", "\x0a\x24", "chirpstack_protobuf", cs.Integration.ID},
			{"bad JSON", "/api/v1/ingest/ttn", ttn.Secret, "application/json", "{", "ingest_invalid_json", ttn.Integration.ID},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				rec := env.ingest(t, tc.path, tc.secret, tc.contentType, []byte(tc.body))
				assert.Equal(t, http.StatusBadRequest, rec.Code)
				assert.Equal(t, tc.code, errorCode(t, rec))
				got := status(t, tc.id)
				require.NotNil(t, got.LastError)
				assert.Equal(t, tc.code, got.LastError.Code)
			})
		}
	})

	t.Run("other events answer 204 and store nothing", func(t *testing.T) {
		rec := env.ingest(t, "/api/v1/ingest/chirpstack?event=join", cs.Secret, "application/json", readTestdata(t, "chirpstack_join.json"))
		assert.Equal(t, http.StatusNoContent, rec.Code)
		rec = env.ingest(t, "/api/v1/ingest/ttn", ttn.Secret, "application/json", readTestdata(t, "ttn_join_accept.json"))
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("a regenerated secret ends the old one at once", func(t *testing.T) {
		in := env.integration(t, org, kindChirpStack)
		rec := env.do(t, org, http.MethodPost, integrationPath(in.Integration.ID, "/secret"), nil)
		fresh := decode[regenerateResponse](t, rec, http.StatusOK)
		body := chirpStackBody(randomDevEUI(), 1, "2026-11-25T09:40:00Z")
		assert.Equal(t, http.StatusUnauthorized, env.ingest(t, chirpStackUpPath, in.Secret, "application/json", body).Code)
		assert.Equal(t, http.StatusNoContent, env.ingest(t, chirpStackUpPath, fresh.Secret, "application/json", body).Code)
	})

	t.Run("a deleted integration keeps its uplinks", func(t *testing.T) {
		in := env.integration(t, org, kindChirpStack)
		eui := randomDevEUI()
		d := env.device(t, org, deviceRequest{DevEUI: eui, Name: "d", DeviceModelID: m.ID})
		env.ingest(t, chirpStackUpPath, in.Secret, "application/json", chirpStackBody(eui, 1, "2026-11-25T09:50:00Z"))
		require.Equal(t, http.StatusNoContent, env.do(t, org, http.MethodDelete, integrationPath(in.Integration.ID, ""), nil).Code)
		assert.Len(t, uplinksOf(t, d.ID), 1)
		assert.Equal(t, http.StatusUnauthorized,
			env.ingest(t, chirpStackUpPath, in.Secret, "application/json", chirpStackBody(eui, 2, "2026-11-25T09:51:00Z")).Code)
	})

	t.Run("the body limit", func(t *testing.T) {
		big := []byte(`{"x":"` + strings.Repeat("a", maxIngestBytes) + `"}`)
		rec := env.ingest(t, "/api/v1/ingest/ttn", ttn.Secret, "application/json", big)
		assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	})
}
