# core

Organizations, users, the admin, logins, and sessions. `CONTEXT.md` defines the terms.

| File                      | Content                                                              |
| ------------------------- | -------------------------------------------------------------------- |
| `module.go`               | `Module`: provides the services and handlers and binds the routes.   |
| `auth_service.go`         | Login, session resolve and extension, logout, password change.       |
| `admin_service.go`        | Organizations and their users, as the admin manages them; `CreateAdmin`. |
| `auth_middleware.go`      | The session cookie, the access levels, and the request principal.    |
| `auth_handler.go`         | `/api/v1/auth/*`: login, logout, me, password.                       |
| `admin_handler.go`        | `/api/v1/admin/*`: list, create, deactivate, reactivate, reset password. |
| `passwords.go`            | bcrypt, the password rules, temporary passwords, the login rule.     |
| `errors.go`               | The rule errors and their HTTP status.                               |
| `*_repo.go`               | The SQL of the `organizations`, `users`, and `sessions` tables.      |

## Sessions

- A login makes a random 32-byte token. The cookie `lora_session` holds the token. The
  `sessions` table holds only its SHA-256, so a database read gives no usable token.
- The cookie is `HttpOnly` and `SameSite=Lax`. It is `Secure` when `site.url` starts with
  `https://`.
- A session lives for `auth.session_ttl` after its last use. A request moves the expiry only
  when it moves by one hour or more, so a busy session writes at most once an hour. The
  middleware then sends the cookie again with a new `Max-Age`.
- A deactivation, a password reset, and a password change end sessions by deleting their rows.
  A deleted row makes the next request answer 401.

## Access levels

`authMiddleware.wrap` takes one level for each route:

- `accessPublic`: no session.
- `accessTemporary`: a session, also with a temporary password. Only `me`, `password`, and
  `logout` use it.
- `accessUser`: a session with a password the user chose.
- `accessAdmin`: as `accessUser`, and the user is an admin.

## Login rules

- A login is stored in lower case. A login request ignores case and outer spaces.
- An unknown login and a wrong password give the same answer and take the same time.
- After `auth.lockout_attempts` failed logins in a row, the login is locked for
  `auth.lockout_duration`. A locked login answers `login_locked` before the password check.
- A member of a deactivated organization gets `organization_deactivated` only after a right
  password.

## Errors

A service returns a `ruleError`. A handler returns it; `wrap` maps it to its status in
`ruleStatus` and writes `{"code", "message"}`. The frontend matches on `code`.

## Tests

The tests use the test database through `pkg/pgtest`, so they skip under `-short`. Run them
with `task test:full`, or with `LORA_CONFIG` set to the absolute path of `config/test.yml`.
