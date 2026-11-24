package bootstrap

import (
	"context"
	"errors"

	"github.com/urfave/cli/v3"

	"github.com/waneock/app-lora/pkg/appconf"
)

const (
	configFlag   = "config"
	logLevelFlag = "log-level"
)

// NewCLI builds the root command. The global --config and --log-level flags
// live here, so every command reads them the same way.
func NewCLI(commands []*cli.Command) *cli.Command {
	return &cli.Command{
		Name:  "lora",
		Usage: "lora platform CLI",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    configFlag,
				Usage:   "load YAML configuration from `FILE`",
				Value:   "config/dev.yml",
				Sources: cli.EnvVars("LORA_CONFIG"),
			},
			&cli.StringFlag{
				Name:    logLevelFlag,
				Usage:   "log at `LEVEL` (debug, info, warn, error); overrides logging.level",
				Sources: cli.EnvVars("LORA_LOG_LEVEL"),
			},
		},
		Commands: commands,
		// main picks the exit status with ExitCode. Without this handler, cli
		// exits by itself on an error that names a status.
		ExitErrHandler: func(context.Context, *cli.Command, error) {},
	}
}

// LoadConfig reads the file that the --config flag names. It applies the
// --log-level flag over logging.level.
func LoadConfig(cmd *cli.Command) (*appconf.Config, error) {
	conf, err := appconf.Load(cmd.String(configFlag))
	if err != nil {
		return nil, err
	}
	if level := cmd.String(logLevelFlag); level != "" {
		if _, err := appconf.ParseLogLevel(level); err != nil {
			return nil, err
		}
		conf.Logging.Level = level
	}
	return conf, nil
}

type exitCoder interface {
	error
	ExitCode() int
}

// ExitCode returns the exit status for err, which the root command returned.
// An error that names a status gives it, also when it is wrapped. Every other
// error exits 1.
func ExitCode(err error) int {
	if coded, ok := errors.AsType[exitCoder](err); ok {
		return coded.ExitCode()
	}
	return 1
}
