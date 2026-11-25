package device

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDevice(t *testing.T) {
	env := newTestEnv(t)
	org := env.organization(t)
	m := env.model(t, org, "Milesight", "EM300-DI")

	t.Run("stores the normalized fields", func(t *testing.T) {
		raw := strings.ToUpper(randomDevEUI())
		written := raw[0:2] + ":" + raw[2:4] + ":" + raw[4:]
		d := env.device(t, org, deviceRequest{
			DevEUI: written, Name: "  Bloc 3 ", DeviceModelID: m.ID, MeterType: "Apă",
			Address: " str. Mare 1 ",
		})
		assert.Equal(t, strings.ToLower(raw), d.DevEUI)
		assert.Equal(t, "Bloc 3", d.Name)
		assert.Equal(t, modelRefResponse{ID: m.ID, Vendor: "Milesight", Model: "EM300-DI"}, d.DeviceModel)
		require.NotNil(t, d.MeterType)
		assert.Equal(t, "water", *d.MeterType)
		require.NotNil(t, d.Address)
		assert.Equal(t, "str. Mare 1", *d.Address)
		assert.Nil(t, d.Note, "an empty optional field is null")
	})

	t.Run("checks the fields", func(t *testing.T) {
		cases := []struct {
			name string
			in   deviceRequest
			code string
		}{
			{"bad DevEUI", deviceRequest{DevEUI: "12", Name: "x", DeviceModelID: m.ID}, "dev_eui_invalid"},
			{"no name", deviceRequest{DevEUI: randomDevEUI(), Name: " ", DeviceModelID: m.ID}, "device_name_required"},
			{"no model", deviceRequest{DevEUI: randomDevEUI(), Name: "x"}, "model_not_found"},
			{"bad meter type", deviceRequest{DevEUI: randomDevEUI(), Name: "x", DeviceModelID: m.ID, MeterType: "steam"}, "meter_type_invalid"},
			{"long note", deviceRequest{DevEUI: randomDevEUI(), Name: "x", DeviceModelID: m.ID, Note: strings.Repeat("n", maxNoteChars+1)}, "note_too_long"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				rec := env.do(t, org, http.MethodPost, "/api/v1/devices", tc.in)
				assert.Equal(t, http.StatusBadRequest, rec.Code)
				assert.Equal(t, tc.code, errorCode(t, rec))
			})
		}
	})

	t.Run("a model of another organization is refused", func(t *testing.T) {
		other := env.organization(t)
		theirs := env.model(t, other, "Other", "Model")
		rec := env.do(t, org, http.MethodPost, "/api/v1/devices",
			deviceRequest{DevEUI: randomDevEUI(), Name: "x", DeviceModelID: theirs.ID})
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "model_not_found", errorCode(t, rec))
	})

	t.Run("a DevEUI is unique in the organization only", func(t *testing.T) {
		eui := randomDevEUI()
		env.device(t, org, deviceRequest{DevEUI: eui, Name: "first", DeviceModelID: m.ID})

		rec := env.do(t, org, http.MethodPost, "/api/v1/devices",
			deviceRequest{DevEUI: strings.ToUpper(eui), Name: "second", DeviceModelID: m.ID})
		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Equal(t, "dev_eui_taken", errorCode(t, rec))

		other := env.organization(t)
		theirs := env.model(t, other, "Milesight", "EM300-DI")
		env.device(t, other, deviceRequest{DevEUI: eui, Name: "theirs", DeviceModelID: theirs.ID})
	})

	t.Run("a DevEUI of an archived device says so", func(t *testing.T) {
		eui := randomDevEUI()
		d := env.device(t, org, deviceRequest{DevEUI: eui, Name: "old", DeviceModelID: m.ID})
		require.Equal(t, http.StatusNoContent, env.do(t, org, http.MethodPost, devicePath(d.ID, "/archive"), nil).Code)

		rec := env.do(t, org, http.MethodPost, "/api/v1/devices",
			deviceRequest{DevEUI: eui, Name: "new", DeviceModelID: m.ID})
		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Equal(t, "dev_eui_archived", errorCode(t, rec))
	})
}

func TestUpdateDevice(t *testing.T) {
	env := newTestEnv(t)
	org := env.organization(t)
	m := env.model(t, org, "Milesight", "EM300-DI")
	d := env.device(t, org, deviceRequest{DevEUI: randomDevEUI(), Name: "before", DeviceModelID: m.ID, Note: "n"})

	t.Run("writes every field and clears an empty one", func(t *testing.T) {
		eui := randomDevEUI()
		rec := env.do(t, org, http.MethodPut, devicePath(d.ID, ""),
			deviceRequest{DevEUI: eui, Name: "after", DeviceModelID: m.ID, MeterType: "gaz"})
		got := decode[deviceResponse](t, rec, http.StatusOK)
		assert.Equal(t, eui, got.DevEUI, "a device with no uplink takes a new DevEUI")
		assert.Equal(t, "after", got.Name)
		require.NotNil(t, got.MeterType)
		assert.Equal(t, "gas", *got.MeterType)
		assert.Nil(t, got.Note)
	})

	t.Run("a device of another organization is not found", func(t *testing.T) {
		other := env.organization(t)
		rec := env.do(t, other, http.MethodPut, devicePath(d.ID, ""),
			deviceRequest{DevEUI: randomDevEUI(), Name: "x", DeviceModelID: m.ID})
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, http.StatusNotFound, env.do(t, other, http.MethodGet, devicePath(d.ID, ""), nil).Code)
	})
}

