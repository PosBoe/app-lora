package device

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/waneock/app-lora/core"
	"github.com/waneock/app-lora/pkg/clock"
	"github.com/waneock/app-lora/pkg/httpkit"
	"github.com/waneock/app-lora/pkg/pgtest"
)

// orgHeader names the organization of a test request. testGuard reads it in
// place of a session.
const orgHeader = "X-Test-Organization"

// testGuard lets every request in as a member of the organization in
// orgHeader. The core tests cover the real session guard.
type testGuard struct {
	logger *slog.Logger
}

func (g testGuard) Member(fn func(w http.ResponseWriter, req *http.Request) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		orgID, _ := strconv.ParseInt(req.Header.Get(orgHeader), 10, 64)
		ctx := core.ContextWithCaller(req.Context(), core.Caller{UserID: 1, OrganizationID: orgID})
		req = req.WithContext(ctx)
		if err := fn(w, req); err != nil {
			httpkit.WriteError(w, req, g.logger, err)
		}
	})
}

// testEnv is the device package on the test database, with a fixed clock
// and the routes on a real mux.
type testEnv struct {
	pool  *pgxpool.Pool
	clock *clock.Fixed
	mux   *http.ServeMux
	stats *Stats
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	pool := pgtest.Pool(t)
	clk := clock.NewFixed(time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC))
	g := testGuard{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

	mux := http.NewServeMux()
	registerDeviceHandler(mux, &deviceHandler{devices: newDeviceService(pool, clk), guard: g})
	registerModelHandler(mux, &modelHandler{models: newModelService(pool), guard: g})
	registerGroupHandler(mux, &groupHandler{groups: newGroupService(pool), guard: g})
	registerImportHandler(mux, &importHandler{imports: newImportService(pool), guard: g})
	return &testEnv{pool: pool, clock: clk, mux: mux, stats: newStats(pool)}
}

// organization creates an organization for one test. The core context has
// no API to create one outside the admin routes, so the test writes the row.
func (e *testEnv) organization(t *testing.T) int64 {
	t.Helper()
	var id int64
	err := e.pool.QueryRow(context.Background(),
		`INSERT INTO organizations (name) VALUES ($1) RETURNING id`, "device "+t.Name()).Scan(&id)
	require.NoError(t, err)
	return id
}

// randomDevEUI returns a DevEUI no other test run uses.
func randomDevEUI() string {
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

// do sends a request as a member of orgID. A non-nil body is sent as JSON.
func (e *testEnv) do(t *testing.T, orgID int64, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(data)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set(orgHeader, strconv.FormatInt(orgID, 10))
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	return rec
}

// decode reads a JSON answer and checks its status.
func decode[T any](t *testing.T, rec *httptest.ResponseRecorder, status int) T {
	t.Helper()
	require.Equal(t, status, rec.Code, rec.Body.String())
	var out T
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	return out
}

// errorCode returns the code of an error answer.
func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), rec.Body.String())
	return body.Code
}

// model creates a device model in orgID.
func (e *testEnv) model(t *testing.T, orgID int64, vendor, model string) modelResponse {
	t.Helper()
	rec := e.do(t, orgID, http.MethodPost, "/api/v1/device-models", modelRequest{Vendor: vendor, Model: model})
	return decode[modelResponse](t, rec, http.StatusCreated)
}

// device creates a device in orgID.
func (e *testEnv) device(t *testing.T, orgID int64, in deviceRequest) deviceResponse {
	t.Helper()
	rec := e.do(t, orgID, http.MethodPost, "/api/v1/devices", in)
	return decode[deviceResponse](t, rec, http.StatusCreated)
}

// list reads the device list of orgID with a query string.
func (e *testEnv) list(t *testing.T, orgID int64, query string) listResponse {
	t.Helper()
	rec := e.do(t, orgID, http.MethodGet, "/api/v1/devices?"+query, nil)
	return decode[listResponse](t, rec, http.StatusOK)
}

func devicePath(id int64, suffix string) string {
	return fmt.Sprintf("/api/v1/devices/%d%s", id, suffix)
}

// group creates a device group in orgID.
func (e *testEnv) group(t *testing.T, orgID int64, name string) groupResponse {
	t.Helper()
	rec := e.do(t, orgID, http.MethodPost, "/api/v1/device-groups", groupRequest{Name: name})
	return decode[groupResponse](t, rec, http.StatusCreated)
}

// doCSV sends a CSV body as a member of orgID.
func (e *testEnv) doCSV(t *testing.T, orgID int64, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "text/csv")
	req.Header.Set(orgHeader, strconv.FormatInt(orgID, 10))
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	return rec
}
