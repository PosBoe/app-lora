package serve

import (
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/waneock/app-lora/bootstrap"
	"github.com/waneock/app-lora/pkg/appconf"
)

// TestModulesGraph checks that the serve assembly resolves: every
// dependency has a provider and no type has two. It starts nothing.
func TestModulesGraph(t *testing.T) {
	conf, err := appconf.Parse([]byte("pg:\n  dsn: postgres://lora:lora@127.0.0.1:1/lora\n"))
	require.NoError(t, err)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	err = fx.ValidateApp(bootstrap.Options(conf, logger), fx.Options(Modules()...))
	require.NoError(t, err)
}
