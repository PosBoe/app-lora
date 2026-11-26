package device

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waneock/app-lora/core"
	"github.com/waneock/app-lora/device/decoder"
	"github.com/waneock/app-lora/pkg/httpkit"
)

// OrganizationNames names the organizations for the admin device model
// page. The core context implements it; the composition root binds it, so
// device reads no table of core.
type OrganizationNames interface {
	OrganizationNames(ctx context.Context) (map[int64]string, error)
}

// adminGuard lets in only an admin. *core.Guard is the production guard.
type adminGuard interface {
	Admin(fn func(w http.ResponseWriter, req *http.Request) error) http.Handler
}

type adminModelHandler struct {
	pool  *pgxpool.Pool
	names OrganizationNames
	guard adminGuard
}

func newAdminModelHandler(pool *pgxpool.Pool, names OrganizationNames, g *core.Guard) *adminModelHandler {
	return &adminModelHandler{pool: pool, names: names, guard: g}
}

func registerAdminModelHandler(mux *http.ServeMux, h *adminModelHandler) {
	mux.Handle("GET /api/v1/admin/device-models", h.admin(h.list))
	mux.Handle("PUT /api/v1/admin/device-models/{id}/decoder", h.admin(h.setDecoder))
	mux.Handle("GET /api/v1/admin/decoders", h.admin(h.decoders))
}

// admin wraps a route with the admin guard and maps the rule errors.
func (h *adminModelHandler) admin(fn func(w http.ResponseWriter, req *http.Request) error) http.Handler {
	return h.guard.Admin(func(w http.ResponseWriter, req *http.Request) error {
		return toHTTP(fn(w, req))
	})
}

// adminModelResponse is a device model of any organization. The frontend
// mirrors it in vue/src/device/use-admin-models.ts.
type adminModelResponse struct {
	ID               int64   `json:"id"`
	OrganizationID   int64   `json:"organization_id"`
	OrganizationName string  `json:"organization_name"`
	Vendor           string  `json:"vendor"`
	Model            string  `json:"model"`
	Decoder          *string `json:"decoder"`
	DeviceCount      int     `json:"device_count"`
	PendingUplinks   int     `json:"pending_uplinks"`
}

type adminModelListResponse struct {
	DeviceModels []adminModelResponse `json:"device_models"`
}

func (h *adminModelHandler) list(w http.ResponseWriter, req *http.Request) error {
	models, err := newModelRepo(h.pool).listAll(req.Context())
	if err != nil {
		return err
	}
	names, err := h.names.OrganizationNames(req.Context())
	if err != nil {
		return err
	}
	resp := adminModelListResponse{DeviceModels: make([]adminModelResponse, len(models))}
	for i, m := range models {
		resp.DeviceModels[i] = adminModelResponse{
			ID: m.ID, OrganizationID: m.OrganizationID, OrganizationName: names[m.OrganizationID],
			Vendor: m.Vendor, Model: m.Model, Decoder: m.Decoder,
			DeviceCount: m.DeviceCount, PendingUplinks: m.PendingUplinks,
		}
	}
	httpkit.JSON(w, http.StatusOK, resp)
	return nil
}

type setDecoderRequest struct {
	Decoder *string `json:"decoder"`
}

// setDecoder assigns a decoder to a device model, or removes it with null,
// and marks the uplinks of its devices to decode again (device:G7).
func (h *adminModelHandler) setDecoder(w http.ResponseWriter, req *http.Request) error {
	id, err := pathID(req)
	if err != nil {
		return err
	}
	var in setDecoderRequest
	if err := httpkit.DecodeJSON(w, req, &in); err != nil {
		return err
	}
	if in.Decoder != nil {
		if _, ok := decoder.Get(*in.Decoder); !ok {
			return errDecoderUnknown
		}
	}
	err = pgx.BeginFunc(req.Context(), h.pool, func(tx pgx.Tx) error {
		if err := newModelRepo(tx).setDecoder(req.Context(), id, in.Decoder); err != nil {
			return err
		}
		return newDecodeRepo(tx).markModel(req.Context(), id)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return errNotFound
	}
	if err != nil {
		return err
	}
	httpkit.NoContent(w)
	return nil
}

type decoderResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type decoderListResponse struct {
	Decoders []decoderResponse `json:"decoders"`
}

func (h *adminModelHandler) decoders(w http.ResponseWriter, _ *http.Request) error {
	all := decoder.All()
	resp := decoderListResponse{Decoders: make([]decoderResponse, len(all))}
	for i, d := range all {
		resp.Decoders[i] = decoderResponse{ID: d.ID(), Name: d.Name()}
	}
	httpkit.JSON(w, http.StatusOK, resp)
	return nil
}
