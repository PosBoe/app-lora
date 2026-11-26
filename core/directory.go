package core

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Directory names the organizations for the admin views of other contexts.
// The composition root binds it to the interface that such a context
// declares.
type Directory struct {
	organizations *organizationRepo
}

func newDirectory(pool *pgxpool.Pool) *Directory {
	return &Directory{organizations: newOrganizationRepo(pool)}
}

// OrganizationNames returns the name of every organization by its id.
func (d *Directory) OrganizationNames(ctx context.Context) (map[int64]string, error) {
	return d.organizations.names(ctx)
}
