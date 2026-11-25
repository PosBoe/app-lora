package core

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogin(t *testing.T) {
	env := newTestEnv(t)

	t.Run("right password", func(t *testing.T) {
		login, userID, orgID := env.member(t)
		rec := env.do(t, http.MethodPost, "/api/v1/auth/login", loginRequest{Login: login, Password: testPassword}, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		var me meResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &me))
		assert.Equal(t, userID, me.ID)
		assert.False(t, me.IsAdmin)
		require.NotNil(t, me.Organization)
		assert.Equal(t, orgID, me.Organization.ID)

		cookie := sessionCookie(t, rec)
		assert.True(t, cookie.HttpOnly)
		assert.True(t, cookie.Secure, "site.url is https")
		assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
		assert.Equal(t, int((30 * 24 * time.Hour).Seconds()), cookie.MaxAge)
	})

	t.Run("login ignores case and spaces", func(t *testing.T) {
		login, _, _ := env.member(t)
		env.login(t, "  "+upper(login)+" ", testPassword)
	})

	t.Run("wrong password and unknown login give one answer", func(t *testing.T) {
		login, _, _ := env.member(t)
		wrong := env.do(t, http.MethodPost, "/api/v1/auth/login", loginRequest{Login: login, Password: "wrong-password"}, "")
		unknown := env.do(t, http.MethodPost, "/api/v1/auth/login", loginRequest{Login: uniqueLogin(t), Password: testPassword}, "")
		assert.Equal(t, http.StatusBadRequest, wrong.Code)
		assert.Equal(t, wrong.Code, unknown.Code)
		assert.Equal(t, wrong.Body.String(), unknown.Body.String())
		assert.Equal(t, "invalid_credentials", errorCode(t, wrong))
	})

	t.Run("lockout after the failed attempts and release after the duration", func(t *testing.T) {
		login, _, _ := env.member(t)
		for range env.conf.Auth.LockoutAttempts {
			rec := env.do(t, http.MethodPost, "/api/v1/auth/login", loginRequest{Login: login, Password: "wrong-password"}, "")
			require.Equal(t, "invalid_credentials", errorCode(t, rec))
		}

		locked := env.do(t, http.MethodPost, "/api/v1/auth/login", loginRequest{Login: login, Password: testPassword}, "")
		assert.Equal(t, http.StatusTooManyRequests, locked.Code)
		assert.Equal(t, "login_locked", errorCode(t, locked))

		env.clock.Advance(env.conf.Auth.LockoutDuration)
		env.login(t, login, testPassword)
		env.clock.Advance(-env.conf.Auth.LockoutDuration)
	})

	t.Run("a right password resets the failed count", func(t *testing.T) {
		login, _, _ := env.member(t)
		for range env.conf.Auth.LockoutAttempts - 1 {
			env.do(t, http.MethodPost, "/api/v1/auth/login", loginRequest{Login: login, Password: "wrong-password"}, "")
		}
		env.login(t, login, testPassword)
		rec := env.do(t, http.MethodPost, "/api/v1/auth/login", loginRequest{Login: login, Password: "wrong-password"}, "")
		assert.Equal(t, "invalid_credentials", errorCode(t, rec), "one more failure does not lock")
	})

	t.Run("deactivated organization", func(t *testing.T) {
		login, _, orgID := env.member(t)
		require.NoError(t, env.admin.deactivate(context.Background(), orgID))

		wrong := env.do(t, http.MethodPost, "/api/v1/auth/login", loginRequest{Login: login, Password: "wrong-password"}, "")
		assert.Equal(t, "invalid_credentials", errorCode(t, wrong), "a wrong password tells nothing more")

		right := env.do(t, http.MethodPost, "/api/v1/auth/login", loginRequest{Login: login, Password: testPassword}, "")
		assert.Equal(t, http.StatusForbidden, right.Code)
		assert.Equal(t, "organization_deactivated", errorCode(t, right))
	})

	t.Run("form body is refused", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/v1/auth/login", nil, "")
		assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
	})
}

