package core

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/waneock/app-lora/pkg/pgdb"
)

// organization is a row of the organizations table.
type organization struct {
	ID            int64
	Name          string
	DeactivatedAt *time.Time
	CreatedAt     time.Time
}

type organizationRepo struct {
	q pgdb.Querier
}

func newOrganizationRepo(q pgdb.Querier) *organizationRepo {
	return &organizationRepo{q: q}
}

const organizationColumns = `id, name, deactivated_at, created_at`

func scanOrganization(row pgx.Row) (*organization, error) {
	o := new(organization)
	if err := row.Scan(&o.ID, &o.Name, &o.DeactivatedAt, &o.CreatedAt); err != nil {
		return nil, err
	}
	return o, nil
}

// insert creates an organization with name.
func (r *organizationRepo) insert(ctx context.Context, name string) (*organization, error) {
	return scanOrganization(r.q.QueryRow(ctx, `
		INSERT INTO organizations (name) VALUES ($1)
		RETURNING `+organizationColumns, name))
}

// selectByID returns pgx.ErrNoRows when no organization has id.
func (r *organizationRepo) selectByID(ctx context.Context, id int64) (*organization, error) {
	return scanOrganization(r.q.QueryRow(ctx,
		`SELECT `+organizationColumns+` FROM organizations WHERE id = $1`, id))
}

// setDeactivatedAt sets or clears the deactivation time. A deactivation keeps
// the first time. It returns pgx.ErrNoRows when no organization has id.
func (r *organizationRepo) setDeactivatedAt(ctx context.Context, id int64, at *time.Time) error {
	tag, err := r.q.Exec(ctx, `
		UPDATE organizations
		SET deactivated_at = CASE WHEN $2::timestamptz IS NULL THEN NULL
			ELSE coalesce(deactivated_at, $2) END
		WHERE id = $1`, id, at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// list returns all organizations, the newest first.
func (r *organizationRepo) list(ctx context.Context) ([]*organization, error) {
	rows, err := r.q.Query(ctx, `SELECT `+organizationColumns+` FROM organizations
		ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*organization, error) {
		return scanOrganization(row)
	})
}

// names returns the name of every organization.
func (r *organizationRepo) names(ctx context.Context) (map[int64]string, error) {
	rows, err := r.q.Query(ctx, `SELECT id, name FROM organizations`)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]string)
	var id int64
	var name string
	_, err = pgx.ForEachRow(rows, []any{&id, &name}, func() error {
		out[id] = name
		return nil
	})
	return out, err
}
