// Command lora is the platform binary.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v3"

	"github.com/waneock/app-lora/bootstrap"
	"github.com/waneock/app-lora/cmd/admin"
	"github.com/waneock/app-lora/cmd/migrate"
	"github.com/waneock/app-lora/cmd/pg"
	"github.com/waneock/app-lora/cmd/serve"
	"github.com/waneock/app-lora/pkg/version"
)

func main() {
	os.Exit(run())
}

// run returns the exit status, so its deferred calls run before main exits.
func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	root := bootstrap.NewCLI([]*cli.Command{
		serve.Command(),
		migrate.Command(),
		admin.Command(),
		pg.Command(),
		{
			Name:  "version",
			Usage: "print the version",
			Action: func(context.Context, *cli.Command) error {
				fmt.Println(version.Version)
				return nil
			},
		},
	})

	if err := root.Run(ctx, os.Args); err != nil {
		// fmt keeps the newlines of a multi-line error.
		fmt.Fprintln(os.Stderr, err)
		return bootstrap.ExitCode(err)
	}
	return 0
}