func TestSession(t *testing.T) {
	env := newTestEnv(t)

	t.Run("me needs a session", func(t *testing.T) {
		rec := env.do(t, http.MethodGet, "/api/v1/auth/me", nil, "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, "unauthenticated", errorCode(t, rec))

		bad := env.do(t, http.MethodGet, "/api/v1/auth/me", nil, "not-a-token")
		assert.Equal(t, http.StatusUnauthorized, bad.Code)
	})

	t.Run("me returns the user", func(t *testing.T) {
		login, userID, _ := env.member(t)
		token := env.login(t, login, testPassword)
		rec := env.do(t, http.MethodGet, "/api/v1/auth/me", nil, token)
		require.Equal(t, http.StatusOK, rec.Code)
		var me meResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &me))
		assert.Equal(t, userID, me.ID)
		assert.Equal(t, "ro", me.Language)
	})

	t.Run("the session lives for the TTL after the last use", func(t *testing.T) {
		login, _, _ := env.member(t)
		token := env.login(t, login, testPassword)
		start := env.clock.Now()
		defer func() { env.clock.Advance(start.Sub(env.clock.Now())) }()

		env.clock.Advance(20 * 24 * time.Hour)
		rec := env.do(t, http.MethodGet, "/api/v1/auth/me", nil, token)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, int(env.conf.Auth.SessionTTL.Seconds()), sessionCookie(t, rec).MaxAge,
			"the moved expiry sends the cookie again")

		env.clock.Advance(20 * 24 * time.Hour)
		assert.Equal(t, http.StatusOK, env.do(t, http.MethodGet, "/api/v1/auth/me", nil, token).Code,
			"40 days after login, 20 days after the last use")

		env.clock.Advance(env.conf.Auth.SessionTTL)
		assert.Equal(t, http.StatusUnauthorized, env.do(t, http.MethodGet, "/api/v1/auth/me", nil, token).Code)
	})

	t.Run("a use within the extend step writes no cookie", func(t *testing.T) {
		login, _, _ := env.member(t)
		token := env.login(t, login, testPassword)
		rec := env.do(t, http.MethodGet, "/api/v1/auth/me", nil, token)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Empty(t, rec.Result().Cookies())
	})

	t.Run("logout ends the session", func(t *testing.T) {
		login, _, _ := env.member(t)
		token := env.login(t, login, testPassword)
		rec := env.do(t, http.MethodPost, "/api/v1/auth/logout", nil, token)
		require.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, -1, sessionCookie(t, rec).MaxAge)
		assert.Equal(t, http.StatusUnauthorized, env.do(t, http.MethodGet, "/api/v1/auth/me", nil, token).Code)
	})

	t.Run("deactivation ends a live session", func(t *testing.T) {
		login, _, orgID := env.member(t)
		token := env.login(t, login, testPassword)
		require.NoError(t, env.admin.deactivate(context.Background(), orgID))
		rec := env.do(t, http.MethodGet, "/api/v1/auth/me", nil, token)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestTemporaryPassword(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	om, temporary, err := env.admin.createOrganization(ctx, "Temporary Org", uniqueLogin(t))
	require.NoError(t, err)
	login := om.users[0].Login
	token := env.login(t, login, temporary)

	t.Run("me tells the frontend to ask for a new password", func(t *testing.T) {
		rec := env.do(t, http.MethodGet, "/api/v1/auth/me", nil, token)
		require.Equal(t, http.StatusOK, rec.Code)
		var me meResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &me))
		assert.True(t, me.PasswordTemporary)
	})

	t.Run("other routes are refused", func(t *testing.T) {
		rec := env.do(t, http.MethodGet, "/api/v1/admin/organizations", nil, token)
		assert.Equal(t, http.StatusForbidden, rec.Code)
		assert.Equal(t, "password_change_required", errorCode(t, rec))
	})

	t.Run("password rules", func(t *testing.T) {
		short := env.do(t, http.MethodPost, "/api/v1/auth/password",
			passwordRequest{CurrentPassword: temporary, NewPassword: "short"}, token)
		assert.Equal(t, "password_too_short", errorCode(t, short))

		wrong := env.do(t, http.MethodPost, "/api/v1/auth/password",
			passwordRequest{CurrentPassword: "not-the-password", NewPassword: "a-new-password"}, token)
		assert.Equal(t, "invalid_current_password", errorCode(t, wrong))
	})

	t.Run("a new password opens the routes and ends other sessions", func(t *testing.T) {
		other := env.login(t, login, temporary)
		rec := env.do(t, http.MethodPost, "/api/v1/auth/password",
			passwordRequest{CurrentPassword: temporary, NewPassword: "a-new-password"}, token)
		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())

		var me meResponse
		meRec := env.do(t, http.MethodGet, "/api/v1/auth/me", nil, token)
		require.NoError(t, json.Unmarshal(meRec.Body.Bytes(), &me))
		assert.False(t, me.PasswordTemporary)
		assert.Equal(t, http.StatusUnauthorized, env.do(t, http.MethodGet, "/api/v1/auth/me", nil, other).Code)
		env.login(t, login, "a-new-password")
	})
}

// upper returns s in upper case, for the case-insensitive login check.
func upper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 'a' + 'A'
		}
	}
	return string(b)
}
