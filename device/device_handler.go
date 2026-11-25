package device

import (
	"net/http"
	"strconv"
	"time"

	"github.com/waneock/app-lora/core"
	"github.com/waneock/app-lora/pkg/httpkit"
)

const (
	defaultPerPage = 50
	maxPerPage     = 200
)

type deviceHandler struct {
	devices *deviceService
	guard   guard
}

func newDeviceHandler(devices *deviceService, g *core.Guard) *deviceHandler {
	return &deviceHandler{devices: devices, guard: g}
}

func registerDeviceHandler(mux *http.ServeMux, h *deviceHandler) {
	mux.Handle("GET /api/v1/devices", member(h.guard, h.list))
	mux.Handle("POST /api/v1/devices", member(h.guard, h.create))
	mux.Handle("POST /api/v1/devices/archive", member(h.guard, h.archiveMany))
	mux.Handle("GET /api/v1/devices/{id}", member(h.guard, h.get))
	mux.Handle("PUT /api/v1/devices/{id}", member(h.guard, h.update))
	mux.Handle("POST /api/v1/devices/{id}/archive", member(h.guard, h.archiveOne))
	mux.Handle("POST /api/v1/devices/{id}/restore", member(h.guard, h.restore))
}

// deviceResponse is a device. The frontend mirrors it in
// vue/src/device/use-devices.ts.
type deviceResponse struct {
	ID          int64              `json:"id"`
	DevEUI      string             `json:"dev_eui"`
	Name        string             `json:"name"`
	DeviceModel modelRefResponse   `json:"device_model"`
	MeterType   *string            `json:"meter_type"`
	Address     *string            `json:"address"`
	Note        *string            `json:"note"`
	ArchivedAt  *time.Time         `json:"archived_at"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	Groups      []groupRefResponse `json:"groups"`
}

type groupRefResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type modelRefResponse struct {
	ID     int64  `json:"id"`
	Vendor string `json:"vendor"`
	Model  string `json:"model"`
}

func newDeviceResponse(d *device) deviceResponse {
	groups := make([]groupRefResponse, len(d.Groups))
	for i, g := range d.Groups {
		groups[i] = groupRefResponse(g)
	}
	return deviceResponse{
		Groups:      groups,
		ID:          d.ID,
		DevEUI:      d.DevEUI,
		Name:        d.Name,
		DeviceModel: modelRefResponse{ID: d.DeviceModelID, Vendor: d.Vendor, Model: d.Model},
		MeterType:   d.MeterType,
		Address:     d.Address,
		Note:        d.Note,
		ArchivedAt:  d.ArchivedAt,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

// deviceRequest is the body of a create and of an update. An empty optional
// field clears it.
type deviceRequest struct {
	DevEUI        string  `json:"dev_eui"`
	Name          string  `json:"name"`
	DeviceModelID int64   `json:"device_model_id"`
	MeterType     string  `json:"meter_type"`
	Address       string  `json:"address"`
	Note          string  `json:"note"`
	GroupIDs      []int64 `json:"group_ids"`
}

func (r deviceRequest) fields() deviceFields {
	return deviceFields(r)
}

type listResponse struct {
	Devices []deviceResponse `json:"devices"`
	Total   int              `json:"total"`
}

func (h *deviceHandler) list(w http.ResponseWriter, req *http.Request, orgID int64) error {
	q, err := parseListQuery(req)
	if err != nil {
		return err
	}
	devices, total, err := h.devices.list(req.Context(), orgID, q)
	if err != nil {
		return err
	}
	resp := listResponse{Devices: make([]deviceResponse, len(devices)), Total: total}
	for i, d := range devices {
		resp.Devices[i] = newDeviceResponse(d)
	}
	httpkit.JSON(w, http.StatusOK, resp)
	return nil
}

// parseListQuery reads q, model_id, group_id, meter_type, archived, page, and
// per_page.
// A page or a size out of range falls back to the first page or the default
// size.
func parseListQuery(req *http.Request) (listQuery, error) {
	v := req.URL.Query()
	q := listQuery{Limit: defaultPerPage, Archived: v.Get("archived") == "true"}

	if s := v.Get("q"); s != "" {
		q.Pattern = searchPattern(s)
	}
	if id, err := strconv.ParseInt(v.Get("model_id"), 10, 64); err == nil && id > 0 {
		q.ModelID = id
	}
	if id, err := strconv.ParseInt(v.Get("group_id"), 10, 64); err == nil && id > 0 {
		q.GroupID = id
	}
	meterType, err := normalizeMeterType(v.Get("meter_type"))
	if err != nil {
		return q, err
	}
	q.MeterType = meterType
	if n, err := strconv.Atoi(v.Get("per_page")); err == nil && n > 0 && n <= maxPerPage {
		q.Limit = n
	}
	if page, err := strconv.Atoi(v.Get("page")); err == nil && page > 1 {
		q.Offset = (page - 1) * q.Limit
	}
	return q, nil
}

func (h *deviceHandler) create(w http.ResponseWriter, req *http.Request, orgID int64) error {
	var in deviceRequest
	if err := httpkit.DecodeJSON(w, req, &in); err != nil {
		return err
	}
	d, err := h.devices.create(req.Context(), orgID, in.fields())
	if err != nil {
		return err
	}
	httpkit.JSON(w, http.StatusCreated, newDeviceResponse(d))
	return nil
}

func (h *deviceHandler) get(w http.ResponseWriter, req *http.Request, orgID int64) error {
	id, err := pathID(req)
	if err != nil {
		return err
	}
	d, err := h.devices.get(req.Context(), orgID, id)
	if err != nil {
		return err
	}
	httpkit.JSON(w, http.StatusOK, newDeviceResponse(d))
	return nil
}

func (h *deviceHandler) update(w http.ResponseWriter, req *http.Request, orgID int64) error {
	id, err := pathID(req)
	if err != nil {
		return err
	}
	var in deviceRequest
	if err := httpkit.DecodeJSON(w, req, &in); err != nil {
		return err
	}
	d, err := h.devices.update(req.Context(), orgID, id, in.fields())
	if err != nil {
		return err
	}
	httpkit.JSON(w, http.StatusOK, newDeviceResponse(d))
	return nil
}

func (h *deviceHandler) archiveOne(w http.ResponseWriter, req *http.Request, orgID int64) error {
	id, err := pathID(req)
	if err != nil {
		return err
	}
	if _, err := h.devices.get(req.Context(), orgID, id); err != nil {
		return err
	}
	if _, err := h.devices.archive(req.Context(), orgID, []int64{id}); err != nil {
		return err
	}
	httpkit.NoContent(w)
	return nil
}

type archiveRequest struct {
	IDs []int64 `json:"ids"`
}

type archiveResponse struct {
	Archived int64 `json:"archived"`
}

func (h *deviceHandler) archiveMany(w http.ResponseWriter, req *http.Request, orgID int64) error {
	var in archiveRequest
	if err := httpkit.DecodeJSON(w, req, &in); err != nil {
		return err
	}
	n, err := h.devices.archive(req.Context(), orgID, in.IDs)
	if err != nil {
		return err
	}
	httpkit.JSON(w, http.StatusOK, archiveResponse{Archived: n})
	return nil
}

func (h *deviceHandler) restore(w http.ResponseWriter, req *http.Request, orgID int64) error {
	id, err := pathID(req)
	if err != nil {
		return err
	}
	if err := h.devices.restore(req.Context(), orgID, id); err != nil {
		return err
	}
	httpkit.NoContent(w)
	return nil
}

// pathID reads the {id} path value. A value that is not a positive integer
// names nothing, so it answers errNotFound.
func pathID(req *http.Request) (int64, error) {
	id, err := strconv.ParseInt(req.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errNotFound
	}
	return id, nil
}
