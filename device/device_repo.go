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
	// HasDecoder tells whether the device model has a decoder.
	HasDecoder bool
	// LastUplinkAt is the receive time of the newest uplink, or nil.
	LastUplinkAt *time.Time
	// ActiveAlarms is filled by the service: the kinds of the active alarms.
	ActiveAlarms []string
	// Groups is filled by the service, not by the device queries.
	Groups []groupRef
}

// listQuery selects a page of the devices of an organization.
type listQuery struct {
	// Pattern is a LIKE pattern on the search column, or "" for no search.
	Pattern   string
	ModelID   int64
	GroupID   int64
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
	d.meter_type, d.address, d.note, d.archived_at, d.created_at, d.updated_at,
	m.decoder IS NOT NULL,
	(SELECT max(u.received_at) FROM uplinks u
	 WHERE u.organization_id = d.organization_id AND u.device_id = d.id)`

const deviceFrom = `devices d JOIN device_models m
	ON m.organization_id = d.organization_id AND m.id = d.device_model_id`

func scanDevice(row pgx.Row, extra ...any) (*device, error) {
	d := new(device)
	dest := append([]any{&d.ID, &d.DevEUI, &d.Name, &d.DeviceModelID, &d.Vendor, &d.Model,
		&d.MeterType, &d.Address, &d.Note, &d.ArchivedAt, &d.CreatedAt, &d.UpdatedAt,
		&d.HasDecoder, &d.LastUplinkAt}, extra...)
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
		WHERE `+listWhere+`
		ORDER BY lower(d.name), d.id
		LIMIT $7 OFFSET $8`,
		orgID, q.Archived, q.Pattern, q.ModelID, q.MeterType, q.GroupID, q.Limit, q.Offset)
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
	err := r.q.QueryRow(ctx, `SELECT count(*) FROM devices d WHERE `+listWhere,
		orgID, q.Archived, q.Pattern, q.ModelID, q.MeterType, q.GroupID).Scan(&n)
	return n, err
}

// listWhere is the filter of list and count. Its parameters are $1 the
// organization, $2 archived, $3 the search pattern, $4 the model, $5 the
// meter type, and $6 the group. A zero or empty value does not filter.
const listWhere = `d.organization_id = $1
	AND (d.archived_at IS NOT NULL) = $2
	AND ($3 = '' OR d.search LIKE $3)
	AND ($4 = 0 OR d.device_model_id = $4)
	AND ($5 = '' OR d.meter_type = $5)
	AND ($6 = 0 OR EXISTS (
		SELECT 1 FROM device_group_members gm
		WHERE gm.organization_id = d.organization_id AND gm.device_id = d.id AND gm.group_id = $6))`

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

// existingDevEUIs returns the DevEUIs of devEUIs that the organization has,
// archived devices too.
func (r *deviceRepo) existingDevEUIs(ctx context.Context, orgID int64, devEUIs []string) (map[string]bool, error) {
	rows, err := r.q.Query(ctx, `
		SELECT dev_eui FROM devices WHERE organization_id = $1 AND dev_eui = ANY($2)`,
		orgID, devEUIs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool)
	var eui string
	_, err = pgx.ForEachRow(rows, []any{&eui}, func() error {
		out[eui] = true
		return nil
	})
	return out, err
}

// insertMany creates devices with one statement and returns the id of each
// DevEUI. The fields must be normalized and the models of the organization.
// It returns errDuplicate when the organization has one of the DevEUIs.
func (r *deviceRepo) insertMany(ctx context.Context, orgID int64, fields []deviceFields) (map[string]int64, error) {
	n := len(fields)
	euis, names, types, addresses, notes := make([]string, n), make([]string, n), make([]string, n), make([]string, n), make([]string, n)
	models := make([]int64, n)
	for i, f := range fields {
		euis[i], names[i], models[i] = f.DevEUI, f.Name, f.DeviceModelID
		types[i], addresses[i], notes[i] = f.MeterType, f.Address, f.Note
	}
	rows, err := r.q.Query(ctx, `
		INSERT INTO devices (organization_id, dev_eui, name, device_model_id, meter_type, address, note)
		SELECT $1, e, n, m, nullif(t, ''), nullif(a, ''), nullif(o, '')
		FROM unnest($2::text[], $3::text[], $4::bigint[], $5::text[], $6::text[], $7::text[])
			AS u(e, n, m, t, a, o)
		RETURNING id, dev_eui`,
		orgID, euis, names, models, types, addresses, notes)
	if err != nil {
		return nil, pgSentinel(err)
	}
	ids := make(map[string]int64, n)
	var id int64
	var eui string
	_, err = pgx.ForEachRow(rows, []any{&id, &eui}, func() error {
		ids[eui] = id
		return nil
	})
	return ids, pgSentinel(err)
}

// lockForUpdate locks the row of a device until the transaction ends. It
// returns pgx.ErrNoRows when the organization has no such device.
func (r *deviceRepo) lockForUpdate(ctx context.Context, orgID, id int64) error {
	var got int64
	return r.q.QueryRow(ctx, `SELECT id FROM devices WHERE organization_id = $1 AND id = $2 FOR UPDATE`,
		orgID, id).Scan(&got)
}
