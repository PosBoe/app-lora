package device

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// modelService holds the rules of the device models of an organization.
type modelService struct {
	models *modelRepo
}

func newModelService(pool *pgxpool.Pool) *modelService {
	return &modelService{models: newModelRepo(pool)}
}

// create adds a device model to the organization.
func (s *modelService) create(ctx context.Context, orgID int64, vendor, model string) (*deviceModel, error) {
	vendor, model, err := normalizeModel(vendor, model)
	if err != nil {
		return nil, err
	}
	m, err := s.models.insert(ctx, orgID, vendor, model)
	if errors.Is(err, errDuplicate) {
		return nil, errModelTaken
	}
	return m, err
}

// list returns the device models of the organization.
func (s *modelService) list(ctx context.Context, orgID int64) ([]*deviceModel, error) {
	return s.models.list(ctx, orgID)
}

// rename sets the vendor and the model name of a device model.
func (s *modelService) rename(ctx context.Context, orgID, id int64, vendor, model string) error {
	vendor, model, err := normalizeModel(vendor, model)
	if err != nil {
		return err
	}
	err = s.models.rename(ctx, orgID, id, vendor, model)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return errNotFound
	case errors.Is(err, errDuplicate):
		return errModelTaken
	}
	return err
}

// delete removes a device model that no device uses, archived devices too.
func (s *modelService) delete(ctx context.Context, orgID, id int64) error {
	err := s.models.delete(ctx, orgID, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return errNotFound
	case errors.Is(err, errReferenced):
		return errModelInUse
	}
	return err
}
