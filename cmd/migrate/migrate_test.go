package migrate

import (
	"io/fs"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waneock/app-lora/cmd/lora/migrations"
)

// TestMigrationNames checks that every migration has a 14-digit version and
// a snake_case name.
func TestMigrationNames(t *testing.T) {
	re := regexp.MustCompile(`^\d{14}_[a-z0-9]+(_[a-z0-9]+)*\.sql$`)

	names, err := fs.Glob(migrations.FS, "*.sql")
	require.NoError(t, err)
	require.NotEmpty(t, names)

	for _, name := range names {
		assert.Regexp(t, re, name)
	}
}
