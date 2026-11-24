// Package pgdb opens the PostgreSQL connection pool and provides it to the
// fx graph.
package pgdb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"github.com/waneock/app-lora/pkg/appconf"
)

// Module provides *pgxpool.Pool. The pool connects on first use, so the
// process starts when the database is down.
var Module = fx.Module("pgdb",
	fx.Provide(newPool),
)

func newPool(lc fx.Lifecycle, conf *appconf.Config) (*pgxpool.Pool, error) {
	pool, err := Open(conf)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.StopHook(pool.Close))
	return pool, nil
}

// Open creates a pool for conf.PG.DSN. It does not dial the database.
func Open(conf *appconf.Config) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(conf.PG.DSN)
	if err != nil {
		return nil, fmt.Errorf("pgdb: parse pg.dsn: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("pgdb: %w", err)
	}
	return pool, nil
}

// Maintenance returns the name of the configured database and a connection
// config that points at the "postgres" maintenance database of the same
// server. Use it to create or drop the configured database.
func Maintenance(conf *appconf.Config) (dbName string, _ *pgx.ConnConfig, _ error) {
	cfg, err := pgx.ParseConfig(conf.PG.DSN)
	if err != nil {
		return "", nil, fmt.Errorf("pgdb: parse pg.dsn: %w", err)
	}
	dbName = cfg.Database
	if dbName == "" {
		return "", nil, fmt.Errorf("pgdb: pg.dsn names no database")
	}
	cfg.Database = "postgres"
	return dbName, cfg, nil
}
