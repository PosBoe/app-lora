package device

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/waneock/app-lora/pkg/pgdb"
)

// device is a row of devices with the names of its device model.
type device struct {
	ID            int64
	DevEUI        string
	Name          string
	DeviceModelID int64
	Vendor        string
	Model         string
	MeterType     *string
	Address       *string
	Note          *string
	ArchivedAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// listQuery selects a page of the devices of an organization.
type listQuery struct {
	// Pattern is a LIKE pattern on the search column, or "" for no search.
	Pattern   string
	ModelID   int64
	MeterType string
	Archived  bool
	Limit     int
	Offset    int
}

type deviceRepo struct {
	q pgdb.Querier
}

func newDeviceRepo(q pgdb.Querier) *deviceRepo {
	return &deviceRepo{q: q}
}

const deviceColumns = `d.id, d.dev_eui, d.name, d.device_model_id, m.vendor, m.model,
	d.meter_type, d.address, d.note, d.archived_at, d.created_at, d.updated_at`

const deviceFrom = `devices d JOIN device_models m
	ON m.organization_id = d.organization_id AND m.id = d.device_model_id`

func scanDevice(row pgx.Row, extra ...any) (*device, error) {
	d := new(device)
	dest := append([]any{&d.ID, &d.DevEUI, &d.Name, &d.DeviceModelID, &d.Vendor, &d.Model,
		&d.MeterType, &d.Address, &d.Note, &d.ArchivedAt, &d.CreatedAt, &d.UpdatedAt}, extra...)
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	return d, nil
}

// nullable stores "" as NULL.
func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// insert creates a device and returns its id. It returns errDuplicate when
// the organization has the DevEUI, and errReferenced when the device model is
// not in the organization.
func (r *deviceRepo) insert(ctx context.Context, orgID int64, f deviceFields) (int64, error) {
	var id int64
	err := r.q.QueryRow(ctx, `
		INSERT INTO devices (organization_id, dev_eui, name, device_model_id, meter_type, address, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
		orgID, f.DevEUI, f.Name, f.DeviceModelID, nullable(f.MeterType), nullable(f.Address),
		nullable(f.Note),
	).Scan(&id)
	return id, pgSentinel(err)
}

// update writes the fields of a device. It returns pgx.ErrNoRows when the
// organization has no such device, and the insert sentinels on a conflict.
func (r *deviceRepo) update(ctx context.Context, orgID, id int64, f deviceFields, at time.Time) error {
	tag, err := r.q.Exec(ctx, `
		UPDATE devices SET dev_eui = $3, name = $4, device_model_id = $5, meter_type = $6,
			address = $7, note = $8, updated_at = $9
		WHERE organization_id = $1 AND id = $2`,
		orgID, id, f.DevEUI, f.Name, f.DeviceModelID, nullable(f.MeterType), nullable(f.Address),
		nullable(f.Note), at)
	if err != nil {
		return pgSentinel(err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// selectByID returns pgx.ErrNoRows when the organization has no such device.
func (r *deviceRepo) selectByID(ctx context.Context, orgID, id int64) (*device, error) {
	return scanDevice(r.q.QueryRow(ctx, `SELECT `+deviceColumns+` FROM `+deviceFrom+`
		WHERE d.organization_id = $1 AND d.id = $2`, orgID, id))
}

// selectByDevEUI returns pgx.ErrNoRows when the organization has no device
// with the DevEUI.
func (r *deviceRepo) selectByDevEUI(ctx context.Context, orgID int64, devEUI string) (*device, error) {
	return scanDevice(r.q.QueryRow(ctx, `SELECT `+deviceColumns+` FROM `+deviceFrom+`
		WHERE d.organization_id = $1 AND d.dev_eui = $2`, orgID, devEUI))
}

// list returns one page of devices by name, and the number of devices that
// match the query on all pages.
func (r *deviceRepo) list(ctx context.Context, orgID int64, q listQuery) ([]*device, int, error) {
	rows, err := r.q.Query(ctx, `
		SELECT `+deviceColumns+`, count(*) OVER ()
		FROM `+deviceFrom+`
		WHERE d.organization_id = $1
			AND (d.archived_at IS NOT NULL) = $2
			AND ($3 = '' OR d.search LIKE $3)
			AND ($4 = 0 OR d.device_model_id = $4)
			AND ($5 = '' OR d.meter_type = $5)
		ORDER BY lower(d.name), d.id
		LIMIT $6 OFFSET $7`,
		orgID, q.Archived, q.Pattern, q.ModelID, q.MeterType, q.Limit, q.Offset)
	if err != nil {
		return nil, 0, err
	}
	var total int
	devices, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (*device, error) {
		return scanDevice(row, &total)
	})
	if err != nil {
		return nil, 0, err
	}
	if len(devices) == 0 && q.Offset > 0 {
		// A page after the last one has no row to carry the total.
		total, err = r.count(ctx, orgID, q)
	}
	return devices, total, err
}

func (r *deviceRepo) count(ctx context.Context, orgID int64, q listQuery) (int, error) {
	var n int
	err := r.q.QueryRow(ctx, `
		SELECT count(*) FROM devices d
		WHERE d.organization_id = $1
			AND (d.archived_at IS NOT NULL) = $2
			AND ($3 = '' OR d.search LIKE $3)
			AND ($4 = 0 OR d.device_model_id = $4)
			AND ($5 = '' OR d.meter_type = $5)`,
		orgID, q.Archived, q.Pattern, q.ModelID, q.MeterType).Scan(&n)
	return n, err
}

// setArchived archives the devices with ids at the time at, or restores them
// when at is nil. It changes only devices of the organization whose state
// differs, and returns how many changed.
func (r *deviceRepo) setArchived(ctx context.Context, orgID int64, ids []int64, at *time.Time) (int64, error) {
	tag, err := r.q.Exec(ctx, `
		UPDATE devices SET archived_at = $3
		WHERE organization_id = $1 AND id = ANY($2)
			AND (archived_at IS NULL) = ($3::timestamptz IS NOT NULL)`,
		orgID, ids, at)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
