package device

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/waneock/app-lora/pkg/pgdb"
)

// uplink is a row of uplinks as a device page shows it.
type uplink struct {
	ID         int64
	FCnt       int64
	Payload    []byte
	ReceivedAt time.Time
}

// unmatchedDevEUI sums up the unmatched uplinks of one DevEUI.
type unmatchedDevEUI struct {
	DevEUI          string
	Count           int
	FirstStoredAt   time.Time
	LastReceivedAt  time.Time
	IntegrationName *string
}

type uplinkRepo struct {
	q pgdb.Querier
}

func newUplinkRepo(q pgdb.Querier) *uplinkRepo {
	return &uplinkRepo{q: q}
}

// insert stores an uplink of an integration's organization on the live
// device with its DevEUI, or as an unmatched uplink. An uplink of a device is
// marked for the decode worker. A copy of a stored uplink is not stored
// again; insert then reports false.
//
// The device row is share-locked, so a DevEUI change waits for this insert
// (devEUIFree). When dedupSince is set, a stored uplink with the same DevEUI
// and frame counter since then is taken as a copy: an uplink with no time of
// its own cannot use the unique key on the receive time.
func (r *uplinkRepo) insert(ctx context.Context, ref *integrationRef, msg *uplinkMessage,
	receivedAt, storedAt time.Time, dedupSince *time.Time,
) (bool, error) {
	tag, err := r.q.Exec(ctx, `
		WITH d AS (
			SELECT id FROM devices
			WHERE organization_id = $1 AND dev_eui = $3 AND archived_at IS NULL
			FOR SHARE
		)
		INSERT INTO uplinks (organization_id, integration_id, device_id, dev_eui, f_cnt, payload,
			received_at, stored_at, decode_pending)
		SELECT $1, $2, (SELECT id FROM d), $3, $4, $5, $6, $7, EXISTS (SELECT 1 FROM d)
		WHERE $8::timestamptz IS NULL OR NOT EXISTS (
			SELECT 1 FROM uplinks x
			WHERE x.organization_id = $1 AND x.dev_eui = $3 AND x.f_cnt = $4 AND x.stored_at >= $8)
		ON CONFLICT DO NOTHING`,
		ref.OrganizationID, ref.ID, msg.DevEUI, msg.FCnt, msg.Payload, receivedAt, storedAt, dedupSince)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// attach gives the unmatched uplinks with a DevEUI to a device, marks them
// for the decode worker, and returns how many it gave. The device's alarms
// are computed again, because the given uplinks can be older than its
// newest one.
func (r *uplinkRepo) attach(ctx context.Context, orgID, deviceID int64, devEUI string) (int64, error) {
	tag, err := r.q.Exec(ctx, `
		UPDATE uplinks SET device_id = $2, decode_pending = true
		WHERE organization_id = $1 AND device_id IS NULL AND dev_eui = $3`,
		orgID, deviceID, devEUI)
	if err != nil {
		return 0, err
	}
	n := tag.RowsAffected()
	if n > 0 {
		_, err = r.q.Exec(ctx, `UPDATE devices SET alarms_through = NULL WHERE organization_id = $1 AND id = $2`,
			orgID, deviceID)
	}
	return n, err
}

// attachMany gives the unmatched uplinks of each DevEUI in devEUIs to the
// device at the same position of deviceIDs.
func (r *uplinkRepo) attachMany(ctx context.Context, orgID int64, deviceIDs []int64, devEUIs []string) error {
	_, err := r.q.Exec(ctx, `
		UPDATE uplinks u SET device_id = x.id, decode_pending = true
		FROM unnest($2::bigint[], $3::text[]) AS x(id, eui)
		WHERE u.organization_id = $1 AND u.device_id IS NULL AND u.dev_eui = x.eui`,
		orgID, deviceIDs, devEUIs)
	return err
}

// hasAny reports whether a device has an uplink.
func (r *uplinkRepo) hasAny(ctx context.Context, orgID, deviceID int64) (bool, error) {
	var ok bool
	err := r.q.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM uplinks WHERE organization_id = $1 AND device_id = $2)`,
		orgID, deviceID).Scan(&ok)
	return ok, err
}

// listForDevice returns a page of the uplinks of a device, the newest first,
// and the number of its uplinks.
func (r *uplinkRepo) listForDevice(ctx context.Context, orgID, deviceID int64, limit, offset int) ([]uplink, int, error) {
	var total int
	err := r.q.QueryRow(ctx, `
		SELECT count(*) FROM uplinks WHERE organization_id = $1 AND device_id = $2`,
		orgID, deviceID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	rows, err := r.q.Query(ctx, `
		SELECT id, f_cnt, payload, received_at FROM uplinks
		WHERE organization_id = $1 AND device_id = $2
		ORDER BY received_at DESC, id DESC
		LIMIT $3 OFFSET $4`, orgID, deviceID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	ups, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (uplink, error) {
		var u uplink
		err := row.Scan(&u.ID, &u.FCnt, &u.Payload, &u.ReceivedAt)
		return u, err
	})
	return ups, total, err
}

// unmatched sums up the unmatched uplinks of an organization by DevEUI, the
// most recent first.
func (r *uplinkRepo) unmatched(ctx context.Context, orgID int64) ([]unmatchedDevEUI, error) {
	rows, err := r.q.Query(ctx, `
		SELECT u.dev_eui, count(*), min(u.stored_at), max(u.received_at),
			(SELECT i.name FROM uplinks l JOIN integrations i ON i.id = l.integration_id
			 WHERE l.organization_id = u.organization_id AND l.device_id IS NULL AND l.dev_eui = u.dev_eui
			 ORDER BY l.stored_at DESC LIMIT 1)
		FROM uplinks u
		WHERE u.organization_id = $1 AND u.device_id IS NULL
		GROUP BY u.organization_id, u.dev_eui
		ORDER BY max(u.received_at) DESC`, orgID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (unmatchedDevEUI, error) {
		var x unmatchedDevEUI
		err := row.Scan(&x.DevEUI, &x.Count, &x.FirstStoredAt, &x.LastReceivedAt, &x.IntegrationName)
		return x, err
	})
}

// purgeUnmatched deletes the unmatched uplinks stored before before, of all
// organizations, and returns how many it deleted.
func (r *uplinkRepo) purgeUnmatched(ctx context.Context, before time.Time) (int64, error) {
	tag, err := r.q.Exec(ctx, `DELETE FROM uplinks WHERE device_id IS NULL AND stored_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
