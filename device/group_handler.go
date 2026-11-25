package device

import (
	"context"
	"net/http"
	"time"

	"github.com/waneock/app-lora/core"
	"github.com/waneock/app-lora/pkg/httpkit"
)

type groupHandler struct {
	groups *groupService
	guard  guard
}

func newGroupHandler(groups *groupService, g *core.Guard) *groupHandler {
	return &groupHandler{groups: groups, guard: g}
}

func registerGroupHandler(mux *http.ServeMux, h *groupHandler) {
	mux.Handle("GET /api/v1/device-groups", member(h.guard, h.list))
	mux.Handle("POST /api/v1/device-groups", member(h.guard, h.create))
	mux.Handle("PUT /api/v1/device-groups/{id}", member(h.guard, h.rename))
	mux.Handle("DELETE /api/v1/device-groups/{id}", member(h.guard, h.delete))
	mux.Handle("POST /api/v1/device-groups/{id}/add", member(h.guard, h.add))
	mux.Handle("POST /api/v1/device-groups/{id}/remove", member(h.guard, h.remove))
}

// groupResponse is a device group. The frontend mirrors it in
// vue/src/device/use-device-groups.ts.
type groupResponse struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	DeviceCount int       `json:"device_count"`
	CreatedAt   time.Time `json:"created_at"`
}

func newGroupResponse(g *deviceGroup) groupResponse {
	return groupResponse{ID: g.ID, Name: g.Name, DeviceCount: g.DeviceCount, CreatedAt: g.CreatedAt}
}

type groupRequest struct {
	Name string `json:"name"`
}

type groupListResponse struct {
	DeviceGroups []groupResponse `json:"device_groups"`
}

type membersRequest struct {
	IDs []int64 `json:"ids"`
}

type membersResponse struct {
	Changed int64 `json:"changed"`
}

func (h *groupHandler) list(w http.ResponseWriter, req *http.Request, orgID int64) error {
	groups, err := h.groups.list(req.Context(), orgID)
	if err != nil {
		return err
	}
	resp := groupListResponse{DeviceGroups: make([]groupResponse, len(groups))}
	for i, g := range groups {
		resp.DeviceGroups[i] = newGroupResponse(g)
	}
	httpkit.JSON(w, http.StatusOK, resp)
	return nil
}

func (h *groupHandler) create(w http.ResponseWriter, req *http.Request, orgID int64) error {
	var in groupRequest
	if err := httpkit.DecodeJSON(w, req, &in); err != nil {
		return err
	}
	g, err := h.groups.create(req.Context(), orgID, in.Name)
	if err != nil {
		return err
	}
	httpkit.JSON(w, http.StatusCreated, newGroupResponse(g))
	return nil
}

func (h *groupHandler) rename(w http.ResponseWriter, req *http.Request, orgID int64) error {
	id, err := pathID(req)
	if err != nil {
		return err
	}
	var in groupRequest
	if err := httpkit.DecodeJSON(w, req, &in); err != nil {
		return err
	}
	if err := h.groups.rename(req.Context(), orgID, id, in.Name); err != nil {
		return err
	}
	httpkit.NoContent(w)
	return nil
}

func (h *groupHandler) delete(w http.ResponseWriter, req *http.Request, orgID int64) error {
	id, err := pathID(req)
	if err != nil {
		return err
	}
	if err := h.groups.delete(req.Context(), orgID, id); err != nil {
		return err
	}
	httpkit.NoContent(w)
	return nil
}

func (h *groupHandler) add(w http.ResponseWriter, req *http.Request, orgID int64) error {
	return h.members(w, req, orgID, h.groups.add)
}

func (h *groupHandler) remove(w http.ResponseWriter, req *http.Request, orgID int64) error {
	return h.members(w, req, orgID, h.groups.remove)
}

// members reads the group and the device ids and runs one bulk change.
func (h *groupHandler) members(w http.ResponseWriter, req *http.Request, orgID int64,
	change func(ctx context.Context, orgID, groupID int64, deviceIDs []int64) (int64, error),
) error {
	id, err := pathID(req)
	if err != nil {
		return err
	}
	var in membersRequest
	if err := httpkit.DecodeJSON(w, req, &in); err != nil {
		return err
	}
	n, err := change(req.Context(), orgID, id, uniqueIDs(in.IDs))
	if err != nil {
		return err
	}
	httpkit.JSON(w, http.StatusOK, membersResponse{Changed: n})
	return nil
}
