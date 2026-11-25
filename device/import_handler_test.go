package device

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImport(t *testing.T) {
	env := newTestEnv(t)
	org := env.organization(t)
	m := env.model(t, org, "Milesight", "EM300-DI")
	env.group(t, org, "Botanica")
	present := randomDevEUI()
	env.device(t, org, deviceRequest{DevEUI: present, Name: "present", DeviceModelID: m.ID})
	archived := env.device(t, org, deviceRequest{DevEUI: randomDevEUI(), Name: "archived", DeviceModelID: m.ID})
	env.do(t, org, http.MethodPost, devicePath(archived.ID, "/archive"), nil)

	newA, newB := randomDevEUI(), randomDevEUI()
	file := bom + "dev_eui;name;vendor;model;type;groups\r\n" +
		newA + ";Bloc A;milesight;em300-di;apă;\"botanica;Sector 1\"\r\n" +
		newB + ";Cazangerie;Elvaco;CMi4110;газ;Sector 1\r\n" +
		present + ";skipped;Elvaco;Other;;Never\r\n" +
		archived.DevEUI + ";skipped too;V;M;;\r\n" +
		"12;broken;V;M;;\r\n"

	t.Run("preview tells each row and changes nothing", func(t *testing.T) {
		got := decode[importPreviewResponse](t, env.doCSV(t, org, "/api/v1/devices/import/preview", file), http.StatusOK)
		assert.Equal(t, importTotalsResponse{New: 2, Exists: 2, Error: 1, NewModels: 1, NewGroups: 1}, got.Totals,
			"the existing model and group match in any case; the skipped rows create nothing")
		require.Len(t, got.Rows, 5)
		assert.Equal(t, "new", got.Rows[0].Status)
		assert.False(t, got.Rows[0].NewModel)
		assert.Equal(t, []string{"Sector 1"}, got.Rows[0].NewGroups)
		assert.True(t, got.Rows[1].NewModel)
		assert.Equal(t, "exists", got.Rows[2].Status)
		assert.Equal(t, "exists", got.Rows[3].Status, "an archived device holds its DevEUI")
		assert.Equal(t, "error", got.Rows[4].Status)
		assert.Equal(t, "dev_eui_invalid", got.Rows[4].Code)
		assert.Equal(t, 6, got.Rows[4].Line)
		assert.Equal(t, 2, env.list(t, org, "").Total+env.list(t, org, "archived=true").Total)
	})

	t.Run("commit creates the new rows with their model and groups", func(t *testing.T) {
		got := decode[importTotalsResponse](t, env.doCSV(t, org, "/api/v1/devices/import", file), http.StatusOK)
		assert.Equal(t, importTotalsResponse{New: 2, Exists: 2, Error: 1, NewModels: 1, NewGroups: 1}, got)

		r := env.list(t, org, "q="+newB)
		require.Len(t, r.Devices, 1)
		d := r.Devices[0]
		assert.Equal(t, "Cazangerie", d.Name)
		assert.Equal(t, "Elvaco", d.DeviceModel.Vendor)
		require.NotNil(t, d.MeterType)
		assert.Equal(t, "gas", *d.MeterType)
		assert.Equal(t, []string{"Sector 1"}, groupNames(d.Groups))

		a := env.list(t, org, "q="+newA).Devices[0]
		assert.Equal(t, m.ID, a.DeviceModel.ID, "the existing model in another case")
		assert.Equal(t, []string{"Botanica", "Sector 1"}, groupNames(a.Groups))

		groups := decode[groupListResponse](t, env.do(t, org, http.MethodGet, "/api/v1/device-groups", nil), http.StatusOK)
		assert.Len(t, groups.DeviceGroups, 2, "Never belongs to a skipped row")
	})

	t.Run("a second commit of the same file creates nothing", func(t *testing.T) {
		got := decode[importTotalsResponse](t, env.doCSV(t, org, "/api/v1/devices/import", file), http.StatusOK)
		assert.Equal(t, importTotalsResponse{New: 0, Exists: 4, Error: 1}, got)
	})

	t.Run("another organization imports the same DevEUIs into its own registry", func(t *testing.T) {
		other := env.organization(t)
		got := decode[importTotalsResponse](t, env.doCSV(t, other, "/api/v1/devices/import", file), http.StatusOK)
		assert.Equal(t, 4, got.New)
		assert.Equal(t, 4, env.list(t, other, "").Total)
	})

	t.Run("file errors", func(t *testing.T) {
		rec := env.doCSV(t, org, "/api/v1/devices/import/preview", "dev_eui,name\n")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "import_missing_column", errorCode(t, rec))

		rec = env.do(t, org, http.MethodPost, "/api/v1/devices/import/preview", map[string]string{"a": "b"})
		assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)

		big := "dev_eui,name,vendor,model\n" + strings.Repeat("x", maxImportBytes)
		rec = env.doCSV(t, org, "/api/v1/devices/import/preview", big)
		assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	})

	t.Run("5000 rows in one request", func(t *testing.T) {
		org := env.organization(t)
		var b strings.Builder
		b.WriteString("dev_eui,name,vendor,model,groups\n")
		for i := range maxImportRows {
			fmt.Fprintf(&b, "%s,device %d,V,M,G%d\n", randomDevEUI(), i, i%10)
		}
		got := decode[importTotalsResponse](t, env.doCSV(t, org, "/api/v1/devices/import", b.String()), http.StatusOK)
		assert.Equal(t, importTotalsResponse{New: maxImportRows, NewModels: 1, NewGroups: 10}, got)
		assert.Equal(t, maxImportRows, env.list(t, org, "per_page=1").Total)
	})

	t.Run("the template parses as a valid file", func(t *testing.T) {
		rows, err := parseCSV([]byte(importTemplate))
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Nil(t, rows[0].Err)
		assert.Equal(t, []string{"Botanica", "Bloc 3"}, rows[0].Groups)
	})
}
