package core

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waneock/app-lora/pkg/appconf"
	"github.com/waneock/app-lora/pkg/clock"
)

// sessionExtendStep is the smallest step by which a request moves a session
// expiry. It bounds the session writes to one an hour for each session.
const sessionExtendStep = time.Hour

// sessionTokenBytes is the size of the random session token.
const sessionTokenBytes = 32

// currentUser is the logged-in user as the frontend sees it.
type currentUser struct {
	ID                int64
	Login             string
	IsAdmin           bool
	PasswordTemporary bool
	Language          string
	Organization      *organizationRef
}

// organizationRef names the organization of a user.
type organizationRef struct {
	ID   int64
	Name string
}

// principal is the user of a request, as the auth middleware resolves it.
type principal struct {
	currentUser
	// tokenHash identifies the session of the request.
	tokenHash []byte
}

// authService logs users in and out, resolves sessions, and changes
// passwords.
type authService struct {
	pool      *pgxpool.Pool
	users     *userRepo
	sessions  *sessionRepo
	passwords *passwords
	clock     clock.Clock
	conf      appconf.AuthConfig
}

func newAuthService(pool *pgxpool.Pool, pw *passwords, clk clock.Clock, conf *appconf.Config) *authService {
	return &authService{
		pool:      pool,
		users:     newUserRepo(pool),
		sessions:  newSessionRepo(pool),
		passwords: pw,
		clock:     clk,
		conf:      conf.Auth,
	}
}

// login checks a login and a password and starts a session. It returns the
// session token for the cookie.
//
// An unknown login and a wrong password give the same error. A locked login
// gives errLoginLocked before the password is checked, so a locked login
// tells nothing about the password. A deactivated organization is reported
// only after a right password.
func (s *authService) login(ctx context.Context, login, password string) (string, *principal, error) {
	now := s.clock.Now()

	u, err := s.lookup(ctx, login)
	if err != nil {
		return "", nil, err
	}
	if u == nil {
		s.passwords.matches("", password)
		return "", nil, errInvalidCredentials
	}
	if u.LockedUntil != nil && u.LockedUntil.After(now) {
		s.passwords.matches("", password)
		return "", nil, errLoginLocked
	}
	if !s.passwords.matches(u.PasswordHash, password) {
		lockUntil := now.Add(s.conf.LockoutDuration)
		if err := s.users.recordFailedLogin(ctx, u.ID, s.conf.LockoutAttempts, lockUntil); err != nil {
			return "", nil, err
		}
		return "", nil, errInvalidCredentials
	}

	ref, err := s.organizationOf(ctx, u)
	if err != nil {
		return "", nil, err
	}

	if err := s.users.recordLogin(ctx, u.ID, now); err != nil {
		return "", nil, err
	}
	if err := s.sessions.deleteExpiredByUser(ctx, u.ID, now); err != nil {
		return "", nil, err
	}
	token, hash := newSessionToken()
	if err := s.sessions.insert(ctx, hash, u.ID, now, now.Add(s.conf.SessionTTL)); err != nil {
		return "", nil, err
	}

	p := &principal{
		currentUser: currentUser{
			ID:                u.ID,
			Login:             u.Login,
			IsAdmin:           u.IsAdmin,
			PasswordTemporary: u.PasswordTemporary,
			Language:          u.Language,
			Organization:      ref,
		},
		tokenHash: hash,
	}
	return token, p, nil
}

// lookup returns the user with login, or nil when there is none. A login
// that breaks the login rules has no user.
func (s *authService) lookup(ctx context.Context, login string) (*user, error) {
	login, err := normalizeLogin(login)
	if err != nil {
		return nil, nil
	}
	u, err := s.users.selectByLogin(ctx, login)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

// organizationOf returns the organization of a member, nil for an admin, and
// errOrganizationDeactivated for a member of a deactivated organization.
func (s *authService) organizationOf(ctx context.Context, u *user) (*organizationRef, error) {
	if u.OrganizationID == nil {
		return nil, nil
	}
	org, err := newOrganizationRepo(s.pool).selectByID(ctx, *u.OrganizationID)
	if err != nil {
		return nil, err
	}
	if org.DeactivatedAt != nil {
		return nil, errOrganizationDeactivated
	}
	return &organizationRef{ID: org.ID, Name: org.Name}, nil
}

// resolve returns the principal of a session token. It moves the session
// expiry to now + TTL when the expiry is more than one step behind, and then
// reports extended. A missing, expired, or unknown token gives
// errUnauthenticated. A session of a deactivated organization gives
// errOrganizationDeactivated.
func (s *authService) resolve(ctx context.Context, token string) (p *principal, extended bool, _ error) {
	hash, ok := hashSessionToken(token)
	if !ok {
		return nil, false, errUnauthenticated
	}
	now := s.clock.Now()

	su, err := s.sessions.selectLive(ctx, hash, now)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, errUnauthenticated
	}
	if err != nil {
		return nil, false, err
	}
	if su.OrganizationDeactivated {
		return nil, false, errOrganizationDeactivated
	}

	next := now.Add(s.conf.SessionTTL)
	if next.Sub(su.ExpiresAt) >= sessionExtendStep {
		if err := s.sessions.extend(ctx, hash, next); err != nil {
			return nil, false, err
		}
		extended = true
	}

	p = &principal{
		currentUser: currentUser{
			ID:                su.UserID,
			Login:             su.Login,
			IsAdmin:           su.IsAdmin,
			PasswordTemporary: su.PasswordTemporary,
			Language:          su.Language,
		},
		tokenHash: hash,
	}
	if su.OrganizationID != nil {
		p.Organization = &organizationRef{ID: *su.OrganizationID, Name: *su.OrganizationName}
	}
	return p, extended, nil
}

// logout ends the session of p.
func (s *authService) logout(ctx context.Context, p *principal) error {
	return s.sessions.delete(ctx, p.tokenHash)
}

// changePassword replaces the password of p. It clears the temporary flag
// and ends the other sessions of the user.
func (s *authService) changePassword(ctx context.Context, p *principal, current, next string) error {
	u, err := s.users.selectByID(ctx, p.ID)
	if err != nil {
		return err
	}
	if !s.passwords.matches(u.PasswordHash, current) {
		return errWrongCurrentPassword
	}
	hash, err := s.passwords.hash(next)
	if err != nil {
		return err
	}

	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := newUserRepo(tx).setPassword(ctx, u.ID, hash, false); err != nil {
			return err
		}
		return newSessionRepo(tx).deleteByUser(ctx, u.ID, p.tokenHash)
	})
}

// newSessionToken returns a random token for the cookie and its hash for
// the database.
func newSessionToken() (token string, hash []byte) {
	buf := make([]byte, sessionTokenBytes)
	_, _ = rand.Read(buf)
	token = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256(buf)
	return token, sum[:]
}

// hashSessionToken returns the hash of a cookie token, and false for a token
// that newSessionToken cannot have made.
func hashSessionToken(token string) ([]byte, bool) {
	buf, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(buf) != sessionTokenBytes {
		return nil, false
	}
	sum := sha256.Sum256(buf)
	return sum[:], true
}
