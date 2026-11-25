package core

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminAccess(t *testing.T) {
	env := newTestEnv(t)

	t.Run("no session", func(t *testing.T) {
		rec := env.do(t, http.MethodGet, "/api/v1/admin/organizations", nil, "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("a member is not an admin", func(t *testing.T) {
		login, _, _ := env.member(t)
		token := env.login(t, login, testPassword)
		rec := env.do(t, http.MethodGet, "/api/v1/admin/organizations", nil, token)
		assert.Equal(t, http.StatusForbidden, rec.Code)
		assert.Equal(t, "forbidden", errorCode(t, rec))
	})
}

func TestAdminOrganizations(t *testing.T) {
	env := newTestEnv(t)
	token := env.login(t, env.adminUser(t), testPassword)

	create := func(t *testing.T, name, login string) createOrganizationResponse {
		t.Helper()
		rec := env.do(t, http.MethodPost, "/api/v1/admin/organizations",
			createOrganizationRequest{Name: name, Login: login}, token)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var resp createOrganizationResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		return resp
	}

	t.Run("create gives a temporary password that logs in", func(t *testing.T) {
		login := uniqueLogin(t)
		resp := create(t, "  Apă Canal  ", login)
		assert.Equal(t, "Apă Canal", resp.Organization.Name)
		require.Len(t, resp.Organization.Users, 1)
		assert.Equal(t, login, resp.Organization.Users[0].Login)
		assert.True(t, resp.Organization.Users[0].PasswordTemporary)
		assert.Len(t, resp.TemporaryPassword, temporaryPasswordChars)

		var me meResponse
		rec := env.do(t, http.MethodGet, "/api/v1/auth/me", nil, env.login(t, login, resp.TemporaryPassword))
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &me))
		assert.True(t, me.PasswordTemporary)
	})

	t.Run("create checks its input", func(t *testing.T) {
		cases := []struct {
			name, org, login, code string
			status                 int
		}{
			{"empty name", " ", uniqueLogin(t), "name_required", http.StatusBadRequest},
			{"bad login", "Org", "a b", "login_invalid", http.StatusBadRequest},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				rec := env.do(t, http.MethodPost, "/api/v1/admin/organizations",
					createOrganizationRequest{Name: tc.org, Login: tc.login}, token)
				assert.Equal(t, tc.status, rec.Code)
				assert.Equal(t, tc.code, errorCode(t, rec))
			})
		}
	})

	t.Run("a taken login creates nothing", func(t *testing.T) {
		login := uniqueLogin(t)
		create(t, "First", login)
		rec := env.do(t, http.MethodPost, "/api/v1/admin/organizations",
			createOrganizationRequest{Name: "Second " + login, Login: login}, token)
		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Equal(t, "login_taken", errorCode(t, rec))

		orgs, err := env.admin.listOrganizations(context.Background())
		require.NoError(t, err)
		for _, om := range orgs {
			assert.NotEqual(t, "Second "+login, om.organization.Name, "the transaction rolled back")
		}
	})

	t.Run("list shows the organization with its user", func(t *testing.T) {
		login := uniqueLogin(t)
		created := create(t, "Listed", login)
		rec := env.do(t, http.MethodGet, "/api/v1/admin/organizations", nil, token)
		require.Equal(t, http.StatusOK, rec.Code)
		var resp listOrganizationsResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

		var found *organizationResponse
		for i := range resp.Organizations {
			if resp.Organizations[i].ID == created.Organization.ID {
				found = &resp.Organizations[i]
			}
		}
		require.NotNil(t, found)
		require.Len(t, found.Users, 1)
		assert.Equal(t, login, found.Users[0].Login)
		assert.Nil(t, found.DeactivatedAt)
	})

	t.Run("deactivate and reactivate", func(t *testing.T) {
		login, _, orgID := env.member(t)
		path := fmt.Sprintf("/api/v1/admin/organizations/%d/", orgID)

		require.Equal(t, http.StatusNoContent, env.do(t, http.MethodPost, path+"deactivate", nil, token).Code)
		rec := env.do(t, http.MethodPost, "/api/v1/auth/login", loginRequest{Login: login, Password: testPassword}, "")
		assert.Equal(t, "organization_deactivated", errorCode(t, rec))

		require.Equal(t, http.StatusNoContent, env.do(t, http.MethodPost, path+"reactivate", nil, token).Code)
		env.login(t, login, testPassword)
	})

	t.Run("unknown organization", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/v1/admin/organizations/0/deactivate", nil, token)
		assert.Equal(t, http.StatusNotFound, rec.Code)
		rec = env.do(t, http.MethodPost, "/api/v1/admin/organizations/999999999/deactivate", nil, token)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("reset password ends the sessions and unlocks", func(t *testing.T) {
		login, userID, _ := env.member(t)
		old := env.login(t, login, testPassword)
		for range env.conf.Auth.LockoutAttempts {
			env.do(t, http.MethodPost, "/api/v1/auth/login", loginRequest{Login: login, Password: "wrong-password"}, "")
		}

		rec := env.do(t, http.MethodPost, fmt.Sprintf("/api/v1/admin/users/%d/reset-password", userID), nil, token)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var resp resetPasswordResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

		assert.Equal(t, http.StatusUnauthorized, env.do(t, http.MethodGet, "/api/v1/auth/me", nil, old).Code)
		env.login(t, login, resp.TemporaryPassword)
	})

	t.Run("reset password of an admin is not found", func(t *testing.T) {
		adminLogin := env.adminUser(t)
		u, err := newUserRepo(env.pool).selectByLogin(context.Background(), adminLogin)
		require.NoError(t, err)
		rec := env.do(t, http.MethodPost, fmt.Sprintf("/api/v1/admin/users/%d/reset-password", u.ID), nil, token)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestCreateAdmin(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	login := uniqueLogin(t)

	password, err := CreateAdmin(ctx, env.pool, login)
	require.NoError(t, err)

	var me meResponse
	rec := env.do(t, http.MethodGet, "/api/v1/auth/me", nil, env.login(t, login, password))
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &me))
	assert.True(t, me.IsAdmin)
	assert.True(t, me.PasswordTemporary)
	assert.Nil(t, me.Organization)

	_, err = CreateAdmin(ctx, env.pool, login)
	require.ErrorIs(t, err, errLoginTaken)
}
