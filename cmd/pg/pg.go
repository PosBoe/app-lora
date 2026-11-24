// Package pg holds the maintenance commands for the PostgreSQL database.
// It boots no fx application.
package pg

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/urfave/cli/v3"

	"github.com/waneock/app-lora/bootstrap"
	"github.com/waneock/app-lora/pkg/pgdb"
)

// Command returns the pg command group.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "pg",
		Usage: "manage the PostgreSQL database",
		Commands: []*cli.Command{
			{
				Name:   "create_db",
				Usage:  "create the configured database",
				Action: maintenance("CREATE DATABASE %s"),
			},
			{
				Name:   "drop_db",
				Usage:  "drop the configured database if it exists",
				Action: maintenance("DROP DATABASE IF EXISTS %s WITH (FORCE)"),
			},
			{
				Name:   "ping",
				Usage:  "check that the configured database answers",
				Action: ping,
			},
		},
	}
}

// maintenance runs a statement on the maintenance database. The statement
// takes the quoted name of the configured database.
func maintenance(query string) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		conf, err := bootstrap.LoadConfig(cmd)
		if err != nil {
			return err
		}
		dbName, cfg, err := pgdb.Maintenance(conf)
		if err != nil {
			return err
		}

		conn, err := pgx.ConnectConfig(ctx, cfg)
		if err != nil {
			return err
		}
		defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

		stmt := fmt.Sprintf(query, pgx.Identifier{dbName}.Sanitize())
		if _, err := conn.Exec(ctx, stmt); err != nil {
			return err
		}
		fmt.Println(stmt)
		return nil
	}
}

func ping(ctx context.Context, cmd *cli.Command) error {
	conf, err := bootstrap.LoadConfig(cmd)
	if err != nil {
		return err
	}
	conn, err := pgx.Connect(ctx, conf.PG.DSN)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

	if err := conn.Ping(ctx); err != nil {
		return err
	}
	fmt.Println("ok")
	return nil
}
