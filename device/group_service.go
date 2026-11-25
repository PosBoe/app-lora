package device

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxGroupChars = 100

// groupService holds the rules of the device groups of an organization.
type groupService struct {
	groups *groupRepo
}

func newGroupService(pool *pgxpool.Pool) *groupService {
	return &groupService{groups: newGroupRepo(pool)}
}

// normalizeGroupName trims a group name and checks the rules.
func normalizeGroupName(name string) (string, error) {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return "", errGroupNameRequired
	case utf8.RuneCountInString(name) > maxGroupChars:
		return "", errGroupNameTooLong
	}
	return name, nil
}

func (s *groupService) create(ctx context.Context, orgID int64, name string) (*deviceGroup, error) {
	name, err := normalizeGroupName(name)
	if err != nil {
		return nil, err
	}
	g, err := s.groups.insert(ctx, orgID, name)
	if errors.Is(err, errDuplicate) {
		return nil, errGroupTaken
	}
	return g, err
}

func (s *groupService) list(ctx context.Context, orgID int64) ([]*deviceGroup, error) {
	return s.groups.list(ctx, orgID)
}

func (s *groupService) rename(ctx context.Context, orgID, id int64, name string) error {
	name, err := normalizeGroupName(name)
	if err != nil {
		return err
	}
	err = s.groups.rename(ctx, orgID, id, name)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return errGroupNotFound
	case errors.Is(err, errDuplicate):
		return errGroupTaken
	}
	return err
}

// delete removes a group. Its devices stay.
func (s *groupService) delete(ctx context.Context, orgID, id int64) error {
	err := s.groups.delete(ctx, orgID, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return errGroupNotFound
	}
	return err
}

// add puts devices in a group and returns how many it added. Ids of other
// organizations are left alone.
func (s *groupService) add(ctx context.Context, orgID, groupID int64, deviceIDs []int64) (int64, error) {
	if err := s.check(ctx, orgID, groupID, deviceIDs); err != nil {
		return 0, err
	}
	n, err := s.groups.add(ctx, orgID, groupID, deviceIDs)
	if errors.Is(err, errReferenced) {
		// The group was deleted after the check.
		return 0, errGroupNotFound
	}
	return n, err
}

// remove takes devices out of a group and returns how many it removed.
func (s *groupService) remove(ctx context.Context, orgID, groupID int64, deviceIDs []int64) (int64, error) {
	if err := s.check(ctx, orgID, groupID, deviceIDs); err != nil {
		return 0, err
	}
	return s.groups.remove(ctx, orgID, groupID, deviceIDs)
}

// check refuses too many ids and a group that is not in the organization. A
// bulk change that matches no device must still answer 404 for such a group.
func (s *groupService) check(ctx context.Context, orgID, groupID int64, deviceIDs []int64) error {
	if len(deviceIDs) > maxBulkIDs {
		return errTooManyIDs
	}
	ok, err := s.groups.exists(ctx, orgID, groupID)
	if err != nil {
		return err
	}
	if !ok {
		return errGroupNotFound
	}
	return nil
}
