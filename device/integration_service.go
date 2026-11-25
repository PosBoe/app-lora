package device

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waneock/app-lora/pkg/clock"
)

const (
	maxIntegrationNameChars = 100
	// secretPrefix marks an integration secret, so a user can tell it from
	// other keys in the settings of a network server.
	secretPrefix = "lora_"
	secretBytes  = 32
	// recentWindow is the period of the uplink count on the status of an
	// integration.
	recentWindow = 24 * time.Hour
)

// integrationService holds the integrations of an organization.
type integrationService struct {
	integrations *integrationRepo
	clock        clock.Clock
}

func newIntegrationService(pool *pgxpool.Pool, clk clock.Clock) *integrationService {
	return &integrationService{integrations: newIntegrationRepo(pool), clock: clk}
}

func normalizeIntegrationName(name string) (string, error) {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return "", errIntegrationNameRequired
	case utf8.RuneCountInString(name) > maxIntegrationNameChars:
		return "", errIntegrationNameTooLong
	}
	return name, nil
}

// create adds an integration and returns its secret. The secret is shown
// this one time; only its hash is stored.
func (s *integrationService) create(ctx context.Context, orgID int64, name, kind string) (*integration, string, error) {
	name, err := normalizeIntegrationName(name)
	if err != nil {
		return nil, "", err
	}
	if kind != kindChirpStack && kind != kindTTN {
		return nil, "", errIntegrationKindInvalid
	}
	secret, hash := newSecret()
	in, err := s.integrations.insert(ctx, orgID, name, kind, hash)
	if err != nil {
		return nil, "", err
	}
	return in, secret, nil
}

func (s *integrationService) list(ctx context.Context, orgID int64) ([]*integration, error) {
	return s.integrations.list(ctx, orgID, s.clock.Now().Add(-recentWindow))
}

func (s *integrationService) rename(ctx context.Context, orgID, id int64, name string) error {
	name, err := normalizeIntegrationName(name)
	if err != nil {
		return err
	}
	return notFoundAs(s.integrations.rename(ctx, orgID, id, name), errIntegrationNotFound)
}

// regenerate gives an integration a new secret. The old secret stops at
// once, so uplinks are refused until the network server has the new one.
func (s *integrationService) regenerate(ctx context.Context, orgID, id int64) (string, error) {
	secret, hash := newSecret()
	if err := notFoundAs(s.integrations.setSecret(ctx, orgID, id, hash), errIntegrationNotFound); err != nil {
		return "", err
	}
	return secret, nil
}

// delete removes an integration. Its uplinks stay.
func (s *integrationService) delete(ctx context.Context, orgID, id int64) error {
	return notFoundAs(s.integrations.delete(ctx, orgID, id), errIntegrationNotFound)
}

// newSecret returns a random secret and its hash.
func newSecret() (string, []byte) {
	buf := make([]byte, secretBytes)
	_, _ = rand.Read(buf)
	secret := secretPrefix + base64.RawURLEncoding.EncodeToString(buf)
	return secret, hashSecret(secret)
}

// hashSecret returns the hash under which a secret is stored.
func hashSecret(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

// notFoundAs turns pgx.ErrNoRows into notFound.
func notFoundAs(err, notFound error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound
	}
	return err
}
