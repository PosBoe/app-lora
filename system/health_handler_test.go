package system

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthDatabaseDown(t *testing.T) {
	// Port 1 refuses the connection at once, so the ping fails fast.
	pool, err := pgxpool.New(t.Context(), "postgres://lora:lora@127.0.0.1:1/lora?connect_timeout=1")
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	mux := http.NewServeMux()
	registerHealthHandler(mux, newHealthHandler(pool, slog.New(slog.NewTextHandler(io.Discard, nil))))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp healthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "degraded", resp.Status)
	assert.Equal(t, "error", resp.DB)
}
