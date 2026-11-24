package system

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waneock/app-lora/pkg/version"
)

const pingTimeout = 2 * time.Second

type healthHandler struct {
	pg     *pgxpool.Pool
	logger *slog.Logger
}

func newHealthHandler(pg *pgxpool.Pool, logger *slog.Logger) *healthHandler {
	return &healthHandler{pg: pg, logger: logger}
}

func registerHealthHandler(mux *http.ServeMux, h *healthHandler) {
	mux.HandleFunc("GET /api/v1/health", h.health)
}

type healthResponse struct {
	Status  string `json:"status"`
	DB      string `json:"db"`
	Version string `json:"version"`
}

// health answers 200 when the database answers a ping, and 503 when it does
// not.
func (h *healthHandler) health(w http.ResponseWriter, req *http.Request) {
	resp := healthResponse{Status: "ok", DB: "ok", Version: version.Version}
	status := http.StatusOK

	ctx, cancel := context.WithTimeout(req.Context(), pingTimeout)
	defer cancel()
	if err := h.pg.Ping(ctx); err != nil {
		h.logger.WarnContext(ctx, "health: pg ping failed", slog.Any("err", err))
		resp.Status = "degraded"
		resp.DB = "error"
		status = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}
