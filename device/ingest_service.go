package device

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waneock/app-lora/pkg/clock"
)

// ingestService stores the uplinks that the integrations deliver.
type ingestService struct {
	integrations *integrationRepo
	uplinks      *uplinkRepo
	clock        clock.Clock
}

func newIngestService(pool *pgxpool.Pool, clk clock.Clock) *ingestService {
	return &ingestService{
		integrations: newIntegrationRepo(pool),
		uplinks:      newUplinkRepo(pool),
		clock:        clk,
	}
}

// authenticate returns the integration of a secret. An unknown secret gives
// errInvalidSecret and leaves no trace: it names no organization.
func (s *ingestService) authenticate(ctx context.Context, secret string) (*integrationRef, error) {
	if secret == "" {
		return nil, errInvalidSecret
	}
	ref, err := s.integrations.bySecret(ctx, hashSecret(secret))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errInvalidSecret
	}
	return ref, err
}

// store keeps an uplink in the organization of its integration (device:D1),
// on the live device with its DevEUI or as an unmatched uplink. An uplink
// with no time of its own gets the arrival time.
func (s *ingestService) store(ctx context.Context, ref *integrationRef, msg *uplinkMessage) error {
	now := s.clock.Now()
	receivedAt := now
	if msg.ReceivedAt != nil {
		receivedAt = *msg.ReceivedAt
	}
	if _, err := s.uplinks.insert(ctx, ref, msg, receivedAt, now); err != nil {
		return err
	}
	return s.integrations.recordUplink(ctx, ref.ID, now)
}

// fail keeps the reason of a request that ingestion could not read on the
// integration, so its page tells the user what to fix.
func (s *ingestService) fail(ctx context.Context, ref *integrationRef, rerr *ruleError) error {
	return s.integrations.recordError(ctx, ref.ID, s.clock.Now(), rerr.code, rerr.message)
}
