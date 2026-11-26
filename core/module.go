// Package core holds who uses the platform and what they own: organizations,
// users, the admin, logins, and sessions. README.md states the session
// mechanism; CONTEXT.md states the terms.
package core

import "go.uber.org/fx"

// Module provides the core services and handlers and binds their routes.
var Module = fx.Module("core",
	fx.Provide(
		newPasswords,
		newAuthService,
		newAdminService,
		newGuard,
		newAuthHandler,
		newAdminHandler,
		newDirectory,
	),
	fx.Invoke(registerAuthHandler, registerAdminHandler),
)
