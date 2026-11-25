package core

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/waneock/app-lora/pkg/pgdb"
)

// uniqueViolation is the PostgreSQL error code of a unique index conflict.
const uniqueViolation = "23505"

// user is a row of the users table.
type user struct {
	ID                int64
	OrganizationID    *int64
	IsAdmin           bool
	Login             string
	PasswordHash      string
	PasswordTemporary bool
	FailedLogins      int
	LockedUntil       *time.Time
	LastLoginAt       *time.Time
	FirstName         *string
	LastName          *string
	Email             *string
	Phone             *string
	Language          string
	CreatedAt         time.Time
}

// errDuplicateLogin is the repo's answer to a login that another user has.
var errDuplicateLogin = errors.New("core: duplicate login")

type userRepo struct {
	q pgdb.Querier
}

func newUserRepo(q pgdb.Querier) *userRepo {
	return &userRepo{q: q}
}

const userColumns = `id, organization_id, is_admin, login, password_hash, password_temporary,
	failed_logins, locked_until, last_login_at, first_name, last_name, email, phone,
	language, created_at`

func scanUser(row pgx.Row) (*user, error) {
	u := new(user)
	err := row.Scan(&u.ID, &u.OrganizationID, &u.IsAdmin, &u.Login, &u.PasswordHash,
		&u.PasswordTemporary, &u.FailedLogins, &u.LockedUntil, &u.LastLoginAt,
		&u.FirstName, &u.LastName, &u.Email, &u.Phone, &u.Language, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// selectByLogin returns pgx.ErrNoRows when no user has login.
func (r *userRepo) selectByLogin(ctx context.Context, login string) (*user, error) {
	return scanUser(r.q.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE login = $1`, login))
}

// selectByID returns pgx.ErrNoRows when no user has id.
func (r *userRepo) selectByID(ctx context.Context, id int64) (*user, error) {
	return scanUser(r.q.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id))
}

// insert writes u and sets u.ID and u.CreatedAt. It returns errDuplicateLogin
// when another user has the login.
func (r *userRepo) insert(ctx context.Context, u *user) error {
	err := r.q.QueryRow(ctx, `
		INSERT INTO users (organization_id, is_admin, login, password_hash, password_temporary)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, language`,
		u.OrganizationID, u.IsAdmin, u.Login, u.PasswordHash, u.PasswordTemporary,
	).Scan(&u.ID, &u.CreatedAt, &u.Language)
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == uniqueViolation {
		return errDuplicateLogin
	}
	return err
}

// recordFailedLogin counts one failed login. When the count reaches
// attempts, it locks the login until lockUntil and starts the count again.
func (r *userRepo) recordFailedLogin(ctx context.Context, id int64, attempts int, lockUntil time.Time) error {
	_, err := r.q.Exec(ctx, `
		UPDATE users SET
			failed_logins = CASE WHEN failed_logins + 1 >= $2 THEN 0 ELSE failed_logins + 1 END,
			locked_until = CASE WHEN failed_logins + 1 >= $2 THEN $3 ELSE locked_until END
		WHERE id = $1`, id, attempts, lockUntil)
	return err
}

// recordLogin clears the failed login count and sets the last login time.
func (r *userRepo) recordLogin(ctx context.Context, id int64, at time.Time) error {
	_, err := r.q.Exec(ctx, `
		UPDATE users SET failed_logins = 0, locked_until = NULL, last_login_at = $2
		WHERE id = $1`, id, at)
	return err
}

// setPassword sets the password hash and the temporary flag, and clears a
// lockout. It returns pgx.ErrNoRows when no user has id.
func (r *userRepo) setPassword(ctx context.Context, id int64, hash string, temporary bool) error {
	tag, err := r.q.Exec(ctx, `
		UPDATE users SET password_hash = $2, password_temporary = $3,
			failed_logins = 0, locked_until = NULL
		WHERE id = $1`, id, hash, temporary)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// listMembers returns the users of all organizations, ordered by id.
func (r *userRepo) listMembers(ctx context.Context) ([]*user, error) {
	rows, err := r.q.Query(ctx, `SELECT `+userColumns+` FROM users
		WHERE organization_id IS NOT NULL ORDER BY id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*user, error) {
		return scanUser(row)
	})
}
