package device

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waneock/app-lora/pkg/clock"
)

// maxBulkIDs bounds the ids of one bulk archive.
const maxBulkIDs = 1000

// deviceService holds the rules of the devices of an organization. Every
// method takes the organization of the caller, never one from the request.
type deviceService struct {
	pool    *pgxpool.Pool
	devices *deviceRepo
	groups  *groupRepo
	uplinks *uplinkRepo
	clock   clock.Clock
}

func newDeviceService(pool *pgxpool.Pool, clk clock.Clock) *deviceService {
	return &deviceService{
		pool:    pool,
		devices: newDeviceRepo(pool),
		groups:  newGroupRepo(pool),
		uplinks: newUplinkRepo(pool),
		clock:   clk,
	}
}

// create adds a device to the organization with its groups, in one
// transaction. The device gets the unmatched uplinks with its DevEUI
// (device:G8).
func (s *deviceService) create(ctx context.Context, orgID int64, f deviceFields) (*device, error) {
	f, err := f.normalize()
	if err != nil {
		return nil, err
	}
	var id int64
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		if id, err = newDeviceRepo(tx).insert(ctx, orgID, f); err != nil {
			return err
		}
		if err := setGroups(ctx, tx, orgID, id, f.GroupIDs); err != nil {
			return err
		}
		_, err = newUplinkRepo(tx).attach(ctx, orgID, id, f.DevEUI)
		return err
	})
	if err != nil {
		return nil, s.writeError(ctx, orgID, f.DevEUI, err)
	}
	return s.get(ctx, orgID, id)
}

// setGroups makes groupIDs the groups of a device in tx.
func setGroups(ctx context.Context, tx pgx.Tx, orgID, deviceID int64, groupIDs []int64) error {
	err := newGroupRepo(tx).setForDevice(ctx, orgID, deviceID, groupIDs)
	if errors.Is(err, errReferenced) {
		return errGroupNotFound
	}
	return err
}

// update writes the fields of a device. The DevEUI changes only while the
// device has no uplink, because the DevEUI identifies the history of the
// device after its first uplink.
func (s *deviceService) update(ctx context.Context, orgID, id int64, f deviceFields) (*device, error) {
	f, err := f.normalize()
	if err != nil {
		return nil, err
	}
	current, err := s.get(ctx, orgID, id)
	if err != nil {
		return nil, err
	}
	if f.DevEUI != current.DevEUI {
		has, err := s.hasUplinks(ctx, orgID, id)
		if err != nil {
			return nil, err
		}
		if has {
			return nil, errDevEUILocked
		}
	}

	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := newDeviceRepo(tx).update(ctx, orgID, id, f, s.clock.Now()); err != nil {
			return err
		}
		if err := setGroups(ctx, tx, orgID, id, f.GroupIDs); err != nil {
			return err
		}
		if f.DevEUI == current.DevEUI || current.ArchivedAt != nil {
			return nil
		}
		_, err := newUplinkRepo(tx).attach(ctx, orgID, id, f.DevEUI)
		return err
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errNotFound
	}
	if err != nil {
		return nil, s.writeError(ctx, orgID, f.DevEUI, err)
	}
	return s.get(ctx, orgID, id)
}

// hasUplinks reports whether a device has an uplink.
func (s *deviceService) hasUplinks(ctx context.Context, orgID, id int64) (bool, error) {
	return s.uplinks.hasAny(ctx, orgID, id)
}

// writeError turns a constraint error of an insert or an update into a rule
// error. A taken DevEUI tells whether the device that has it is archived, so
// the user can restore that device.
func (s *deviceService) writeError(ctx context.Context, orgID int64, devEUI string, err error) error {
	switch {
	case errors.Is(err, errReferenced):
		return errModelNotFound
	case errors.Is(err, errDuplicate):
		other, lookupErr := s.devices.selectByDevEUI(ctx, orgID, devEUI)
		if lookupErr == nil && other.ArchivedAt != nil {
			return errDevEUIArchived
		}
		return errDevEUITaken
	}
	return err
}

// get returns a device of the organization with its groups, or
// errNotFound.
func (s *deviceService) get(ctx context.Context, orgID, id int64) (*device, error) {
	d, err := s.devices.selectByID(ctx, orgID, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := s.fillGroups(ctx, orgID, []*device{d}); err != nil {
		return nil, err
	}
	return d, nil
}

// list returns a page of devices with their groups, and the number of
// devices on all pages.
func (s *deviceService) list(ctx context.Context, orgID int64, q listQuery) ([]*device, int, error) {
	devices, total, err := s.devices.list(ctx, orgID, q)
	if err != nil {
		return nil, 0, err
	}
	if err := s.fillGroups(ctx, orgID, devices); err != nil {
		return nil, 0, err
	}
	return devices, total, nil
}

// fillGroups sets the Groups of each device with one query. It mutates the
// devices.
func (s *deviceService) fillGroups(ctx context.Context, orgID int64, devices []*device) error {
	if len(devices) == 0 {
		return nil
	}
	ids := make([]int64, len(devices))
	for i, d := range devices {
		ids[i] = d.ID
	}
	byDevice, err := s.groups.byDevice(ctx, orgID, ids)
	if err != nil {
		return err
	}
	for _, d := range devices {
		d.Groups = byDevice[d.ID]
	}
	return nil
}

// archive archives the devices with ids. Ids of other organizations and
// devices that are archived already are left alone. It returns how many
// devices it archived.
func (s *deviceService) archive(ctx context.Context, orgID int64, ids []int64) (int64, error) {
	if len(ids) > maxBulkIDs {
		return 0, errTooManyIDs
	}
	now := s.clock.Now()
	return s.devices.setArchived(ctx, orgID, ids, &now)
}

// restore takes a device out of the archive with the unmatched uplinks of
// its DevEUI.
func (s *deviceService) restore(ctx context.Context, orgID, id int64) error {
	d, err := s.get(ctx, orgID, id)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := newDeviceRepo(tx).setArchived(ctx, orgID, []int64{id}, nil); err != nil {
			return err
		}
		// The uplinks that arrived while the device was archived are
		// unmatched; the restored device takes them back.
		_, err := newUplinkRepo(tx).attach(ctx, orgID, id, d.DevEUI)
		return err
	})
}
