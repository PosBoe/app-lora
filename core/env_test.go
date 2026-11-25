package core

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/waneock/app-lora/pkg/appconf"
	"github.com/waneock/app-lora/pkg/clock"
	"github.com/waneock/app-lora/pkg/pgtest"
)

// testPassword is the password that testEnv.member gives a user.
const testPassword = "correct-horse-battery"

// testEnv is the core package wired on the test database, with a fixed
// clock and the routes on a real mux.
type testEnv struct {
	pool  *pgxpool.Pool
	clock *clock.Fixed
	auth  *authService
	admin *adminService
	conf  *appconf.Config
	mux   *http.ServeMux
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	pool := pgtest.Pool(t)

	conf, err := appconf.Parse([]byte("pg:\n  dsn: unused\nsite:\n  url: https://lora.test\n"))
	require.NoError(t, err)
	clk := clock.NewFixed(time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC))
	pw, err := newPasswordsWithCost(bcrypt.MinCost)
	require.NoError(t, err)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	auth := newAuthService(pool, pw, clk, conf)
	admin := newAdminService(pool, pw, clk, fixedCounts{})
	middleware := newGuard(auth, conf, logger)
	mux := http.NewServeMux()
	registerAuthHandler(mux, newAuthHandler(auth, middleware))
	registerAdminHandler(mux, newAdminHandler(admin, middleware))

	return &testEnv{pool: pool, clock: clk, auth: auth, admin: admin, conf: conf, mux: mux}
}

// uniqueLogin returns a login no other test run uses.
func uniqueLogin(t *testing.T) string {
	t.Helper()
	buf := make([]byte, 6)
	_, _ = rand.Read(buf)
	return "t-" + hex.EncodeToString(buf)
}

// member creates an organization with one user whose password is
// testPassword and is not temporary.
func (e *testEnv) member(t *testing.T) (login string, userID, organizationID int64) {
	t.Helper()
	ctx := context.Background()
	om, _, err := e.admin.createOrganization(ctx, "Org "+t.Name(), uniqueLogin(t))
	require.NoError(t, err)
	u := om.users[0]
	hash, err := e.auth.passwords.hash(testPassword)
	require.NoError(t, err)
	require.NoError(t, newUserRepo(e.pool).setPassword(ctx, u.ID, hash, false))
	return u.Login, u.ID, om.organization.ID
}

// adminUser creates an admin whose password is testPassword and is not
// temporary.
func (e *testEnv) adminUser(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	login := uniqueLogin(t)
	_, err := CreateAdmin(ctx, e.pool, login)
	require.NoError(t, err)
	u, err := newUserRepo(e.pool).selectByLogin(ctx, login)
	require.NoError(t, err)
	hash, err := e.auth.passwords.hash(testPassword)
	require.NoError(t, err)
	require.NoError(t, newUserRepo(e.pool).setPassword(ctx, u.ID, hash, false))
	return login
}

// do sends a request to the mux. A non-nil body is sent as JSON. A non-empty
// token is sent as the session cookie.
func (e *testEnv) do(t *testing.T, method, path string, body any, token string) *httptest.ResponseRecorder {
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
	if token != "" {
		req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	}
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	return rec
}

// login logs in through the API and returns the session token.
func (e *testEnv) login(t *testing.T, login, password string) string {
	t.Helper()
	rec := e.do(t, http.MethodPost, "/api/v1/auth/login", loginRequest{Login: login, Password: password}, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	return sessionCookie(t, rec).Value
}

// sessionCookie returns the session cookie that rec sets.
func sessionCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookieName {
			return c
		}
	}
	require.FailNow(t, "no session cookie in the response")
	return nil
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

// guard returns the Guard of the environment, for the member route tests.
func (e *testEnv) guard() *Guard {
	conf := e.conf
	return newGuard(e.auth, conf, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

// fixedCounts is a DeviceCounter with set counts. The device context tests
// the real one.
type fixedCounts map[int64]int

func (c fixedCounts) DevicesByOrganization(context.Context) (map[int64]int, error) {
	return c, nil
}
