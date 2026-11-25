// Package device holds the device registry of an organization: devices and
// device models. README.md states the rules; CONTEXT.md states the terms;
// DESIGN.md states the goals.
package device

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

// Module provides the device services, handlers, and Stats, and binds the
// routes. It needs *core.Guard from the core module.
var Module = fx.Module("device",
	fx.Provide(
		newDeviceService,
		newModelService,
		newGroupService,
		newImportService,
		newIntegrationService,
		newIngestService,
		newDeviceHandler,
		newModelHandler,
		newGroupHandler,
		newImportHandler,
		newIntegrationHandler,
		newIngestHandler,
		newUplinkHandler,
		newPurger,
		func(pool *pgxpool.Pool) *Stats { return newStats(pool) },
	),
	fx.Invoke(registerDeviceHandler, registerModelHandler, registerGroupHandler,
		registerImportHandler,
		registerIntegrationHandler,
		registerIngestHandler,
		registerUplinkHandler,
		runPurger),
)
