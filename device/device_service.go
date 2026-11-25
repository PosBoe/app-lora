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
	devices *deviceRepo
	clock   clock.Clock
}

func newDeviceService(pool *pgxpool.Pool, clk clock.Clock) *deviceService {
	return &deviceService{devices: newDeviceRepo(pool), clock: clk}
}

// create adds a device to the organization.
func (s *deviceService) create(ctx context.Context, orgID int64, f deviceFields) (*device, error) {
	f, err := f.normalize()
	if err != nil {
		return nil, err
	}
	id, err := s.devices.insert(ctx, orgID, f)
	if err != nil {
		return nil, s.writeError(ctx, orgID, f.DevEUI, err)
	}
	return s.devices.selectByID(ctx, orgID, id)
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

	err = s.devices.update(ctx, orgID, id, f, s.clock.Now())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errNotFound
	}
	if err != nil {
		return nil, s.writeError(ctx, orgID, f.DevEUI, err)
	}
	return s.devices.selectByID(ctx, orgID, id)
}

// hasUplinks reports whether a device has an uplink. The ingestion of
// uplinks does not exist yet, so no device has one.
func (s *deviceService) hasUplinks(context.Context, int64, int64) (bool, error) {
	return false, nil
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

// get returns a device of the organization, or errNotFound.
func (s *deviceService) get(ctx context.Context, orgID, id int64) (*device, error) {
	d, err := s.devices.selectByID(ctx, orgID, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errNotFound
	}
	return d, err
}

// list returns a page of devices and the number of devices on all pages.
func (s *deviceService) list(ctx context.Context, orgID int64, q listQuery) ([]*device, int, error) {
	return s.devices.list(ctx, orgID, q)
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

// restore takes a device out of the archive.
func (s *deviceService) restore(ctx context.Context, orgID, id int64) error {
	if _, err := s.get(ctx, orgID, id); err != nil {
		return err
	}
	_, err := s.devices.setArchived(ctx, orgID, []int64{id}, nil)
	return err
}
