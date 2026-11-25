package core

import (
	"errors"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waneock/app-lora/pkg/httpkit"
)

func TestGuardMember(t *testing.T) {
	env := newTestEnv(t)
	env.mux.Handle("GET /api/v1/test/member", env.guard().Member(func(w http.ResponseWriter, req *http.Request) error {
		c, ok := CallerFrom(req.Context())
		if !ok {
			return errors.New("no caller in the context")
		}
		httpkit.JSON(w, http.StatusOK, c)
		return nil
	}))

	t.Run("a member gets its organization", func(t *testing.T) {
		login, userID, orgID := env.member(t)
		rec := env.do(t, http.MethodGet, "/api/v1/test/member", nil, env.login(t, login, testPassword))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.JSONEq(t,
			`{"UserID":`+itoa(userID)+`,"OrganizationID":`+itoa(orgID)+`,"IsAdmin":false}`,
			rec.Body.String())
	})

	t.Run("an admin is not a member", func(t *testing.T) {
		rec := env.do(t, http.MethodGet, "/api/v1/test/member", nil, env.login(t, env.adminUser(t), testPassword))
		assert.Equal(t, http.StatusForbidden, rec.Code)
		assert.Equal(t, "forbidden", errorCode(t, rec))
	})

	t.Run("no session", func(t *testing.T) {
		rec := env.do(t, http.MethodGet, "/api/v1/test/member", nil, "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("outside a guard handler", func(t *testing.T) {
		_, ok := CallerFrom(t.Context())
		assert.False(t, ok)
	})
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