func TestListDevices(t *testing.T) {
	env := newTestEnv(t)
	org := env.organization(t)
	water := env.model(t, org, "Milesight", "EM300-DI")
	gas := env.model(t, org, "Elvaco", "CMi4110")

	eui := "70b3d57ed0001234"
	env.device(t, org, deviceRequest{DevEUI: eui, Name: "Bloc A", DeviceModelID: water.ID, MeterType: "water", Address: "str. Mare 1"})
	env.device(t, org, deviceRequest{DevEUI: randomDevEUI(), Name: "bloc b", DeviceModelID: water.ID, MeterType: "water"})
	env.device(t, org, deviceRequest{DevEUI: randomDevEUI(), Name: "Cazangerie", DeviceModelID: gas.ID, MeterType: "gas", Address: "str. Mică 7"})
	archived := env.device(t, org, deviceRequest{DevEUI: randomDevEUI(), Name: "Vechi", DeviceModelID: gas.ID})
	env.do(t, org, http.MethodPost, devicePath(archived.ID, "/archive"), nil)

	names := func(r listResponse) []string {
		out := make([]string, len(r.Devices))
		for i, d := range r.Devices {
			out[i] = d.Name
		}
		return out
	}

	t.Run("live devices by name, with the total", func(t *testing.T) {
		r := env.list(t, org, "")
		assert.Equal(t, []string{"Bloc A", "bloc b", "Cazangerie"}, names(r))
		assert.Equal(t, 3, r.Total)
	})

	t.Run("pages keep the total", func(t *testing.T) {
		first := env.list(t, org, "per_page=2")
		assert.Equal(t, []string{"Bloc A", "bloc b"}, names(first))
		assert.Equal(t, 3, first.Total)
		second := env.list(t, org, "per_page=2&page=2")
		assert.Equal(t, []string{"Cazangerie"}, names(second))
		past := env.list(t, org, "per_page=2&page=9")
		assert.Empty(t, past.Devices)
		assert.Equal(t, 3, past.Total, "a page after the last one still has the total")
	})

	t.Run("search", func(t *testing.T) {
		assert.Equal(t, []string{"Bloc A", "bloc b"}, names(env.list(t, org, "q=BLOC")))
		assert.Equal(t, []string{"Bloc A"}, names(env.list(t, org, "q=70:B3:D5:7E")), "a DevEUI with separators")
		assert.Equal(t, []string{"Cazangerie"}, names(env.list(t, org, "q=mică")), "an address")
		assert.Empty(t, env.list(t, org, "q=%25").Devices, "a LIKE wildcard matches only itself")
	})

	t.Run("filters", func(t *testing.T) {
		assert.Equal(t, []string{"Cazangerie"}, names(env.list(t, org, "meter_type=gaz")))
		assert.Equal(t, []string{"Cazangerie"}, names(env.list(t, org, "model_id="+itoa(gas.ID))))
		assert.Equal(t, []string{"Vechi"}, names(env.list(t, org, "archived=true")))
	})

	t.Run("another organization sees none of them", func(t *testing.T) {
		other := env.organization(t)
		r := env.list(t, other, "")
		assert.Empty(t, r.Devices)
		assert.Zero(t, r.Total)
	})
}

func TestArchiveDevices(t *testing.T) {
	env := newTestEnv(t)
	org := env.organization(t)
	m := env.model(t, org, "Milesight", "EM300-DI")
	a := env.device(t, org, deviceRequest{DevEUI: randomDevEUI(), Name: "a", DeviceModelID: m.ID})
	b := env.device(t, org, deviceRequest{DevEUI: randomDevEUI(), Name: "b", DeviceModelID: m.ID})

	other := env.organization(t)
	theirModel := env.model(t, other, "X", "Y")
	theirs := env.device(t, other, deviceRequest{DevEUI: randomDevEUI(), Name: "theirs", DeviceModelID: theirModel.ID})

	t.Run("bulk archive leaves other organizations alone", func(t *testing.T) {
		rec := env.do(t, org, http.MethodPost, "/api/v1/devices/archive",
			archiveRequest{IDs: []int64{a.ID, b.ID, theirs.ID}})
		assert.Equal(t, archiveResponse{Archived: 2}, decode[archiveResponse](t, rec, http.StatusOK))
		assert.Empty(t, env.list(t, org, "").Devices)
		assert.Len(t, env.list(t, other, "").Devices, 1)
	})

	t.Run("restore brings a device back with its fields", func(t *testing.T) {
		require.Equal(t, http.StatusNoContent, env.do(t, org, http.MethodPost, devicePath(a.ID, "/restore"), nil).Code)
		got := decode[deviceResponse](t, env.do(t, org, http.MethodGet, devicePath(a.ID, ""), nil), http.StatusOK)
		assert.Nil(t, got.ArchivedAt)
		assert.Equal(t, a.DevEUI, got.DevEUI)
	})

	t.Run("archive of another organization's device is not found", func(t *testing.T) {
		rec := env.do(t, org, http.MethodPost, devicePath(theirs.ID, "/archive"), nil)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("stats count live devices", func(t *testing.T) {
		counts, err := env.stats.DevicesByOrganization(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 1, counts[org])
		assert.Equal(t, 1, counts[other])
	})
}
