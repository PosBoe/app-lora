# bootstrap

Assembles a runnable process from configuration: the CLI root, config loading, logging, and the
fx application.

**Only `cmd/*` imports this package.** A domain package gets its dependencies from the fx graph.
If a domain package needs something from here, move that thing down to `pkg/`.

| File | Content |
| --- | --- |
| `cli.go` | `NewCLI` (the root command and its `--config`/`--log-level` flags), `LoadConfig`, `ExitCode`. |
| `logging.go` | `NewLogger`: the `*slog.Logger` from `logging.level` and `logging.format`. |
| `app.go` | `Options` (the config, the logger, the fx event log) and `Serve`, which runs a long-lived graph until a signal. |

## Writing a command

- A **long-lived** command (a server, a worker) calls `bootstrap.Serve(ctx, cmd, modules...)`.
  `cmd/serve/serve.go` is the pattern; its `Modules()` is the assembly.
- A **one-shot** command that needs only the config and a connection boots no fx application. It
  calls `bootstrap.LoadConfig(cmd)` and opens what it needs. `cmd/migrate` and `cmd/pg` do this.

The config path comes from `--config`, or `LORA_CONFIG`, or `config/dev.yml` by default.
