// Package pgtest opens the test database for an integration test.
package pgtest

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/waneock/app-lora/pkg/appconf"
	"github.com/waneock/app-lora/pkg/pgdb"
)

// pingTimeout bounds the first dial, so a missing database fails fast.
const pingTimeout = 5 * time.Second

// Pool returns a pool on the database that LORA_CONFIG names, and closes it
// when the test ends. It skips the test under -short, and when LORA_CONFIG is
// not set. Run `task test:setup` first to create the migrated database.
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("pgtest: the test needs PostgreSQL; run without -short")
	}
	path := os.Getenv("LORA_CONFIG")
	if path == "" {
		t.Skip("pgtest: LORA_CONFIG is not set; use the test tasks")
	}

	conf, err := appconf.Load(path)
	require.NoError(t, err)
	pool, err := pgdb.Open(conf)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	require.NoError(t, pool.Ping(ctx), "pgtest: run `task up` and `task test:setup`")
	return pool
}
