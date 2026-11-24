// Package migrations holds the PostgreSQL migrations. Each file is named
// <version>_<name>.sql, where the version is the 14 digits YYYYMMDDhhmmss of
// the UTC time the file was created. Create one with `task dev:migrate:create -- <name>`.
package migrations

import "embed"

// FS holds the migration files.
//
//go:embed *.sql
var FS embed.FS
