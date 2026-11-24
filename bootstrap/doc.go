// Package bootstrap assembles a runnable process: the CLI root, config
// loading, logging, and the fx application.
//
// Only cmd/* imports this package. A domain package gets its dependencies
// from the fx graph, not from bootstrap.
package bootstrap
