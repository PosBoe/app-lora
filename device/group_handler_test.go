package device

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func groupPath(id int64, suffix string) string {
	return fmt.Sprintf("/api/v1/device-groups/%d%s", id, suffix)
}

func groupNames(groups []groupRefResponse) []string {
	out := make([]string, len(groups))
	for i, g := range groups {
		out[i] = g.Name
	}
	return out
}

func TestDeviceGroups(t *testing.T) {
	env := newTestEnv(t)
	org := env.organization(t)
	m := env.model(t, org, "Milesight", "EM300-DI")

	t.Run("name rules", func(t *testing.T) {
		env.group(t, org, "Sector 1")
		rec := env.do(t, org, http.MethodPost, "/api/v1/device-groups", groupRequest{Name: " sector 1 "})
		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Equal(t, "group_taken", errorCode(t, rec))
		rec = env.do(t, org, http.MethodPost, "/api/v1/device-groups", groupRequest{Name: " "})
		assert.Equal(t, "group_name_required", errorCode(t, rec))
	})

	t.Run("a device takes its groups on create and update", func(t *testing.T) {
		a := env.group(t, org, "Botanica")
		b := env.group(t, org, "Ciocana")
		d := env.device(t, org, deviceRequest{
			DevEUI: randomDevEUI(), Name: "d", DeviceModelID: m.ID, GroupIDs: []int64{b.ID, a.ID, a.ID},
		})
		assert.Equal(t, []string{"Botanica", "Ciocana"}, groupNames(d.Groups), "by name, no repeat")

		rec := env.do(t, org, http.MethodPut, devicePath(d.ID, ""),
			deviceRequest{DevEUI: d.DevEUI, Name: "d", DeviceModelID: m.ID, GroupIDs: []int64{b.ID}})
		assert.Equal(t, []string{"Ciocana"}, groupNames(decode[deviceResponse](t, rec, http.StatusOK).Groups))

		rec = env.do(t, org, http.MethodPut, devicePath(d.ID, ""),
			deviceRequest{DevEUI: d.DevEUI, Name: "d", DeviceModelID: m.ID})
		assert.Empty(t, decode[deviceResponse](t, rec, http.StatusOK).Groups)
	})

	t.Run("a group of another organization is refused and nothing is written", func(t *testing.T) {
		other := env.organization(t)
		theirs := env.group(t, other, "Theirs")
		eui := randomDevEUI()
		rec := env.do(t, org, http.MethodPost, "/api/v1/devices",
			deviceRequest{DevEUI: eui, Name: "d", DeviceModelID: m.ID, GroupIDs: []int64{theirs.ID}})
		assert.Equal(t, "group_not_found", errorCode(t, rec))
		assert.Empty(t, env.list(t, org, "q="+eui).Devices, "the transaction rolled back")
	})

	t.Run("bulk add and remove, filter, and counts", func(t *testing.T) {
		org := env.organization(t)
		m := env.model(t, org, "V", "M")
		g := env.group(t, org, "Bulk")
		d1 := env.device(t, org, deviceRequest{DevEUI: randomDevEUI(), Name: "one", DeviceModelID: m.ID})
		d2 := env.device(t, org, deviceRequest{DevEUI: randomDevEUI(), Name: "two", DeviceModelID: m.ID})
		env.device(t, org, deviceRequest{DevEUI: randomDevEUI(), Name: "three", DeviceModelID: m.ID})

		other := env.organization(t)
		om := env.model(t, other, "V", "M")
		theirs := env.device(t, other, deviceRequest{DevEUI: randomDevEUI(), Name: "theirs", DeviceModelID: om.ID})

		rec := env.do(t, org, http.MethodPost, groupPath(g.ID, "/add"), membersRequest{IDs: []int64{d1.ID, d2.ID, theirs.ID}})
		assert.Equal(t, membersResponse{Changed: 2}, decode[membersResponse](t, rec, http.StatusOK), "another organization's id is left alone")
		rec = env.do(t, org, http.MethodPost, groupPath(g.ID, "/add"), membersRequest{IDs: []int64{d1.ID}})
		assert.Equal(t, membersResponse{Changed: 0}, decode[membersResponse](t, rec, http.StatusOK), "already in the group")

		r := env.list(t, org, fmt.Sprintf("group_id=%d", g.ID))
		assert.Equal(t, 2, r.Total)

		groups := decode[groupListResponse](t, env.do(t, org, http.MethodGet, "/api/v1/device-groups", nil), http.StatusOK)
		require.Len(t, groups.DeviceGroups, 1)
		assert.Equal(t, 2, groups.DeviceGroups[0].DeviceCount)

		env.do(t, org, http.MethodPost, devicePath(d2.ID, "/archive"), nil)
		groups = decode[groupListResponse](t, env.do(t, org, http.MethodGet, "/api/v1/device-groups", nil), http.StatusOK)
		assert.Equal(t, 1, groups.DeviceGroups[0].DeviceCount, "an archived device is not counted")

		rec = env.do(t, org, http.MethodPost, groupPath(g.ID, "/remove"), membersRequest{IDs: []int64{d1.ID, d2.ID}})
		assert.Equal(t, membersResponse{Changed: 2}, decode[membersResponse](t, rec, http.StatusOK))
		assert.Zero(t, env.list(t, org, fmt.Sprintf("group_id=%d", g.ID)).Total)
	})

	t.Run("delete keeps the devices", func(t *testing.T) {
		g := env.group(t, org, "Temporary")
		d := env.device(t, org, deviceRequest{DevEUI: randomDevEUI(), Name: "kept", DeviceModelID: m.ID, GroupIDs: []int64{g.ID}})
		require.Equal(t, http.StatusNoContent, env.do(t, org, http.MethodDelete, groupPath(g.ID, ""), nil).Code)

		got := decode[deviceResponse](t, env.do(t, org, http.MethodGet, devicePath(d.ID, ""), nil), http.StatusOK)
		assert.Empty(t, got.Groups)
	})

	t.Run("another organization cannot use a group", func(t *testing.T) {
		g := env.group(t, org, "Private")
		other := env.organization(t)
		assert.Equal(t, http.StatusNotFound, env.do(t, other, http.MethodPut, groupPath(g.ID, ""), groupRequest{Name: "x"}).Code)
		assert.Equal(t, http.StatusNotFound, env.do(t, other, http.MethodDelete, groupPath(g.ID, ""), nil).Code)
		assert.Equal(t, http.StatusNotFound, env.do(t, other, http.MethodPost, groupPath(g.ID, "/add"), membersRequest{IDs: []int64{1}}).Code)
		assert.Equal(t, http.StatusNotFound, env.do(t, other, http.MethodPost, groupPath(g.ID, "/remove"), membersRequest{IDs: []int64{1}}).Code)
	})
}
