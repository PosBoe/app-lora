package device

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/waneock/app-lora/pkg/pgdb"
)

// Stats sums up devices and uplinks for the admin views of the core context.
type Stats struct {
	q pgdb.Querier
}

func newStats(q pgdb.Querier) *Stats {
	return &Stats{q: q}
}

// DevicesByOrganization returns the number of devices that are not archived,
// for each organization that has one.
func (s *Stats) DevicesByOrganization(ctx context.Context) (map[int64]int, error) {
	rows, err := s.q.Query(ctx, `
		SELECT organization_id, count(*) FROM devices
		WHERE archived_at IS NULL GROUP BY organization_id`)
	if err != nil {
		return nil, err
	}
	counts := make(map[int64]int)
	var orgID int64
	var n int
	_, err = pgx.ForEachRow(rows, []any{&orgID, &n}, func() error {
		counts[orgID] = n
		return nil
	})
	return counts, err
}

// LastUplinkByOrganization returns the arrival time of the newest uplink of
// each organization that has one.
func (s *Stats) LastUplinkByOrganization(ctx context.Context) (map[int64]time.Time, error) {
	rows, err := s.q.Query(ctx, `SELECT organization_id, max(stored_at) FROM uplinks GROUP BY organization_id`)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]time.Time)
	var orgID int64
	var at time.Time
	_, err = pgx.ForEachRow(rows, []any{&orgID, &at}, func() error {
		out[orgID] = at
		return nil
	})
	return out, err
}
