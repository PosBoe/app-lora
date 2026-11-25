package device

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/waneock/app-lora/pkg/pgdb"
)

// Integration kinds. They are persisted in integrations.kind and name the
// ingest route of each network server.
const (
	kindChirpStack = "chirpstack"
	kindTTN        = "ttn"
)

// integration is a row of integrations, with the number of uplinks it
// delivered since a time.
type integration struct {
	ID               int64
	Name             string
	Kind             string
	CreatedAt        time.Time
	LastUplinkAt     *time.Time
	LastErrorAt      *time.Time
	LastErrorCode    *string
	LastErrorMessage *string
	RecentUplinks    int
}

// integrationRef is what ingestion needs of the integration of a secret.
type integrationRef struct {
	ID             int64
	OrganizationID int64
	Kind           string
}

type integrationRepo struct {
	q pgdb.Querier
}

func newIntegrationRepo(q pgdb.Querier) *integrationRepo {
	return &integrationRepo{q: q}
}

func (r *integrationRepo) insert(ctx context.Context, orgID int64, name, kind string, secretHash []byte) (*integration, error) {
	in := &integration{Name: name, Kind: kind}
	err := r.q.QueryRow(ctx, `
		INSERT INTO integrations (organization_id, name, kind, secret_hash) VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`, orgID, name, kind, secretHash).Scan(&in.ID, &in.CreatedAt)
	if err != nil {
		return nil, err
	}
	return in, nil
}

// list returns the integrations of an organization by name, each with the
// number of uplinks it delivered since since.
func (r *integrationRepo) list(ctx context.Context, orgID int64, since time.Time) ([]*integration, error) {
	rows, err := r.q.Query(ctx, `
		SELECT i.id, i.name, i.kind, i.created_at, i.last_uplink_at, i.last_error_at,
			i.last_error_code, i.last_error_message,
			(SELECT count(*) FROM uplinks u WHERE u.integration_id = i.id AND u.stored_at >= $2)
		FROM integrations i
		WHERE i.organization_id = $1
		ORDER BY lower(i.name), i.id`, orgID, since)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*integration, error) {
		in := new(integration)
		err := row.Scan(&in.ID, &in.Name, &in.Kind, &in.CreatedAt, &in.LastUplinkAt,
			&in.LastErrorAt, &in.LastErrorCode, &in.LastErrorMessage, &in.RecentUplinks)
		return in, err
	})
}

// rename returns pgx.ErrNoRows when the organization has no such integration.
func (r *integrationRepo) rename(ctx context.Context, orgID, id int64, name string) error {
	return affectedOne(r.q.Exec(ctx, `UPDATE integrations SET name = $3 WHERE organization_id = $1 AND id = $2`,
		orgID, id, name))
}

// setSecret replaces the secret hash, so the old secret stops at once. It
// returns pgx.ErrNoRows when the organization has no such integration.
func (r *integrationRepo) setSecret(ctx context.Context, orgID, id int64, secretHash []byte) error {
	return affectedOne(r.q.Exec(ctx, `
		UPDATE integrations SET secret_hash = $3 WHERE organization_id = $1 AND id = $2`,
		orgID, id, secretHash))
}

// delete removes an integration. Its uplinks stay, with no integration. It
// returns pgx.ErrNoRows when the organization has no such integration.
func (r *integrationRepo) delete(ctx context.Context, orgID, id int64) error {
	return affectedOne(r.q.Exec(ctx, `DELETE FROM integrations WHERE organization_id = $1 AND id = $2`,
		orgID, id))
}

// bySecret returns the integration of a secret hash, or pgx.ErrNoRows.
func (r *integrationRepo) bySecret(ctx context.Context, secretHash []byte) (*integrationRef, error) {
	ref := new(integrationRef)
	err := r.q.QueryRow(ctx, `SELECT id, organization_id, kind FROM integrations WHERE secret_hash = $1`,
		secretHash).Scan(&ref.ID, &ref.OrganizationID, &ref.Kind)
	if err != nil {
		return nil, err
	}
	return ref, nil
}

// recordUplink moves the last uplink time forward to at.
func (r *integrationRepo) recordUplink(ctx context.Context, id int64, at time.Time) error {
	_, err := r.q.Exec(ctx, `
		UPDATE integrations SET last_uplink_at = greatest(last_uplink_at, $2) WHERE id = $1`, id, at)
	return err
}

// recordError keeps the last request of an integration that ingestion could
// not read.
func (r *integrationRepo) recordError(ctx context.Context, id int64, at time.Time, code, message string) error {
	_, err := r.q.Exec(ctx, `
		UPDATE integrations SET last_error_at = $2, last_error_code = $3, last_error_message = $4
		WHERE id = $1`, id, at, code, message)
	return err
}

// affectedOne turns an update or a delete that changed no row into
// pgx.ErrNoRows.
func affectedOne(tag interface{ RowsAffected() int64 }, err error) error {
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
