package device

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/waneock/app-lora/pkg/pgdb"
)

// pulsePoint is the pulse count that one decoded uplink reported.
type pulsePoint struct {
	At         time.Time
	PulseCount int64
}

// dayPulse is the last pulse count of one day.
type dayPulse struct {
	Day  time.Time
	Last int64
}

// alarmRow is a stored alarm as a device page shows it.
type alarmRow struct {
	ID        int64
	Kind      string
	RaisedAt  time.Time
	ClearedAt *time.Time
}

// readingRepo reads the decoded pulse counts and the alarms of devices.
type readingRepo struct {
	q pgdb.Querier
}

func newReadingRepo(q pgdb.Querier) *readingRepo {
	return &readingRepo{q: q}
}

// points returns the pulse counts of a device received at or after from,
// in time order.
func (r *readingRepo) points(ctx context.Context, orgID, deviceID int64, from time.Time) ([]pulsePoint, error) {
	rows, err := r.q.Query(ctx, `
		SELECT received_at, pulse_count FROM uplinks
		WHERE organization_id = $1 AND device_id = $2 AND pulse_count IS NOT NULL AND received_at >= $3
		ORDER BY received_at, id`, orgID, deviceID, from)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (pulsePoint, error) {
		var p pulsePoint
		err := row.Scan(&p.At, &p.PulseCount)
		return p, err
	})
}

// days returns the last pulse count of each day at or after from, in the
// time zone tz. A pulse count only increases (device:N8), so the last one
// of a day is its maximum.
func (r *readingRepo) days(ctx context.Context, orgID, deviceID int64, from time.Time, tz string) ([]dayPulse, error) {
	rows, err := r.q.Query(ctx, `
		SELECT (received_at AT TIME ZONE $4)::date AS day, max(pulse_count)
		FROM uplinks
		WHERE organization_id = $1 AND device_id = $2 AND pulse_count IS NOT NULL AND received_at >= $3
		GROUP BY day ORDER BY day`, orgID, deviceID, from, tz)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (dayPulse, error) {
		var d dayPulse
		err := row.Scan(&d.Day, &d.Last)
		return d, err
	})
}

// lastBefore returns the last pulse count of a device before before, or nil.
// It reads one row through the partial index of decoded uplinks.
func (r *readingRepo) lastBefore(ctx context.Context, orgID, deviceID int64, before time.Time) (*int64, error) {
	var n int64
	err := r.q.QueryRow(ctx, `
		SELECT pulse_count FROM uplinks
		WHERE organization_id = $1 AND device_id = $2 AND pulse_count IS NOT NULL AND received_at < $3
		ORDER BY received_at DESC, id DESC LIMIT 1`,
		orgID, deviceID, before).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// alarms returns a page of the alarms of a device, the newest first, and
// the number of its alarms.
func (r *readingRepo) alarms(ctx context.Context, orgID, deviceID int64, limit, offset int) ([]alarmRow, int, error) {
	var total int
	if err := r.q.QueryRow(ctx, `SELECT count(*) FROM alarms WHERE organization_id = $1 AND device_id = $2`,
		orgID, deviceID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.q.Query(ctx, `
		SELECT id, kind, raised_at, cleared_at FROM alarms
		WHERE organization_id = $1 AND device_id = $2
		ORDER BY raised_at DESC, id DESC LIMIT $3 OFFSET $4`, orgID, deviceID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (alarmRow, error) {
		var a alarmRow
		err := row.Scan(&a.ID, &a.Kind, &a.RaisedAt, &a.ClearedAt)
		return a, err
	})
	return out, total, err
}

// activeKinds returns the kinds of the active alarms of each device in
// deviceIDs.
func (r *readingRepo) activeKinds(ctx context.Context, orgID int64, deviceIDs []int64) (map[int64][]string, error) {
	rows, err := r.q.Query(ctx, `
		SELECT device_id, kind FROM alarms
		WHERE organization_id = $1 AND device_id = ANY($2) AND cleared_at IS NULL
		ORDER BY kind`, orgID, deviceIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[int64][]string)
	var id int64
	var kind string
	_, err = pgx.ForEachRow(rows, []any{&id, &kind}, func() error {
		out[id] = append(out[id], kind)
		return nil
	})
	return out, err
}
