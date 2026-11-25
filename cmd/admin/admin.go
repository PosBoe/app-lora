// Package admin holds the commands that manage the admins of the platform.
// The UI creates no admins. It boots no fx application.
package admin

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/waneock/app-lora/bootstrap"
	"github.com/waneock/app-lora/core"
	"github.com/waneock/app-lora/pkg/pgdb"
)

// Command returns the admin command group.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "admin",
		Usage: "manage the admins of the platform",
		Commands: []*cli.Command{
			{
				Name:  "create",
				Usage: "create an admin and print its temporary password",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "login",
						Usage:    "the login of the new admin",
						Required: true,
					},
				},
				Action: create,
			},
		},
	}
}

func create(ctx context.Context, cmd *cli.Command) error {
	conf, err := bootstrap.LoadConfig(cmd)
	if err != nil {
		return err
	}
	pool, err := pgdb.Open(conf)
	if err != nil {
		return err
	}
	defer pool.Close()

	password, err := core.CreateAdmin(ctx, pool, cmd.String("login"))
	if err != nil {
		return fmt.Errorf("admin create: %w", err)
	}
	fmt.Printf("Temporary password: %s\n", password)
	fmt.Println("It is shown one time. Replace it at the first login.")
	return nil
}
