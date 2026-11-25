package device

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func integrationPath(id int64, suffix string) string {
	return fmt.Sprintf("/api/v1/integrations/%d%s", id, suffix)
}

func TestIntegrations(t *testing.T) {
	env := newTestEnv(t)
	org := env.organization(t)

	t.Run("create shows the secret and the URL of the kind", func(t *testing.T) {
		got := env.integration(t, org, kindTTN)
		assert.True(t, strings.HasPrefix(got.Secret, secretPrefix))
		assert.Equal(t, "https://lora.test/api/v1/ingest/ttn", got.Integration.IngestURL)

		list := decode[integrationListResponse](t, env.do(t, org, http.MethodGet, "/api/v1/integrations", nil), http.StatusOK)
		require.Len(t, list.Integrations, 1)
		assert.Nil(t, list.Integrations[0].LastUplinkAt)
		assert.NotContains(t, fmt.Sprint(list), got.Secret, "the list never holds the secret")
	})

	t.Run("checks the fields", func(t *testing.T) {
		rec := env.do(t, org, http.MethodPost, "/api/v1/integrations", integrationRequest{Name: " ", Kind: kindTTN})
		assert.Equal(t, "integration_name_required", errorCode(t, rec))
		rec = env.do(t, org, http.MethodPost, "/api/v1/integrations", integrationRequest{Name: "x", Kind: "loriot"})
		assert.Equal(t, "integration_kind_invalid", errorCode(t, rec))
	})

	t.Run("regenerate gives a new secret", func(t *testing.T) {
		created := env.integration(t, org, kindChirpStack)
		rec := env.do(t, org, http.MethodPost, integrationPath(created.Integration.ID, "/secret"), nil)
		got := decode[regenerateResponse](t, rec, http.StatusOK)
		assert.NotEqual(t, created.Secret, got.Secret)
	})

	t.Run("another organization cannot see or change an integration", func(t *testing.T) {
		created := env.integration(t, org, kindTTN)
		other := env.organization(t)
		list := decode[integrationListResponse](t, env.do(t, other, http.MethodGet, "/api/v1/integrations", nil), http.StatusOK)
		assert.Empty(t, list.Integrations)
		for _, rec := range []int{
			env.do(t, other, http.MethodPut, integrationPath(created.Integration.ID, ""), integrationRequest{Name: "x"}).Code,
			env.do(t, other, http.MethodPost, integrationPath(created.Integration.ID, "/secret"), nil).Code,
			env.do(t, other, http.MethodDelete, integrationPath(created.Integration.ID, ""), nil).Code,
		} {
			assert.Equal(t, http.StatusNotFound, rec)
		}
	})
}
