package device

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviceModels(t *testing.T) {
	env := newTestEnv(t)
	org := env.organization(t)

	t.Run("a duplicate in any case is refused", func(t *testing.T) {
		env.model(t, org, "Milesight", "EM300-DI")
		rec := env.do(t, org, http.MethodPost, "/api/v1/device-models",
			modelRequest{Vendor: " milesight ", Model: "em300-di"})
		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Equal(t, "model_taken", errorCode(t, rec))
	})

	t.Run("vendor and model name are required", func(t *testing.T) {
		rec := env.do(t, org, http.MethodPost, "/api/v1/device-models", modelRequest{Vendor: "", Model: "x"})
		assert.Equal(t, "vendor_required", errorCode(t, rec))
		rec = env.do(t, org, http.MethodPost, "/api/v1/device-models", modelRequest{Vendor: "x", Model: " "})
		assert.Equal(t, "model_name_required", errorCode(t, rec))
	})

	t.Run("list counts the devices of each model", func(t *testing.T) {
		org := env.organization(t)
		used := env.model(t, org, "B-Vendor", "Used")
		env.model(t, org, "a-vendor", "Free")
		env.device(t, org, deviceRequest{DevEUI: randomDevEUI(), Name: "d", DeviceModelID: used.ID})

		got := decode[modelListResponse](t, env.do(t, org, http.MethodGet, "/api/v1/device-models", nil), http.StatusOK)
		assert.Len(t, got.DeviceModels, 2)
		assert.Equal(t, "a-vendor", got.DeviceModels[0].Vendor, "sorted by vendor in any case")
		assert.Equal(t, 0, got.DeviceModels[0].DeviceCount)
		assert.Equal(t, 1, got.DeviceModels[1].DeviceCount)
	})

	t.Run("rename and delete", func(t *testing.T) {
		org := env.organization(t)
		used := env.model(t, org, "V", "Used")
		free := env.model(t, org, "V", "Free")
		env.device(t, org, deviceRequest{DevEUI: randomDevEUI(), Name: "d", DeviceModelID: used.ID})

		rec := env.do(t, org, http.MethodPut, modelPath(free.ID), modelRequest{Vendor: "V", Model: "Renamed"})
		assert.Equal(t, http.StatusNoContent, rec.Code)
		rec = env.do(t, org, http.MethodPut, modelPath(free.ID), modelRequest{Vendor: "v", Model: "used"})
		assert.Equal(t, "model_taken", errorCode(t, rec))

		rec = env.do(t, org, http.MethodDelete, modelPath(used.ID), nil)
		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Equal(t, "model_in_use", errorCode(t, rec))
		assert.Equal(t, http.StatusNoContent, env.do(t, org, http.MethodDelete, modelPath(free.ID), nil).Code)
	})

	t.Run("another organization cannot see or change a model", func(t *testing.T) {
		mine := env.model(t, org, "Mine", "Only")
		other := env.organization(t)
		got := decode[modelListResponse](t, env.do(t, other, http.MethodGet, "/api/v1/device-models", nil), http.StatusOK)
		assert.Empty(t, got.DeviceModels)
		assert.Equal(t, http.StatusNotFound, env.do(t, other, http.MethodPut, modelPath(mine.ID), modelRequest{Vendor: "a", Model: "b"}).Code)
		assert.Equal(t, http.StatusNotFound, env.do(t, other, http.MethodDelete, modelPath(mine.ID), nil).Code)
	})
}

func modelPath(id int64) string {
	return fmt.Sprintf("/api/v1/device-models/%d", id)
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
