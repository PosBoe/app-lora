package bootstrap

import (
	"io"
	"log/slog"

	"github.com/waneock/app-lora/pkg/appconf"
)

// NewLogger builds the process logger from logging.level and logging.format.
// LoadConfig has validated both, so NewLogger cannot fail.
func NewLogger(w io.Writer, conf *appconf.Config) *slog.Logger {
	level, _ := appconf.ParseLogLevel(conf.Logging.Level)
	opts := &slog.HandlerOptions{Level: level}

	if conf.Logging.Format == "json" {
		return slog.New(slog.NewJSONHandler(w, opts))
	}
	return slog.New(slog.NewTextHandler(w, opts))
}
