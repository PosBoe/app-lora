package serve

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"go.uber.org/fx"

	lora "github.com/waneock/app-lora"
	"github.com/waneock/app-lora/pkg/appconf"
)

// Module provides the shared *http.ServeMux and runs the HTTP server on it.
// A domain module binds its routes on the mux.
var Module = fx.Module("serve",
	fx.Provide(http.NewServeMux),
	fx.Invoke(serveVue),
	fx.Invoke(runHTTPServer),
)

func serveVue(mux *http.ServeMux) {
	distFS := lora.VueDistFS()
	if distFS == nil {
		return
	}
	mux.Handle("GET /", newVueHandler(distFS))
}

func runHTTPServer(lc fx.Lifecycle, conf *appconf.Config, mux *http.ServeMux, logger *slog.Logger) {
	srv := &http.Server{
		Addr:              conf.Listen.HTTP,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			var lcfg net.ListenConfig
			ln, err := lcfg.Listen(ctx, "tcp", srv.Addr)
			if err != nil {
				return err
			}
			logger.Info("http server started", slog.String("addr", ln.Addr().String()))
			go func() {
				if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Error("http server failed", slog.Any("err", err))
				}
			}()
			return nil
		},
		OnStop: srv.Shutdown,
	})
}
