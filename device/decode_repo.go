package device

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/waneock/app-lora/pkg/pgdb"
)

// pendingUplink is an uplink that the decode worker must decode, with the
// decoder of its device's model.
type pendingUplink struct {
	ID             int64
	OrganizationID int64
	DeviceID       int64
	Payload        []byte
	ReceivedAt     time.Time
	// Decoder is the decoder of the device model, or nil.
	Decoder *string
}

// decodedValues are the values that a decode writes to an uplink.
type decodedValues struct {
	Decoder    *string
	PulseCount *int64
	Reported   []string
	Raised     []string
	Error      *string
}

// deviceKey names a device of an organization.
type deviceKey struct {
	OrganizationID int64
	DeviceID       int64
}

type decodeRepo struct {
	q pgdb.Querier
}

func newDecodeRepo(q pgdb.Querier) *decodeRepo {
	return &decodeRepo{q: q}
}

// pending locks and returns up to limit pending uplinks, the oldest first.
// Run it in a transaction; SKIP LOCKED lets a second worker take other rows.
func (r *decodeRepo) pending(ctx context.Context, limit int) ([]pendingUplink, error) {
	rows, err := r.q.Query(ctx, `
		SELECT u.id, u.organization_id, u.device_id, u.payload, u.received_at, m.decoder
		FROM uplinks u
		JOIN devices d ON d.organization_id = u.organization_id AND d.id = u.device_id
		JOIN device_models m ON m.organization_id = d.organization_id AND m.id = d.device_model_id
		WHERE u.decode_pending
		ORDER BY u.id
		LIMIT $1
		FOR UPDATE OF u SKIP LOCKED`, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (pendingUplink, error) {
		var p pendingUplink
		err := row.Scan(&p.ID, &p.OrganizationID, &p.DeviceID, &p.Payload, &p.ReceivedAt, &p.Decoder)
		return p, err
	})
}

// save writes the decoded values of each uplink in ids and clears its
// pending mark.
func (r *decodeRepo) save(ctx context.Context, ids []int64, values []decodedValues) error {
	batch := &pgx.Batch{}
	for i, id := range ids {
		v := values[i]
		batch.Queue(`
			UPDATE uplinks SET decoder = $2, pulse_count = $3, alarms_reported = $4,
				alarms_raised = $5, decode_error = $6, decode_pending = false
			WHERE id = $1`, id, v.Decoder, v.PulseCount, v.Reported, v.Raised, v.Error)
	}
	return r.q.SendBatch(ctx, batch).Close()
}

