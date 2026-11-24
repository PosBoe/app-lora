// Package system serves the endpoints that report on the process itself.
package system

import "go.uber.org/fx"

// Module provides the system handlers and binds their routes.
var Module = fx.Module("system",
	fx.Provide(newHealthHandler),
	fx.Invoke(registerHealthHandler),
)
