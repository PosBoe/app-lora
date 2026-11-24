package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/urfave/cli/v3"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/waneock/app-lora/pkg/appconf"
)

const (
	startTimeout = 30 * time.Second
	stopTimeout  = 30 * time.Second
)

// Options returns the fx options every application starts from: the config,
// the logger, and the fx event log at debug level.
func Options(conf *appconf.Config, logger *slog.Logger) fx.Option {
	return fx.Options(
		fx.Supply(conf, logger),
		fx.WithLogger(func() fxevent.Logger {
			l := &fxevent.SlogLogger{Logger: logger}
			l.UseLogLevel(slog.LevelDebug)
			return l
		}),
	)
}

// Serve runs a long-lived application. It loads the config, starts the fx
// graph that opts describe, and blocks until ctx is done or the process gets
// SIGINT or SIGTERM. Then it stops the graph.
func Serve(ctx context.Context, cmd *cli.Command, opts ...fx.Option) error {
	conf, err := LoadConfig(cmd)
	if err != nil {
		return err
	}
	logger := NewLogger(os.Stderr, conf)

	app := fx.New(Options(conf, logger), fx.Options(opts...))
	if err := app.Err(); err != nil {
		return err
	}

	startCtx, cancel := context.WithTimeout(ctx, startTimeout)
	defer cancel()
	if err := app.Start(startCtx); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	select {
	case <-ctx.Done():
	case sig := <-app.Wait():
		logger.Info("shutting down", slog.String("signal", sig.Signal.String()))
	}

	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), stopTimeout)
	defer cancel()
	return app.Stop(stopCtx)
}
