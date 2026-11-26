package device

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/waneock/app-lora/pkg/pgdb"
)

// PostgreSQL error codes that the repos turn into sentinels.
const (
	uniqueViolation     = "23505"
	foreignKeyViolation = "23503"
)

var (
	// errDuplicate is the repo answer to a unique index conflict.
	errDuplicate = errors.New("device: duplicate")
	// errReferenced is the repo answer to a row that another row refers to,
	// or to a reference to a missing row.
	errReferenced = errors.New("device: foreign key")
)

// pgSentinel turns a constraint error into errDuplicate or errReferenced, and
// returns any other error unchanged.
func pgSentinel(err error) error {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	if !ok {
		return err
	}
	switch pgErr.Code {
	case uniqueViolation:
		return errDuplicate
	case foreignKeyViolation:
		return errReferenced
	}
	return err
}

// deviceModel is a row of device_models, with the number of devices that use
// it.
type deviceModel struct {
	ID          int64
	Vendor      string
	Model       string
	DeviceCount int
	CreatedAt   time.Time
}

type modelRepo struct {
	q pgdb.Querier
}

func newModelRepo(q pgdb.Querier) *modelRepo {
	return &modelRepo{q: q}
}

// insert creates a device model. It returns errDuplicate when the
// organization has the same vendor and model name in any case.
func (r *modelRepo) insert(ctx context.Context, orgID int64, vendor, model string) (*deviceModel, error) {
	m := &deviceModel{Vendor: vendor, Model: model}
	err := r.q.QueryRow(ctx, `
		INSERT INTO device_models (organization_id, vendor, model) VALUES ($1, $2, $3)
		RETURNING id, created_at`, orgID, vendor, model).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return nil, pgSentinel(err)
	}
	return m, nil
}

// list returns the device models of an organization by vendor and model name,
// with the number of devices, archived devices too, that use each one.
func (r *modelRepo) list(ctx context.Context, orgID int64) ([]*deviceModel, error) {
	rows, err := r.q.Query(ctx, `
		SELECT m.id, m.vendor, m.model, count(d.id), m.created_at
		FROM device_models m
		LEFT JOIN devices d ON d.organization_id = m.organization_id AND d.device_model_id = m.id
		WHERE m.organization_id = $1
		GROUP BY m.id
		ORDER BY lower(m.vendor), lower(m.model), m.id`, orgID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*deviceModel, error) {
		m := new(deviceModel)
		err := row.Scan(&m.ID, &m.Vendor, &m.Model, &m.DeviceCount, &m.CreatedAt)
		return m, err
	})
}

// rename sets the vendor and the model name. It returns pgx.ErrNoRows when
// the organization has no such model, and errDuplicate on a name conflict.
func (r *modelRepo) rename(ctx context.Context, orgID, id int64, vendor, model string) error {
	tag, err := r.q.Exec(ctx, `
		UPDATE device_models SET vendor = $3, model = $4
		WHERE organization_id = $1 AND id = $2`, orgID, id, vendor, model)
	if err != nil {
		return pgSentinel(err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// delete removes a device model. It returns pgx.ErrNoRows when the
// organization has no such model, and errReferenced when a device uses it.
func (r *modelRepo) delete(ctx context.Context, orgID, id int64) error {
	tag, err := r.q.Exec(ctx, `DELETE FROM device_models WHERE organization_id = $1 AND id = $2`,
		orgID, id)
	if err != nil {
		return pgSentinel(err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// setDecoder sets the decoder of a device model, nil for none. It returns
// pgx.ErrNoRows when there is no such model. Device models of every
// organization are open to it: only the admin changes a decoder.
func (r *modelRepo) setDecoder(ctx context.Context, id int64, decoder *string) error {
	return affectedOne(r.q.Exec(ctx, `UPDATE device_models SET decoder = $2 WHERE id = $1`, id, decoder))
}
