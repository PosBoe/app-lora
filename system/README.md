# system

Endpoints that report on the process itself. This package is also the
reference shape for a new domain package.

| File                    | Content                                               |
| ----------------------- | ----------------------------------------------------- |
| `module.go`             | `Module`: provides the handlers and binds the routes. |
| `health_handler.go`     | `GET /api/v1/health`: pings PostgreSQL.               |
| `health_handler_test.go`| Health answers 503 when the database is down.         |

A new package copies this shape: an unexported `new<Name>Handler`, an
unexported `register<Name>Handler`, and one `Module` that `cmd/serve` adds to
the graph.
