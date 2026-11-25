// Package serve runs the HTTP server.
package serve

import (
	"context"

	"github.com/urfave/cli/v3"
	"go.uber.org/fx"

	"github.com/waneock/app-lora/bootstrap"
	"github.com/waneock/app-lora/core"
	"github.com/waneock/app-lora/device"
	"github.com/waneock/app-lora/pkg/clock"
	"github.com/waneock/app-lora/pkg/pgdb"
	"github.com/waneock/app-lora/system"
)

// Command returns the serve command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "run the HTTP server",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return bootstrap.Serve(ctx, cmd, Modules()...)
		},
	}
}

// Modules is the fx assembly of the serve command. Add the Module of a new
// domain package here.
func Modules() []fx.Option {
	return []fx.Option{
		pgdb.Module,
		clock.Module,
		system.Module,
		core.Module,
		device.Module,
		Module,
	}
}
