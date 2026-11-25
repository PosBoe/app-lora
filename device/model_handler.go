package device

import (
	"net/http"
	"time"

	"github.com/waneock/app-lora/core"
	"github.com/waneock/app-lora/pkg/httpkit"
)

type modelHandler struct {
	models *modelService
	guard  guard
}

func newModelHandler(models *modelService, g *core.Guard) *modelHandler {
	return &modelHandler{models: models, guard: g}
}

func registerModelHandler(mux *http.ServeMux, h *modelHandler) {
	mux.Handle("GET /api/v1/device-models", member(h.guard, h.list))
	mux.Handle("POST /api/v1/device-models", member(h.guard, h.create))
	mux.Handle("PUT /api/v1/device-models/{id}", member(h.guard, h.rename))
	mux.Handle("DELETE /api/v1/device-models/{id}", member(h.guard, h.delete))
}

// modelResponse is a device model. The frontend mirrors it in
// vue/src/device/use-device-models.ts.
type modelResponse struct {
	ID          int64     `json:"id"`
	Vendor      string    `json:"vendor"`
	Model       string    `json:"model"`
	DeviceCount int       `json:"device_count"`
	CreatedAt   time.Time `json:"created_at"`
}

func newModelResponse(m *deviceModel) modelResponse {
	return modelResponse{
		ID:          m.ID,
		Vendor:      m.Vendor,
		Model:       m.Model,
		DeviceCount: m.DeviceCount,
		CreatedAt:   m.CreatedAt,
	}
}

type modelRequest struct {
	Vendor string `json:"vendor"`
	Model  string `json:"model"`
}

type modelListResponse struct {
	DeviceModels []modelResponse `json:"device_models"`
}

func (h *modelHandler) list(w http.ResponseWriter, req *http.Request, orgID int64) error {
	models, err := h.models.list(req.Context(), orgID)
	if err != nil {
		return err
	}
	resp := modelListResponse{DeviceModels: make([]modelResponse, len(models))}
	for i, m := range models {
		resp.DeviceModels[i] = newModelResponse(m)
	}
	httpkit.JSON(w, http.StatusOK, resp)
	return nil
}

func (h *modelHandler) create(w http.ResponseWriter, req *http.Request, orgID int64) error {
	var in modelRequest
	if err := httpkit.DecodeJSON(w, req, &in); err != nil {
		return err
	}
	m, err := h.models.create(req.Context(), orgID, in.Vendor, in.Model)
	if err != nil {
		return err
	}
	httpkit.JSON(w, http.StatusCreated, newModelResponse(m))
	return nil
}

func (h *modelHandler) rename(w http.ResponseWriter, req *http.Request, orgID int64) error {
	id, err := pathID(req)
	if err != nil {
		return err
	}
	var in modelRequest
	if err := httpkit.DecodeJSON(w, req, &in); err != nil {
		return err
	}
	if err := h.models.rename(req.Context(), orgID, id, in.Vendor, in.Model); err != nil {
		return err
	}
	httpkit.NoContent(w)
	return nil
}

func (h *modelHandler) delete(w http.ResponseWriter, req *http.Request, orgID int64) error {
	id, err := pathID(req)
	if err != nil {
		return err
	}
	if err := h.models.delete(req.Context(), orgID, id); err != nil {
		return err
	}
	httpkit.NoContent(w)
	return nil
}
