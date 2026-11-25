package device

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/waneock/app-lora/pkg/pgdb"
)

// deviceGroup is a row of device_groups, with the number of its devices that
// are not archived.
type deviceGroup struct {
	ID          int64
	Name        string
	DeviceCount int
	CreatedAt   time.Time
}

// groupRef names a group of a device.
type groupRef struct {
	ID   int64
	Name string
}

type groupRepo struct {
	q pgdb.Querier
}

func newGroupRepo(q pgdb.Querier) *groupRepo {
	return &groupRepo{q: q}
}

// insert creates a group. It returns errDuplicate when the organization has
// the name in any case.
func (r *groupRepo) insert(ctx context.Context, orgID int64, name string) (*deviceGroup, error) {
	g := &deviceGroup{Name: name}
	err := r.q.QueryRow(ctx, `
		INSERT INTO device_groups (organization_id, name) VALUES ($1, $2)
		RETURNING id, created_at`, orgID, name).Scan(&g.ID, &g.CreatedAt)
	if err != nil {
		return nil, pgSentinel(err)
	}
	return g, nil
}

// list returns the groups of an organization by name.
func (r *groupRepo) list(ctx context.Context, orgID int64) ([]*deviceGroup, error) {
	rows, err := r.q.Query(ctx, `
		SELECT g.id, g.name, count(d.id), g.created_at
		FROM device_groups g
		LEFT JOIN device_group_members m ON m.group_id = g.id
		LEFT JOIN devices d ON d.organization_id = m.organization_id AND d.id = m.device_id
			AND d.archived_at IS NULL
		WHERE g.organization_id = $1
		GROUP BY g.id
		ORDER BY lower(g.name), g.id`, orgID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*deviceGroup, error) {
		g := new(deviceGroup)
		err := row.Scan(&g.ID, &g.Name, &g.DeviceCount, &g.CreatedAt)
		return g, err
	})
}

// exists reports whether the organization has the group.
func (r *groupRepo) exists(ctx context.Context, orgID, id int64) (bool, error) {
	var ok bool
	err := r.q.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM device_groups WHERE organization_id = $1 AND id = $2)`,
		orgID, id).Scan(&ok)
	return ok, err
}

// rename sets the name of a group. It returns pgx.ErrNoRows when the
// organization has no such group, and errDuplicate on a name conflict.
func (r *groupRepo) rename(ctx context.Context, orgID, id int64, name string) error {
	tag, err := r.q.Exec(ctx, `UPDATE device_groups SET name = $3 WHERE organization_id = $1 AND id = $2`,
		orgID, id, name)
	if err != nil {
		return pgSentinel(err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// delete removes a group and its memberships. The devices stay. It returns
// pgx.ErrNoRows when the organization has no such group.
func (r *groupRepo) delete(ctx context.Context, orgID, id int64) error {
	tag, err := r.q.Exec(ctx, `DELETE FROM device_groups WHERE organization_id = $1 AND id = $2`,
		orgID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// add puts the devices with deviceIDs in a group. Ids that are not devices of
// the organization, and devices already in the group, are left alone. It
// returns errReferenced when the group is not in the organization, and how
// many devices it added.
func (r *groupRepo) add(ctx context.Context, orgID, groupID int64, deviceIDs []int64) (int64, error) {
	tag, err := r.q.Exec(ctx, `
		INSERT INTO device_group_members (organization_id, group_id, device_id)
		SELECT $1, $2, d.id FROM devices d WHERE d.organization_id = $1 AND d.id = ANY($3)
		ON CONFLICT DO NOTHING`, orgID, groupID, deviceIDs)
	if err != nil {
		return 0, pgSentinel(err)
	}
	return tag.RowsAffected(), nil
}

// remove takes the devices with deviceIDs out of a group, and returns how
// many it removed.
func (r *groupRepo) remove(ctx context.Context, orgID, groupID int64, deviceIDs []int64) (int64, error) {
	tag, err := r.q.Exec(ctx, `
		DELETE FROM device_group_members
		WHERE organization_id = $1 AND group_id = $2 AND device_id = ANY($3)`,
		orgID, groupID, deviceIDs)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// setForDevice makes groupIDs the groups of a device. It returns
// errReferenced when a group is not in the organization.
func (r *groupRepo) setForDevice(ctx context.Context, orgID, deviceID int64, groupIDs []int64) error {
	_, err := r.q.Exec(ctx, `
		DELETE FROM device_group_members
		WHERE organization_id = $1 AND device_id = $2 AND NOT (group_id = ANY($3))`,
		orgID, deviceID, groupIDs)
	if err != nil {
		return err
	}
	_, err = r.q.Exec(ctx, `
		INSERT INTO device_group_members (organization_id, group_id, device_id)
		SELECT $1, g, $2 FROM unnest($3::bigint[]) AS g
		ON CONFLICT DO NOTHING`, orgID, deviceID, groupIDs)
	return pgSentinel(err)
}

// byDevice returns the groups of each device in deviceIDs, by name.
func (r *groupRepo) byDevice(ctx context.Context, orgID int64, deviceIDs []int64) (map[int64][]groupRef, error) {
	rows, err := r.q.Query(ctx, `
		SELECT m.device_id, g.id, g.name
		FROM device_group_members m
		JOIN device_groups g ON g.id = m.group_id
		WHERE m.organization_id = $1 AND m.device_id = ANY($2)
		ORDER BY lower(g.name), g.id`, orgID, deviceIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[int64][]groupRef)
	var deviceID int64
	var g groupRef
	_, err = pgx.ForEachRow(rows, []any{&deviceID, &g.ID, &g.Name}, func() error {
		out[deviceID] = append(out[deviceID], g)
		return nil
	})
	return out, err
}
