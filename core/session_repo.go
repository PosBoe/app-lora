package core

import (
	"context"
	"time"

	"github.com/waneock/app-lora/pkg/pgdb"
)

// sessionUser is a live session with its user and organization.
type sessionUser struct {
	UserID                  int64
	Login                   string
	IsAdmin                 bool
	PasswordTemporary       bool
	Language                string
	OrganizationID          *int64
	OrganizationName        *string
	OrganizationDeactivated bool
	ExpiresAt               time.Time
}

type sessionRepo struct {
	q pgdb.Querier
}

func newSessionRepo(q pgdb.Querier) *sessionRepo {
	return &sessionRepo{q: q}
}

// insert stores a session by the hash of its token.
func (r *sessionRepo) insert(ctx context.Context, tokenHash []byte, userID int64, createdAt, expiresAt time.Time) error {
	_, err := r.q.Exec(ctx, `
		INSERT INTO sessions (token_hash, user_id, created_at, expires_at)
		VALUES ($1, $2, $3, $4)`, tokenHash, userID, createdAt, expiresAt)
	return err
}

// selectLive returns the session with tokenHash that expires after now. It
// returns pgx.ErrNoRows when there is none.
func (r *sessionRepo) selectLive(ctx context.Context, tokenHash []byte, now time.Time) (*sessionUser, error) {
	s := new(sessionUser)
	err := r.q.QueryRow(ctx, `
		SELECT u.id, u.login, u.is_admin, u.password_temporary, u.language,
			o.id, o.name, o.deactivated_at IS NOT NULL, s.expires_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		LEFT JOIN organizations o ON o.id = u.organization_id
		WHERE s.token_hash = $1 AND s.expires_at > $2`, tokenHash, now,
	).Scan(&s.UserID, &s.Login, &s.IsAdmin, &s.PasswordTemporary, &s.Language,
		&s.OrganizationID, &s.OrganizationName, &s.OrganizationDeactivated, &s.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// extend sets a new expiry time on a session.
func (r *sessionRepo) extend(ctx context.Context, tokenHash []byte, expiresAt time.Time) error {
	_, err := r.q.Exec(ctx, `UPDATE sessions SET expires_at = $2 WHERE token_hash = $1`,
		tokenHash, expiresAt)
	return err
}

// delete removes one session.
func (r *sessionRepo) delete(ctx context.Context, tokenHash []byte) error {
	_, err := r.q.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

// deleteByUser removes the sessions of a user, except the session with
// keepHash. A nil keepHash keeps none.
func (r *sessionRepo) deleteByUser(ctx context.Context, userID int64, keepHash []byte) error {
	_, err := r.q.Exec(ctx, `
		DELETE FROM sessions
		WHERE user_id = $1 AND ($2::bytea IS NULL OR token_hash <> $2)`, userID, keepHash)
	return err
}

// deleteExpiredByUser removes the sessions of a user that expired at or
// before now.
func (r *sessionRepo) deleteExpiredByUser(ctx context.Context, userID int64, now time.Time) error {
	_, err := r.q.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1 AND expires_at <= $2`,
		userID, now)
	return err
}

// deleteByOrganization removes the sessions of all users of an
// organization.
func (r *sessionRepo) deleteByOrganization(ctx context.Context, organizationID int64) error {
	_, err := r.q.Exec(ctx, `
		DELETE FROM sessions
		WHERE user_id IN (SELECT id FROM users WHERE organization_id = $1)`, organizationID)
	return err
}
