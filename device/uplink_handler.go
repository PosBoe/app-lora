package device

import (
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/waneock/app-lora/core"
	"github.com/waneock/app-lora/pkg/httpkit"
)

// unmatchedKeep is how long an unmatched uplink stays (device:D4).
const unmatchedKeep = 30 * 24 * time.Hour

type uplinkHandler struct {
	devices *deviceService
	uplinks *uplinkRepo
	guard   guard
}

func newUplinkHandler(devices *deviceService, g *core.Guard) *uplinkHandler {
	return &uplinkHandler{devices: devices, uplinks: devices.uplinks, guard: g}
}

func registerUplinkHandler(mux *http.ServeMux, h *uplinkHandler) {
	mux.Handle("GET /api/v1/devices/{id}/uplinks", member(h.guard, h.forDevice))
	mux.Handle("GET /api/v1/unmatched", member(h.guard, h.unmatched))
}

// uplinkResponse is one uplink of a device. The frontend mirrors it in
// vue/src/device/use-uplinks.ts.
type uplinkResponse struct {
	ID         int64     `json:"id"`
	ReceivedAt time.Time `json:"received_at"`
	FCnt       int64     `json:"f_cnt"`
	PayloadHex string    `json:"payload_hex"`
}

type uplinkListResponse struct {
	Uplinks []uplinkResponse `json:"uplinks"`
	Total   int              `json:"total"`
}

func (h *uplinkHandler) forDevice(w http.ResponseWriter, req *http.Request, orgID int64) error {
	id, err := pathID(req)
	if err != nil {
		return err
	}
	if _, err := h.devices.get(req.Context(), orgID, id); err != nil {
		return err
	}
	offset := 0
	if page, err := strconv.Atoi(req.URL.Query().Get("page")); err == nil && page > 1 {
		offset = (page - 1) * defaultPerPage
	}
	ups, total, err := h.uplinks.listForDevice(req.Context(), orgID, id, defaultPerPage, offset)
	if err != nil {
		return err
	}
	resp := uplinkListResponse{Uplinks: make([]uplinkResponse, len(ups)), Total: total}
	for i, u := range ups {
		resp.Uplinks[i] = uplinkResponse{
			ID:         u.ID,
			ReceivedAt: u.ReceivedAt,
			FCnt:       u.FCnt,
			PayloadHex: hex.EncodeToString(u.Payload),
		}
	}
	httpkit.JSON(w, http.StatusOK, resp)
	return nil
}

// unmatchedResponse sums up the unmatched uplinks of one DevEUI. The
// frontend mirrors it in vue/src/device/use-uplinks.ts.
type unmatchedResponse struct {
	DevEUI          string    `json:"dev_eui"`
	Count           int       `json:"count"`
	LastReceivedAt  time.Time `json:"last_received_at"`
	IntegrationName *string   `json:"integration_name"`
	// DeletesAt is when the oldest of these uplinks is deleted.
	DeletesAt time.Time `json:"deletes_at"`
}

type unmatchedListResponse struct {
	DevEUIs []unmatchedResponse `json:"dev_euis"`
}

func (h *uplinkHandler) unmatched(w http.ResponseWriter, req *http.Request, orgID int64) error {
	xs, err := h.uplinks.unmatched(req.Context(), orgID)
	if err != nil {
		return err
	}
	resp := unmatchedListResponse{DevEUIs: make([]unmatchedResponse, len(xs))}
	for i, x := range xs {
		resp.DevEUIs[i] = unmatchedResponse{
			DevEUI:          x.DevEUI,
			Count:           x.Count,
			LastReceivedAt:  x.LastReceivedAt,
			IntegrationName: x.IntegrationName,
			DeletesAt:       x.FirstStoredAt.Add(unmatchedKeep),
		}
	}
	httpkit.JSON(w, http.StatusOK, resp)
	return nil
}
