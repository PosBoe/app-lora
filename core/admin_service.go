package core

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waneock/app-lora/pkg/clock"
)

// organizationMembers is an organization with its users.
type organizationMembers struct {
	organization *organization
	users        []*user
}

// adminService holds what the admin does to organizations and their users.
type adminService struct {
	pool      *pgxpool.Pool
	passwords *passwords
	clock     clock.Clock
}

func newAdminService(pool *pgxpool.Pool, pw *passwords, clk clock.Clock) *adminService {
	return &adminService{pool: pool, passwords: pw, clock: clk}
}

// createOrganization creates an organization and its first user, in one
// transaction. It returns the temporary password of the user.
func (s *adminService) createOrganization(ctx context.Context, name, login string) (*organizationMembers, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, "", errNameRequired
	}
	login, err := normalizeLogin(login)
	if err != nil {
		return nil, "", err
	}
	password := newTemporaryPassword()
	hash, err := s.passwords.hash(password)
	if err != nil {
		return nil, "", err
	}

	out := new(organizationMembers)
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		org, err := newOrganizationRepo(tx).insert(ctx, name)
		if err != nil {
			return err
		}
		u := &user{OrganizationID: &org.ID, Login: login, PasswordHash: hash, PasswordTemporary: true}
		if err := newUserRepo(tx).insert(ctx, u); err != nil {
			return err
		}
		out.organization = org
		out.users = []*user{u}
		return nil
	})
	if errors.Is(err, errDuplicateLogin) {
		return nil, "", errLoginTaken
	}
	if err != nil {
		return nil, "", err
	}
	return out, password, nil
}

// resetPassword gives a member of an organization a new temporary password,
// clears a lockout, and ends the sessions of the user. The admin of the
// platform is not a member, so this cannot reset an admin.
func (s *adminService) resetPassword(ctx context.Context, userID int64) (string, error) {
	u, err := newUserRepo(s.pool).selectByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && u.IsAdmin) {
		return "", errNotFound
	}
	if err != nil {
		return "", err
	}
	password := newTemporaryPassword()
	hash, err := s.passwords.hash(password)
	if err != nil {
		return "", err
	}

	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := newUserRepo(tx).setPassword(ctx, u.ID, hash, true); err != nil {
			return err
		}
		return newSessionRepo(tx).deleteByUser(ctx, u.ID, nil)
	})
	if err != nil {
		return "", err
	}
	return password, nil
}

// deactivate stops the logins of an organization and ends the sessions of
// its users. Its integrations still deliver uplinks.
func (s *adminService) deactivate(ctx context.Context, organizationID int64) error {
	now := s.clock.Now()
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := newOrganizationRepo(tx).setDeactivatedAt(ctx, organizationID, &now); err != nil {
			return err
		}
		return newSessionRepo(tx).deleteByOrganization(ctx, organizationID)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return errNotFound
	}
	return err
}

// reactivate lets the users of an organization log in again.
func (s *adminService) reactivate(ctx context.Context, organizationID int64) error {
	err := newOrganizationRepo(s.pool).setDeactivatedAt(ctx, organizationID, nil)
	if errors.Is(err, pgx.ErrNoRows) {
		return errNotFound
	}
	return err
}

// listOrganizations returns all organizations, the newest first, each with
// its users.
func (s *adminService) listOrganizations(ctx context.Context) ([]*organizationMembers, error) {
	orgs, err := newOrganizationRepo(s.pool).list(ctx)
	if err != nil {
		return nil, err
	}
	members, err := newUserRepo(s.pool).listMembers(ctx)
	if err != nil {
		return nil, err
	}

	byOrg := make(map[int64][]*user, len(orgs))
	for _, u := range members {
		byOrg[*u.OrganizationID] = append(byOrg[*u.OrganizationID], u)
	}
	out := make([]*organizationMembers, len(orgs))
	for i, org := range orgs {
		out[i] = &organizationMembers{organization: org, users: byOrg[org.ID]}
	}
	return out, nil
}

// CreateAdmin creates an admin of the platform with login and returns the
// temporary password. The `lora admin create` command calls it; the UI
// creates no admins.
func CreateAdmin(ctx context.Context, pool *pgxpool.Pool, login string) (string, error) {
	login, err := normalizeLogin(login)
	if err != nil {
		return "", err
	}
	pw, err := newPasswords()
	if err != nil {
		return "", err
	}
	password := newTemporaryPassword()
	hash, err := pw.hash(password)
	if err != nil {
		return "", err
	}

	u := &user{IsAdmin: true, Login: login, PasswordHash: hash, PasswordTemporary: true}
	err = newUserRepo(pool).insert(ctx, u)
	if errors.Is(err, errDuplicateLogin) {
		return "", errLoginTaken
	}
	if err != nil {
		return "", err
	}
	return password, nil
}
