package device

import (
	"net/http"
	"strings"
	"time"

	"github.com/waneock/app-lora/core"
	"github.com/waneock/app-lora/pkg/appconf"
	"github.com/waneock/app-lora/pkg/httpkit"
)

type integrationHandler struct {
	integrations *integrationService
	guard        guard
	// siteURL is the public address of the platform, for the ingest URL.
	siteURL string
}

func newIntegrationHandler(integrations *integrationService, g *core.Guard, conf *appconf.Config) *integrationHandler {
	return &integrationHandler{
		integrations: integrations,
		guard:        g,
		siteURL:      strings.TrimRight(conf.Site.URL, "/"),
	}
}

func registerIntegrationHandler(mux *http.ServeMux, h *integrationHandler) {
	mux.Handle("GET /api/v1/integrations", member(h.guard, h.list))
	mux.Handle("POST /api/v1/integrations", member(h.guard, h.create))
	mux.Handle("PUT /api/v1/integrations/{id}", member(h.guard, h.rename))
	mux.Handle("DELETE /api/v1/integrations/{id}", member(h.guard, h.delete))
	mux.Handle("POST /api/v1/integrations/{id}/secret", member(h.guard, h.regenerate))
}

// ingestPath is the route that receives the uplinks of an integration kind.
func ingestPath(kind string) string {
	return "/api/v1/ingest/" + kind
}

// integrationResponse is an integration with its status. The frontend
// mirrors it in vue/src/device/use-integrations.ts.
type integrationResponse struct {
	ID            int64                `json:"id"`
	Name          string               `json:"name"`
	Kind          string               `json:"kind"`
	IngestURL     string               `json:"ingest_url"`
	CreatedAt     time.Time            `json:"created_at"`
	LastUplinkAt  *time.Time           `json:"last_uplink_at"`
	RecentUplinks int                  `json:"uplinks_24h"`
	LastError     *ingestErrorResponse `json:"last_error"`
}

type ingestErrorResponse struct {
	At      time.Time `json:"at"`
	Code    string    `json:"code"`
	Message string    `json:"message"`
}

func (h *integrationHandler) response(in *integration) integrationResponse {
	resp := integrationResponse{
		ID:            in.ID,
		Name:          in.Name,
		Kind:          in.Kind,
		IngestURL:     h.siteURL + ingestPath(in.Kind),
		CreatedAt:     in.CreatedAt,
		LastUplinkAt:  in.LastUplinkAt,
		RecentUplinks: in.RecentUplinks,
	}
	if in.LastErrorAt != nil && in.LastErrorCode != nil {
		resp.LastError = &ingestErrorResponse{At: *in.LastErrorAt, Code: *in.LastErrorCode}
		if in.LastErrorMessage != nil {
			resp.LastError.Message = *in.LastErrorMessage
		}
	}
	return resp
}

type integrationRequest struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type integrationListResponse struct {
	Integrations []integrationResponse `json:"integrations"`
}

// secretResponse holds a secret, shown one time.
type secretResponse struct {
	Integration integrationResponse `json:"integration"`
	Secret      string              `json:"secret"`
}

func (h *integrationHandler) list(w http.ResponseWriter, req *http.Request, orgID int64) error {
	ins, err := h.integrations.list(req.Context(), orgID)
	if err != nil {
		return err
	}
	resp := integrationListResponse{Integrations: make([]integrationResponse, len(ins))}
	for i, in := range ins {
		resp.Integrations[i] = h.response(in)
	}
	httpkit.JSON(w, http.StatusOK, resp)
	return nil
}

func (h *integrationHandler) create(w http.ResponseWriter, req *http.Request, orgID int64) error {
	var in integrationRequest
	if err := httpkit.DecodeJSON(w, req, &in); err != nil {
		return err
	}
	created, secret, err := h.integrations.create(req.Context(), orgID, in.Name, in.Kind)
	if err != nil {
		return err
	}
	httpkit.JSON(w, http.StatusCreated, secretResponse{Integration: h.response(created), Secret: secret})
	return nil
}

func (h *integrationHandler) rename(w http.ResponseWriter, req *http.Request, orgID int64) error {
	id, err := pathID(req)
	if err != nil {
		return err
	}
	var in integrationRequest
	if err := httpkit.DecodeJSON(w, req, &in); err != nil {
		return err
	}
	if err := h.integrations.rename(req.Context(), orgID, id, in.Name); err != nil {
		return err
	}
	httpkit.NoContent(w)
	return nil
}

func (h *integrationHandler) delete(w http.ResponseWriter, req *http.Request, orgID int64) error {
	id, err := pathID(req)
	if err != nil {
		return err
	}
	if err := h.integrations.delete(req.Context(), orgID, id); err != nil {
		return err
	}
	httpkit.NoContent(w)
	return nil
}

type regenerateResponse struct {
	Secret string `json:"secret"`
}

func (h *integrationHandler) regenerate(w http.ResponseWriter, req *http.Request, orgID int64) error {
	id, err := pathID(req)
	if err != nil {
		return err
	}
	secret, err := h.integrations.regenerate(req.Context(), orgID, id)
	if err != nil {
		return err
	}
	httpkit.JSON(w, http.StatusOK, regenerateResponse{Secret: secret})
	return nil
}