// hasPending reports whether a device has an uplink to decode.
func (r *decodeRepo) hasPending(ctx context.Context, k deviceKey) (bool, error) {
	var ok bool
	err := r.q.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM uplinks
			WHERE organization_id = $1 AND device_id = $2 AND decode_pending)`,
		k.OrganizationID, k.DeviceID).Scan(&ok)
	return ok, err
}

// alarmsThrough returns the receive time up to which the alarms of a device
// are computed, or nil when they must be computed again.
func (r *decodeRepo) alarmsThrough(ctx context.Context, k deviceKey) (*time.Time, error) {
	var at *time.Time
	err := r.q.QueryRow(ctx, `SELECT alarms_through FROM devices WHERE organization_id = $1 AND id = $2`,
		k.OrganizationID, k.DeviceID).Scan(&at)
	return at, err
}

func (r *decodeRepo) setAlarmsThrough(ctx context.Context, k deviceKey, at *time.Time) error {
	_, err := r.q.Exec(ctx, `UPDATE devices SET alarms_through = $3 WHERE organization_id = $1 AND id = $2`,
		k.OrganizationID, k.DeviceID, at)
	return err
}

// decodedRows returns the uplinks of a device that report an alarm kind, in
// time order.
func (r *decodeRepo) decodedRows(ctx context.Context, k deviceKey) ([]decodedRow, error) {
	rows, err := r.q.Query(ctx, `
		SELECT id, received_at, alarms_reported, coalesce(alarms_raised, '{}')
		FROM uplinks
		WHERE organization_id = $1 AND device_id = $2 AND cardinality(alarms_reported) > 0
		ORDER BY received_at, id`, k.OrganizationID, k.DeviceID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (decodedRow, error) {
		var d decodedRow
		err := row.Scan(&d.UplinkID, &d.ReceivedAt, &d.Reported, &d.Raised)
		return d, err
	})
}

// lastReceivedAt returns the receive time of the newest uplink of a device,
// or nil when it has none.
func (r *decodeRepo) lastReceivedAt(ctx context.Context, k deviceKey) (*time.Time, error) {
	var at *time.Time
	err := r.q.QueryRow(ctx, `
		SELECT max(received_at) FROM uplinks WHERE organization_id = $1 AND device_id = $2`,
		k.OrganizationID, k.DeviceID).Scan(&at)
	return at, err
}

// activeAlarms returns the active alarms of a device.
func (r *decodeRepo) activeAlarms(ctx context.Context, k deviceKey) ([]*alarmPeriod, error) {
	rows, err := r.q.Query(ctx, `
		SELECT id, kind, raised_at, coalesce(raised_uplink_id, 0)
		FROM alarms WHERE organization_id = $1 AND device_id = $2 AND cleared_at IS NULL`,
		k.OrganizationID, k.DeviceID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*alarmPeriod, error) {
		a := new(alarmPeriod)
		err := row.Scan(&a.ID, &a.Kind, &a.RaisedAt, &a.RaisedUplinkID)
		return a, err
	})
}

// deleteAlarms removes the alarms of a device before a replay.
func (r *decodeRepo) deleteAlarms(ctx context.Context, k deviceKey) error {
	_, err := r.q.Exec(ctx, `DELETE FROM alarms WHERE organization_id = $1 AND device_id = $2`,
		k.OrganizationID, k.DeviceID)
	return err
}

// writeAlarms stores the alarms that applyAlarms touched: it inserts the new
// ones and clears the changed ones.
func (r *decodeRepo) writeAlarms(ctx context.Context, k deviceKey, alarms []*alarmPeriod) error {
	batch := &pgx.Batch{}
	for _, a := range alarms {
		var cleared *int64
		if a.ClearedUplinkID != 0 {
			cleared = &a.ClearedUplinkID
		}
		if a.ID == 0 {
			var raised *int64
			if a.RaisedUplinkID != 0 {
				raised = &a.RaisedUplinkID
			}
			batch.Queue(`
				INSERT INTO alarms (organization_id, device_id, kind, raised_at, cleared_at,
					raised_uplink_id, cleared_uplink_id)
				VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				k.OrganizationID, k.DeviceID, a.Kind, a.RaisedAt, a.ClearedAt, raised, cleared)
			continue
		}
		batch.Queue(`UPDATE alarms SET cleared_at = $2, cleared_uplink_id = $3 WHERE id = $1`,
			a.ID, a.ClearedAt, cleared)
	}
	return r.q.SendBatch(ctx, batch).Close()
}

// markDevice marks every uplink of a device to decode again, and its alarms
// to compute again: its device model changed.
func (r *decodeRepo) markDevice(ctx context.Context, k deviceKey) error {
	if _, err := r.q.Exec(ctx, `
		UPDATE uplinks SET decode_pending = true WHERE organization_id = $1 AND device_id = $2`,
		k.OrganizationID, k.DeviceID); err != nil {
		return err
	}
	return r.setAlarmsThrough(ctx, k, nil)
}

// markModel marks every uplink of the devices of a device model to decode
// again, and their alarms to compute again: the decoder of the model
// changed.
func (r *decodeRepo) markModel(ctx context.Context, modelID int64) error {
	if _, err := r.q.Exec(ctx, `
		UPDATE uplinks u SET decode_pending = true
		FROM devices d
		WHERE d.organization_id = u.organization_id AND d.id = u.device_id AND d.device_model_id = $1`,
		modelID); err != nil {
		return err
	}
	_, err := r.q.Exec(ctx, `UPDATE devices SET alarms_through = NULL WHERE device_model_id = $1`, modelID)
	return err
}
