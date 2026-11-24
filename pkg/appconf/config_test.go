package appconf_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waneock/app-lora/pkg/appconf"
)

func TestParse(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		conf, err := appconf.Parse([]byte("pg:\n  dsn: postgres://localhost/lora\n"))
		require.NoError(t, err)
		assert.Equal(t, ":8080", conf.Listen.HTTP)
		assert.Equal(t, "info", conf.Logging.Level)
		assert.Equal(t, "text", conf.Logging.Format)
	})

	t.Run("env expansion", func(t *testing.T) {
		t.Setenv("LORA_TEST_DSN", "postgres://env/lora")
		conf, err := appconf.Parse([]byte("pg:\n  dsn: ${LORA_TEST_DSN}\n"))
		require.NoError(t, err)
		assert.Equal(t, "postgres://env/lora", conf.PG.DSN)
	})

	t.Run("unknown key", func(t *testing.T) {
		_, err := appconf.Parse([]byte("pg:\n  dsn: x\nunknown: 1\n"))
		require.Error(t, err)
	})

	t.Run("missing dsn", func(t *testing.T) {
		_, err := appconf.Parse([]byte("listen:\n  http: :9000\n"))
		require.ErrorContains(t, err, "pg.dsn")
	})

	t.Run("bad level", func(t *testing.T) {
		_, err := appconf.Parse([]byte("pg:\n  dsn: x\nlogging:\n  level: loud\n"))
		require.ErrorContains(t, err, "logging.level")
	})
}
